// Package main Symphony服务入口
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dministrator/symphony/internal/cli"
	"github.com/dministrator/symphony/internal/common/errors"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/server"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/radovskyb/watcher"
)

var (
	workflowPath string
	port         int
)

func init() {
	flag.StringVar(&workflowPath, "workflow", "", "路径到WORKFLOW.md文件")
	flag.StringVar(&workflowPath, "w", "", "路径到WORKFLOW.md文件 (简写)")
	flag.IntVar(&port, "port", 0, "HTTP服务器端口 (0表示禁用)")
}

func main() {
	// 检查是否是子命令模式
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			runInitCommand()
			return
		case "start":
			// 移除 "start" 参数后继续执行
			os.Args = append(os.Args[:1], os.Args[2:]...)
		case "help", "-h", "--help":
			printUsage()
			return
		case "version", "-v", "--version":
			printVersion()
			return
		}
	}

	flag.Parse()

	// 确定工作流文件路径
	if workflowPath == "" {
		// 检查默认的 .sym/workflow.md
		if _, err := os.Stat(".sym/workflow.md"); err == nil {
			workflowPath = ".sym/workflow.md"
		} else {
			workflowPath = "./WORKFLOW.md"
		}
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法解析工作流路径: %v\n", err)
		os.Exit(1)
	}
	workflowPath = absPath

	// 加载工作流
	loader := workflow.NewLoader(workflowPath)
	def, err := loader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 加载工作流失败: %v\n", err)
		os.Exit(1)
	}

	// 解析配置
	cfg, err := def.ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %s\n", errors.WrapError("config", "parse_failed", "解析配置失败", err))
		os.Exit(1)
	}

	// 验证配置（基础调度配置）
	validation := cfg.ValidateDispatchConfig()
	if !validation.Valid {
		fmt.Fprintf(os.Stderr, "%s:\n", errors.ErrConfigInvalid.Error())
		for _, e := range validation.Errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	// 验证 Symphony 特定配置（Agent CLI、prompt 文件等）
	symphonyValidation := cfg.ValidateSymphonyConfig()
	if !symphonyValidation.Valid {
		fmt.Fprintf(os.Stderr, "%s:\n", errors.ErrConfigInvalid.Error())
		for _, e := range symphonyValidation.Errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Println("  Symphony - 编码代理编排服务")
	fmt.Println("========================================")
	fmt.Printf("工作流文件: %s\n", workflowPath)
	fmt.Printf("跟踪器: %s\n", cfg.Tracker.Kind)
	fmt.Printf("工作空间根目录: %s\n", cfg.Workspace.Root)
	fmt.Printf("轮询间隔: %dms\n", cfg.Polling.IntervalMs)
	fmt.Printf("最大并发: %d\n", cfg.Agent.MaxConcurrentAgents)
	if port > 0 || (cfg.Server != nil && cfg.Server.Port > 0) {
		httpPort := port
		if httpPort == 0 && cfg.Server != nil {
			httpPort = cfg.Server.Port
		}
		fmt.Printf("HTTP端口: %d\n", httpPort)
	}
	fmt.Println("========================================")

	// 检查 Tracker 可用性
	fmt.Println("检查 Tracker 可用性...")
	trackerClient := tracker.NewTracker(cfg)
	if err := trackerClient.CheckAvailability(); err != nil {
		fmt.Fprintf(os.Stderr, "Tracker 不可用: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Tracker %s 可用\n", cfg.Tracker.Kind)

	// 创建编排器
	orch := orchestrator.New(cfg, def.PromptTemplate)

	// 创建恢复管理器并执行任务恢复
	recoveryMgr := orchestrator.NewRecoveryManager(cfg, trackerClient, orch)
	fmt.Println("检查需要恢复的任务...")
	recoveredTasks, err := recoveryMgr.RestoreAll(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "任务恢复检查失败: %v\n", err)
		// 不退出，继续启动服务
	}
	if len(recoveredTasks) > 0 {
		fmt.Printf("发现 %d 个需要恢复的任务\n", len(recoveredTasks))
		// 执行恢复动作
		for _, task := range recoveredTasks {
			if execErr := recoveryMgr.ExecuteRecovery(context.Background(), task); execErr != nil {
				fmt.Fprintf(os.Stderr, "恢复任务 %s 失败: %v\n", task.Issue.Identifier, execErr)
			}
		}
	}

	// 设置工作流文件监视
	go watchWorkflow(workflowPath, orch, cfg)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// HTTP 服务器引用（用于优雅关闭）
	var srv *server.Server

	// 处理信号 - 实现优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		fmt.Printf("\n接收到信号 %v，开始优雅关闭...\n", sig)

		// 创建关闭超时上下文（30秒）
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// 1. 关闭编排器
		if err := orch.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "编排器关闭错误: %v\n", err)
		}

		// 2. 关闭 HTTP 服务器
		if srv != nil {
			if err := srv.Shutdown(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "HTTP服务器关闭错误: %v\n", err)
			}
		}

		// 3. 取消主上下文
		cancel()
	}()

	// 启动HTTP服务器（如果配置了）
	if port > 0 || (cfg.Server != nil && cfg.Server.Port > 0) {
		httpPort := port
		if httpPort == 0 && cfg.Server != nil {
			httpPort = cfg.Server.Port
		}
		srv = server.NewServer(orch, httpPort)
		go func() {
			fmt.Printf("HTTP服务器启动在端口 %d\n", httpPort)
			fmt.Printf("访问 http://localhost:%d 查看 Web 看板\n", httpPort)
			if err := srv.Run(); err != nil {
				// 服务器关闭时可能会返回正常错误，不报告
				if orch.IsShuttingDown() {
					return
				}
				fmt.Fprintf(os.Stderr, "HTTP服务器错误: %v\n", err)
			}
		}()
	}

	// 运行编排器
	fmt.Println("启动编排器...")
	if err := orch.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "编排器错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("服务已停止")
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("Symphony - 编码代理编排服务")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  symphony [命令] [选项]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  init        初始化项目配置（交互式向导）")
	fmt.Println("  start       启动编排服务（默认命令）")
	fmt.Println("  help        显示帮助信息")
	fmt.Println("  version     显示版本信息")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -w, --workflow <path>  工作流文件路径（默认: ./WORKFLOW.md 或 .sym/workflow.md）")
	fmt.Println("  --port <port>          HTTP服务器端口（0表示禁用）")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  symphony init                    # 交互式初始化")
	fmt.Println("  symphony init --tracker mock     # 指定 tracker 类型")
	fmt.Println("  symphony start                   # 使用默认配置启动")
	fmt.Println("  symphony start -w workflow.md    # 使用指定工作流文件启动")
	fmt.Println("  symphony start --port 8080       # 启动并开启 HTTP 服务")
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Println("Symphony v1.0.0")
}

// runInitCommand 执行 init 命令
func runInitCommand() {
	// 解析 init 子命令的参数
	initFlags := flag.NewFlagSet("init", flag.ExitOnError)
	trackerType := initFlags.String("tracker", "", "Tracker 类型 (linear, github, mock, beads)")
	agentType := initFlags.String("agent", "", "AI Agent CLI 类型 (codex, claude, opencode)")
	projectPath := initFlags.String("path", "", "项目路径（默认: 当前目录）")
	nonInteractive := initFlags.Bool("non-interactive", false, "非交互模式")

	initFlags.Parse(os.Args[2:])

	opts := cli.InitOptions{
		TrackerType:    *trackerType,
		AgentType:      *agentType,
		ProjectPath:    *projectPath,
		NonInteractive: *nonInteractive,
	}

	cmd := cli.NewInitCommand(opts)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}
}

// watchWorkflow 监视工作流文件变更
func watchWorkflow(path string, orch *orchestrator.Orchestrator, currentCfg *config.Config) {
	w := watcher.New()
	w.SetMaxEvents(1)

	if err := w.Add(path); err != nil {
		fmt.Fprintf(os.Stderr, "无法监视工作流文件: %v\n", err)
		return
	}

	go func() {
		for {
			select {
			case <-w.Event:
				fmt.Println("检测到工作流文件变更，重新加载...")

				loader := workflow.NewLoader(path)
				def, err := loader.Load()
				if err != nil {
					fmt.Fprintf(os.Stderr, "工作流加载错误: %v\n", err)
					continue
				}

				cfg, err := def.ParseConfig()
				if err != nil {
					fmt.Fprintf(os.Stderr, "配置解析错误: %v\n", err)
					continue
				}

				// 验证基础调度配置
				validation := cfg.ValidateDispatchConfig()
				if !validation.Valid {
					fmt.Fprintf(os.Stderr, "%s:\n", errors.ErrConfigInvalid.Error())
					for _, e := range validation.Errors {
						fmt.Fprintf(os.Stderr, "  - %s\n", e)
					}
					continue
				}

				// 验证 Symphony 特定配置
				symphonyValidation := cfg.ValidateSymphonyConfig()
				if !symphonyValidation.Valid {
					fmt.Fprintf(os.Stderr, "%s:\n", errors.ErrConfigInvalid.Error())
					for _, e := range symphonyValidation.Errors {
						fmt.Fprintf(os.Stderr, "  - %s\n", e)
					}
					continue
				}

				// 更新编排器
				orch.UpdateConfig(cfg, def.PromptTemplate)
				fmt.Println("工作流已重新加载")

			case <-w.Closed:
				return
			}
		}
	}()

	// 每100ms检查一次
	if err := w.Start(100 * time.Millisecond); err != nil {
		fmt.Fprintf(os.Stderr, "文件监视错误: %v\n", err)
	}
}

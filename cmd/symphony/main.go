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

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/server"
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
	flag.Parse()

	// 确定工作流文件路径
	if workflowPath == "" {
		workflowPath = "./WORKFLOW.md"
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
		fmt.Fprintf(os.Stderr, "错误: 解析配置失败: %v\n", err)
		os.Exit(1)
	}

	// 验证配置
	validation := cfg.ValidateDispatchConfig()
	if !validation.Valid {
		fmt.Fprintf(os.Stderr, "配置验证失败:\n")
		for _, e := range validation.Errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Println("  Symphony - 编码代理编排服务")
	fmt.Println("========================================")
	fmt.Printf("工作流文件: %s\n", workflowPath)
	fmt.Printf("跟踪器: %s (项目: %s)\n", cfg.Tracker.Kind, cfg.Tracker.ProjectSlug)
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

	// 创建编排器
	orch := orchestrator.New(cfg, def.PromptTemplate)

	// 设置工作流文件监视
	go watchWorkflow(workflowPath, orch, cfg)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n接收到终止信号，正在关闭...")
		cancel()
	}()

	// 启动HTTP服务器（如果配置了）
	if port > 0 || (cfg.Server != nil && cfg.Server.Port > 0) {
		httpPort := port
		if httpPort == 0 && cfg.Server != nil {
			httpPort = cfg.Server.Port
		}
		srv := server.NewServer(orch, httpPort)
		go func() {
			fmt.Printf("HTTP服务器启动在端口 %d\n", httpPort)
			if err := srv.Run(); err != nil {
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

				// 验证配置
				validation := cfg.ValidateDispatchConfig()
				if !validation.Valid {
					fmt.Fprintf(os.Stderr, "配置验证失败: %v\n", validation.Errors)
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
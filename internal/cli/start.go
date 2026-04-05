// Package cli 提供命令行界面功能
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dministrator/symphony/internal/common/errors"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/logging"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/server"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/radovskyb/watcher"
)

// StartOptions 包含 start 命令的选项
type StartOptions struct {
	// WorkflowPath 工作流文件路径
	WorkflowPath string
	// Port HTTP 服务器端口 (0 表示禁用)
	Port int
	// ConfigPath 配置文件路径 (默认 .sym/config.yaml)
	ConfigPath string
}

// StartCommand 实现 symphony start 命令
type StartCommand struct {
	options StartOptions
}

// NewStartCommand 创建新的 start 命令
func NewStartCommand(opts StartOptions) *StartCommand {
	return &StartCommand{
		options: opts,
	}
}

// RunResult 包含运行结果
type RunResult struct {
	// Config 加载的配置
	Config *config.Config
	// WorkflowPath 工作流文件路径
	WorkflowPath string
}

// Run 执行 start 命令
func (c *StartCommand) Run(ctx context.Context) error {
	// 1. 确定工作流文件路径
	workflowPath := c.resolveWorkflowPath()

	// 转换为绝对路径
	absPath, err := filepath.Abs(workflowPath)
	if err != nil {
		return fmt.Errorf("start.path_resolve: 无法解析工作流路径: %w", err)
	}
	workflowPath = absPath

	// 2. 加载工作流
	loader := workflow.NewLoader(workflowPath)
	def, err := loader.Load()
	if err != nil {
		return fmt.Errorf("start.workflow_load: 加载工作流失败: %w", err)
	}

	// 3. 解析配置
	cfg, err := def.ParseConfig()
	if err != nil {
		return fmt.Errorf("start.config_parse: %s", errors.WrapError("config", "parse_failed", "解析配置失败", err))
	}

	// 4. 验证配置（基础调度配置）
	validation := cfg.ValidateDispatchConfig()
	if !validation.Valid {
		return fmt.Errorf("start.config_invalid: %s: %v", errors.ErrConfigInvalid.Error(), validation.Errors)
	}

	// 5. 验证 Symphony 特定配置（Agent CLI、prompt 文件等）
	symphonyValidation := cfg.ValidateSymphonyConfig()
	if !symphonyValidation.Valid {
		return fmt.Errorf("start.symphony_config_invalid: %s: %v", errors.ErrConfigInvalid.Error(), symphonyValidation.Errors)
	}

	// 6. 输出启动信息
	c.printStartupInfo(cfg, workflowPath)

	// 7. 检查 Tracker 可用性
	fmt.Println("检查 Tracker 可用性...")
	trackerClient := tracker.NewTracker(cfg)
	if err := trackerClient.CheckAvailability(); err != nil {
		return fmt.Errorf("start.tracker_unavailable: Tracker 不可用: %w", err)
	}
	fmt.Printf("Tracker %s 可用\n", cfg.Tracker.Kind)

	// 8. 创建编排器
	orch := orchestrator.New(cfg, def.PromptTemplate)

	// 9. 创建恢复管理器并执行任务恢复
	recoveryMgr := orchestrator.NewRecoveryManager(cfg, trackerClient, orch)
	fmt.Println("检查需要恢复的任务...")
	recoveredTasks, err := recoveryMgr.RestoreAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "任务恢复检查失败: %v\n", err)
		// 不退出，继续启动服务
	}
	if len(recoveredTasks) > 0 {
		fmt.Printf("发现 %d 个需要恢复的任务\n", len(recoveredTasks))
		// 执行恢复动作
		for _, task := range recoveredTasks {
			if execErr := recoveryMgr.ExecuteRecovery(ctx, task); execErr != nil {
				fmt.Fprintf(os.Stderr, "恢复任务 %s 失败: %v\n", task.Issue.Identifier, execErr)
			}
		}
	}

	// 10. 设置工作流文件监视
	go c.watchWorkflow(workflowPath, orch, cfg)

	// 11. 创建可取消的上下文
	runCtx, cancel := context.WithCancel(ctx)

	// 12. HTTP 服务器引用（用于优雅关闭）
	var srv *server.Server

	// 13. 处理信号 - 实现优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		fmt.Printf("\n接收到信号 %v，开始优雅关闭...\n", sig)

		// 创建关闭超时上下文（30秒）
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// 关闭编排器
		if err := orch.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "编排器关闭错误: %v\n", err)
		}

		// 关闭 HTTP 服务器
		if srv != nil {
			if err := srv.Shutdown(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "HTTP服务器关闭错误: %v\n", err)
			}
		}

		// 取消主上下文
		cancel()
	}()

	// 14. 启动 HTTP 服务器（如果配置了）
	httpPort := c.resolveHTTPPort(cfg)
	if httpPort > 0 {
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

	// 15. 运行编排器
	fmt.Println("启动编排器...")
	if err := orch.Run(runCtx); err != nil && err != context.Canceled {
		return fmt.Errorf("start.orchestrator_error: 编排器错误: %w", err)
	}

	fmt.Println("服务已停止")
	return nil
}

// resolveWorkflowPath 确定工作流文件路径
func (c *StartCommand) resolveWorkflowPath() string {
	if c.options.WorkflowPath != "" {
		return c.options.WorkflowPath
	}

	// 检查默认的 .sym/workflow.md
	if _, err := os.Stat(".sym/workflow.md"); err == nil {
		return ".sym/workflow.md"
	}

	return "./WORKFLOW.md"
}

// resolveHTTPPort 确定 HTTP 端口
func (c *StartCommand) resolveHTTPPort(cfg *config.Config) int {
	// CLI 参数优先
	if c.options.Port > 0 {
		return c.options.Port
	}

	// 配置文件中的端口
	if cfg.Server != nil && cfg.Server.Port > 0 {
		return cfg.Server.Port
	}

	return 0
}

// printStartupInfo 打印启动信息
func (c *StartCommand) printStartupInfo(cfg *config.Config, workflowPath string) {
	fmt.Println("========================================")
	fmt.Println("  Symphony - 编码代理编排服务")
	fmt.Println("========================================")
	fmt.Printf("工作流文件: %s\n", workflowPath)
	fmt.Printf("跟踪器: %s\n", cfg.Tracker.Kind)
	fmt.Printf("工作空间根目录: %s\n", cfg.Workspace.Root)
	fmt.Printf("轮询间隔: %dms\n", cfg.Polling.IntervalMs)
	fmt.Printf("最大并发: %d\n", cfg.Agent.MaxConcurrentAgents)

	httpPort := c.resolveHTTPPort(cfg)
	if httpPort > 0 {
		fmt.Printf("HTTP端口: %d\n", httpPort)
	}

	fmt.Println("========================================")
}

// watchWorkflow 监视工作流文件变更
func (c *StartCommand) watchWorkflow(path string, orch *orchestrator.Orchestrator, currentCfg *config.Config) {
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
				logging.Info("工作流已重新加载")

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

// ValidateConfig 验证配置是否有效（用于测试和预检查）
func (c *StartCommand) ValidateConfig() (*config.Config, *RunResult, error) {
	workflowPath := c.resolveWorkflowPath()

	// 转换为绝对路径
	absPath, err := filepath.Abs(workflowPath)
	if err != nil {
		return nil, nil, fmt.Errorf("start.path_resolve: 无法解析工作流路径: %w", err)
	}

	// 加载工作流
	loader := workflow.NewLoader(absPath)
	def, err := loader.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("start.workflow_load: 加载工作流失败: %w", err)
	}

	// 解析配置
	cfg, err := def.ParseConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("start.config_parse: %s", errors.WrapError("config", "parse_failed", "解析配置失败", err))
	}

	// 验证配置
	validation := cfg.ValidateDispatchConfig()
	if !validation.Valid {
		return nil, nil, fmt.Errorf("start.config_invalid: %s: %v", errors.ErrConfigInvalid.Error(), validation.Errors)
	}

	symphonyValidation := cfg.ValidateSymphonyConfig()
	if !symphonyValidation.Valid {
		return nil, nil, fmt.Errorf("start.symphony_config_invalid: %s: %v", errors.ErrConfigInvalid.Error(), symphonyValidation.Errors)
	}

	return cfg, &RunResult{
		Config:       cfg,
		WorkflowPath: absPath,
	}, nil
}
// Package main Symphony服务入口
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/dministrator/symphony/internal/cli"
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

	// 创建 StartCommand
	opts := cli.StartOptions{
		WorkflowPath: workflowPath,
		Port:         port,
	}
	cmd := cli.NewStartCommand(opts)

	// 执行 start 命令
	if err := cmd.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
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

	_ = initFlags.Parse(os.Args[2:])

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

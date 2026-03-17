// Package workspace 提供工作空间管理功能
package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// Manager 工作空间管理器
type Manager struct {
	cfg *config.Config
}

// NewManager 创建新的工作空间管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

// workspaceKeyRe 用于清理工作空间键名的正则表达式
var workspaceKeyRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// CreateForIssue 为问题创建工作空间
func (m *Manager) CreateForIssue(ctx context.Context, identifier string) (*domain.Workspace, error) {
	// 清理标识符作为工作空间键
	workspaceKey := workspaceKeyRe.ReplaceAllString(identifier, "_")

	// 计算工作空间路径
	workspacePath := filepath.Join(m.cfg.Workspace.Root, workspaceKey)

	// 检查是否已存在
	var createdNow bool
	info, err := os.Stat(workspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 创建目录
			if err := os.MkdirAll(workspacePath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create workspace directory: %w", err)
			}
			createdNow = true
		} else {
			return nil, fmt.Errorf("failed to stat workspace: %w", err)
		}
	} else if !info.IsDir() {
		// 路径存在但不是目录
		return nil, fmt.Errorf("workspace path exists but is not a directory: %s", workspacePath)
	}

	workspace := &domain.Workspace{
		Path:         workspacePath,
		WorkspaceKey: workspaceKey,
		CreatedNow:   createdNow,
	}

	// 如果是新创建的，运行 after_create 钩子
	if createdNow && m.cfg.Hooks.AfterCreate != nil {
		if err := m.runHook(ctx, *m.cfg.Hooks.AfterCreate, workspacePath, "after_create"); err != nil {
			// 钩子失败，清理目录
			os.RemoveAll(workspacePath)
			return nil, fmt.Errorf("after_create hook failed: %w", err)
		}
	}

	return workspace, nil
}

// RunBeforeRunHook 运行运行前钩子
func (m *Manager) RunBeforeRunHook(ctx context.Context, workspacePath string) error {
	if m.cfg.Hooks.BeforeRun == nil {
		return nil
	}
	return m.runHook(ctx, *m.cfg.Hooks.BeforeRun, workspacePath, "before_run")
}

// RunAfterRunHook 运行运行后钩子
func (m *Manager) RunAfterRunHook(ctx context.Context, workspacePath string) error {
	if m.cfg.Hooks.AfterRun == nil {
		return nil
	}
	// after_run 钩子失败只记录，不影响执行
	if err := m.runHook(ctx, *m.cfg.Hooks.AfterRun, workspacePath, "after_run"); err != nil {
		// 记录错误但不返回
		fmt.Printf("after_run hook failed (ignored): %v\n", err)
	}
	return nil
}

// RunBeforeRemoveHook 运行删除前钩子
func (m *Manager) RunBeforeRemoveHook(ctx context.Context, workspacePath string) error {
	if m.cfg.Hooks.BeforeRemove == nil {
		return nil
	}
	// before_remove 钩子失败只记录，不阻止删除
	if err := m.runHook(ctx, *m.cfg.Hooks.BeforeRemove, workspacePath, "before_remove"); err != nil {
		fmt.Printf("before_remove hook failed (ignored): %v\n", err)
	}
	return nil
}

// RemoveWorkspace 删除工作空间
func (m *Manager) RemoveWorkspace(ctx context.Context, workspacePath string) error {
	// 验证路径在工作空间根目录内
	if !m.IsWithinRoot(workspacePath) {
		return fmt.Errorf("workspace path is not within workspace root")
	}

	// 检查目录是否存在
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return nil // 目录不存在，无需删除
	}

	// 运行删除前钩子
	m.RunBeforeRemoveHook(ctx, workspacePath)

	// 删除目录
	if err := os.RemoveAll(workspacePath); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	return nil
}

// runHook 执行钩子脚本
func (m *Manager) runHook(ctx context.Context, script, workspacePath, hookName string) error {
	timeout := time.Duration(m.cfg.Hooks.TimeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 使用 bash 执行脚本
	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	cmd.Dir = workspacePath
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SYMPHONY_WORKSPACE=%s", workspacePath),
		fmt.Sprintf("SYMPHONY_HOOK=%s", hookName),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 截断输出
		outputStr := string(output)
		if len(outputStr) > 500 {
			outputStr = outputStr[:500] + "..."
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("hook %s timed out: %s", hookName, outputStr)
		}
		return fmt.Errorf("hook %s failed: %v - %s", hookName, err, outputStr)
	}

	return nil
}

// IsWithinRoot 检查路径是否在工作空间根目录内
func (m *Manager) IsWithinRoot(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(m.cfg.Workspace.Root)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absPath, absRoot+string(filepath.Separator))
}

// GetWorkspacePath 获取问题的工作空间路径
func (m *Manager) GetWorkspacePath(identifier string) string {
	workspaceKey := workspaceKeyRe.ReplaceAllString(identifier, "_")
	return filepath.Join(m.cfg.Workspace.Root, workspaceKey)
}

// CleanupTerminalWorkspaces 清理终态问题的工作空间
func (m *Manager) CleanupTerminalWorkspaces(ctx context.Context, terminalIssues []*domain.Issue) error {
	for _, issue := range terminalIssues {
		workspacePath := m.GetWorkspacePath(issue.Identifier)
		if err := m.RemoveWorkspace(ctx, workspacePath); err != nil {
			fmt.Printf("failed to cleanup workspace for %s: %v\n", issue.Identifier, err)
		} else {
			fmt.Printf("cleaned up workspace for terminal issue: %s\n", issue.Identifier)
		}
	}
	return nil
}
// Package workspace_test 测试工作空间管理
package workspace_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/workspace"
)

func TestCreateForIssue(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir

	mgr := workspace.NewManager(cfg)

	ctx := context.Background()
	ws, err := mgr.CreateForIssue(ctx, "TEST-123")
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// 检查路径
	expectedPath := filepath.Join(tmpDir, "TEST-123")
	if ws.Path != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, ws.Path)
	}

	// 检查是否已创建
	if !ws.CreatedNow {
		t.Error("expected CreatedNow to be true")
	}

	// 检查目录是否存在
	if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
		t.Error("workspace directory was not created")
	}
}

func TestReuseWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir

	mgr := workspace.NewManager(cfg)

	ctx := context.Background()

	// 第一次创建
	ws1, err := mgr.CreateForIssue(ctx, "TEST-456")
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	if !ws1.CreatedNow {
		t.Error("expected first creation to have CreatedNow=true")
	}

	// 第二次创建（复用）
	ws2, err := mgr.CreateForIssue(ctx, "TEST-456")
	if err != nil {
		t.Fatalf("failed to reuse workspace: %v", err)
	}

	if ws2.CreatedNow {
		t.Error("expected reuse to have CreatedNow=false")
	}

	if ws1.Path != ws2.Path {
		t.Error("expected same path for reuse")
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir

	mgr := workspace.NewManager(cfg)

	ctx := context.Background()

	// 测试特殊字符会被替换
	ws, err := mgr.CreateForIssue(ctx, "TEST/123@ABC")
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "TEST_123_ABC")
	if ws.Path != expectedPath {
		t.Errorf("expected sanitized path '%s', got '%s'", expectedPath, ws.Path)
	}
}

func TestRemoveWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir

	mgr := workspace.NewManager(cfg)

	ctx := context.Background()

	// 创建工作空间
	ws, err := mgr.CreateForIssue(ctx, "TEST-789")
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// 确保目录存在
	if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
		t.Fatal("workspace was not created")
	}

	// 删除工作空间
	if err := mgr.RemoveWorkspace(ctx, ws.Path); err != nil {
		t.Fatalf("failed to remove workspace: %v", err)
	}

	// 确保目录已删除
	if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
		t.Error("workspace was not removed")
	}
}

func TestIsWithinRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir

	mgr := workspace.NewManager(cfg)

	// 测试在根目录内
	validPath := filepath.Join(tmpDir, "subdir")
	if !mgr.IsWithinRoot(validPath) {
		t.Error("expected path within root to be valid")
	}

	// 测试在根目录外
	invalidPath := "/tmp/other-dir"
	if mgr.IsWithinRoot(invalidPath) {
		t.Error("expected path outside root to be invalid")
	}
}

func TestGetWorkspacePath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Workspace.Root = "/tmp/symphony"

	mgr := workspace.NewManager(cfg)

	path := mgr.GetWorkspacePath("TEST-123")
	expected := "/tmp/symphony/TEST-123"

	if path != expected {
		t.Errorf("expected path '%s', got '%s'", expected, path)
	}
}

// TestGetWorkspacePathSanitization 测试路径清理功能
func TestGetWorkspacePathSanitization(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "normal identifier",
			identifier: "TEST-123",
			expected:   "/tmp/symphony/TEST-123",
		},
		{
			name:       "special characters",
			identifier: "TEST/123@ABC",
			expected:   "/tmp/symphony/TEST_123_ABC",
		},
		{
			name:       "spaces",
			identifier: "TEST 123",
			expected:   "/tmp/symphony/TEST_123",
		},
		{
			name:       "all special characters",
			identifier: "test@example.com:issue-123",
			expected:   "/tmp/symphony/test_example.com_issue-123",
		},
		{
			name:       "dots and underscores preserved",
			identifier: "test.v1_issue",
			expected:   "/tmp/symphony/test.v1_issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Workspace.Root = "/tmp/symphony"
			mgr := workspace.NewManager(cfg)

			path := mgr.GetWorkspacePath(tt.identifier)
			if path != tt.expected {
				t.Errorf("expected path '%s', got '%s'", tt.expected, path)
			}
		})
	}
}

// TestRunBeforeRunHook 测试运行前钩子
func TestRunBeforeRunHook(t *testing.T) {
	tests := []struct {
		name       string
		script     *string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "no hook configured",
			script:  nil,
			wantErr: false,
		},
		{
			name:    "successful hook execution",
			script:  strPtr("echo 'success'"),
			wantErr: false,
		},
		{
			name:       "hook fails",
			script:     strPtr("exit 1"),
			wantErr:    true,
			errMessage: "hook before_run failed",
		},
		{
			name:       "hook times out",
			script:     strPtr("sleep 10"),
			wantErr:    true,
			errMessage: "timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "symphony-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := config.DefaultConfig()
			cfg.Workspace.Root = tmpDir
			cfg.Hooks.BeforeRun = tt.script
			cfg.Hooks.TimeoutMs = 1000 // 1 second timeout for timeout test

			mgr := workspace.NewManager(cfg)
			ctx := context.Background()

			err = mgr.RunBeforeRunHook(ctx, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunBeforeRunHook() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMessage != "" && err == nil {
				t.Errorf("expected error containing '%s', got nil", tt.errMessage)
			}
			if tt.wantErr && tt.errMessage != "" && err != nil {
				if !contains(err.Error(), tt.errMessage) {
					t.Errorf("error message should contain '%s', got '%s'", tt.errMessage, err.Error())
				}
			}
		})
	}
}

// TestRunAfterRunHook 测试运行后钩子
func TestRunAfterRunHook(t *testing.T) {
	tests := []struct {
		name       string
		script     *string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "no hook configured",
			script:  nil,
			wantErr: false,
		},
		{
			name:    "successful hook execution",
			script:  strPtr("echo 'success'"),
			wantErr: false,
		},
		{
			name:    "hook fails but is ignored",
			script:  strPtr("exit 1"),
			wantErr: false, // after_run 失败不返回错误
		},
		{
			name:    "hook times out but is ignored",
			script:  strPtr("sleep 10"),
			wantErr: false, // after_run 失败不返回错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "symphony-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := config.DefaultConfig()
			cfg.Workspace.Root = tmpDir
			cfg.Hooks.AfterRun = tt.script
			cfg.Hooks.TimeoutMs = 1000

			mgr := workspace.NewManager(cfg)
			ctx := context.Background()

			err = mgr.RunAfterRunHook(ctx, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunAfterRunHook() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRunBeforeRemoveHook 测试删除前钩子
func TestRunBeforeRemoveHook(t *testing.T) {
	tests := []struct {
		name       string
		script     *string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "no hook configured",
			script:  nil,
			wantErr: false,
		},
		{
			name:    "successful hook execution",
			script:  strPtr("echo 'success'"),
			wantErr: false,
		},
		{
			name:    "hook fails but is ignored",
			script:  strPtr("exit 1"),
			wantErr: false, // before_remove 失败不返回错误
		},
		{
			name:    "hook times out but is ignored",
			script:  strPtr("sleep 10"),
			wantErr: false, // before_remove 失败不返回错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "symphony-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := config.DefaultConfig()
			cfg.Workspace.Root = tmpDir
			cfg.Hooks.BeforeRemove = tt.script
			cfg.Hooks.TimeoutMs = 1000

			mgr := workspace.NewManager(cfg)
			ctx := context.Background()

			err = mgr.RunBeforeRemoveHook(ctx, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunBeforeRemoveHook() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRemoveWorkspaceEdgeCases 测试删除工作空间的边界情况
func TestRemoveWorkspaceEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, cfg *config.Config) (workspacePath string)
		wantErr     bool
		errMessage  string
		cleanupFunc func(t *testing.T, path string)
	}{
		{
			name: "remove existing workspace",
			setupFunc: func(t *testing.T, cfg *config.Config) string {
				mgr := workspace.NewManager(cfg)
				ctx := context.Background()
				ws, err := mgr.CreateForIssue(ctx, "TEST-REMOVE-1")
				if err != nil {
					t.Fatalf("failed to create workspace: %v", err)
				}
				return ws.Path
			},
			wantErr: false,
		},
		{
			name: "remove non-existent workspace",
			setupFunc: func(t *testing.T, cfg *config.Config) string {
				return filepath.Join(cfg.Workspace.Root, "non-existent")
			},
			wantErr: false, // 不存在的目录不返回错误
		},
		{
			name: "path outside root is rejected",
			setupFunc: func(t *testing.T, cfg *config.Config) string {
				return "/tmp/outside-root"
			},
			wantErr:    true,
			errMessage: "not within workspace root",
		},
		{
			name: "remove with before_remove hook",
			setupFunc: func(t *testing.T, cfg *config.Config) string {
				cfg.Hooks.BeforeRemove = strPtr("echo 'before remove'")
				mgr := workspace.NewManager(cfg)
				ctx := context.Background()
				ws, err := mgr.CreateForIssue(ctx, "TEST-REMOVE-2")
				if err != nil {
					t.Fatalf("failed to create workspace: %v", err)
				}
				return ws.Path
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "symphony-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := config.DefaultConfig()
			cfg.Workspace.Root = tmpDir

			workspacePath := tt.setupFunc(t, cfg)

			mgr := workspace.NewManager(cfg)
			ctx := context.Background()

			err = mgr.RemoveWorkspace(ctx, workspacePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveWorkspace() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMessage != "" && err == nil {
				t.Errorf("expected error containing '%s', got nil", tt.errMessage)
			}
			if tt.wantErr && tt.errMessage != "" && err != nil {
				if !contains(err.Error(), tt.errMessage) {
					t.Errorf("error message should contain '%s', got '%s'", tt.errMessage, err.Error())
				}
			}

			// 如果期望成功删除，验证目录不存在
			if !tt.wantErr && !contains(tt.name, "outside root") && !contains(tt.name, "non-existent") {
				if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
					t.Errorf("workspace directory should be removed, but it still exists or stat failed: %v", err)
				}
			}
		})
	}
}

// TestCreateForIssueWithAfterCreateHook 测试 after_create 钩子
func TestCreateForIssueWithAfterCreateHook(t *testing.T) {
	tests := []struct {
		name        string
		hookScript  *string
		wantErr     bool
		errMessage  string
		shouldExist bool // 创建后工作空间是否应该存在
	}{
		{
			name:        "no hook",
			hookScript:  nil,
			wantErr:     false,
			shouldExist: true,
		},
		{
			name:        "successful hook",
			hookScript:  strPtr("echo 'after create' > hook-marker.txt"),
			wantErr:     false,
			shouldExist: true,
		},
		{
			name:        "hook fails, workspace should be removed",
			hookScript:  strPtr("exit 1"),
			wantErr:     true,
			errMessage:  "after_create hook failed",
			shouldExist: false,
		},
		{
			name:        "hook times out, workspace should be removed",
			hookScript:  strPtr("sleep 10"),
			wantErr:     true,
			errMessage:  "after_create hook failed",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "symphony-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := config.DefaultConfig()
			cfg.Workspace.Root = tmpDir
			cfg.Hooks.AfterCreate = tt.hookScript
			cfg.Hooks.TimeoutMs = 1000

			mgr := workspace.NewManager(cfg)
			ctx := context.Background()

			ws, err := mgr.CreateForIssue(ctx, "TEST-HOOK-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateForIssue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMessage != "" && err == nil {
				t.Errorf("expected error containing '%s', got nil", tt.errMessage)
			}
			if tt.wantErr && tt.errMessage != "" && err != nil {
				if !contains(err.Error(), tt.errMessage) {
					t.Errorf("error message should contain '%s', got '%s'", tt.errMessage, err.Error())
				}
			}

			// 验证工作空间是否存在
			if tt.shouldExist {
				if ws == nil {
					t.Fatal("expected workspace to be created, got nil")
				}
				if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
					t.Error("workspace should exist after creation")
				}
			} else {
				// 失败时工作空间应该被清理
				if ws != nil {
					if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
						t.Error("workspace should be removed after hook failure")
					}
				}
			}
		})
	}
}

// TestCleanupTerminalWorkspaces 测试批量清理终态工作空间
func TestCleanupTerminalWorkspaces(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, mgr *workspace.Manager) []*domain.Issue
		wantErr    bool
		validate   func(t *testing.T, mgr *workspace.Manager, issues []*domain.Issue)
	}{
		{
			name: "cleanup multiple workspaces",
			setupFunc: func(t *testing.T, mgr *workspace.Manager) []*domain.Issue {
				ctx := context.Background()
				issues := []*domain.Issue{}

				// 创建3个工作空间
				for i := 1; i <= 3; i++ {
					_, err := mgr.CreateForIssue(ctx, fmt.Sprintf("TEST-CLEANUP-%d", i))
					if err != nil {
						t.Fatalf("failed to create workspace: %v", err)
					}
					issues = append(issues, &domain.Issue{
						ID:         fmt.Sprintf("id-%d", i),
						Identifier: fmt.Sprintf("TEST-CLEANUP-%d", i),
					})
				}

				return issues
			},
			wantErr: false,
			validate: func(t *testing.T, mgr *workspace.Manager, issues []*domain.Issue) {
				// 验证所有工作空间都被删除
				for _, issue := range issues {
					path := mgr.GetWorkspacePath(issue.Identifier)
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("workspace for %s should be removed", issue.Identifier)
					}
				}
			},
		},
		{
			name: "cleanup with some workspaces already removed",
			setupFunc: func(t *testing.T, mgr *workspace.Manager) []*domain.Issue {
				ctx := context.Background()
				issues := []*domain.Issue{}

				// 创建3个工作空间
				for i := 1; i <= 3; i++ {
					ws, err := mgr.CreateForIssue(ctx, fmt.Sprintf("TEST-PARTIAL-%d", i))
					if err != nil {
						t.Fatalf("failed to create workspace: %v", err)
					}
					issues = append(issues, &domain.Issue{
						ID:         fmt.Sprintf("id-%d", i),
						Identifier: fmt.Sprintf("TEST-PARTIAL-%d", i),
					})

					// 手动删除第二个工作空间
					if i == 2 {
						os.RemoveAll(ws.Path)
					}
				}

				return issues
			},
			wantErr: false,
			validate: func(t *testing.T, mgr *workspace.Manager, issues []*domain.Issue) {
				// 验证所有工作空间都被删除（包括已删除的）
				for _, issue := range issues {
					path := mgr.GetWorkspacePath(issue.Identifier)
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("workspace for %s should be removed", issue.Identifier)
					}
				}
			},
		},
		{
			name: "cleanup with failed removal",
			setupFunc: func(t *testing.T, mgr *workspace.Manager) []*domain.Issue {
				ctx := context.Background()
				issues := []*domain.Issue{}

				// 创建一个工作空间并设置权限导致删除失败
				ws, err := mgr.CreateForIssue(ctx, "TEST-FAIL-REMOVE")
				if err != nil {
					t.Fatalf("failed to create workspace: %v", err)
				}

				// 创建一个无法删除的文件
				testFile := filepath.Join(ws.Path, "protected.txt")
				if err := os.WriteFile(testFile, []byte("protected"), 0444); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}

				issues = append(issues, &domain.Issue{
					ID:         "id-fail",
					Identifier: "TEST-FAIL-REMOVE",
				})

				// 创建一个可以正常删除的工作空间
				ws2, err := mgr.CreateForIssue(ctx, "TEST-NORMAL-REMOVE")
				if err != nil {
					t.Fatalf("failed to create workspace: %v", err)
				}
				if err := os.Chmod(ws2.Path, 0755); err != nil {
					t.Fatalf("failed to set permissions: %v", err)
				}

				issues = append(issues, &domain.Issue{
					ID:         "id-normal",
					Identifier: "TEST-NORMAL-REMOVE",
				})

				return issues
			},
			wantErr: false, // CleanupTerminalWorkspaces 不返回错误，即使部分失败
			validate: func(t *testing.T, mgr *workspace.Manager, issues []*domain.Issue) {
				// 验证正常的工作空间被删除
				normalPath := mgr.GetWorkspacePath("TEST-NORMAL-REMOVE")
				if _, err := os.Stat(normalPath); !os.IsNotExist(err) {
					t.Error("normal workspace should be removed")
				}
			},
		},
		{
			name: "cleanup empty list",
			setupFunc: func(t *testing.T, mgr *workspace.Manager) []*domain.Issue {
				return []*domain.Issue{}
			},
			wantErr: false,
			validate: func(t *testing.T, mgr *workspace.Manager, issues []*domain.Issue) {
				// 空列表应该正常完成
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "symphony-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := config.DefaultConfig()
			cfg.Workspace.Root = tmpDir

			mgr := workspace.NewManager(cfg)
			ctx := context.Background()

			issues := tt.setupFunc(t, mgr)

			err = mgr.CleanupTerminalWorkspaces(ctx, issues)
			if (err != nil) != tt.wantErr {
				t.Errorf("CleanupTerminalWorkspaces() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mgr, issues)
			}
		})
	}
}

// TestHookEnvironmentVariables 测试钩子环境变量设置
func TestHookEnvironmentVariables(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir
	cfg.Hooks.BeforeRun = strPtr("echo $SYMPHONY_WORKSPACE > workspace_var.txt && echo $SYMPHONY_HOOK > hook_var.txt")
	cfg.Hooks.TimeoutMs = 5000

	mgr := workspace.NewManager(cfg)
	ctx := context.Background()

	err = mgr.RunBeforeRunHook(ctx, tmpDir)
	if err != nil {
		t.Fatalf("RunBeforeRunHook failed: %v", err)
	}

	// 验证环境变量被正确设置
	workspaceVarFile := filepath.Join(tmpDir, "workspace_var.txt")
	hookVarFile := filepath.Join(tmpDir, "hook_var.txt")

	workspaceVar, err := os.ReadFile(workspaceVarFile)
	if err != nil {
		t.Fatalf("failed to read workspace_var.txt: %v", err)
	}

	hookVar, err := os.ReadFile(hookVarFile)
	if err != nil {
		t.Fatalf("failed to read hook_var.txt: %v", err)
	}

	expectedWorkspace := tmpDir + "\n"
	expectedHook := "before_run\n"

	if string(workspaceVar) != expectedWorkspace {
		t.Errorf("SYMPHONY_WORKSPACE = %q, want %q", string(workspaceVar), expectedWorkspace)
	}

	if string(hookVar) != expectedHook {
		t.Errorf("SYMPHONY_HOOK = %q, want %q", string(hookVar), expectedHook)
	}
}

// TestHookOutputTruncation 测试钩子输出截断
func TestHookOutputTruncation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symphony-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Workspace.Root = tmpDir
	// 创建一个输出超过500字符的钩子
	cfg.Hooks.BeforeRun = strPtr(`for i in {1..100}; do echo "This is a very long line that will exceed the output limit when concatenated many times"; done && exit 1`)
	cfg.Hooks.TimeoutMs = 5000

	mgr := workspace.NewManager(cfg)
	ctx := context.Background()

	err = mgr.RunBeforeRunHook(ctx, tmpDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if len(errMsg) > 1000 {
		t.Errorf("error message too long (>%d), should be truncated", 1000)
	}

	// 验证包含截断标记
	if !contains(errMsg, "...") {
		t.Error("error message should contain truncation marker '...'")
	}
}

// 辅助函数

func strPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
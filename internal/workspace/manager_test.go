// Package workspace_test 测试工作空间管理
package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dministrator/symphony/internal/config"
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
// Package tracker 提供文件系统 Tracker 测试
package tracker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dministrator/symphony/internal/domain"
)

func TestFileClient_CheckAvailability(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	// 检查可用性（目录不存在时会创建）
	err := client.CheckAvailability()
	if err != nil {
		t.Fatalf("CheckAvailability failed: %v", err)
	}

	// 验证目录已创建
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Fatal("directory should be created")
	}
}

func TestFileClient_CreateTask(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()
	issue, err := client.CreateTask(ctx, "测试任务", "这是一个测试任务描述")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 验证任务 ID 已生成
	if issue.ID == "" {
		t.Fatal("task ID should not be empty")
	}

	// 验证任务目录结构已创建
	taskDir := filepath.Join(tmpDir, issue.ID)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Fatal("task directory should be created")
	}

	// 验证状态索引文件已创建
	taskFile := filepath.Join(taskDir, taskFileName)
	if _, err := os.Stat(taskFile); os.IsNotExist(err) {
		t.Fatal("task.md should be created")
	}

	// 验证子任务目录已创建
	for _, subdir := range []string{"Planner", "Generator", "Evaluator"} {
		subDirPath := filepath.Join(taskDir, subdir)
		if _, err := os.Stat(subDirPath); os.IsNotExist(err) {
			t.Fatalf("%s directory should be created", subdir)
		}
	}
}

func TestFileClient_GetTask(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()
	// 先创建任务
	created, err := client.CreateTask(ctx, "获取测试任务", "测试 GetTask")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 获取任务
	issue, err := client.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	// 验证任务信息
	if issue.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", issue.ID, created.ID)
	}

	if issue.Title != "获取测试任务" {
		t.Errorf("Title mismatch: got %s, want %s", issue.Title, "获取测试任务")
	}

	if issue.State != "backlog" {
		t.Errorf("State mismatch: got %s, want backlog", issue.State)
	}
}

func TestFileClient_UpdateStage(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()
	created, err := client.CreateTask(ctx, "更新状态任务", "测试 UpdateStage")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 更新阶段状态
	stage := domain.StageState{
		Name:   "generator",
		Status: "in_progress",
	}

	err = client.UpdateStage(ctx, created.ID, stage)
	if err != nil {
		t.Fatalf("UpdateStage failed: %v", err)
	}

	// 验证状态已更新
	issue, err := client.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if issue.State != "in-progress" {
		t.Errorf("State should be in-progress, got %s", issue.State)
	}
}

func TestFileClient_ListTasksByState(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()

	// 创建多个任务
	task1, err := client.CreateTask(ctx, "任务1", "描述1")
	if err != nil {
		t.Fatalf("CreateTask 1 failed: %v", err)
	}

	_, err = client.CreateTask(ctx, "任务2", "描述2")
	if err != nil {
		t.Fatalf("CreateTask 2 failed: %v", err)
	}

	// 更新 task1 状态为 in-progress
	stage := domain.StageState{
		Name:   "generator",
		Status: "in_progress",
	}
	if err := client.UpdateStage(ctx, task1.ID, stage); err != nil {
		t.Fatalf("UpdateStage failed: %v", err)
	}

	// 按状态查询
	issues, err := client.ListTasksByState(ctx, []string{"backlog"})
	if err != nil {
		t.Fatalf("ListTasksByState failed: %v", err)
	}

	// 应该只有 task2 在 backlog 状态
	if len(issues) != 1 {
		t.Errorf("expected 1 backlog task, got %d", len(issues))
	}

	// 查询 in-progress 状态
	inProgress, err := client.ListTasksByState(ctx, []string{"in-progress"})
	if err != nil {
		t.Fatalf("ListTasksByState failed: %v", err)
	}

	if len(inProgress) != 1 {
		t.Errorf("expected 1 in-progress task, got %d", len(inProgress))
	}
}

func TestFileClient_GetStageState(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()
	created, err := client.CreateTask(ctx, "状态恢复任务", "测试 GetStageState")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 更新阶段
	stage := domain.StageState{
		Name:   "evaluator",
		Status: "in_progress",
	}
	if err := client.UpdateStage(ctx, created.ID, stage); err != nil {
		t.Fatalf("UpdateStage failed: %v", err)
	}

	// 获取阶段状态
	stageState, err := client.GetStageState(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetStageState failed: %v", err)
	}

	if stageState.Name != "evaluator" {
		t.Errorf("Stage name mismatch: got %s, want evaluator", stageState.Name)
	}

	if stageState.Status != "in_progress" {
		t.Errorf("Stage status mismatch: got %s, want in_progress", stageState.Status)
	}
}

func TestParseSubTaskTitle(t *testing.T) {
	tests := []struct {
		title    string
		wantType string
		wantNum  int
		wantName string
	}{
		{"P1: 需求澄清", "planner", 1, "需求澄清"},
		{"G1: BDD测试脚本-v1", "generator", 1, "BDD测试脚本"},
		{"G4: 代码实现", "generator", 4, "代码实现"},
		{"E1: 评估验收-v1", "evaluator", 1, "评估验收"},
	}

	for _, tt := range tests {
		gotType, gotNum, gotName := parseSubTaskTitle(tt.title)
		if gotType != tt.wantType {
			t.Errorf("parseSubTaskTitle(%s) type = %s, want %s", tt.title, gotType, tt.wantType)
		}
		if gotNum != tt.wantNum {
			t.Errorf("parseSubTaskTitle(%s) num = %d, want %d", tt.title, gotNum, tt.wantNum)
		}
		if gotName != tt.wantName {
			t.Errorf("parseSubTaskTitle(%s) name = %s, want %s", tt.title, gotName, tt.wantName)
		}
	}
}

func TestParseSubTaskID(t *testing.T) {
	tests := []struct {
		identifier   string
		wantParent   string
		wantType     string
		wantNum      int
	}{
		{"SYM-001-P1", "SYM-001", "P", 1},
		{"SYM-001-G4", "SYM-001", "G", 4},
		{"SYM-001-E1", "SYM-001", "E", 1},
		{"ABC-123-G2", "ABC-123", "G", 2},
	}

	for _, tt := range tests {
		gotParent, gotType, gotNum := parseSubTaskID(tt.identifier)
		if gotParent != tt.wantParent {
			t.Errorf("parseSubTaskID(%s) parent = %s, want %s", tt.identifier, gotParent, tt.wantParent)
		}
		if gotType != tt.wantType {
			t.Errorf("parseSubTaskID(%s) type = %s, want %s", tt.identifier, gotType, tt.wantType)
		}
		if gotNum != tt.wantNum {
			t.Errorf("parseSubTaskID(%s) num = %d, want %d", tt.identifier, gotNum, tt.wantNum)
		}
	}
}

func TestStatusToMark(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"completed", "✅"},
		{"done", "✅"},
		{"approved", "✅"},
		{"failed", "❌"},
		{"rejected", "❌"},
		{"in-progress", "⏳"},
		{"pending", "⏳"},
		{"backlog", "⬜"},
	}

	for _, tt := range tests {
		got := statusToMark(tt.status)
		if got != tt.want {
			t.Errorf("statusToMark(%s) = %s, want %s", tt.status, got, tt.want)
		}
	}
}

func TestFrontmatterParsing(t *testing.T) {
	content := `---
id: SYM-001
title: 测试任务
status: backlog
phase: backlog
iteration: 1
created: 2026-04-07T10:00:00Z
updated: 2026-04-07T10:00:00Z
---

# Planner

- P1: 需求澄清 ⬜
`

	fm, err := parseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parseFrontmatter failed: %v", err)
	}

	if fm["id"] != "SYM-001" {
		t.Errorf("id mismatch: got %v, want SYM-001", fm["id"])
	}

	if fm["title"] != "测试任务" {
		t.Errorf("title mismatch: got %v, want 测试任务", fm["title"])
	}

	if fm["status"] != "backlog" {
		t.Errorf("status mismatch: got %v, want backlog", fm["status"])
	}
}

func TestFrontmatterWithContent(t *testing.T) {
	content := `---
id: SYM-001
title: 测试任务
---

# Planner

- P1: 需求澄清 ⬜
`

	fm, markdown, err := parseFrontmatterWithContent([]byte(content))
	if err != nil {
		t.Fatalf("parseFrontmatterWithContent failed: %v", err)
	}

	if fm["id"] != "SYM-001" {
		t.Errorf("id mismatch: got %v, want SYM-001", fm["id"])
	}

	if !markdownContains(markdown, "# Planner") {
		t.Errorf("markdown should contain # Planner, got: %s", markdown)
	}
}

func markdownContains(content, substr string) bool {
	return len(content) > 0 && (content == substr || len(content) >= len(substr) && content[:len(substr)] == substr || len(content) > len(substr) && contains(content, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFormatFrontmatter(t *testing.T) {
	fm := map[string]interface{}{
		"id":     "SYM-001",
		"title":  "测试任务",
		"status": "backlog",
	}

	content := "# Planner\n\n- P1: 需求澄清 ⬜"

	data := formatFrontmatter(fm, content)

	// 验证包含 frontmatter 边界
	if !contains(string(data), "---\n") {
		t.Error("formatted data should contain frontmatter boundary")
	}

	// 验证包含 YAML 内容
	if !contains(string(data), "id: SYM-001") {
		t.Error("formatted data should contain id field")
	}

	// 验证包含 markdown 内容
	if !contains(string(data), "# Planner") {
		t.Error("formatted data should contain markdown content")
	}
}

func TestFileClient_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()
	created, err := client.CreateTask(ctx, "并发写入测试", "测试并发写入")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 并发更新状态
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			stage := domain.StageState{
				Name:   fmt.Sprintf("stage-%d", n),
				Status: "in_progress",
			}
			_ = client.UpdateStage(ctx, created.ID, stage)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证文件仍然可读
	issue, err := client.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTask failed after concurrent writes: %v", err)
	}

	if issue.ID != created.ID {
		t.Errorf("ID mismatch after concurrent writes: got %s, want %s", issue.ID, created.ID)
	}
}

func TestFileClient_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewFileClientWithDir(tmpDir)

	ctx := context.Background()
	created, err := client.CreateTask(ctx, "停止测试", "测试 Stop")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 更新状态
	stage := domain.StageState{
		Name:   "test",
		Status: "completed",
	}
	if err := client.UpdateStage(ctx, created.ID, stage); err != nil {
		t.Fatalf("UpdateStage failed: %v", err)
	}

	// 停止客户端
	client.Stop()

	// 停止后更新应该不会 panic（但可能失败）
	// 由于 stopWriteCh 已关闭，写入可能被丢弃或失败
	_ = client.UpdateStage(ctx, created.ID, stage)
}
// Package handlers 提供HTTP处理器功能的导出函数测试
package handlers

import (
	"embed"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestNewSSEHandler 测试 NewSSEHandler 构造函数
func TestNewSSEHandler(t *testing.T) {
	broadcaster := common.NewSSEBroadcaster()
	handler := NewSSEHandler(broadcaster)
	assert.NotNil(t, handler)
}

// TestNewTaskHandler 测试 NewTaskHandler 构造函数
func TestNewTaskHandler(t *testing.T) {
	handler := NewTaskHandler()
	assert.NotNil(t, handler)
}

// TestNewDashboardHandler 测试 NewDashboardHandler 构造函数
func TestNewDashboardHandler(t *testing.T) {
	mockGetter := &mockOrchestratorGetter{}
	handler := NewDashboardHandler(mockGetter)
	assert.NotNil(t, handler)
}

// TestNewStaticHandler 测试 NewStaticHandler 构造函数
func TestNewStaticHandler(t *testing.T) {
	var embedFS embed.FS
	handler := NewStaticHandler(embedFS)
	assert.NotNil(t, handler)
}

// TestNewAPIHandler 测试 NewAPIHandler 构造函数
func TestNewAPIHandler(t *testing.T) {
	mockGetter := &mockOrchestratorGetter{}
	handler := NewAPIHandler(mockGetter)
	assert.NotNil(t, handler)
}

// TestNewAPIHandlerWithCanceler 测试 NewAPIHandlerWithCanceler 构造函数
func TestNewAPIHandlerWithCanceler(t *testing.T) {
	mockGetter := &mockOrchestratorGetter{}
	mockCanceler := &mockOrchestratorCanceler{}
	handler := NewAPIHandlerWithCanceler(mockGetter, mockCanceler)
	assert.NotNil(t, handler)
}

// TestRenderTaskFormHTML 测试 RenderTaskFormHTML 函数
func TestRenderTaskFormHTML(t *testing.T) {
	html := RenderTaskFormHTML()
	assert.Contains(t, html, "form")
	assert.Contains(t, html, "任务")
}

// TestRenderTaskCreatedHTML 测试 RenderTaskCreatedHTML 函数
func TestRenderTaskCreatedHTML(t *testing.T) {
	parent := TaskInfo{
		Identifier: "PARENT-1",
		Title:      "Parent Task",
	}
	subTasks := []TaskInfo{
		{Identifier: "CHILD-1", Title: "Child Task 1"},
	}

	html := RenderTaskCreatedHTML(parent, subTasks)
	assert.Contains(t, html, "PARENT-1")
	assert.Contains(t, html, "CHILD-1")
}

// TestRenderAnswerSubmittedHTML 测试 RenderAnswerSubmittedHTML 函数
func TestRenderAnswerSubmittedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Answer submitted",
	}

	html := RenderAnswerSubmittedHTML(response)
	assert.Contains(t, html, "Answer submitted")
}

// TestRenderBDDApprovedHTML 测试 RenderBDDApprovedHTML 函数
func TestRenderBDDApprovedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "BDD approved",
	}

	html := RenderBDDApprovedHTML(response)
	assert.Contains(t, html, "BDD")
}

// TestRenderBDDRejectedHTML 测试 RenderBDDRejectedHTML 函数
func TestRenderBDDRejectedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "BDD rejected",
	}

	html := RenderBDDRejectedHTML(response)
	assert.Contains(t, html, "BDD")
}

// TestRenderArchitectureApprovedHTML 测试 RenderArchitectureApprovedHTML 函数
func TestRenderArchitectureApprovedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Architecture approved",
	}

	html := RenderArchitectureApprovedHTML(response)
	assert.Contains(t, html, "Architecture")
}

// TestRenderArchitectureRejectedHTML 测试 RenderArchitectureRejectedHTML 函数
func TestRenderArchitectureRejectedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Architecture rejected",
	}

	html := RenderArchitectureRejectedHTML(response)
	assert.Contains(t, html, "Architecture")
}

// TestRenderVerificationApprovedHTML 测试 RenderVerificationApprovedHTML 函数
func TestRenderVerificationApprovedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Verification approved",
	}

	html := RenderVerificationApprovedHTML(response)
	assert.Contains(t, html, "Verification")
}

// TestRenderVerificationRejectedHTML 测试 RenderVerificationRejectedHTML 函数
func TestRenderVerificationRejectedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Verification rejected",
	}

	html := RenderVerificationRejectedHTML(response)
	assert.Contains(t, html, "Verification")
}

// TestRenderTaskResumedHTML 测试 RenderTaskResumedHTML 函数
func TestRenderTaskResumedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Task resumed",
	}

	html := RenderTaskResumedHTML(response)
	assert.Contains(t, html, "resumed")
}

// TestRenderTaskReclarifiedHTML 测试 RenderTaskReclarifiedHTML 函数
func TestRenderTaskReclarifiedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Task reclarified",
	}

	html := RenderTaskReclarifiedHTML(response)
	assert.Contains(t, html, "reclarified")
}

// TestRenderTaskAbandonedHTML 测试 RenderTaskAbandonedHTML 函数
func TestRenderTaskAbandonedHTML(t *testing.T) {
	response := gin.H{
		"success": true,
		"message": "Task abandoned",
	}

	html := RenderTaskAbandonedHTML(response)
	assert.Contains(t, html, "abandoned")
}

// TestRenderExecutionLogsHTML 测试 RenderExecutionLogsHTML 函数
func TestRenderExecutionLogsHTML(t *testing.T) {
	task := &domain.Issue{
		ID:         "1",
		Identifier: "EXEC-1",
		Title:      "Execution Task",
	}

	stageState := &domain.StageState{
		Name:   "implementation",
		Status: "in_progress",
	}

	logs := []workflow.ExecutionLog{
		{Message: "Starting execution", Event: "start", Timestamp: time.Now()},
		{Message: "Task completed", Event: "complete", Timestamp: time.Now()},
	}

	progress := &workflow.ExecutionProgress{
		TaskID:       "1",
		Identifier:   "EXEC-1",
		CurrentStage: workflow.StageImplementation,
		Status:       workflow.StatusInProgress,
	}

	html := RenderExecutionLogsHTML(task, stageState, logs, 2, progress)
	assert.Contains(t, html, "EXEC-1")
	assert.Contains(t, html, "Starting execution")
}

// Mock types for testing

type mockOrchestratorGetter struct{}

func (m *mockOrchestratorGetter) GetState() *domain.OrchestratorState {
	return &domain.OrchestratorState{
		Running:       make(map[string]*domain.RunningEntry),
		RetryAttempts: make(map[string]*domain.RetryEntry),
		CodexTotals:   &domain.CodexTotals{},
	}
}

func (m *mockOrchestratorGetter) GetTracker() tracker.Tracker {
	return nil
}

type mockOrchestratorCanceler struct{}

func (m *mockOrchestratorCanceler) CancelTask(identifier string) (bool, bool, error) {
	return false, false, nil
}

func (m *mockOrchestratorCanceler) GetRunningEntryByIdentifier(identifier string) *domain.RunningEntry {
	return nil
}

func (m *mockOrchestratorCanceler) GetRetryEntryByIdentifier(identifier string) *domain.RetryEntry {
	return nil
}
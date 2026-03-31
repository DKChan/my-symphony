package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/router"
	"github.com/stretchr/testify/assert"
)

func TestAPIHandler_GetState(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/v1/state", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "running")
	assert.Contains(t, response, "counts")
}

func TestAPIHandler_GetIssue_NotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/v1/NONEXISTENT-123", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIHandler_Refresh(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/v1/refresh", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["queued"])
	assert.Equal(t, false, response["coalesced"])
}

func TestDashboardHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestStaticHandler_DashboardCSS(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/dashboard.css", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/css")
}

func TestAPIHandler_GetTasks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	// 添加一些 mock 数据
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "TEST-1", Title: "Test Task 1", State: "Todo"},
		{ID: "2", Identifier: "TEST-2", Title: "Test Task 2", State: "In Progress"},
		{ID: "3", Identifier: "TEST-3", Title: "Test Task 3", State: "Done"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试获取所有任务
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "tasks")
	assert.Contains(t, response, "total_count")
	assert.Contains(t, response, "filter_label")
}

func TestAPIHandler_GetTasks_WithFilter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "TEST-1", Title: "Backlog Task", State: "Todo"},
		{ID: "2", Identifier: "TEST-2", Title: "Active Task", State: "In Progress"},
		{ID: "3", Identifier: "TEST-3", Title: "Done Task", State: "Done"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试筛选 backlog 状态
	req := httptest.NewRequest("GET", "/api/v1/tasks?state=backlog", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "backlog", response["filter"])
	assert.Equal(t, "待开始", response["filter_label"])
}

func TestAPIHandler_GetTasks_MultipleFilters(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "TEST-1", Title: "Backlog Task", State: "Todo"},
		{ID: "2", Identifier: "TEST-2", Title: "Active Task", State: "In Progress"},
		{ID: "3", Identifier: "TEST-3", Title: "Done Task", State: "Done"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试多状态筛选
	req := httptest.NewRequest("GET", "/api/v1/tasks?state=backlog,active", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["filter"], "backlog")
}

func TestAPIHandler_CancelTask_NotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试取消不存在的任务
	req := httptest.NewRequest("POST", "/api/v1/NONEXISTENT-123/cancel", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestAPIHandler_CancelConfirm_NotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试获取不存在的任务确认信息
	req := httptest.NewRequest("GET", "/api/v1/NONEXISTENT-123/cancel/confirm", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}
func TestTaskHandler_NewTaskForm(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/new", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "创建新需求")
	assert.Contains(t, w.Body.String(), "需求标题")
	assert.Contains(t, w.Body.String(), "需求描述")
}

func TestAPIHandler_CreateTask_Success(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 构建请求体
	reqBody := map[string]string{
		"title":       "Test Task",
		"description": "This is a test task description",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "parent_task")
	assert.Contains(t, response, "sub_tasks")
	assert.Contains(t, response, "message")

	// 验证父任务信息
	parentTask := response["parent_task"].(map[string]interface{})
	assert.NotEmpty(t, parentTask["identifier"])
	assert.Equal(t, "Test Task", parentTask["title"])

	// 验证子任务数量
	subTasks := response["sub_tasks"].([]interface{})
	assert.GreaterOrEqual(t, len(subTasks), 1)
}

func TestAPIHandler_CreateTask_ValidationError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试缺少 title
	reqBody := map[string]string{
		"description": "This is a test task description",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_CreateTask_EmptyFields(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 测试空字符串
	reqBody := map[string]string{
		"title":       "",
		"description": "",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestAPIHandler_CreateTask_HTMLResponse(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 构建请求体
	reqBody := map[string]string{
		"title":       "HTML Test Task",
		"description": "Testing HTML response",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "需求创建成功")
	assert.Contains(t, w.Body.String(), "HTML Test Task")
}

func TestTaskHandler_TaskDetail_NotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/NONEXISTENT-123", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "任务不存在")
}

func TestTaskHandler_TaskDetail_Success(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{
			ID:          "1",
			Identifier:  "TEST-123",
			Title:       "测试任务详情",
			Description: "这是一个测试任务的描述内容",
			State:       "In Progress",
		},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/TEST-123", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "任务详情")
	assert.Contains(t, w.Body.String(), "TEST-123")
	assert.Contains(t, w.Body.String(), "测试任务详情")
	assert.Contains(t, w.Body.String(), "澄清进度")
	assert.Contains(t, w.Body.String(), "历史问答记录")
}

func TestTaskHandler_TaskDetail_WithDescription(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{
			ID:          "1",
			Identifier:  "DESC-001",
			Title:       "带描述的任务",
			Description: "详细描述：实现用户登录功能，支持邮箱和手机号登录",
			State:       "Todo",
		},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/DESC-001", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "带描述的任务")
	assert.Contains(t, w.Body.String(), "详细描述：实现用户登录功能")
	assert.Contains(t, w.Body.String(), "任务描述")
}

func TestTaskHandler_TaskDetail_NoDescription(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{
			ID:         "1",
			Identifier: "NDESC-001",
			Title:      "无描述任务",
			State:      "Todo",
		},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/NDESC-001", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "无描述任务")
	// 任务描述区域不会被渲染（没有 h3 标签显示"任务描述"作为标题）
	assert.NotContains(t, w.Body.String(), "<h3 style=\"font-size: 1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;\">任务描述</h3>")
}

// BDD 审核相关测试
func TestAPIHandler_ApproveBDD_BDDReviewNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "BDD-001", Title: "BDD Test Task", State: "Todo"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/BDD-001/bdd/approve", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 BDD 审核管理器，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_RejectBDD_BDDReviewNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "BDD-002", Title: "BDD Test Task", State: "Todo"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	reqBody := map[string]string{"reason": "测试驳回"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/tasks/BDD-002/bdd/reject", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 BDD 审核管理器，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_GetBDDReviewStatus_BDDReviewNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "BDD-003", Title: "BDD Test Task", State: "Todo"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/tasks/BDD-003/bdd", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 BDD 审核管理器，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_ApproveBDD_TaskNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/NONEXISTENT/bdd/approve", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 BDDReviewManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestAPIHandler_RejectBDD_TaskNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	reqBody := map[string]string{"reason": "测试驳回"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/tasks/NONEXISTENT/bdd/reject", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 BDDReviewManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestAPIHandler_GetBDDReviewStatus_TaskNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/tasks/NONEXISTENT/bdd", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 BDDReviewManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Epic 8: 异常处理与人工干预测试 ==========

func TestAPIHandler_GetNeedsAttentionStatus_NeedsAttentionNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "NA-001", Title: "Needs Attention Task", State: "Needs Attention"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/tasks/NA-001/needs-attention", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "needs_attention_not_supported", errorObj["code"])
}

// TestAPIHandler_GetNeedsAttentionStatus_TaskNotFound 测试获取不存在任务的待处理状态
// 注意：由于没有 NeedsAttentionManager，会先返回 500（功能不可用）
// 这是预期行为，因为 TaskNotFound 检查在获取 taskID 之后
func TestAPIHandler_GetNeedsAttentionStatus_NoManager(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/tasks/NONEXISTENT/needs-attention", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "needs_attention_not_supported", errorObj["code"])
}

func TestAPIHandler_ResumeTask_NeedsAttentionNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "RESUME-001", Title: "Resume Task", State: "Needs Attention"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/RESUME-001/resume", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_ResumeTask_NoManager(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/NONEXISTENT/resume", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_ReclarifyTask_NeedsAttentionNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "RECLARIFY-001", Title: "Reclarify Task", State: "Needs Attention"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/RECLARIFY-001/reclarify", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_ReclarifyTask_NoManager(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/NONEXISTENT/reclarify", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_AbandonTask_NeedsAttentionNotSupported(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "ABANDON-001", Title: "Abandon Task", State: "Needs Attention"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/ABANDON-001/abandon", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_AbandonTask_NoManager(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("POST", "/api/tasks/NONEXISTENT/abandon", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 没有 NeedsAttentionManager，应该返回 500（功能不可用）
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_AbandonConfirm_TaskNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/tasks/NONEXISTENT/abandon/confirm", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIHandler_AbandonConfirm_Success(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{ID: "1", Identifier: "CONFIRM-001", Title: "Confirm Abandon Task", State: "Needs Attention"},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/api/tasks/CONFIRM-001/abandon/confirm", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "CONFIRM-001", response["identifier"])
	assert.Equal(t, "Confirm Abandon Task", response["title"])
	assert.True(t, response["requires_confirm"].(bool))
	assert.Contains(t, response["warning"], "放弃操作不可逆")
}

func TestTaskHandler_NeedsAttentionPage_TaskNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/NONEXISTENT/needs-attention", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "任务不存在")
}

func TestTaskHandler_NeedsAttentionPage_Success(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{
			ID:          "1",
			Identifier:  "NA-PAGE-001",
			Title:       "需要人工干预的任务",
			Description: "这是一个测试任务的描述",
			State:       "Needs Attention",
		},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/NA-PAGE-001/needs-attention", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "待人工处理")
	assert.Contains(t, w.Body.String(), "NA-PAGE-001")
	assert.Contains(t, w.Body.String(), "需要人工干预的任务")
}

func TestTaskHandler_NeedsAttentionPage_HTMXResponse(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Tracker.MockIssues = []config.MockIssueConfig{
		{
			ID:          "1",
			Identifier:  "NA-HTMX-001",
			Title:       "HTMX 测试任务",
			Description: "测试 HTMX 请求",
			State:       "Needs Attention",
		},
	}
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	req := httptest.NewRequest("GET", "/tasks/NA-HTMX-001/needs-attention", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "NA-HTMX-001")
}

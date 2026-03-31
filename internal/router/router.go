package router

import (
	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/server/handlers"
	"github.com/dministrator/symphony/internal/server/static"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/gin-gonic/gin"
)

// SetupRouter 设置所有路由
func SetupRouter(orchestrator *orchestrator.Orchestrator, broadcaster *common.SSEBroadcaster, engine *gin.Engine) {
	// 静态资源
	staticHandler := handlers.NewStaticHandler(static.DashboardFS)
	engine.GET("/dashboard.css", staticHandler.HandleDashboardCSS)

	// 主页 - 仪表板
	dashboardHandler := handlers.NewDashboardHandler(orchestrator)
	engine.GET("/", dashboardHandler.Handle)

	// 任务创建表单页面
	taskHandler := handlers.NewTaskHandler()
	engine.GET("/tasks/new", taskHandler.HandleNewTaskForm)

	// 任务详情页面（需要 tracker）
	taskDetailHandler := handlers.NewTaskHandlerWithTracker(orchestrator.GetTracker())
	engine.GET("/tasks/:identifier", taskDetailHandler.HandleTaskDetail)

	// BDD 规则审核页面
	engine.GET("/tasks/:identifier/bdd", taskDetailHandler.HandleBDDReviewPage)

	// 架构设计审核页面（Epic 5）
	engine.GET("/tasks/:identifier/architecture", taskDetailHandler.HandleArchitectureReviewPage)

	// 验收报告页面
	engine.GET("/tasks/:identifier/verification", taskDetailHandler.HandleVerificationPage)

	// 待人工处理页面（Epic 8）
	engine.GET("/tasks/:identifier/needs-attention", taskDetailHandler.HandleNeedsAttentionPage)

	// SSE 端点
	sseHandler := handlers.NewSSEHandler(broadcaster)
	engine.GET("/events", sseHandler.Handle)

	// API 路由
	apiHandler := handlers.NewAPIHandlerWithCanceler(orchestrator, orchestrator)
	api := engine.Group("/api/v1")
	{
		api.GET("/state", apiHandler.HandleGetState)
		api.GET("/tasks", apiHandler.HandleGetTasks) // 任务列表（支持状态筛选）
		api.GET("/:identifier", apiHandler.HandleGetIssue)
		api.POST("/refresh", apiHandler.HandleRefresh)
		// 取消任务相关路由
		api.GET("/:identifier/cancel/confirm", apiHandler.HandleCancelConfirm)
		api.POST("/:identifier/cancel", apiHandler.HandleCancelTask)
		// 澄清相关路由（当有澄清管理器时可用）
		api.GET("/:identifier/clarification", apiHandler.HandleGetClarificationStatus)
		api.POST("/:identifier/skip", apiHandler.HandleSkipClarification)
		// 执行进度和日志 API
		api.GET("/:identifier/progress", handlers.NewExecutionHandler(orchestrator.GetTracker(), nil, orchestrator.GetWorkflowEngine()).HandleGetProgress)
		api.GET("/:identifier/logs", handlers.NewExecutionHandler(orchestrator.GetTracker(), nil, orchestrator.GetWorkflowEngine()).HandleGetLogs)
		api.GET("/:identifier/status", handlers.NewExecutionHandler(orchestrator.GetTracker(), nil, orchestrator.GetWorkflowEngine()).HandleGetStatusDetail)
	}

	// 任务创建 API（不带 v1 前缀，简化 HTMX 调用）
	engine.POST("/api/tasks", apiHandler.HandleCreateTask)
	// 跳过澄清 API（不带 v1 前缀）
	engine.POST("/api/tasks/:identifier/skip", apiHandler.HandleSkipClarification)
	// 提交回答 API（不带 v1 前缀）
	engine.POST("/api/tasks/:identifier/answer", apiHandler.HandleSubmitAnswer)
	// 获取澄清状态 API（不带 v1 前缀）
	engine.GET("/api/tasks/:identifier/clarification", apiHandler.HandleGetClarificationState)
	// BDD 审核相关 API（不带 v1 前缀）
	engine.POST("/api/tasks/:identifier/bdd/approve", apiHandler.HandleApproveBDD)
	engine.POST("/api/tasks/:identifier/bdd/reject", apiHandler.HandleRejectBDD)
	engine.GET("/api/tasks/:identifier/bdd", apiHandler.HandleGetBDDReviewStatus)

	// 架构审核相关 API（不带 v1 前缀，Epic 5）
	engine.POST("/api/tasks/:identifier/architecture/approve", apiHandler.HandleApproveArchitecture)
	engine.POST("/api/tasks/:identifier/architecture/reject", apiHandler.HandleRejectArchitecture)
	engine.GET("/api/tasks/:identifier/architecture", apiHandler.HandleGetArchitectureReviewStatus)

	// 验收审核相关 API（不带 v1 前缀）
	engine.POST("/api/tasks/:identifier/verification/approve", apiHandler.HandleApproveVerification)
	engine.POST("/api/tasks/:identifier/verification/reject", apiHandler.HandleRejectVerification)
	engine.GET("/api/tasks/:identifier/verification", apiHandler.HandleGetVerificationStatus)

	// Epic 8: 异常处理与人工干预 API
	engine.GET("/api/tasks/:identifier/needs-attention", apiHandler.HandleGetNeedsAttentionStatus)
	engine.POST("/api/tasks/:identifier/resume", apiHandler.HandleResumeTask)
	engine.POST("/api/tasks/:identifier/reclarify", apiHandler.HandleReclarifyTask)
	engine.GET("/api/tasks/:identifier/abandon/confirm", apiHandler.HandleAbandonConfirm)
	engine.POST("/api/tasks/:identifier/abandon", apiHandler.HandleAbandonTask)
}

// SetupRouterWithExecution 设置路由（包含执行管理器）
func SetupRouterWithExecution(orchestrator *orchestrator.Orchestrator, broadcaster *common.SSEBroadcaster, engine *gin.Engine, execHandler *handlers.ExecutionHandler) {
	// 静态资源
	staticHandler := handlers.NewStaticHandler(static.DashboardFS)
	engine.GET("/dashboard.css", staticHandler.HandleDashboardCSS)

	// 主页 - 仪表板
	dashboardHandler := handlers.NewDashboardHandler(orchestrator)
	engine.GET("/", dashboardHandler.Handle)

	// 任务创建表单页面
	taskHandler := handlers.NewTaskHandler()
	engine.GET("/tasks/new", taskHandler.HandleNewTaskForm)

	// 任务详情页面（需要 tracker）
	taskDetailHandler := handlers.NewTaskHandlerWithTracker(orchestrator.GetTracker())
	engine.GET("/tasks/:identifier", taskDetailHandler.HandleTaskDetail)

	// BDD 规则审核页面
	engine.GET("/tasks/:identifier/bdd", taskDetailHandler.HandleBDDReviewPage)

	// 架构设计审核页面（Epic 5）
	engine.GET("/tasks/:identifier/architecture", taskDetailHandler.HandleArchitectureReviewPage)

	// 验收报告页面
	engine.GET("/tasks/:identifier/verification", taskDetailHandler.HandleVerificationPage)

	// 待人工处理页面（Epic 8）
	engine.GET("/tasks/:identifier/needs-attention", taskDetailHandler.HandleNeedsAttentionPage)

	// 执行日志页面
	if execHandler != nil {
		engine.GET("/tasks/:identifier/logs", execHandler.HandleGetLogsPage)
	}

	// SSE 端点
	sseHandler := handlers.NewSSEHandler(broadcaster)
	engine.GET("/events", sseHandler.Handle)

	// API 路由
	apiHandler := handlers.NewAPIHandlerWithCanceler(orchestrator, orchestrator)
	api := engine.Group("/api/v1")
	{
		api.GET("/state", apiHandler.HandleGetState)
		api.GET("/tasks", apiHandler.HandleGetTasks)
		api.GET("/:identifier", apiHandler.HandleGetIssue)
		api.POST("/refresh", apiHandler.HandleRefresh)
		api.GET("/:identifier/cancel/confirm", apiHandler.HandleCancelConfirm)
		api.POST("/:identifier/cancel", apiHandler.HandleCancelTask)
		api.GET("/:identifier/clarification", apiHandler.HandleGetClarificationStatus)
		api.POST("/:identifier/skip", apiHandler.HandleSkipClarification)
		// 执行进度和日志 API
		if execHandler != nil {
			api.GET("/:identifier/progress", execHandler.HandleGetProgress)
			api.GET("/:identifier/logs", execHandler.HandleGetLogs)
			api.GET("/:identifier/status", execHandler.HandleGetStatusDetail)
		}
	}

	// 任务创建 API
	engine.POST("/api/tasks", apiHandler.HandleCreateTask)
	engine.POST("/api/tasks/:identifier/skip", apiHandler.HandleSkipClarification)
	engine.POST("/api/tasks/:identifier/answer", apiHandler.HandleSubmitAnswer)
	engine.GET("/api/tasks/:identifier/clarification", apiHandler.HandleGetClarificationState)
	engine.POST("/api/tasks/:identifier/bdd/approve", apiHandler.HandleApproveBDD)
	engine.POST("/api/tasks/:identifier/bdd/reject", apiHandler.HandleRejectBDD)
	engine.GET("/api/tasks/:identifier/bdd", apiHandler.HandleGetBDDReviewStatus)

	// 架构审核相关 API（Epic 5）
	engine.POST("/api/tasks/:identifier/architecture/approve", apiHandler.HandleApproveArchitecture)
	engine.POST("/api/tasks/:identifier/architecture/reject", apiHandler.HandleRejectArchitecture)
	engine.GET("/api/tasks/:identifier/architecture", apiHandler.HandleGetArchitectureReviewStatus)

	// 验收审核相关 API
	engine.POST("/api/tasks/:identifier/verification/approve", apiHandler.HandleApproveVerification)
	engine.POST("/api/tasks/:identifier/verification/reject", apiHandler.HandleRejectVerification)
	engine.GET("/api/tasks/:identifier/verification", apiHandler.HandleGetVerificationStatus)

	// Epic 8: 异常处理与人工干预 API
	engine.GET("/api/tasks/:identifier/needs-attention", apiHandler.HandleGetNeedsAttentionStatus)
	engine.POST("/api/tasks/:identifier/resume", apiHandler.HandleResumeTask)
	engine.POST("/api/tasks/:identifier/reclarify", apiHandler.HandleReclarifyTask)
	engine.GET("/api/tasks/:identifier/abandon/confirm", apiHandler.HandleAbandonConfirm)
	engine.POST("/api/tasks/:identifier/abandon", apiHandler.HandleAbandonTask)
}

// BuildRouter 构建并返回路由器（用于测试）
func BuildRouter(orchestrator *orchestrator.Orchestrator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	broadcaster := common.NewSSEBroadcaster()

	SetupRouter(orchestrator, broadcaster, engine)
	return engine
}

// BuildRouterWithWorkflow 构建并返回路由器（包含工作流引擎）
func BuildRouterWithWorkflow(orchestrator *orchestrator.Orchestrator, wfEngine *workflow.Engine) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	broadcaster := common.NewSSEBroadcaster()

	execHandler := handlers.NewExecutionHandler(orchestrator.GetTracker(), nil, wfEngine)
	SetupRouterWithExecution(orchestrator, broadcaster, engine, execHandler)
	return engine
}

package router

import (
	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/server/handlers"
	"github.com/dministrator/symphony/internal/server/static"
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
}

// BuildRouter 构建并返回路由器（用于测试）
func BuildRouter(orchestrator *orchestrator.Orchestrator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	broadcaster := common.NewSSEBroadcaster()

	SetupRouter(orchestrator, broadcaster, engine)
	return engine
}

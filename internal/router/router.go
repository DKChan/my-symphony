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

	// SSE 端点
	sseHandler := handlers.NewSSEHandler(broadcaster)
	engine.GET("/events", sseHandler.Handle)

	// API 路由
	apiHandler := handlers.NewAPIHandler(orchestrator)
	api := engine.Group("/api/v1")
	{
		api.GET("/state", apiHandler.HandleGetState)
		api.GET("/:identifier", apiHandler.HandleGetIssue)
		api.POST("/refresh", apiHandler.HandleRefresh)
	}
}

// BuildRouter 构建并返回路由器（用于测试）
func BuildRouter(orchestrator *orchestrator.Orchestrator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	broadcaster := common.NewSSEBroadcaster()

	SetupRouter(orchestrator, broadcaster, engine)
	return engine
}

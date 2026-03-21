package handlers

import (
	"net/http"
	"time"

	"github.com/dministrator/symphony/internal/server/components"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 仪表板页面处理器
type DashboardHandler struct {
	orchestrator OrchestratorGetter
}

// NewDashboardHandler 创建新的仪表板处理器
func NewDashboardHandler(orch OrchestratorGetter) *DashboardHandler {
	return &DashboardHandler{
		orchestrator: orch,
	}
}

// Handle 处理仪表板页面请求
func (h *DashboardHandler) Handle(c *gin.Context) {
	state := h.orchestrator.GetState()
	now := time.Now()

	html := components.RenderDashboardHTML(state, now)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

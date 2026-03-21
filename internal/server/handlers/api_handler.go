package handlers

import (
	"net/http"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/presenter"
	"github.com/gin-gonic/gin"
)

// OrchestratorGetter 定义获取 orchestrator 状态的接口
type OrchestratorGetter interface {
	GetState() *domain.OrchestratorState
}

// APIHandler API 处理器，提供状态、问题和刷新相关的 API 端点
type APIHandler struct {
	orchestrator OrchestratorGetter
}

// NewAPIHandler 创建新的 API 处理器
func NewAPIHandler(orch OrchestratorGetter) *APIHandler {
	return &APIHandler{
		orchestrator: orch,
	}
}

// HandleGetState 处理获取状态的请求
func (h *APIHandler) HandleGetState(c *gin.Context) {
	state := h.orchestrator.GetState()
	payload := presenter.BuildStatePayload(state)
	c.JSON(http.StatusOK, payload)
}

// HandleGetIssue 处理获取单个问题的请求
func (h *APIHandler) HandleGetIssue(c *gin.Context) {
	identifier := c.Param("identifier")
	state := h.orchestrator.GetState()

	response, err := presenter.BuildIssuePayload(identifier, state)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "issue_not_found",
				"message": "issue not found in current state",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleRefresh 处理刷新请求
func (h *APIHandler) HandleRefresh(c *gin.Context) {
	response := presenter.BuildRefreshPayload()
	c.JSON(http.StatusAccepted, response)
}

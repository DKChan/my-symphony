package handlers

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/dministrator/symphony/internal/server/static"
	"github.com/gin-gonic/gin"
)

// StaticHandler 静态资源处理器
type StaticHandler struct {
	fs fs.FS
}

// NewStaticHandler 创建新的静态资源处理器
func NewStaticHandler(embedFS embed.FS) *StaticHandler {
	return &StaticHandler{
		fs: embedFS,
	}
}

// HandleDashboardCSS 处理 CSS 文件请求
func (h *StaticHandler) HandleDashboardCSS(c *gin.Context) {
	c.Header("Content-Type", "text/css; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")

	// 从嵌入的文件系统读取 CSS
	data, err := static.DashboardFS.ReadFile("dashboard.css")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load CSS")
		return
	}

	c.Data(http.StatusOK, "text/css; charset=utf-8", data)
}

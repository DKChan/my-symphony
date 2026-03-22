// Package server 提供HTTP服务器实现
package server

import (
	"strconv"

	"github.com/dministrator/symphony/internal/router"
	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/server/presenter"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/gin-gonic/gin"
)

// Server HTTP服务器
type Server struct {
	orchestrator *orchestrator.Orchestrator
	port         int
	engine       *gin.Engine

	// SSE 广播器
	broadcaster *common.SSEBroadcaster
}

// NewServer 创建新的HTTP服务器
func NewServer(orch *orchestrator.Orchestrator, port int) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{
		orchestrator: orch,
		port:         port,
		engine:       engine,
		broadcaster:  common.NewSSEBroadcaster(),
	}

	// 设置状态变更回调
	orch.SetOnStateChange(s.onStateChange)

	// 设置路由
	router.SetupRouter(s.orchestrator, s.broadcaster, s.engine)
	return s
}

// onStateChange 状态变更回调
func (s *Server) onStateChange() {
	state := s.orchestrator.GetState()
	payload := presenter.BuildStatePayload(state)
	s.broadcaster.Broadcast("state", payload)
}

// Run 运行服务器
func (s *Server) Run() error {
	return s.engine.Run(":" + strconv.Itoa(s.port))
}

// Port 返回服务器端口
func (s *Server) Port() int {
	return s.port
}

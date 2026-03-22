package server_test

import (
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	srv := server.NewServer(orch, 8080)

	assert.NotNil(t, srv)
	assert.Equal(t, 8080, srv.Port())
}

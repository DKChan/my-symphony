package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/router"
	"github.com/stretchr/testify/assert"
)

func TestBuildRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")

	engine := router.BuildRouter(orch)

	assert.NotNil(t, engine)
}

func TestRouterEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"GET dashboard", "GET", "/", http.StatusOK},
		{"GET dashboard.css", "GET", "/dashboard.css", http.StatusOK},
		{"GET api state", "GET", "/api/v1/state", http.StatusOK},
		{"GET api refresh", "POST", "/api/v1/refresh", http.StatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

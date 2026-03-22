package handlers_test

import (
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
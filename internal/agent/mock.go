// Package agent 提供编码代理运行器接口和工厂
package agent

import (
	"context"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// mockRunner Mock代理运行器，用于测试时模拟agent执行
type mockRunner struct {
	cfg         *config.Config
	mu          sync.Mutex
	stageTimers map[string]*time.Timer
}

// newMockRunner 创建新的Mock运行器
func newMockRunner(cfg *config.Config) Runner {
	return &mockRunner{
		cfg:         cfg,
		stageTimers: make(map[string]*time.Timer),
	}
}

// RunAttempt 执行一次模拟运行
// 模拟agent执行，10秒后返回成功
func (r *mockRunner) RunAttempt(
	ctx context.Context,
	issue *domain.Issue,
	workspacePath string,
	attempt *int,
	promptTemplate string,
	callback EventCallback,
) (*RunAttemptResult, error) {
	// 发送会话开始事件
	if callback != nil {
		callback("session_started", map[string]any{
			"session_id": "mock-session-" + issue.ID,
			"issue_id":   issue.ID,
		})
	}

	// 使用通道等待模拟完成或取消
	done := make(chan struct{})

	// 创建10秒定时器模拟agent执行
	r.mu.Lock()
	timer := time.NewTimer(10 * time.Second)
	r.stageTimers[issue.ID] = timer
	r.mu.Unlock()

	// 清理定时器
	defer func() {
		r.mu.Lock()
		if t, ok := r.stageTimers[issue.ID]; ok {
			t.Stop()
			delete(r.stageTimers, issue.ID)
		}
		r.mu.Unlock()
	}()

	// 模拟进度更新 (每2秒发送一次)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		turnCount := 0
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				turnCount++
				if callback != nil {
					callback("turn_complete", map[string]any{
						"turn_count": turnCount,
						"message":    "Mock agent progress simulation",
						"usage": map[string]any{
							"input_tokens":  1000 * turnCount,
							"output_tokens": 500 * turnCount,
							"total_tokens":  1500 * turnCount,
						},
					})
				}
			}
		}
	}()

	// 等待完成或取消
	select {
	case <-ctx.Done():
		close(done)
		return &RunAttemptResult{
			Success:   false,
			Error:     ctx.Err(),
			TurnCount: 0,
		}, nil
	case <-timer.C:
		close(done)
		// 发送最终完成事件
		if callback != nil {
			callback("result", map[string]any{
				"success":    true,
				"turn_count": 5,
				"usage": map[string]any{
					"input_tokens":  5000,
					"output_tokens": 2500,
					"total_tokens":  7500,
				},
			})
		}
		return &RunAttemptResult{
			Success:    true,
			Error:      nil,
			TurnCount:  5,
			TokenUsage: &TokenUsage{InputTokens: 5000, OutputTokens: 2500, TotalTokens: 7500},
		}, nil
	}
}
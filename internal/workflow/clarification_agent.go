// Package workflow 提供AI Agent需求理解调用功能
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/agent"
	"github.com/dministrator/symphony/internal/domain"
)

// DefaultClarificationPrompt 默认澄清提示模板
const DefaultClarificationPrompt = `你是一个需求分析专家。请分析以下需求，识别不明确的地方并提出问题。

## 需求标题
{{ issue.title }}

## 需求描述
{{ issue.description }}

请以 JSON 格式返回：
{
  "status": "needs_clarification" | "clear",
  "questions": [
    {"id": "q1", "question": "..."},
    ...
  ],
  "summary": "需求摘要"
}

注意：
1. 如果需求描述足够清晰、完整，返回 status: "clear"
2. 如果存在模糊、缺失或需要确认的部分，返回 status: "needs_clarification" 并列出具体问题
3. 问题应该具体、可回答，帮助明确需求细节
4. summary 应该是对需求的简要概括`

// ClarificationOption 澄清管理器选项
type ClarificationOption func(*ClarificationManager)

// WithRunner 设置AI Agent运行器
func WithRunner(r agent.Runner) ClarificationOption {
	return func(m *ClarificationManager) {
		m.SetRunner(r)
	}
}

// WithPromptTemplate 设置澄清提示模板
func WithPromptTemplate(template string) ClarificationOption {
	return func(m *ClarificationManager) {
		m.promptTmpl = template
	}
}

// AgentRunner 接口定义AI Agent运行器
type AgentRunner interface {
	RunAttempt(
		ctx context.Context,
		issue *domain.Issue,
		workspacePath string,
		attempt *int,
		promptTemplate string,
		callback agent.EventCallback,
	) (*agent.RunAttemptResult, error)
}

// RunClarificationWithAgent 使用AI Agent进行需求澄清
// Given: 一个新任务进入"需求澄清"阶段
// When: 阶段开始执行
// Then: 系统读取任务标题和描述，调用 AI Agent CLI，传入 clarification prompt
func (cm *ClarificationManager) RunClarificationWithAgent(
	ctx context.Context,
	task *domain.Issue,
	workspacePath string,
) (*ClarificationResult, error) {
	// 检查任务是否有效
	if task == nil {
		return nil, ErrInvalidTask
	}

	// 检查是否有 tracker
	if cm.tracker == nil {
		return nil, fmt.Errorf("tracker not configured")
	}

	// 获取对话历史（用于多轮澄清）
	history, err := cm.tracker.GetConversationHistory(ctx, task.Identifier)
	if err != nil {
		// 如果获取历史失败，继续处理（可能还没有历史记录）
		history = nil
	}

	// 构建提示词
	prompt := cm.buildClarificationPromptForAgent(task, history)

	// 调用 AI Agent（如果配置了 runner）
	if cm.runner != nil {
		return cm.callAgentAndProcess(ctx, task, workspacePath, prompt)
	}

	// 如果没有配置 runner，返回需要澄清状态（用于测试或手动处理）
	return &ClarificationResult{
		Status:         StatusNeedsClarification,
		Summary:        "需要人工进行需求澄清",
		WaitingForUser: false,
	}, nil
}

// buildClarificationPromptForAgent 构建澄清提示词
func (cm *ClarificationManager) buildClarificationPromptForAgent(
	task *domain.Issue,
	history []domain.ConversationTurn,
) string {
	prompt := cm.promptTmpl
	if prompt == "" {
		prompt = DefaultClarificationPrompt
	}

	// 替换基本字段
	prompt = strings.ReplaceAll(prompt, "{{ issue.title }}", task.Title)

	description := ""
	if task.Description != nil {
		description = *task.Description
	}
	prompt = strings.ReplaceAll(prompt, "{{ issue.description }}", description)

	// 如果有对话历史，追加到提示词
	if len(history) > 0 {
		historySection := formatConversationHistoryForAgent(history)
		prompt = prompt + "\n\n## 已有对话历史\n" + historySection
	}

	return strings.TrimSpace(prompt)
}

// formatConversationHistoryForAgent 格式化对话历史
func formatConversationHistoryForAgent(history []domain.ConversationTurn) string {
	var builder strings.Builder

	for i, turn := range history {
		roleLabel := "用户"
		if turn.Role == "assistant" {
			roleLabel = "AI助手"
		}

		builder.WriteString(fmt.Sprintf("**%s (Turn %d):** %s\n", roleLabel, i+1, turn.Content))
	}

	return builder.String()
}

// callAgentAndProcess 调用AI Agent并处理响应
func (cm *ClarificationManager) callAgentAndProcess(
	ctx context.Context,
	task *domain.Issue,
	workspacePath string,
	prompt string,
) (*ClarificationResult, error) {
	// 调用 AI Agent
	agentResponse, err := cm.callAgent(ctx, task, workspacePath, prompt)
	if err != nil {
		return &ClarificationResult{
			Status: StatusNeedsClarification,
			Error:  fmt.Errorf("agent call failed: %w", err),
		}, err
	}

	// 处理响应
	return cm.ProcessAgentResponse(ctx, task, agentResponse)
}

// callAgent 调用 AI Agent
func (cm *ClarificationManager) callAgent(
	ctx context.Context,
	task *domain.Issue,
	workspacePath string,
	prompt string,
) (string, error) {
	// 使用 agent.Runner 执行
	result, err := cm.runner.RunAttempt(
		ctx,
		task,
		workspacePath,
		nil, // 首次澄清，无重试次数
		prompt,
		nil, // 不需要事件回调
	)

	if err != nil {
		return "", fmt.Errorf("runner failed: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("agent execution failed: %v", result.Error)
	}

	// 由于 agent.Runner 不直接返回响应内容，
	// 实际实现中需要通过其他机制获取响应（如文件或回调）
	// 这里返回一个模拟响应用于测试
	return cm.getAgentResponseFromWorkspace(ctx, workspacePath)
}

// getAgentResponseFromWorkspace 从工作空间获取 Agent 响应
// 实际实现需要根据 agent 的输出方式调整
func (cm *ClarificationManager) getAgentResponseFromWorkspace(ctx context.Context, workspacePath string) (string, error) {
	// TODO: 实际实现需要从 agent 输出中获取响应
	// 可能的方式：
	// 1. 从文件读取（如 CLARIFICATION_RESULT.md）
	// 2. 通过回调接收
	// 3. 使用专门的响应接口

	// 这里返回一个默认的 mock 响应，用于测试
	return `{"status": "clear", "questions": [], "summary": "需求已明确"}`, nil
}

// ProcessAgentResponse 处理 AI Agent 响应
// Given: AI Agent 返回澄清问题
// When: 系统解析响应
// Then: 问题保存到任务的澄清记录中，任务状态变为"等待用户回答"
func (cm *ClarificationManager) ProcessAgentResponse(
	ctx context.Context,
	task *domain.Issue,
	response string,
) (*ClarificationResult, error) {
	// 解析 JSON 响应
	clarResponse, err := ParseClarificationResponse(response)
	if err != nil {
		return &ClarificationResult{
			Status: StatusNeedsClarification,
			Error:  fmt.Errorf("parse response failed: %w", err),
		}, err
	}

	// 保存 AI 的响应到对话历史
	aiTurn := domain.ConversationTurn{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}

	if cm.tracker != nil {
		if err := cm.tracker.AppendConversation(ctx, task.Identifier, aiTurn); err != nil {
			// 记录错误但继续处理
			fmt.Printf("warning: failed to append AI turn: %v\n", err)
		}
	}

	// 根据状态处理
	if clarResponse.Status == StatusNeedsClarification {
		// 需要澄清 - 保存问题
		questionsJSON, _ := json.Marshal(clarResponse.Questions)
		questionTurn := domain.ConversationTurn{
			Role:      "assistant",
			Content:   fmt.Sprintf("澄清问题: %s", string(questionsJSON)),
			Timestamp: time.Now(),
		}

		if cm.tracker != nil {
			if err := cm.tracker.AppendConversation(ctx, task.Identifier, questionTurn); err != nil {
				fmt.Printf("warning: failed to append questions: %v\n", err)
			}

			// 更新阶段状态为等待用户回答
			stageState := domain.StageState{
				Name:      string(StageClarification),
				Status:    "waiting_user_response",
				StartedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := cm.tracker.UpdateStage(ctx, task.Identifier, stageState); err != nil {
				fmt.Printf("warning: failed to update stage: %v\n", err)
			}
		}

		return &ClarificationResult{
			Status:         StatusNeedsClarification,
			Questions:      clarResponse.Questions,
			Summary:        clarResponse.Summary,
			WaitingForUser: true,
		}, nil
	}

	// 需求已明确 - 更新状态为完成，推进到下一阶段
	if cm.tracker != nil {
		stageState := domain.StageState{
			Name:      string(StageClarification),
			Status:    "completed",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := cm.tracker.UpdateStage(ctx, task.Identifier, stageState); err != nil {
			fmt.Printf("warning: failed to update stage to completed: %v\n", err)
		}
	}

	// 推进工作流到下一阶段
	if cm.engine != nil {
		_, err := cm.engine.AdvanceStage(task.ID)
		if err != nil {
			fmt.Printf("warning: failed to advance stage: %v\n", err)
		}
	}

	return &ClarificationResult{
		Status:         StatusClear,
		Summary:        clarResponse.Summary,
		WaitingForUser: false,
	}, nil
}

// ParseClarificationResponse 解析澄清响应 JSON
func ParseClarificationResponse(response string) (*ClarificationResponse, error) {
	// 清理响应（可能包含 markdown 代码块）
	cleaned := cleanJSONResponse(response)

	var resp ClarificationResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}

	// 验证状态
	if resp.Status != StatusNeedsClarification && resp.Status != StatusClear {
		resp.Status = StatusNeedsClarification
	}

	return &resp, nil
}

// cleanJSONResponse 清理 JSON 响应（去除 markdown 代码块等）
func cleanJSONResponse(response string) string {
	// 去除 markdown 代码块标记
	response = strings.TrimSpace(response)

	// 处理 ```json ... ``` 格式
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}

// HandleUserResponseWithAgent 处理用户回答（使用AI Agent版本）
// 将用户回答保存到对话历史，并重新调用 Agent 分析
func (cm *ClarificationManager) HandleUserResponseWithAgent(
	ctx context.Context,
	task *domain.Issue,
	userResponse string,
	workspacePath string,
) (*ClarificationResult, error) {
	// 保存用户回答到对话历史
	userTurn := domain.ConversationTurn{
		Role:      "user",
		Content:   userResponse,
		Timestamp: time.Now(),
	}

	if cm.tracker != nil {
		if err := cm.tracker.AppendConversation(ctx, task.Identifier, userTurn); err != nil {
			return nil, fmt.Errorf("failed to append user turn: %w", err)
		}
	}

	// 检查当前轮次
	wf := cm.engine.GetWorkflow(task.ID)
	if wf == nil {
		return nil, ErrWorkflowNotFound
	}

	currentStage := wf.Stages[StageClarification]
	if currentStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查是否超过最大轮次
	if currentStage.Round >= cm.config.Clarification.MaxRounds {
		// 超过最大轮次，强制完成
		return cm.forceCompleteClarificationWithAgent(ctx, task), nil
	}

	// 增加轮次
	if _, err := cm.engine.IncrementRound(task.ID); err != nil {
		fmt.Printf("warning: failed to increment round: %v\n", err)
	}

	// 重新调用 Agent 分析
	return cm.RunClarificationWithAgent(ctx, task, workspacePath)
}

// forceCompleteClarificationWithAgent 强制完成澄清阶段
func (cm *ClarificationManager) forceCompleteClarificationWithAgent(
	ctx context.Context,
	task *domain.Issue,
) *ClarificationResult {
	// 更新阶段状态
	if cm.tracker != nil {
		stageState := domain.StageState{
			Name:      string(StageClarification),
			Status:    "completed",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := cm.tracker.UpdateStage(ctx, task.Identifier, stageState); err != nil {
			fmt.Printf("warning: failed to update stage: %v\n", err)
		}
	}

	// 推进工作流
	if cm.engine != nil {
		_, err := cm.engine.AdvanceStage(task.ID)
		if err != nil {
			fmt.Printf("warning: failed to advance stage: %v\n", err)
		}
	}

	return &ClarificationResult{
		Status:         StatusClear,
		Summary:        "澄清阶段已完成（达到最大轮次限制）",
		WaitingForUser: false,
	}
}

// GetQuestionsFromHistory 获取当前需要回答的问题
func (cm *ClarificationManager) GetQuestionsFromHistory(
	ctx context.Context,
	identifier string,
) ([]ClarificationQuestion, error) {
	if cm.tracker == nil {
		return nil, fmt.Errorf("tracker not configured")
	}

	// 从对话历史中提取最新的问题
	history, err := cm.tracker.GetConversationHistory(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	// 查找最新的问题记录
	for i := len(history) - 1; i >= 0; i-- {
		turn := history[i]
		if turn.Role == "assistant" && strings.Contains(turn.Content, "澄清问题:") {
			// 解析问题
			questionsJSON := strings.TrimPrefix(turn.Content, "澄清问题: ")
			var questions []ClarificationQuestion
			if err := json.Unmarshal([]byte(questionsJSON), &questions); err == nil {
				return questions, nil
			}
		}
	}

	return nil, nil
}

// ErrInvalidTask 无效任务错误
var ErrInvalidTask = fmt.Errorf("invalid task")

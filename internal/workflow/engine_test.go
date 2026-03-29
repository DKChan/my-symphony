// Package workflow_test 测试工作流引擎
package workflow_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/workflow"
)

func TestStageConstants(t *testing.T) {
	// 验证阶段顺序
	expectedOrder := []workflow.StageName{
		workflow.StageClarification,
		workflow.StageBDDReview,
		workflow.StageArchitectureReview,
		workflow.StageImplementation,
		workflow.StageVerification,
	}

	if len(workflow.StageOrder) != len(expectedOrder) {
		t.Errorf("expected %d stages, got %d", len(expectedOrder), len(workflow.StageOrder))
	}

	for i, stage := range workflow.StageOrder {
		if stage != expectedOrder[i] {
			t.Errorf("stage at index %d: expected %s, got %s", i, expectedOrder[i], stage)
		}
	}
}

func TestStageStatusConstants(t *testing.T) {
	// 验证状态常量
	statuses := []workflow.StageStatus{
		workflow.StatusPending,
		workflow.StatusInProgress,
		workflow.StatusCompleted,
		workflow.StatusFailed,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("status should not be empty")
		}
	}
}

func TestGetStageName(t *testing.T) {
	tests := []struct {
		input    string
		expected workflow.StageName
		valid    bool
	}{
		{"clarification", workflow.StageClarification, true},
		{"bdd_review", workflow.StageBDDReview, true},
		{"architecture_review", workflow.StageArchitectureReview, true},
		{"implementation", workflow.StageImplementation, true},
		{"verification", workflow.StageVerification, true},
		{"invalid_stage", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			stage, ok := workflow.GetStageName(tt.input)
			if ok != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, ok)
			}
			if ok && stage != tt.expected {
				t.Errorf("expected stage %s, got %s", tt.expected, stage)
			}
		})
	}
}

func TestMustGetStageName(t *testing.T) {
	// 正常情况
	stage := workflow.MustGetStageName("clarification")
	if stage != workflow.StageClarification {
		t.Errorf("expected clarification, got %s", stage)
	}

	// 异常情况应该panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid stage name")
		}
	}()
	workflow.MustGetStageName("invalid")
}

func TestStageDisplayName(t *testing.T) {
	tests := []struct {
		stage    workflow.StageName
		expected string
	}{
		{workflow.StageClarification, "需求澄清"},
		{workflow.StageBDDReview, "BDD评审"},
		{workflow.StageArchitectureReview, "架构评审"},
		{workflow.StageImplementation, "实现"},
		{workflow.StageVerification, "验证"},
	}

	for _, tt := range tests {
		display := workflow.GetStageDisplayName(tt.stage)
		if display != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, display)
		}
	}
}

func TestStatusDisplayName(t *testing.T) {
	tests := []struct {
		status   workflow.StageStatus
		expected string
	}{
		{workflow.StatusPending, "待开始"},
		{workflow.StatusInProgress, "进行中"},
		{workflow.StatusCompleted, "已完成"},
		{workflow.StatusFailed, "失败"},
	}

	for _, tt := range tests {
		display := workflow.GetStatusDisplayName(tt.status)
		if display != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, display)
		}
	}
}

func TestStageStateIsTerminal(t *testing.T) {
	tests := []struct {
		status   workflow.StageStatus
		terminal bool
	}{
		{workflow.StatusPending, false},
		{workflow.StatusInProgress, false},
		{workflow.StatusCompleted, true},
		{workflow.StatusFailed, true},
	}

	for _, tt := range tests {
		state := &workflow.StageState{Status: tt.status}
		if state.IsTerminal() != tt.terminal {
			t.Errorf("status %s: expected terminal=%v, got %v", tt.status, tt.terminal, state.IsTerminal())
		}
	}
}

func TestNewWorkflowEngine(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestNewEngine(t *testing.T) {
	engine := workflow.NewEngine()
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestInitWorkflow(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-1"

	wf := engine.InitWorkflow(taskID)
	if wf == nil {
		t.Fatal("expected non-nil workflow")
	}

	// 验证初始状态
	if wf.TaskID != taskID {
		t.Errorf("expected taskID %s, got %s", taskID, wf.TaskID)
	}

	if wf.CurrentStage != workflow.StageClarification {
		t.Errorf("expected current stage clarification, got %s", wf.CurrentStage)
	}

	// 验证第一个阶段是进行中
	clarificationStage := wf.Stages[workflow.StageClarification]
	if clarificationStage.Status != workflow.StatusInProgress {
		t.Errorf("expected clarification stage in progress, got %s", clarificationStage.Status)
	}

	// 验证其他阶段是待开始
	for _, name := range workflow.StageOrder[1:] {
		stage := wf.Stages[name]
		if stage.Status != workflow.StatusPending {
			t.Errorf("stage %s: expected pending, got %s", name, stage.Status)
		}
	}
}

func TestInitTask(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "test-task-2"

	wf, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	if wf == nil {
		t.Fatal("expected non-nil workflow")
	}

	// 验证初始状态
	if wf.TaskID != taskID {
		t.Errorf("expected taskID %s, got %s", taskID, wf.TaskID)
	}

	// 验证第一个阶段是进行中
	clarificationStage := wf.Stages[workflow.StageClarification]
	if clarificationStage.Status != workflow.StatusInProgress {
		t.Errorf("expected clarification stage in progress, got %s", clarificationStage.Status)
	}

	// 重复初始化应该失败
	_, err = engine.InitTask(taskID)
	if err == nil {
		t.Error("expected error for duplicate init")
	}
}

func TestAdvanceStage(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-3"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 推进阶段
	wf, err := engine.AdvanceStage(taskID)
	if err != nil {
		t.Fatalf("failed to advance stage: %v", err)
	}

	// 验证当前阶段变为BDD评审
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected current stage bdd_review, got %s", wf.CurrentStage)
	}

	// 验证clarification已完成
	clarificationStage := wf.Stages[workflow.StageClarification]
	if clarificationStage.Status != workflow.StatusCompleted {
		t.Errorf("expected clarification completed, got %s", clarificationStage.Status)
	}

	// 验证BDD评审是进行中
	bddStage := wf.Stages[workflow.StageBDDReview]
	if bddStage.Status != workflow.StatusInProgress {
		t.Errorf("expected bdd_review in progress, got %s", bddStage.Status)
	}
}

func TestAdvanceStageSequence(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-4"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 推进所有阶段
	expectedStages := []workflow.StageName{
		workflow.StageBDDReview,
		workflow.StageArchitectureReview,
		workflow.StageImplementation,
		workflow.StageVerification,
	}

	for _, expectedStage := range expectedStages {
		wf, err := engine.AdvanceStage(taskID)
		if err != nil {
			t.Fatalf("failed to advance to %s: %v", expectedStage, err)
		}

		if wf.CurrentStage != expectedStage {
			t.Errorf("expected stage %s, got %s", expectedStage, wf.CurrentStage)
		}
	}

	// 最后一次推进（verification完成）
	wf, err := engine.AdvanceStage(taskID)
	if err != nil {
		t.Fatalf("failed to advance from verification: %v", err)
	}

	// 验证verification已完成
	verificationStage := wf.Stages[workflow.StageVerification]
	if verificationStage.Status != workflow.StatusCompleted {
		t.Errorf("expected verification completed, got %s", verificationStage.Status)
	}

	// 工作流应该完成
	if !wf.IsComplete() {
		t.Error("workflow should be complete")
	}
}

func TestAdvanceStageNotFound(t *testing.T) {
	engine := workflow.NewWorkflowEngine()

	_, err := engine.AdvanceStage("non-existent-task")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestFailStage(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-5"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 标记失败
	reason := "需求澄清失败：无法获取用户需求"
	wf, err := engine.FailStage(taskID, reason)
	if err != nil {
		t.Fatalf("failed to fail stage: %v", err)
	}

	// 验证失败状态
	clarificationStage := wf.Stages[workflow.StageClarification]
	if clarificationStage.Status != workflow.StatusFailed {
		t.Errorf("expected clarification failed, got %s", clarificationStage.Status)
	}

	if clarificationStage.Error != reason {
		t.Errorf("expected error '%s', got '%s'", reason, clarificationStage.Error)
	}

	// 工作流应该标记为失败
	if !wf.IsFailed() {
		t.Error("workflow should be failed")
	}

	// 获取失败的阶段
	failedStage := wf.GetFailedStage()
	if failedStage == nil {
		t.Fatal("expected failed stage")
	}
	if failedStage.Name != workflow.StageClarification {
		t.Errorf("expected failed stage clarification, got %s", failedStage.Name)
	}
}

func TestFailStageNotFound(t *testing.T) {
	engine := workflow.NewWorkflowEngine()

	_, err := engine.FailStage("non-existent-task", "test reason")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestFailStageAndAdvance(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-6"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 标记失败
	engine.FailStage(taskID, "test failure")

	// 尝试推进应该失败
	_, err := engine.AdvanceStage(taskID)
	if err == nil {
		t.Error("expected error when advancing from failed stage")
	}
}

func TestGetWorkflow(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-7"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 获取工作流
	wf := engine.GetWorkflow(taskID)
	if wf == nil {
		t.Fatal("expected non-nil workflow")
	}

	// 获取不存在的工作流
	wf = engine.GetWorkflow("non-existent")
	if wf != nil {
		t.Error("expected nil for non-existent workflow")
	}
}

func TestGetCurrentStage(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-8"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 获取当前阶段
	stage, err := engine.GetCurrentStage(taskID)
	if err != nil {
		t.Fatalf("failed to get current stage: %v", err)
	}

	if stage.Name != workflow.StageClarification {
		t.Errorf("expected clarification, got %s", stage.Name)
	}

	if stage.Status != workflow.StatusInProgress {
		t.Errorf("expected in_progress, got %s", stage.Status)
	}
}

func TestGetStageStatus(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-9"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 获取特定阶段状态
	stage, err := engine.GetStageStatus(taskID, workflow.StageImplementation)
	if err != nil {
		t.Fatalf("failed to get stage status: %v", err)
	}

	if stage.Status != workflow.StatusPending {
		t.Errorf("expected pending, got %s", stage.Status)
	}
}

func TestSetStageRound(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-10"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 设置轮次
	err := engine.SetStageRound(taskID, 2)
	if err != nil {
		t.Fatalf("failed to set round: %v", err)
	}

	// 验证轮次
	stage, _ := engine.GetCurrentStage(taskID)
	if stage.Round != 2 {
		t.Errorf("expected round 2, got %d", stage.Round)
	}
}

func TestIncrementStageRound(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-11"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 增加轮次
	round, err := engine.IncrementStageRound(taskID)
	if err != nil {
		t.Fatalf("failed to increment round: %v", err)
	}

	if round != 1 {
		t.Errorf("expected round 1, got %d", round)
	}

	// 再增加一次
	round, err = engine.IncrementStageRound(taskID)
	if err != nil {
		t.Fatalf("failed to increment round: %v", err)
	}

	if round != 2 {
		t.Errorf("expected round 2, got %d", round)
	}
}

func TestResetStage(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-12"

	// 初始化工作流并推进
	engine.InitWorkflow(taskID)
	engine.AdvanceStage(taskID) // clarification -> bdd_review

	// 重置clarification阶段
	err := engine.ResetStage(taskID, workflow.StageClarification)
	if err != nil {
		t.Fatalf("failed to reset stage: %v", err)
	}

	// 验证clarification状态
	wf := engine.GetWorkflow(taskID)
	clarification := wf.Stages[workflow.StageClarification]
	if clarification.Status != workflow.StatusPending {
		t.Errorf("expected pending, got %s", clarification.Status)
	}
}

func TestResetCurrentStage(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-13"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 重置当前阶段（clarification）
	err := engine.ResetStage(taskID, workflow.StageClarification)
	if err != nil {
		t.Fatalf("failed to reset current stage: %v", err)
	}

	// 当前阶段应该重新设置为进行中
	wf := engine.GetWorkflow(taskID)
	clarification := wf.Stages[workflow.StageClarification]
	if clarification.Status != workflow.StatusInProgress {
		t.Errorf("expected in_progress for current stage, got %s", clarification.Status)
	}
}

func TestRemoveWorkflow(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-14"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 移除工作流
	engine.RemoveWorkflow(taskID)

	// 验证已移除
	wf := engine.GetWorkflow(taskID)
	if wf != nil {
		t.Error("expected nil after removal")
	}
}

func TestGetAllWorkflows(t *testing.T) {
	engine := workflow.NewWorkflowEngine()

	// 初始化多个工作流
	engine.InitWorkflow("task-1")
	engine.InitWorkflow("task-2")
	engine.InitWorkflow("task-3")

	// 获取所有工作流
	workflows := engine.GetAllWorkflows()
	if len(workflows) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(workflows))
	}
}

func TestTaskWorkflowMethods(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "test-task-15"

	// 初始化工作流
	wf := engine.InitWorkflow(taskID)

	// 测试GetNextStage
	nextStage := wf.GetNextStage()
	if nextStage != workflow.StageBDDReview {
		t.Errorf("expected next stage bdd_review, got %s", nextStage)
	}

	// 测试GetAllStages
	stages := wf.GetAllStages()
	if len(stages) != 5 {
		t.Errorf("expected 5 stages, got %d", len(stages))
	}

	// 测试GetStage
	clarification := wf.GetStage(workflow.StageClarification)
	if clarification == nil {
		t.Fatal("expected clarification stage")
	}

	// 推进到最后阶段
	engine.AdvanceStage(taskID)
	engine.AdvanceStage(taskID)
	engine.AdvanceStage(taskID)
	engine.AdvanceStage(taskID)

	wf = engine.GetWorkflow(taskID)

	// 测试GetNextStage（最后阶段）
	nextStage = wf.GetNextStage()
	if nextStage != "" {
		t.Errorf("expected empty next stage for verification, got %s", nextStage)
	}
}

func TestEngineWithPersistPath(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "workflows.json")

	// 创建引擎
	engine := workflow.NewEngine(workflow.WithPersistPath(persistPath))

	// 初始化任务
	taskID := "persist-test-task"
	wf, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(persistPath); os.IsNotExist(err) {
		t.Error("expected persist file to be created")
	}

	// 推进阶段
	engine.AdvanceStage(taskID)

	// 创建新引擎加载持久化数据
	engine2 := workflow.NewEngine(workflow.WithPersistPath(persistPath))
	wf2 := engine2.GetWorkflow(taskID)

	if wf2 == nil {
		t.Fatal("expected workflow to be loaded from persist file")
	}

	if wf2.CurrentStage != wf.CurrentStage {
		t.Errorf("expected current stage %s, got %s", wf.CurrentStage, wf2.CurrentStage)
	}
}

func TestEngineIncrementRound(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "test-round"

	// 初始化任务
	engine.InitTask(taskID)

	// 增加轮次
	round, err := engine.IncrementRound(taskID)
	if err != nil {
		t.Fatalf("failed to increment round: %v", err)
	}

	if round != 1 {
		t.Errorf("expected round 1, got %d", round)
	}
}

func TestEngineSetStageStatus(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "test-status"

	// 初始化任务
	engine.InitTask(taskID)

	// 设置阶段状态
	err := engine.SetStageStatus(taskID, workflow.StageClarification, workflow.StatusCompleted)
	if err != nil {
		t.Fatalf("failed to set stage status: %v", err)
	}

	// 验证状态
	stage, err := engine.GetCurrentStage(taskID)
	if err != nil {
		t.Fatalf("failed to get current stage: %v", err)
	}

	if stage.Status != workflow.StatusCompleted {
		t.Errorf("expected completed, got %s", stage.Status)
	}
}

func TestEngineGetStageHistory(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "test-history"

	// 初始化任务
	engine.InitTask(taskID)

	// 推进两个阶段
	engine.AdvanceStage(taskID)
	engine.AdvanceStage(taskID)

	// 获取历史
	history, err := engine.GetStageHistory(taskID)
	if err != nil {
		t.Fatalf("failed to get stage history: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("expected 5 stages in history, got %d", len(history))
	}

	// 验证前两个阶段已完成
	if history[0].Status != workflow.StatusCompleted {
		t.Errorf("expected clarification completed, got %s", history[0].Status)
	}
	if history[1].Status != workflow.StatusCompleted {
		t.Errorf("expected bdd_review completed, got %s", history[1].Status)
	}
}

func TestEngineListWorkflows(t *testing.T) {
	engine := workflow.NewEngine()

	// 初始化多个任务
	engine.InitTask("task-1")
	engine.InitTask("task-2")
	engine.InitTask("task-3")

	// 推进task-1到完成
	engine.AdvanceStage("task-1")
	engine.AdvanceStage("task-1")
	engine.AdvanceStage("task-1")
	engine.AdvanceStage("task-1")
	engine.AdvanceStage("task-1")

	// 标记task-2失败
	engine.FailStage("task-2", "test failure")

	// 测试ListWorkflows
	all := engine.ListWorkflows()
	if len(all) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(all))
	}

	// 测试ListActiveWorkflows
	active := engine.ListActiveWorkflows()
	if len(active) != 1 {
		t.Errorf("expected 1 active workflow, got %d", len(active))
	}

	// 测试ListFailedWorkflows
	failed := engine.ListFailedWorkflows()
	if len(failed) != 1 {
		t.Errorf("expected 1 failed workflow, got %d", len(failed))
	}

	// 测试ListCompletedWorkflows
	completed := engine.ListCompletedWorkflows()
	if len(completed) != 1 {
		t.Errorf("expected 1 completed workflow, got %d", len(completed))
	}
}

func TestEngineGetWorkflowStats(t *testing.T) {
	engine := workflow.NewEngine()

	// 初始化多个任务
	engine.InitTask("task-1")
	engine.InitTask("task-2")
	engine.InitTask("task-3")

	// 推进task-1到完成
	for i := 0; i < 5; i++ {
		engine.AdvanceStage("task-1")
	}

	// 标记task-2失败
	engine.FailStage("task-2", "test failure")

	// 获取统计
	stats := engine.GetWorkflowStats()

	if stats["total"] != 3 {
		t.Errorf("expected total 3, got %d", stats["total"])
	}
	if stats["active"] != 1 {
		t.Errorf("expected active 1, got %d", stats["active"])
	}
	if stats["completed"] != 1 {
		t.Errorf("expected completed 1, got %d", stats["completed"])
	}
	if stats["failed"] != 1 {
		t.Errorf("expected failed 1, got %d", stats["failed"])
	}
}

func TestEngineRecoverTask(t *testing.T) {
	engine := workflow.NewEngine()

	// 恢复任务到BDD评审阶段进行中
	taskID := "recovered-task"
	wf, err := engine.RecoverTask(taskID, workflow.StageBDDReview, workflow.StatusInProgress)
	if err != nil {
		t.Fatalf("failed to recover task: %v", err)
	}

	if wf.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected current stage bdd_review, got %s", wf.CurrentStage)
	}

	// 验证clarification已完成
	clarification := wf.Stages[workflow.StageClarification]
	if clarification.Status != workflow.StatusCompleted {
		t.Errorf("expected clarification completed, got %s", clarification.Status)
	}

	// 验证BDD评审进行中
	bdd := wf.Stages[workflow.StageBDDReview]
	if bdd.Status != workflow.StatusInProgress {
		t.Errorf("expected bdd_review in progress, got %s", bdd.Status)
	}

	// 验证后续阶段待开始
	for _, name := range workflow.StageOrder[2:] {
		stage := wf.Stages[name]
		if stage.Status != workflow.StatusPending {
			t.Errorf("stage %s: expected pending, got %s", name, stage.Status)
		}
	}
}

func TestEngineRemoveTask(t *testing.T) {
	engine := workflow.NewEngine()

	// 初始化任务
	taskID := "remove-test"
	engine.InitTask(taskID)

	// 移除任务
	engine.RemoveTask(taskID)

	// 验证已移除
	wf := engine.GetWorkflow(taskID)
	if wf != nil {
		t.Error("expected nil after removal")
	}
}

func TestWorkflowErrors(t *testing.T) {
	// 测试错误常量
	if workflow.ErrWorkflowNotFound == nil {
		t.Error("ErrWorkflowNotFound should not be nil")
	}
	if workflow.ErrInvalidStage == nil {
		t.Error("ErrInvalidStage should not be nil")
	}
	if workflow.ErrInvalidTransition == nil {
		t.Error("ErrInvalidTransition should not be nil")
	}
	if workflow.ErrStageAlreadyComplete == nil {
		t.Error("ErrStageAlreadyComplete should not be nil")
	}
	if workflow.ErrStageNotInProgress == nil {
		t.Error("ErrStageNotInProgress should not be nil")
	}
	if workflow.ErrWorkflowAlreadyComplete == nil {
		t.Error("ErrWorkflowAlreadyComplete should not be nil")
	}
}

func TestCompleteStage(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "complete-test"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 完成阶段（等同于AdvanceStage）
	wf, err := engine.CompleteStage(taskID)
	if err != nil {
		t.Fatalf("failed to complete stage: %v", err)
	}

	// 验证推进成功
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected current stage bdd_review, got %s", wf.CurrentStage)
	}
}

func TestTaskWorkflowCompleteAndFailed(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "complete-failed-test"

	// 初始化工作流
	wf := engine.InitWorkflow(taskID)

	// 初始状态不应完成或失败
	if wf.IsComplete() {
		t.Error("workflow should not be complete initially")
	}
	if wf.IsFailed() {
		t.Error("workflow should not be failed initially")
	}

	// 推进到完成
	for i := 0; i < 5; i++ {
		engine.AdvanceStage(taskID)
	}

	wf = engine.GetWorkflow(taskID)
	if !wf.IsComplete() {
		t.Error("workflow should be complete after advancing all stages")
	}

	// 测试失败场景
	taskID2 := "failed-test"
	engine.InitWorkflow(taskID2)
	engine.FailStage(taskID2, "test reason")

	wf2 := engine.GetWorkflow(taskID2)
	if !wf2.IsFailed() {
		t.Error("workflow should be failed after FailStage")
	}
}

func TestStageStateTimeFields(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "time-test"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 获取当前阶段
	stage, _ := engine.GetCurrentStage(taskID)

	// 验证StartedAt已设置
	if stage.StartedAt == nil {
		t.Error("StartedAt should be set for in_progress stage")
	}

	// 验证UpdatedAt已设置
	if stage.UpdatedAt == nil {
		t.Error("UpdatedAt should be set")
	}

	// 推进阶段
	engine.AdvanceStage(taskID)

	// 获取clarification阶段（现在已完成）
	wf := engine.GetWorkflow(taskID)
	completedStage := wf.Stages[workflow.StageClarification]

	// 验证CompletedAt已设置
	if completedStage.CompletedAt == nil {
		t.Error("CompletedAt should be set for completed stage")
	}
}

func TestEngineGetCurrentStage(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "engine-current-stage"

	// 初始化任务
	engine.InitTask(taskID)

	// 获取当前阶段
	stage, err := engine.GetCurrentStage(taskID)
	if err != nil {
		t.Fatalf("failed to get current stage: %v", err)
	}

	if stage.Name != workflow.StageClarification {
		t.Errorf("expected clarification, got %s", stage.Name)
	}

	// 不存在的任务应该返回错误
	_, err = engine.GetCurrentStage("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestEngineSetRound(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "engine-set-round"

	// 初始化任务
	engine.InitTask(taskID)

	// 设置轮次
	err := engine.SetRound(taskID, 5)
	if err != nil {
		t.Fatalf("failed to set round: %v", err)
	}

	// 验证轮次
	stage, _ := engine.GetCurrentStage(taskID)
	if stage.Round != 5 {
		t.Errorf("expected round 5, got %d", stage.Round)
	}
}

func TestConcurrentAccess(t *testing.T) {
	engine := workflow.NewWorkflowEngine()
	taskID := "concurrent-test"

	// 初始化工作流
	engine.InitWorkflow(taskID)

	// 并发推进阶段
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			engine.AdvanceStage(taskID)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("timeout waiting for concurrent access")
		}
	}

	// 验证最终状态是有效的
	wf := engine.GetWorkflow(taskID)
	if wf == nil {
		t.Fatal("workflow should still exist")
	}

	// 验证阶段状态是有效的
	for _, stage := range wf.Stages {
		validStatuses := []workflow.StageStatus{
			workflow.StatusPending,
			workflow.StatusInProgress,
			workflow.StatusCompleted,
			workflow.StatusFailed,
		}
		valid := false
		for _, s := range validStatuses {
			if stage.Status == s {
				valid = true
				break
			}
		}
		if !valid {
			t.Errorf("invalid stage status: %s", stage.Status)
		}
	}
}

// BDD 审核相关测试
func TestEngine_ApproveBDD(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "bdd-approve-test"

	// 初始化任务并推进到 BDD 审核阶段
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // clarification -> bdd_review

	// 验证当前阶段
	wf := engine.GetWorkflow(taskID)
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Fatalf("expected bdd_review, got %s", wf.CurrentStage)
	}

	// 通过 BDD 审核
	wf, err := engine.ApproveBDD(taskID)
	if err != nil {
		t.Fatalf("failed to approve BDD: %v", err)
	}

	// 验证状态流转
	if wf.CurrentStage != workflow.StageArchitectureReview {
		t.Errorf("expected architecture_review, got %s", wf.CurrentStage)
	}

	// 验证 BDD 阶段已完成
	bddStage := wf.Stages[workflow.StageBDDReview]
	if bddStage.Status != workflow.StatusCompleted {
		t.Errorf("expected bdd_review completed, got %s", bddStage.Status)
	}

	// 验证架构审核阶段进行中
	archStage := wf.Stages[workflow.StageArchitectureReview]
	if archStage.Status != workflow.StatusInProgress {
		t.Errorf("expected architecture_review in_progress, got %s", archStage.Status)
	}
}

func TestEngine_ApproveBDD_NotInBDDReviewStage(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "bdd-approve-invalid"

	// 初始化任务（在澄清阶段）
	engine.InitTask(taskID)

	// 尝试通过 BDD 审核（应该失败）
	_, err := engine.ApproveBDD(taskID)
	if err == nil {
		t.Error("expected error when not in bdd_review stage")
	}
}

func TestEngine_ApproveBDD_WorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()

	// 尝试通过不存在的工作流
	_, err := engine.ApproveBDD("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestEngine_RejectBDD(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "bdd-reject-test"

	// 初始化任务并推进到 BDD 审核阶段
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // clarification -> bdd_review

	// 验证当前阶段
	wf := engine.GetWorkflow(taskID)
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Fatalf("expected bdd_review, got %s", wf.CurrentStage)
	}

	// 驳回 BDD 审核
	reason := "BDD 规则不符合要求"
	wf, err := engine.RejectBDD(taskID, reason)
	if err != nil {
		t.Fatalf("failed to reject BDD: %v", err)
	}

	// 验证状态流转回澄清阶段
	if wf.CurrentStage != workflow.StageClarification {
		t.Errorf("expected clarification, got %s", wf.CurrentStage)
	}

	// 验证 BDD 阶段已失败
	bddStage := wf.Stages[workflow.StageBDDReview]
	if bddStage.Status != workflow.StatusFailed {
		t.Errorf("expected bdd_review failed, got %s", bddStage.Status)
	}

	// 验证驳回原因已记录
	if bddStage.Error != reason {
		t.Errorf("expected error '%s', got '%s'", reason, bddStage.Error)
	}

	// 验证澄清阶段重新开始
	clarificationStage := wf.Stages[workflow.StageClarification]
	if clarificationStage.Status != workflow.StatusInProgress {
		t.Errorf("expected clarification in_progress, got %s", clarificationStage.Status)
	}

	// 验证轮次已重置
	if clarificationStage.Round != 0 {
		t.Errorf("expected round 0, got %d", clarificationStage.Round)
	}
}

func TestEngine_RejectBDD_NotInBDDReviewStage(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "bdd-reject-invalid"

	// 初始化任务（在澄清阶段）
	engine.InitTask(taskID)

	// 尝试驳回 BDD 审核（应该失败）
	_, err := engine.RejectBDD(taskID, "test reason")
	if err == nil {
		t.Error("expected error when not in bdd_review stage")
	}
}

func TestEngine_RejectBDD_WorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()

	// 尝试驳回不存在的工作流
	_, err := engine.RejectBDD("non-existent", "test reason")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestEngine_RejectBDD_EmptyReason(t *testing.T) {
	engine := workflow.NewEngine()
	taskID := "bdd-reject-empty"

	// 初始化任务并推进到 BDD 审核阶段
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID)

	// 驳回 BDD 审核（空原因）
	wf, err := engine.RejectBDD(taskID, "")
	if err != nil {
		t.Fatalf("failed to reject BDD: %v", err)
	}

	// 验证驳回成功（空原因也被接受）
	bddStage := wf.Stages[workflow.StageBDDReview]
	if bddStage.Status != workflow.StatusFailed {
		t.Errorf("expected bdd_review failed, got %s", bddStage.Status)
	}
}
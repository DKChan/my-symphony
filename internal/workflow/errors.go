// Package workflow 提供工作流阶段管理功能
package workflow

import "errors"

var (
	// ErrWorkflowNotFound 工作流未找到错误
	ErrWorkflowNotFound = errors.New("workflow not found")
	// ErrInvalidStage 无效阶段错误
	ErrInvalidStage = errors.New("invalid stage")
	// ErrInvalidTransition 无效状态转换错误
	ErrInvalidTransition = errors.New("invalid stage transition")
	// ErrStageAlreadyComplete 阶段已完成错误
	ErrStageAlreadyComplete = errors.New("stage already complete")
	// ErrStageNotInProgress 阶段未在进行中错误
	ErrStageNotInProgress = errors.New("stage not in progress")
	// ErrWorkflowAlreadyComplete 工作流已完成错误
	ErrWorkflowAlreadyComplete = errors.New("workflow already complete")
	// ErrRoundLimitReached 澄清轮次已达上限错误
	ErrRoundLimitReached = errors.New("clarification round limit reached")
	// ErrCannotSkipClarification 无法跳过澄清错误
	ErrCannotSkipClarification = errors.New("cannot skip clarification")
	// ErrNotInClarificationStage 不在澄清阶段错误
	ErrNotInClarificationStage = errors.New("not in clarification stage")
	// ErrNotInBDDReviewStage 不在 BDD 审核阶段错误
	ErrNotInBDDReviewStage = errors.New("not in bdd_review stage")
	// ErrBDDReviewAlreadyApproved BDD 审核已通过错误
	ErrBDDReviewAlreadyApproved = errors.New("bdd review already approved")
	// ErrBDDReviewAlreadyRejected BDD 审核已驳回错误
	ErrBDDReviewAlreadyRejected = errors.New("bdd review already rejected")
	// ErrInvalidBDDRules 无效的BDD规则错误
	ErrInvalidBDDRules = errors.New("invalid bdd rules")
	// ErrBDDFileNotFound BDD文件未找到错误
	ErrBDDFileNotFound = errors.New("bdd file not found")
	// ErrBDDGenerationFailed BDD生成失败错误
	ErrBDDGenerationFailed = errors.New("bdd generation failed")
)
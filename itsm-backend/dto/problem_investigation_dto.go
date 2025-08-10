package dto

import (
	"time"
)

// ProblemInvestigationStatus 问题调查状态
type ProblemInvestigationStatus string

const (
	InvestigationStatusNotStarted ProblemInvestigationStatus = "not_started"
	InvestigationStatusInProgress ProblemInvestigationStatus = "in_progress"
	InvestigationStatusOnHold     ProblemInvestigationStatus = "on_hold"
	InvestigationStatusCompleted  ProblemInvestigationStatus = "completed"
	InvestigationStatusCancelled  ProblemInvestigationStatus = "cancelled"
)

// ProblemInvestigationStepStatus 调查步骤状态
type ProblemInvestigationStepStatus string

const (
	StepStatusPending    ProblemInvestigationStepStatus = "pending"
	StepStatusInProgress ProblemInvestigationStepStatus = "in_progress"
	StepStatusCompleted  ProblemInvestigationStepStatus = "completed"
	StepStatusBlocked    ProblemInvestigationStepStatus = "blocked"
	StepStatusCancelled  ProblemInvestigationStepStatus = "cancelled"
)

// ConfidenceLevel 置信度级别
type ConfidenceLevel string

const (
	ConfidenceLow    ConfidenceLevel = "low"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceHigh   ConfidenceLevel = "high"
)

// SolutionType 解决方案类型
type SolutionType string

const (
	SolutionTypeWorkaround SolutionType = "workaround"
	SolutionTypeFix        SolutionType = "fix"
	SolutionTypePrevention SolutionType = "prevention"
	SolutionTypeProcess    SolutionType = "process"
)

// SolutionStatus 解决方案状态
type SolutionStatus string

const (
	SolutionStatusProposed    SolutionStatus = "proposed"
	SolutionStatusApproved    SolutionStatus = "approved"
	SolutionStatusInProgress  SolutionStatus = "in_progress"
	SolutionStatusImplemented SolutionStatus = "implemented"
	SolutionStatusRejected    SolutionStatus = "rejected"
	SolutionStatusCancelled   SolutionStatus = "cancelled"
)

// ImplementationStatus 实施状态
type ImplementationStatus string

const (
	ImplementationStatusNotStarted ImplementationStatus = "not_started"
	ImplementationStatusInProgress ImplementationStatus = "in_progress"
	ImplementationStatusOnHold     ImplementationStatus = "on_hold"
	ImplementationStatusCompleted  ImplementationStatus = "completed"
	ImplementationStatusFailed     ImplementationStatus = "failed"
)

// CreateProblemInvestigationRequest 创建问题调查请求
type CreateProblemInvestigationRequest struct {
	ProblemID               int        `json:"problem_id" binding:"required"`
	InvestigatorID          int        `json:"investigator_id" binding:"required"`
	EstimatedCompletionDate *time.Time `json:"estimated_completion_date"`
	InvestigationSummary    string     `json:"investigation_summary"`
}

// UpdateProblemInvestigationRequest 更新问题调查请求
type UpdateProblemInvestigationRequest struct {
	Status                  *ProblemInvestigationStatus `json:"status"`
	EstimatedCompletionDate *time.Time                  `json:"estimated_completion_date"`
	ActualCompletionDate    *time.Time                  `json:"actual_completion_date"`
	InvestigationSummary    *string                     `json:"investigation_summary"`
}

// CreateInvestigationStepRequest 创建调查步骤请求
type CreateInvestigationStepRequest struct {
	InvestigationID int    `json:"investigation_id" binding:"required"`
	StepNumber      int    `json:"step_number" binding:"required"`
	StepTitle       string `json:"step_title" binding:"required"`
	StepDescription string `json:"step_description" binding:"required"`
	AssignedTo      *int   `json:"assigned_to"`
	Notes           string `json:"notes"`
}

// UpdateInvestigationStepRequest 更新调查步骤请求
type UpdateInvestigationStepRequest struct {
	StepTitle       *string                         `json:"step_title"`
	StepDescription *string                         `json:"step_description"`
	Status          *ProblemInvestigationStepStatus `json:"status"`
	AssignedTo      *int                            `json:"assigned_to"`
	StartDate       *time.Time                      `json:"start_date"`
	CompletionDate  *time.Time                      `json:"completion_date"`
	Notes           *string                         `json:"notes"`
}

// CreateRootCauseAnalysisRequest 创建根本原因分析请求
type CreateRootCauseAnalysisRequest struct {
	ProblemID            int             `json:"problem_id" binding:"required"`
	AnalystID            int             `json:"analyst_id" binding:"required"`
	AnalysisMethod       string          `json:"analysis_method" binding:"required"`
	RootCauseDescription string          `json:"root_cause_description" binding:"required"`
	ContributingFactors  string          `json:"contributing_factors"`
	Evidence             string          `json:"evidence"`
	ConfidenceLevel      ConfidenceLevel `json:"confidence_level" binding:"required"`
}

// UpdateRootCauseAnalysisRequest 更新根本原因分析请求
type UpdateRootCauseAnalysisRequest struct {
	AnalysisMethod       *string          `json:"analysis_method"`
	RootCauseDescription *string          `json:"root_cause_description"`
	ContributingFactors  *string          `json:"contributing_factors"`
	Evidence             *string          `json:"evidence"`
	ConfidenceLevel      *ConfidenceLevel `json:"confidence_level"`
	ReviewedBy           *int             `json:"reviewed_by"`
	ReviewDate           *time.Time       `json:"review_date"`
}

// CreateProblemSolutionRequest 创建问题解决方案请求
type CreateProblemSolutionRequest struct {
	ProblemID            int          `json:"problem_id" binding:"required"`
	SolutionType         SolutionType `json:"solution_type" binding:"required"`
	SolutionDescription  string       `json:"solution_description" binding:"required"`
	Priority             string       `json:"priority" binding:"required,oneof=low medium high critical"`
	ProposedBy           int          `json:"proposed_by" binding:"required"`
	EstimatedEffortHours *int         `json:"estimated_effort_hours"`
	EstimatedCost        *float64     `json:"estimated_cost"`
	RiskAssessment       string       `json:"risk_assessment"`
}

// UpdateProblemSolutionRequest 更新问题解决方案请求
type UpdateProblemSolutionRequest struct {
	SolutionType         *SolutionType   `json:"solution_type"`
	SolutionDescription  *string         `json:"solution_description"`
	Priority             *string         `json:"priority"`
	Status               *SolutionStatus `json:"status"`
	EstimatedEffortHours *int            `json:"estimated_effort_hours"`
	EstimatedCost        *float64        `json:"estimated_cost"`
	RiskAssessment       *string         `json:"risk_assessment"`
	ApprovalStatus       *string         `json:"approval_status"`
	ApprovedBy           *int            `json:"approved_by"`
	ApprovalDate         *time.Time      `json:"approval_date"`
}

// CreateSolutionImplementationRequest 创建解决方案实施请求
type CreateSolutionImplementationRequest struct {
	SolutionID          int        `json:"solution_id" binding:"required"`
	ImplementerID       int        `json:"implementer_id" binding:"required"`
	StartDate           *time.Time `json:"start_date"`
	ImplementationNotes string     `json:"implementation_notes"`
}

// UpdateSolutionImplementationRequest 更新解决方案实施请求
type UpdateSolutionImplementationRequest struct {
	ImplementationStatus  *ImplementationStatus `json:"implementation_status"`
	StartDate             *time.Time            `json:"start_date"`
	CompletionDate        *time.Time            `json:"completion_date"`
	ActualEffortHours     *int                  `json:"actual_effort_hours"`
	ActualCost            *float64              `json:"actual_cost"`
	ImplementationNotes   *string               `json:"implementation_notes"`
	ChallengesEncountered *string               `json:"challenges_encountered"`
	LessonsLearned        *string               `json:"lessons_learned"`
}

// CreateProblemRelationshipRequest 创建问题关联请求
type CreateProblemRelationshipRequest struct {
	ProblemID        int    `json:"problem_id" binding:"required"`
	RelatedType      string `json:"related_type" binding:"required"`
	RelatedID        int    `json:"related_id" binding:"required"`
	RelationshipType string `json:"relationship_type" binding:"required"`
	Description      string `json:"description"`
}

// CreateProblemKnowledgeArticleRequest 创建问题知识库文章请求
type CreateProblemKnowledgeArticleRequest struct {
	ProblemID      int      `json:"problem_id" binding:"required"`
	ArticleTitle   string   `json:"article_title" binding:"required"`
	ArticleContent string   `json:"article_content" binding:"required"`
	ArticleType    string   `json:"article_type" binding:"required"`
	Tags           []string `json:"tags"`
}

// UpdateProblemKnowledgeArticleRequest 更新问题知识库文章请求
type UpdateProblemKnowledgeArticleRequest struct {
	ArticleTitle   *string   `json:"article_title"`
	ArticleContent *string   `json:"article_content"`
	ArticleType    *string   `json:"article_type"`
	Status         *string   `json:"status"`
	Tags           *[]string `json:"tags"`
}

// ProblemInvestigationResponse 问题调查响应
type ProblemInvestigationResponse struct {
	ID                      int                        `json:"id"`
	ProblemID               int                        `json:"problem_id"`
	InvestigatorID          int                        `json:"investigator_id"`
	InvestigatorName        string                     `json:"investigator_name"`
	Status                  ProblemInvestigationStatus `json:"status"`
	StartDate               time.Time                  `json:"start_date"`
	EstimatedCompletionDate *time.Time                 `json:"estimated_completion_date"`
	ActualCompletionDate    *time.Time                 `json:"actual_completion_date"`
	InvestigationSummary    *string                    `json:"investigation_summary"`
	CreatedAt               time.Time                  `json:"created_at"`
	UpdatedAt               time.Time                  `json:"updated_at"`
}

// InvestigationStepResponse 调查步骤响应
type InvestigationStepResponse struct {
	ID              int                            `json:"id"`
	InvestigationID int                            `json:"investigation_id"`
	StepNumber      int                            `json:"step_number"`
	StepTitle       string                         `json:"step_title"`
	StepDescription string                         `json:"step_description"`
	Status          ProblemInvestigationStepStatus `json:"status"`
	AssignedTo      *int                           `json:"assigned_to"`
	AssignedToName  *string                        `json:"assigned_to_name"`
	StartDate       *time.Time                     `json:"start_date"`
	CompletionDate  *time.Time                     `json:"completion_date"`
	Notes           *string                        `json:"notes"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
}

// RootCauseAnalysisResponse 根本原因分析响应
type RootCauseAnalysisResponse struct {
	ID                   int             `json:"id"`
	ProblemID            int             `json:"problem_id"`
	AnalystID            int             `json:"analyst_id"`
	AnalystName          string          `json:"analyst_name"`
	AnalysisMethod       string          `json:"analysis_method"`
	RootCauseDescription string          `json:"root_cause_description"`
	ContributingFactors  *string         `json:"contributing_factors"`
	Evidence             *string         `json:"evidence"`
	ConfidenceLevel      ConfidenceLevel `json:"confidence_level"`
	AnalysisDate         time.Time       `json:"analysis_date"`
	ReviewedBy           *int            `json:"reviewed_by"`
	ReviewedByName       *string         `json:"reviewed_by_name"`
	ReviewDate           *time.Time      `json:"review_date"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// ProblemSolutionResponse 问题解决方案响应
type ProblemSolutionResponse struct {
	ID                   int            `json:"id"`
	ProblemID            int            `json:"problem_id"`
	SolutionType         SolutionType   `json:"solution_type"`
	SolutionDescription  string         `json:"solution_description"`
	ProposedBy           int            `json:"proposed_by"`
	ProposedByName       string         `json:"proposed_by_name"`
	ProposedDate         time.Time      `json:"proposed_date"`
	Status               SolutionStatus `json:"status"`
	Priority             string         `json:"priority"`
	EstimatedEffortHours *int           `json:"estimated_effort_hours"`
	EstimatedCost        *float64       `json:"estimated_cost"`
	RiskAssessment       *string        `json:"risk_assessment"`
	ApprovalStatus       string         `json:"approval_status"`
	ApprovedBy           *int           `json:"approved_by"`
	ApprovedByName       *string        `json:"approved_by_name"`
	ApprovalDate         *time.Time     `json:"approval_date"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// SolutionImplementationResponse 解决方案实施响应
type SolutionImplementationResponse struct {
	ID                    int                  `json:"id"`
	SolutionID            int                  `json:"solution_id"`
	ImplementerID         int                  `json:"implementer_id"`
	ImplementerName       string               `json:"implementer_name"`
	ImplementationStatus  ImplementationStatus `json:"implementation_status"`
	StartDate             *time.Time           `json:"start_date"`
	CompletionDate        *time.Time           `json:"completion_date"`
	ActualEffortHours     *int                 `json:"actual_effort_hours"`
	ActualCost            *float64             `json:"actual_cost"`
	ImplementationNotes   *string              `json:"implementation_notes"`
	ChallengesEncountered *string              `json:"challenges_encountered"`
	LessonsLearned        *string              `json:"lessons_learned"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

// ProblemRelationshipResponse 问题关联响应
type ProblemRelationshipResponse struct {
	ID               int       `json:"id"`
	ProblemID        int       `json:"problem_id"`
	RelatedType      string    `json:"related_type"`
	RelatedID        int       `json:"related_id"`
	RelatedTitle     string    `json:"related_title"`
	RelationshipType string    `json:"relationship_type"`
	Description      *string   `json:"description"`
	CreatedAt        time.Time `json:"created_at"`
}

// ProblemKnowledgeArticleResponse 问题知识库文章响应
type ProblemKnowledgeArticleResponse struct {
	ID             int        `json:"id"`
	ProblemID      int        `json:"problem_id"`
	ArticleTitle   string     `json:"article_title"`
	ArticleContent string     `json:"article_content"`
	ArticleType    string     `json:"article_type"`
	AuthorID       int        `json:"author_id"`
	AuthorName     string     `json:"author_name"`
	Status         string     `json:"status"`
	PublishedDate  *time.Time `json:"published_date"`
	Tags           []string   `json:"tags"`
	ViewCount      int        `json:"view_count"`
	HelpfulCount   int        `json:"helpful_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ProblemInvestigationSummaryResponse 问题调查摘要响应
type ProblemInvestigationSummaryResponse struct {
	Investigation     *ProblemInvestigationResponse      `json:"investigation"`
	Steps             []*InvestigationStepResponse       `json:"steps"`
	RootCauseAnalysis *RootCauseAnalysisResponse         `json:"root_cause_analysis"`
	Solutions         []*ProblemSolutionResponse         `json:"solutions"`
	Implementations   []*SolutionImplementationResponse  `json:"implementations"`
	Relationships     []*ProblemRelationshipResponse     `json:"relationships"`
	KnowledgeArticles []*ProblemKnowledgeArticleResponse `json:"knowledge_articles"`
}

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
	SolutionStatusProposed              SolutionStatus = "proposed"
	SolutionStatusApproved              SolutionStatus = "approved"
	SolutionStatusPendingImplementation SolutionStatus = "pending_implementation"
	SolutionStatusInProgress            SolutionStatus = "in_progress"
	SolutionStatusImplemented           SolutionStatus = "implemented"
	SolutionStatusRejected              SolutionStatus = "rejected"
	SolutionStatusCancelled             SolutionStatus = "cancelled"
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
	ProblemID               int        `json:"problemId" binding:"required"`
	InvestigatorID          int        `json:"investigatorId" binding:"required"`
	EstimatedCompletionDate *time.Time `json:"estimatedCompletionDate"`
	InvestigationSummary    string     `json:"investigationSummary"`
}

// UpdateProblemInvestigationRequest 更新问题调查请求
type UpdateProblemInvestigationRequest struct {
	Status                  *ProblemInvestigationStatus `json:"status"`
	EstimatedCompletionDate *time.Time                  `json:"estimatedCompletionDate"`
	ActualCompletionDate    *time.Time                  `json:"actualCompletionDate"`
	InvestigationSummary    *string                     `json:"investigationSummary"`
}

// CreateInvestigationStepRequest 创建调查步骤请求
type CreateInvestigationStepRequest struct {
	InvestigationID int    `json:"investigationId" binding:"required"`
	StepNumber      int    `json:"stepNumber" binding:"required"`
	StepTitle       string `json:"stepTitle" binding:"required"`
	StepDescription string `json:"stepDescription" binding:"required"`
	AssignedTo      *int   `json:"assignedTo"`
	Notes           string `json:"notes"`
}

// UpdateInvestigationStepRequest 更新调查步骤请求
type UpdateInvestigationStepRequest struct {
	StepTitle       *string                         `json:"stepTitle"`
	StepDescription *string                         `json:"stepDescription"`
	Status          *ProblemInvestigationStepStatus `json:"status"`
	AssignedTo      *int                            `json:"assignedTo"`
	StartDate       *time.Time                      `json:"startDate"`
	CompletionDate  *time.Time                      `json:"completionDate"`
	Notes           *string                         `json:"notes"`
}

// CreateRootCauseAnalysisRequest 创建根本原因分析请求
type CreateRootCauseAnalysisRequest struct {
	ProblemID            int             `json:"problemId" binding:"required"`
	AnalystID            int             `json:"analystId" binding:"required"`
	AnalysisMethod       string          `json:"analysisMethod" binding:"required"`
	RootCauseDescription string          `json:"rootCauseDescription" binding:"required"`
	ContributingFactors  string          `json:"contributingFactors"`
	Evidence             string          `json:"evidence"`
	ConfidenceLevel      ConfidenceLevel `json:"confidenceLevel" binding:"required"`
}

// UpdateRootCauseAnalysisRequest 更新根本原因分析请求
type UpdateRootCauseAnalysisRequest struct {
	AnalysisMethod       *string          `json:"analysisMethod"`
	RootCauseDescription *string          `json:"rootCauseDescription"`
	ContributingFactors  *string          `json:"contributingFactors"`
	Evidence             *string          `json:"evidence"`
	ConfidenceLevel      *ConfidenceLevel `json:"confidenceLevel"`
	ReviewedBy           *int             `json:"reviewedBy"`
	ReviewDate           *time.Time       `json:"reviewDate"`
}

// CreateProblemSolutionRequest 创建问题解决方案请求
type CreateProblemSolutionRequest struct {
	ProblemID            int          `json:"problemId" binding:"required"`
	SolutionType         SolutionType `json:"solutionType" binding:"required"`
	SolutionDescription  string       `json:"solutionDescription" binding:"required"`
	Priority             string       `json:"priority" binding:"required,oneof=low medium high critical"`
	ProposedBy           int          `json:"proposedBy" binding:"required"`
	EstimatedEffortHours *int         `json:"estimatedEffortHours"`
	EstimatedCost        *float64     `json:"estimatedCost"`
	RiskAssessment       string       `json:"riskAssessment"`
}

// UpdateProblemSolutionRequest 更新问题解决方案请求
type UpdateProblemSolutionRequest struct {
	SolutionType         *SolutionType   `json:"solutionType"`
	SolutionDescription  *string         `json:"solutionDescription"`
	Priority             *string         `json:"priority"`
	Status               *SolutionStatus `json:"status"`
	EstimatedEffortHours *int            `json:"estimatedEffortHours"`
	EstimatedCost        *float64        `json:"estimatedCost"`
	RiskAssessment       *string         `json:"riskAssessment"`
	ApprovalStatus       *string         `json:"approvalStatus"`
	ApprovedBy           *int            `json:"approvedBy"`
	ApprovalDate         *time.Time      `json:"approvalDate"`
}

// CreateSolutionImplementationRequest 创建解决方案实施请求
type CreateSolutionImplementationRequest struct {
	SolutionID          int        `json:"solutionId" binding:"required"`
	ImplementerID       int        `json:"implementerId" binding:"required"`
	StartDate           *time.Time `json:"startDate"`
	ImplementationNotes string     `json:"implementationNotes"`
}

// UpdateSolutionImplementationRequest 更新解决方案实施请求
type UpdateSolutionImplementationRequest struct {
	ImplementationStatus  *ImplementationStatus `json:"implementationStatus"`
	StartDate             *time.Time            `json:"startDate"`
	CompletionDate        *time.Time            `json:"completionDate"`
	ActualEffortHours     *int                  `json:"actualEffortHours"`
	ActualCost            *float64              `json:"actualCost"`
	ImplementationNotes   *string               `json:"implementationNotes"`
	ChallengesEncountered *string               `json:"challengesEncountered"`
	LessonsLearned        *string               `json:"lessonsLearned"`
}

// CreateProblemRelationshipRequest 创建问题关联请求
type CreateProblemRelationshipRequest struct {
	ProblemID        int    `json:"problemId" binding:"required"`
	RelatedType      string `json:"relatedType" binding:"required"`
	RelatedID        int    `json:"relatedId" binding:"required"`
	RelationshipType string `json:"relationshipType" binding:"required"`
	Description      string `json:"description"`
}

// CreateProblemKnowledgeArticleRequest 创建问题知识库文章请求
type CreateProblemKnowledgeArticleRequest struct {
	ProblemID      int      `json:"problemId" binding:"required"`
	ArticleTitle   string   `json:"articleTitle" binding:"required"`
	ArticleContent string   `json:"articleContent" binding:"required"`
	ArticleType    string   `json:"articleType" binding:"required"`
	Tags           []string `json:"tags"`
}

// UpdateProblemKnowledgeArticleRequest 更新问题知识库文章请求
type UpdateProblemKnowledgeArticleRequest struct {
	ArticleTitle   *string   `json:"articleTitle"`
	ArticleContent *string   `json:"articleContent"`
	ArticleType    *string   `json:"articleType"`
	Status         *string   `json:"status"`
	Tags           *[]string `json:"tags"`
}

// ProblemInvestigationResponse 问题调查响应
type ProblemInvestigationResponse struct {
	ID                      int                        `json:"id"`
	ProblemID               int                        `json:"problemId"`
	InvestigatorID          int                        `json:"investigatorId"`
	InvestigatorName        string                     `json:"investigatorName"`
	Status                  ProblemInvestigationStatus `json:"status"`
	StartDate               time.Time                  `json:"startDate"`
	EstimatedCompletionDate *time.Time                 `json:"estimatedCompletionDate"`
	ActualCompletionDate    *time.Time                 `json:"actualCompletionDate"`
	InvestigationSummary    *string                    `json:"investigationSummary"`
	CreatedAt               time.Time                  `json:"createdAt"`
	UpdatedAt               time.Time                  `json:"updatedAt"`
}

// InvestigationStepResponse 调查步骤响应
type InvestigationStepResponse struct {
	ID              int                            `json:"id"`
	InvestigationID int                            `json:"investigationId"`
	StepNumber      int                            `json:"stepNumber"`
	StepTitle       string                         `json:"stepTitle"`
	StepDescription string                         `json:"stepDescription"`
	Status          ProblemInvestigationStepStatus `json:"status"`
	AssignedTo      *int                           `json:"assignedTo"`
	AssignedToName  *string                        `json:"assignedToName"`
	StartDate       *time.Time                     `json:"startDate"`
	CompletionDate  *time.Time                     `json:"completionDate"`
	Notes           *string                        `json:"notes"`
	CreatedAt       time.Time                      `json:"createdAt"`
	UpdatedAt       time.Time                      `json:"updatedAt"`
}

// RootCauseAnalysisResponse 根本原因分析响应
type RootCauseAnalysisResponse struct {
	ID                   int             `json:"id"`
	ProblemID            int             `json:"problemId"`
	AnalystID            int             `json:"analystId"`
	AnalystName          string          `json:"analystName"`
	AnalysisMethod       string          `json:"analysisMethod"`
	RootCauseDescription string          `json:"rootCauseDescription"`
	ContributingFactors  *string         `json:"contributingFactors"`
	Evidence             *string         `json:"evidence"`
	ConfidenceLevel      ConfidenceLevel `json:"confidenceLevel"`
	AnalysisDate         time.Time       `json:"analysisDate"`
	ReviewedBy           *int            `json:"reviewedBy"`
	ReviewedByName       *string         `json:"reviewedByName"`
	ReviewDate           *time.Time      `json:"reviewDate"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

// ProblemSolutionResponse 问题解决方案响应
type ProblemSolutionResponse struct {
	ID                   int            `json:"id"`
	ProblemID            int            `json:"problemId"`
	SolutionType         SolutionType   `json:"solutionType"`
	SolutionDescription  string         `json:"solutionDescription"`
	ProposedBy           int            `json:"proposedBy"`
	ProposedByName       string         `json:"proposedByName"`
	ProposedDate         time.Time      `json:"proposedDate"`
	Status               SolutionStatus `json:"status"`
	Priority             string         `json:"priority"`
	EstimatedEffortHours *int           `json:"estimatedEffortHours"`
	EstimatedCost        *float64       `json:"estimatedCost"`
	RiskAssessment       *string        `json:"riskAssessment"`
	ApprovalStatus       string         `json:"approvalStatus"`
	ApprovedBy           *int           `json:"approvedBy"`
	ApprovedByName       *string        `json:"approvedByName"`
	ApprovalDate         *time.Time     `json:"approvalDate"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

// SolutionImplementationResponse 解决方案实施响应
type SolutionImplementationResponse struct {
	ID                    int                  `json:"id"`
	SolutionID            int                  `json:"solutionId"`
	ImplementerID         int                  `json:"implementerId"`
	ImplementerName       string               `json:"implementerName"`
	ImplementationStatus  ImplementationStatus `json:"implementationStatus"`
	StartDate             *time.Time           `json:"startDate"`
	CompletionDate        *time.Time           `json:"completionDate"`
	ActualEffortHours     *int                 `json:"actualEffortHours"`
	ActualCost            *float64             `json:"actualCost"`
	ImplementationNotes   *string              `json:"implementationNotes"`
	ChallengesEncountered *string              `json:"challengesEncountered"`
	LessonsLearned        *string              `json:"lessonsLearned"`
	CreatedAt             time.Time            `json:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt"`
}

// ProblemRelationshipResponse 问题关联响应
type ProblemRelationshipResponse struct {
	ID               int       `json:"id"`
	ProblemID        int       `json:"problemId"`
	RelatedType      string    `json:"relatedType"`
	RelatedID        int       `json:"relatedId"`
	RelatedTitle     string    `json:"relatedTitle"`
	RelationshipType string    `json:"relationshipType"`
	Description      *string   `json:"description"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ProblemKnowledgeArticleResponse 问题知识库文章响应
type ProblemKnowledgeArticleResponse struct {
	ID             int        `json:"id"`
	ProblemID      int        `json:"problemId"`
	ArticleTitle   string     `json:"articleTitle"`
	ArticleContent string     `json:"articleContent"`
	ArticleType    string     `json:"articleType"`
	AuthorID       int        `json:"authorId"`
	AuthorName     string     `json:"authorName"`
	Status         string     `json:"status"`
	PublishedDate  *time.Time `json:"publishedDate"`
	Tags           []string   `json:"tags"`
	ViewCount      int        `json:"viewCount"`
	HelpfulCount   int        `json:"helpfulCount"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// ProblemInvestigationSummaryResponse 问题调查摘要响应
type ProblemInvestigationSummaryResponse struct {
	Investigation     *ProblemInvestigationResponse      `json:"investigation"`
	Steps             []*InvestigationStepResponse       `json:"steps"`
	RootCauseAnalysis *RootCauseAnalysisResponse         `json:"rootCauseAnalysis"`
	Solutions         []*ProblemSolutionResponse         `json:"solutions"`
	Implementations   []*SolutionImplementationResponse  `json:"implementations"`
	Relationships     []*ProblemRelationshipResponse     `json:"relationships"`
	KnowledgeArticles []*ProblemKnowledgeArticleResponse `json:"knowledgeArticles"`
}

package dto

import (
	"itsm-backend/ent"
	"time"
)

// CreateProblemRequest 创建问题请求
type CreateProblemRequest struct {
	Title       string `json:"title" binding:"required,min=2,max=200"`
	Description string `json:"description" binding:"required,min=10,max=5000"`
	Priority    string `json:"priority" binding:"required,oneof=low medium high critical"`
	Category    string `json:"category" binding:"required"`
	RootCause   string `json:"root_cause" binding:"required"`
	Impact      string `json:"impact" binding:"required"`
	CreatedBy   int    `json:"created_by" binding:"required"`
}

// UpdateProblemRequest 更新问题请求
type UpdateProblemRequest struct {
	Title       string `json:"title" binding:"omitempty,min=2,max=200"`
	Description string `json:"description" binding:"omitempty,min=10,max=5000"`
	Priority    string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Status      string `json:"status" binding:"omitempty,oneof=open in_progress resolved closed"`
	Category    string `json:"category" binding:"omitempty"`
	RootCause   string `json:"root_cause" binding:"omitempty"`
	Impact      string `json:"impact" binding:"omitempty"`
}

// ListProblemsRequest 获取问题列表请求
type ListProblemsRequest struct {
	Page      int        `json:"page" form:"page"`
	PageSize  int        `json:"page_size" form:"page_size"`
	Status    string     `json:"status" form:"status"`
	Priority  string     `json:"priority" form:"priority"`
	Category  string     `json:"category" form:"category"`
	Keyword   string     `json:"keyword" form:"keyword"`
	DateFrom  *time.Time `json:"date_from" form:"date_from"`
	DateTo    *time.Time `json:"date_to" form:"date_to"`
	SortBy    string     `json:"sort_by" form:"sort_by"`
	SortOrder string     `json:"sort_order" form:"sort_order"`
}

// ListProblemsResponse 问题列表响应
type ListProblemsResponse struct {
	Problems []*ent.Problem `json:"problems"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// ProblemStatsResponse 问题统计响应
type ProblemStatsResponse struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"in_progress"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	HighPriority int `json:"high_priority"`
}

// ProblemDetailResponse 问题详情响应
type ProblemDetailResponse struct {
	Problem *ent.Problem `json:"problem"`
}

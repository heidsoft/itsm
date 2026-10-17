package dto

// AddTeamMemberRequest 添加团队成员请求。
type AddTeamMemberRequest struct {
	UserID int `json:"userId" binding:"required,min=1"`
}

// RemoveTeamMemberRequest 移除团队成员请求。
type RemoveTeamMemberRequest struct {
	UserID int `json:"userId" binding:"required,min=1"`
}

package dto

import "time"

type FeishuOAuthAuthURLResponse struct {
	AuthURL     string `json:"authUrl"`
	RedirectURI string `json:"redirectUri"`
	State       string `json:"state"`
}

type FeishuOAuthCallbackResponse struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int    `json:"expiresIn,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
	Scope        string `json:"scope,omitempty"`
	UserID       string `json:"userId,omitempty"`
	OpenID       string `json:"openId,omitempty"`
	UnionID      string `json:"unionId,omitempty"`
}

type FeishuTaskResponse struct {
	GUID        string                 `json:"guid"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	Completed   bool                   `json:"completed"`
	CreatorID   string                 `json:"creatorId,omitempty"`
	Assignees   []string               `json:"assignees,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

type FeishuTicketSyncResponse struct {
	ID                int                 `json:"id,omitempty"`
	TenantID          int                 `json:"tenantId"`
	TicketID          int                 `json:"ticketId"`
	TicketNumber      string              `json:"ticketNumber,omitempty"`
	FeishuTaskID      string              `json:"feishuTaskId"`
	FeishuTaskGUID    string              `json:"feishuTaskGuid,omitempty"`
	SyncStatus        string              `json:"syncStatus"`
	LastSyncDirection string              `json:"lastSyncDirection,omitempty"`
	LastSyncedAt      *time.Time          `json:"lastSyncedAt,omitempty"`
	ErrorMessage      string              `json:"errorMessage,omitempty"`
	Task              *FeishuTaskResponse `json:"task,omitempty"`
}

type FeishuWebhookResponse struct {
	EventType string                    `json:"eventType,omitempty"`
	Action    string                    `json:"action,omitempty"`
	Sync      *FeishuTicketSyncResponse `json:"sync,omitempty"`
}

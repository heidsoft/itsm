package dto

import (
	"time"

	"itsm-backend/ent"
)

// SSOInitiateRequest 登录发起请求。
// TenantCode 可选：多租户部署下同一提供方编码可能被多个租户启用，
// 携带租户编码即把解析范围收敛到该租户；不带时按编码全局解析，
// 命中多个启用中的提供方则拒绝（与登录租户消歧同一口径，绝不任选一个）。
type SSOInitiateRequest struct {
	Provider   string `json:"provider"`
	TenantCode string `json:"tenantCode,omitempty"`
}

// SSOInitiateResponse 只回跳转地址与 state；state 由前端随回调原样带回，
// 服务端凭它取回 PKCE verifier / nonce / provider 绑定。
type SSOInitiateResponse struct {
	RedirectURL string `json:"redirectUrl"`
	State       string `json:"state"`
	Provider    string `json:"provider"`
}

// SSOCallbackRequest 回调请求。code/state 来自 IdP 重定向后的查询串，
// provider 取自查询参数（与前端 sso/callback 页契约一致）。
type SSOCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// SSOSessionResponse 是 SSO 回调的会话载荷。
// 令牌只进 httpOnly cookie（json:"-"），浏览器可读响应体不得出现凭证；
// expiresIn 由服务端 TTL 给出，前端据此安排续签。
type SSOSessionResponse struct {
	AccessToken  string             `json:"-"`
	RefreshToken string             `json:"-"`
	ExpiresIn    int                `json:"expiresIn"`
	User         *LoginUserResponse `json:"user"`
	Tenant       *ent.Tenant        `json:"tenant"`
	// Provisioned 标记本次登录是否 JIT 新建了账号，便于前端提示首次登录。
	Provisioned bool `json:"provisioned"`
}

// SSOProviderResponse 提供方配置的对外投影。
// ClientSecretEncrypted 一律不回显（连密文也不回显）：管理端更新时可留空
// 表示不改动，读取时无法反推原文，避免任何路径把 secret 泄进响应体。
type SSOProviderResponse struct {
	ID                    int        `json:"id"`
	TenantID              int        `json:"tenantId"`
	Code                  string     `json:"code"`
	Name                  string     `json:"name"`
	Protocol              string     `json:"protocol"`
	ClientID              string     `json:"clientId"`
	HasClientSecret       bool       `json:"hasClientSecret"`
	Issuer                string     `json:"issuer"`
	AuthorizationEndpoint string     `json:"authorizationEndpoint,omitempty"`
	TokenEndpoint         string     `json:"tokenEndpoint,omitempty"`
	UserinfoEndpoint      string     `json:"userinfoEndpoint,omitempty"`
	JWKSURI               string     `json:"jwksUri,omitempty"`
	Scopes                string     `json:"scopes"`
	RedirectURI           string     `json:"redirectUri"`
	AutoProvision         bool       `json:"autoProvision"`
	DefaultRole           string     `json:"defaultRole"`
	AllowedEmailDomains   string     `json:"allowedEmailDomains,omitempty"`
	Enabled               bool       `json:"enabled"`
	Status                string     `json:"status"`
	LastError             string     `json:"lastError,omitempty"`
	LastHealthCheckAt     *time.Time `json:"lastHealthCheckAt,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

// SSOCreateProviderRequest 创建提供方。ClientSecret 明文仅存在于请求体，
// 落库前由 service 加密；留空表示公开客户端。
type SSOCreateProviderRequest struct {
	Code                  string `json:"code" binding:"required,max=50"`
	Name                  string `json:"name" binding:"required,max=100"`
	Protocol              string `json:"protocol" binding:"omitempty,max=20"`
	ClientID              string `json:"clientId" binding:"required,max=200"`
	ClientSecret          string `json:"clientSecret" binding:"omitempty,max=500"`
	Issuer                string `json:"issuer" binding:"required,max=500"`
	AuthorizationEndpoint string `json:"authorizationEndpoint" binding:"omitempty,max=1000"`
	TokenEndpoint         string `json:"tokenEndpoint" binding:"omitempty,max=1000"`
	UserinfoEndpoint      string `json:"userinfoEndpoint" binding:"omitempty,max=1000"`
	JWKSURI               string `json:"jwksUri" binding:"omitempty,max=1000"`
	Scopes                string `json:"scopes" binding:"omitempty,max=500"`
	RedirectURI           string `json:"redirectUri" binding:"required,max=1000"`
	AutoProvision         *bool  `json:"autoProvision"`
	DefaultRole           string `json:"defaultRole" binding:"omitempty,max=50"`
	AllowedEmailDomains   string `json:"allowedEmailDomains" binding:"omitempty,max=1000"`
	Enabled               *bool  `json:"enabled"`
}

// SSOUpdateProviderRequest 更新提供方。
// ClientSecret 为空表示保持原 secret 不变（管理端无法读回原文，
// 因此「留空即不改」是唯一可用语义；要轮换必须显式传新值）。
type SSOUpdateProviderRequest struct {
	Name                  *string `json:"name" binding:"omitempty,max=100"`
	ClientID              *string `json:"clientId" binding:"omitempty,max=200"`
	ClientSecret          *string `json:"clientSecret" binding:"omitempty,max=500"`
	Issuer                *string `json:"issuer" binding:"omitempty,max=500"`
	AuthorizationEndpoint *string `json:"authorizationEndpoint" binding:"omitempty,max=1000"`
	TokenEndpoint         *string `json:"tokenEndpoint" binding:"omitempty,max=1000"`
	UserinfoEndpoint      *string `json:"userinfoEndpoint" binding:"omitempty,max=1000"`
	JWKSURI               *string `json:"jwksUri" binding:"omitempty,max=1000"`
	Scopes                *string `json:"scopes" binding:"omitempty,max=500"`
	RedirectURI           *string `json:"redirectUri" binding:"omitempty,max=1000"`
	AutoProvision         *bool   `json:"autoProvision"`
	DefaultRole           *string `json:"defaultRole" binding:"omitempty,max=50"`
	AllowedEmailDomains   *string `json:"allowedEmailDomains" binding:"omitempty,max=1000"`
	Enabled               *bool   `json:"enabled"`
}

// SSOBindRequest 已登录用户把当前会话绑定到外部身份。
// code/state 来自一次专用的 SSO 发起（state 记录 mode=bind）。
type SSOBindRequest struct {
	Provider string `json:"provider" binding:"required,max=50"`
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
}

// SSOUnbindRequest 解绑请求。
type SSOUnbindRequest struct {
	Provider string `json:"provider" binding:"required,max=50"`
}

// SSOBrief 是登录页可见的提供方摘要（不含任何端点或凭证信息）。
type SSOBrief struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

// SSOIdentityResponse 当前用户的身份绑定投影。
type SSOIdentityResponse struct {
	ProviderCode     string     `json:"providerCode"`
	ProviderName     string     `json:"providerName"`
	Subject          string     `json:"subject"`
	ExternalUsername string     `json:"externalUsername,omitempty"`
	ExternalEmail    string     `json:"externalEmail,omitempty"`
	LastLoginAt      *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

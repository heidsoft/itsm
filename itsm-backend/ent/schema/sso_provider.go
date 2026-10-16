package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SSOProvider 租户级 SSO（OIDC）身份提供方配置。
//
// client_secret 只以 middleware.EncryptionService 密文落库（client_secret_encrypted），
// 与 ConnectorConfig.encrypted_credentials 同一口径；任何读取路径都不得回显明文。
// 端点字段（authorization/token/userinfo/jwks）可为空：为空时按 OIDC Discovery
// （issuer + /.well-known/openid-configuration）自动派生。
type SSOProvider struct{ ent.Schema }

func (SSOProvider) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("code").NotEmpty().MaxLen(50).
			Comment("提供方编码：登录发起与回调按编码解析，租户内唯一"),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("protocol").Default("oidc").MaxLen(20),
		field.String("client_id").NotEmpty().MaxLen(200),
		field.Text("client_secret_encrypted").Optional().
			Comment("AES-GCM 密文；为空表示公开客户端（无 secret）"),
		field.String("issuer").NotEmpty().MaxLen(500),
		field.String("authorization_endpoint").Optional().MaxLen(1000),
		field.String("token_endpoint").Optional().MaxLen(1000),
		field.String("userinfo_endpoint").Optional().MaxLen(1000),
		field.String("jwks_uri").Optional().MaxLen(1000),
		field.String("scopes").Default("openid profile email").MaxLen(500),
		field.String("redirect_uri").NotEmpty().MaxLen(1000).
			Comment("在本平台注册的回调地址，授权请求与令牌交换都必须原样携带"),
		field.Bool("auto_provision").Default(true).
			Comment("身份未绑定时是否按 default_role JIT 建号"),
		field.String("default_role").Default("end_user").MaxLen(50),
		field.String("allowed_email_domains").Optional().MaxLen(1000).
			Comment("逗号分隔的邮箱域白名单；空表示不限制"),
		field.Bool("enabled").Default(false),
		field.String("status").Default("configured").MaxLen(40),
		field.String("last_error").Optional().MaxLen(2000),
		field.Time("last_health_check_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SSOProvider) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "code").Unique(),
		// 登录发起发生在认证之前，没有租户上下文：按 code 解析启用中的提供方。
		index.Fields("code", "enabled"),
	}
}

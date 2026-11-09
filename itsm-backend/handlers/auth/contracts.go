package auth

// 域内窄契约：认证域只消费邮件服务的密码重置投递能力。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import "context"

type EmailService interface {
	SendPasswordResetEmail(ctx context.Context, to []string, resetToken, baseURL string) error
}

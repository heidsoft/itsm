package auth

import (
	"context"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

// =====================================================================
// 测试夹具
// 从 service/auth_service_ext_test.go 迁移（AuthService 随 361f03976 迁入
// handlers/auth 并更名 Service 后，原测试文件滞留 service 包导致构建失败）。
// 仅迁移仍然存在的方法：SwitchTenant/Register/ForgotPassword/ValidateResetToken/
// ResetPassword。Logout/ValidateUser/RevokeUserTokens 等方法已在架构迁移中
// 移至 HTTP 层（handlers/common）与 middleware，对应测试不再适用。
// =====================================================================

type authFixture struct {
	client  *ent.Client
	service *Service
	tenant  *ent.Tenant
	tenant2 *ent.Tenant
	user    *ent.User
	ctx     context.Context
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	ctx := context.Background()

	client := enttest.Open(t, "sqlite3", "file:auth_ext?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	svc := &Service{
		client:    client,
		jwtSecret: "test-secret-key",
		logger:    logger,
		baseURL:   "http://localhost:3000",
	}

	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tenant2, err := client.Tenant.Create().
		SetName("Tenant Two").
		SetCode("tenant2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("alice").
		SetEmail("alice@example.com").
		SetName("Alice").
		SetPasswordHash(string(hashed)).
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return &authFixture{
		client:  client,
		service: svc,
		tenant:  tenant,
		tenant2: tenant2,
		user:    user,
		ctx:     ctx,
	}
}

// =====================================================================
// SwitchTenant
// =====================================================================

func TestService_SwitchTenant(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("切换到当前所属租户应成功", func(t *testing.T) {
		resp, err := fx.service.SwitchTenant(fx.ctx, fx.user.ID, fx.tenant.ID)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, fx.user.ID, resp.User.ID)
		assert.Equal(t, fx.tenant.ID, resp.User.TenantID)
		assert.Equal(t, fx.tenant.ID, resp.Tenant.ID)
		// end_user 角色应有非空 permissions（来自 RolePermissions）
		assert.NotEmpty(t, resp.User.Permissions)
	})

	t.Run("用户不在目标租户返回无权限错误", func(t *testing.T) {
		// tenant2 中没有该用户 → 应返回 "无权限访问该租户"
		_, err := fx.service.SwitchTenant(fx.ctx, fx.user.ID, fx.tenant2.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无权限访问该租户")
	})

	t.Run("租户不存在", func(t *testing.T) {
		_, err := fx.service.SwitchTenant(fx.ctx, fx.user.ID, 99999)
		require.Error(t, err)
	})
}

// =====================================================================
// Register
// =====================================================================

func TestService_Register(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("多活跃租户时未指定租户代码须被拒绝", func(t *testing.T) {
		// fixture 有两个 active 租户(tenant/tenant2)，注册不指定 TenantCode
		// 必须失败 closed，禁止回退硬编码默认租户
		req := &dto.RegisterRequest{
			Username:    "bob",
			Email:       "bob@example.com",
			Password:    "securePass1",
			DisplayName: "Bob Builder",
			Phone:       "13900000000",
			Company:     "Acme",
			Role:        "end_user",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "租户")
	})

	t.Run("单一活跃租户时未指定租户代码注册到该租户", func(t *testing.T) {
		// 挂起 tenant2 模拟单租户私有部署
		_, err := fx.client.Tenant.UpdateOneID(fx.tenant2.ID).
			SetStatus("suspended").
			Save(fx.ctx)
		require.NoError(t, err)

		req := &dto.RegisterRequest{
			Username:    "bob",
			Email:       "bob@example.com",
			Password:    "securePass1",
			DisplayName: "Bob Builder",
			Phone:       "13900000000",
			Company:     "Acme",
			Role:        "end_user",
		}
		resp, err := fx.service.Register(fx.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "bob", resp.Username)
		assert.Equal(t, "bob@example.com", resp.Email)
		assert.Contains(t, resp.Message, "成功")

		// 验证数据库中确实创建，且落入唯一 active 租户而非硬编码租户 1
		saved, err := fx.client.User.Get(fx.ctx, resp.ID)
		require.NoError(t, err)
		assert.Equal(t, "bob", saved.Username)
		assert.Equal(t, "Bob Builder", saved.Name)
		assert.Equal(t, "13900000000", saved.Phone)
		assert.Equal(t, "Acme", saved.Department)
		assert.True(t, saved.Active)
		assert.Equal(t, fx.tenant.ID, saved.TenantID)
	})

	t.Run("用户名重复返回错误", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username:    "alice", // 已存在
			Email:       "new@example.com",
			Password:    "securePass1",
			DisplayName: "Imposter",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "用户名已被注册")
	})

	t.Run("邮箱重复返回错误", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username:    "charlie",
			Email:       "alice@example.com", // 已存在
			Password:    "securePass1",
			DisplayName: "Charlie",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "邮箱已被注册")
	})

	t.Run("指定租户 code 成功注册", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username:    "diana",
			Email:       "diana@example.com",
			Password:    "securePass1",
			DisplayName: "Diana",
			TenantCode:  "tenant2",
		}
		resp, err := fx.service.Register(fx.ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "diana", resp.Username)

		saved, err := fx.client.User.Get(fx.ctx, resp.ID)
		require.NoError(t, err)
		assert.Equal(t, fx.tenant2.ID, saved.TenantID)
	})

	t.Run("指定不存在的租户 code 返回错误", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username:    "eve",
			Email:       "eve@example.com",
			Password:    "securePass1",
			DisplayName: "Eve",
			TenantCode:  "ghost-tenant",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "租户不存在")
	})
}

// =====================================================================
// ForgotPassword
// =====================================================================

func TestService_ForgotPassword(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("用户存在且 email service 为 nil 时静默成功", func(t *testing.T) {
		req := &dto.ForgotPasswordRequest{Email: "alice@example.com"}
		resp, err := fx.service.ForgotPassword(fx.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "如果该邮箱已注册")

		// 验证 alice 的 token 已生成
		count, err := fx.client.PasswordResetToken.Query().Count(fx.ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1, "应该至少生成一个重置令牌")
	})

	t.Run("用户不存在时仍返回成功（安全考虑）", func(t *testing.T) {
		req := &dto.ForgotPasswordRequest{Email: "ghost@example.com"}
		resp, err := fx.service.ForgotPassword(fx.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "如果该邮箱已注册")

		// 查询 ghost email 的 token（应该为 0，因为用户不存在）
		all, err := fx.client.PasswordResetToken.Query().All(fx.ctx)
		require.NoError(t, err)
		for _, tok := range all {
			assert.NotEqual(t, "ghost@example.com", tok.Email,
				"不存在的 email 不应该生成 token")
		}
	})

	t.Run("指定租户 code 且租户不存在时也应返回通用成功（不泄露存在性）", func(t *testing.T) {
		// 安全考虑：不应区分"租户不存在"和"用户不存在"，避免泄露租户存在性。
		req := &dto.ForgotPasswordRequest{
			Email:      "alice@example.com",
			TenantCode: "ghost-tenant",
		}
		resp, err := fx.service.ForgotPassword(fx.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "如果该邮箱已注册")
	})

	t.Run("指定正确租户 code 生成 token", func(t *testing.T) {
		req := &dto.ForgotPasswordRequest{
			Email:      "alice@example.com",
			TenantCode: "test",
		}
		resp, err := fx.service.ForgotPassword(fx.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		count, _ := fx.client.PasswordResetToken.Query().Count(fx.ctx)
		assert.GreaterOrEqual(t, count, 1)
	})
}

// =====================================================================
// ValidateResetToken
// =====================================================================

func TestService_ValidateResetToken(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	// 创建有效 token
	expiresAt := time.Now().Add(1 * time.Hour)
	validToken, err := fx.client.PasswordResetToken.Create().
		SetUserID(fx.user.ID).
		SetEmail(fx.user.Email).
		SetToken("valid-token-abc").
		SetExpiresAt(expiresAt).
		Save(fx.ctx)
	require.NoError(t, err)

	// 创建过期 token
	_, err = fx.client.PasswordResetToken.Create().
		SetUserID(fx.user.ID).
		SetEmail(fx.user.Email).
		SetToken("expired-token-xyz").
		SetExpiresAt(time.Now().Add(-1 * time.Hour)).
		Save(fx.ctx)
	require.NoError(t, err)

	// 创建已使用 token
	_, err = fx.client.PasswordResetToken.Create().
		SetUserID(fx.user.ID).
		SetEmail(fx.user.Email).
		SetToken("used-token-111").
		SetExpiresAt(expiresAt).
		SetUsed(true).
		Save(fx.ctx)
	require.NoError(t, err)

	t.Run("有效 token 返回 Valid=true", func(t *testing.T) {
		resp, err := fx.service.ValidateResetToken(fx.ctx, &dto.ValidateResetTokenRequest{
			Token: validToken.Token,
			Email: fx.user.Email,
		})
		require.NoError(t, err)
		assert.True(t, resp.Valid)
		assert.Equal(t, fx.user.Email, resp.Email)
	})

	t.Run("不存在 token 返回 Valid=false", func(t *testing.T) {
		resp, err := fx.service.ValidateResetToken(fx.ctx, &dto.ValidateResetTokenRequest{
			Token: "not-exist",
			Email: fx.user.Email,
		})
		require.NoError(t, err)
		assert.False(t, resp.Valid)
	})

	t.Run("过期 token 返回 Valid=false", func(t *testing.T) {
		resp, err := fx.service.ValidateResetToken(fx.ctx, &dto.ValidateResetTokenRequest{
			Token: "expired-token-xyz",
			Email: fx.user.Email,
		})
		require.NoError(t, err)
		assert.False(t, resp.Valid)
	})

	t.Run("已使用 token 返回 Valid=false", func(t *testing.T) {
		resp, err := fx.service.ValidateResetToken(fx.ctx, &dto.ValidateResetTokenRequest{
			Token: "used-token-111",
			Email: fx.user.Email,
		})
		require.NoError(t, err)
		assert.False(t, resp.Valid)
	})
}

// =====================================================================
// ResetPassword
// =====================================================================

func TestService_ResetPassword(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("两次密码不一致返回错误", func(t *testing.T) {
		req := &dto.PasswordResetRequest{
			Token:           "any",
			Email:           fx.user.Email,
			Password:        "newPass123",
			PasswordConfirm: "differentPass123",
		}
		_, err := fx.service.ResetPassword(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "两次输入的密码不一致")
	})

	t.Run("token 不存在返回错误", func(t *testing.T) {
		req := &dto.PasswordResetRequest{
			Token:           "not-exist-token",
			Email:           fx.user.Email,
			Password:        "newPass123",
			PasswordConfirm: "newPass123",
		}
		_, err := fx.service.ResetPassword(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "令牌无效或已使用")
	})

	t.Run("成功重置密码并标记 token 已使用", func(t *testing.T) {
		// 创建有效 token
		tok, err := fx.client.PasswordResetToken.Create().
			SetUserID(fx.user.ID).
			SetEmail(fx.user.Email).
			SetToken("happy-token-999").
			SetExpiresAt(time.Now().Add(time.Hour)).
			Save(fx.ctx)
		require.NoError(t, err)

		req := &dto.PasswordResetRequest{
			Token:           tok.Token,
			Email:           fx.user.Email,
			Password:        "brand-new-pass",
			PasswordConfirm: "brand-new-pass",
		}
		resp, err := fx.service.ResetPassword(fx.ctx, req)
		require.NoError(t, err)
		assert.Contains(t, resp.Message, "成功")

		// 验证密码已更新
		updated, err := fx.client.User.Get(fx.ctx, fx.user.ID)
		require.NoError(t, err)
		bcryptErr := bcrypt.CompareHashAndPassword(
			[]byte(updated.PasswordHash), []byte("brand-new-pass"),
		)
		assert.NoError(t, bcryptErr, "新密码应可验证")

		// 验证 token 已被标记为已使用
		reloaded, err := fx.client.PasswordResetToken.Get(fx.ctx, tok.ID)
		require.NoError(t, err)
		assert.True(t, reloaded.Used, "token 应该被标记为已使用")
	})

	t.Run("使用过期 token 返回错误", func(t *testing.T) {
		tok, err := fx.client.PasswordResetToken.Create().
			SetUserID(fx.user.ID).
			SetEmail(fx.user.Email).
			SetToken("old-token-888").
			SetExpiresAt(time.Now().Add(-time.Hour)).
			Save(fx.ctx)
		require.NoError(t, err)

		req := &dto.PasswordResetRequest{
			Token:           tok.Token,
			Email:           fx.user.Email,
			Password:        "newPass123",
			PasswordConfirm: "newPass123",
		}
		_, err = fx.service.ResetPassword(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "令牌无效或已使用")
	})
}

// =====================================================================
// generateResetToken 唯一性（内部函数间接测试）
// =====================================================================

func TestGenerateResetToken_Distinct(t *testing.T) {
	// 通过多次调用 ForgotPassword 触发 token 生成，间接验证 token 唯一性
	fx := newAuthFixture(t)
	defer fx.client.Close()

	const N = 5
	seen := make(map[string]struct{}, N)
	for i := 0; i < N; i++ {
		// 用同一个 email 多次请求
		_, err := fx.service.ForgotPassword(fx.ctx, &dto.ForgotPasswordRequest{
			Email: fx.user.Email,
		})
		require.NoError(t, err)
	}

	tokens, err := fx.client.PasswordResetToken.Query().All(fx.ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tokens), N)

	for _, tok := range tokens {
		// token 是 32 字节 hex = 64 字符
		assert.Len(t, tok.Token, 64, "reset token 应该是 32 字节 hex (64字符)")
		if _, dup := seen[tok.Token]; dup {
			t.Fatalf("reset token 不应重复: %s", tok.Token)
		}
		seen[tok.Token] = struct{}{}
	}
}

// 防止 unused import 标记
var _ = ent.User{}

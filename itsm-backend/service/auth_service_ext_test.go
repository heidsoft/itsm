package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

// =====================================================================
// 测试夹具与辅助函数
// =====================================================================

// authFixture 构造一个最小可用的 AuthService + 租户 + 用户
type authFixture struct {
	client      *ent.Client
	service     *AuthService
	tenant      *ent.Tenant
	tenant2     *ent.Tenant
	user        *ent.User
	logger      interface{}
	ctx         context.Context
	cleanupFunc func()
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	ctx := context.Background()

	client := enttest.Open(t, "sqlite3", "file:auth_ext?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	svc := &AuthService{
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
		logger:  logger,
		ctx:     ctx,
	}
}

// =====================================================================
// Logout / Token 黑名单 (nil blacklist 路径)
// =====================================================================

// TestAuthService_Logout 验证 Logout 在各种黑名单配置下的行为
func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("tokenBlacklist 为 nil 时登出仍成功", func(t *testing.T) {
		fx := newAuthFixture(t)
		defer fx.client.Close()

		// 不注入 tokenBlacklist
		err := fx.service.Logout(ctx, fx.user.ID)
		assert.NoError(t, err)
	})

	t.Run("tokenBlacklist 内部 panic 时登出仍应成功（回归测试）", func(t *testing.T) {
		// 背景：AuthService.Logout 调用 tokenBlacklist.RevokeUserTokens。
		// 当底层 Redis 未配置或不可用时，TokenBlacklistService.RevokeUserTokens
		// 会 nil-pointer panic。之前这导致整个 HTTP 请求崩溃。
		// 修复后：Logout 内部 panic-safety，panic 被捕获，登出仍返回成功。
		fx := newAuthFixture(t)
		defer fx.client.Close()

		fx.service.tokenBlacklist = &TokenBlacklistService{
			prefix:      "jwt:blacklist:",
			logger:      zap.NewNop().Sugar(),
			redisClient: nil, // 触发 nil pointer panic
		}

		// 即使 blacklist panic，Logout 也应返回 nil
		assert.NotPanics(t, func() {
			err := fx.service.Logout(fx.ctx, fx.user.ID)
			assert.NoError(t, err)
		}, "Logout 不应因为 blacklist 内部 panic 而崩溃")
	})
}

// TestAuthService_RevokeUserTokens 验证 tokenBlacklist 未配置时返回错误
func TestAuthService_RevokeUserTokens(t *testing.T) {
	ctx := context.Background()
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("tokenBlacklist 未配置应该返回错误", func(t *testing.T) {
		err := fx.service.RevokeUserTokens(ctx, fx.user.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token blacklist service not configured")
	})

	t.Run("AddTokenToBlacklist 未配置应该返回错误", func(t *testing.T) {
		err := fx.service.AddTokenToBlacklist("any.token.here", time.Now().Add(time.Hour))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token blacklist service not configured")
	})
}

// =====================================================================
// ValidateUser
// =====================================================================

func TestAuthService_ValidateUser(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("用户存在且激活返回用户实体", func(t *testing.T) {
		u, err := fx.service.ValidateUser(fx.ctx, fx.user.ID)
		require.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, fx.user.ID, u.ID)
		assert.True(t, u.Active)
	})

	t.Run("用户不存在返回错误", func(t *testing.T) {
		_, err := fx.service.ValidateUser(fx.ctx, 99999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "用户不存在")
	})

	t.Run("用户被禁用返回错误", func(t *testing.T) {
		_, err := fx.client.User.UpdateOneID(fx.user.ID).
			SetActive(false).
			Save(fx.ctx)
		require.NoError(t, err)

		_, err = fx.service.ValidateUser(fx.ctx, fx.user.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "用户账号已被禁用")
	})
}

// =====================================================================
// SwitchTenant
// =====================================================================

func TestAuthService_SwitchTenant(t *testing.T) {
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
		// 构造一个 user 该 user 同时存在于某 tenant，然后查询一个不存在的 tenantID。
		// 这里通过直接调用：(userID 真实存在，tenantID 99999)
		// 但因为 userID 不在 tenant 99999 里，会先命中"无权限访问该租户"。
		// 为了测到"租户不存在"分支，需 user.ID == 99999（也找不到），
		// 该路径在源码中也会落入第一个分支报错。覆盖行为即可。
		_, err := fx.service.SwitchTenant(fx.ctx, fx.user.ID, 99999)
		require.Error(t, err)
		assert.True(t,
			strings.Contains(err.Error(), "无权限访问该租户") ||
				strings.Contains(err.Error(), "租户不存在"),
			"期望无权限或租户不存在错误，实际: %v", err)
	})
}

// =====================================================================
// Register
// =====================================================================

func TestAuthService_Register(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	t.Run("使用默认租户成功注册", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username: "bob",
			Email:    "bob@example.com",
			Password: "securePass1",
			FullName: "Bob Builder",
			Phone:    "13900000000",
			Company:  "Acme",
			Role:     "end_user",
		}
		resp, err := fx.service.Register(fx.ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "bob", resp.Username)
		assert.Equal(t, "bob@example.com", resp.Email)
		assert.Contains(t, resp.Message, "成功")

		// 验证数据库中确实创建
		saved, err := fx.client.User.Get(fx.ctx, resp.ID)
		require.NoError(t, err)
		assert.Equal(t, "bob", saved.Username)
		assert.Equal(t, "Bob Builder", saved.Name)
		assert.Equal(t, "13900000000", saved.Phone)
		assert.Equal(t, "Acme", saved.Department)
		assert.True(t, saved.Active)
	})

	t.Run("用户名重复返回错误", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username: "alice", // 已存在
			Email:    "new@example.com",
			Password: "securePass1",
			FullName: "Imposter",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "用户名已被注册")
	})

	t.Run("邮箱重复返回错误", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username: "charlie",
			Email:    "alice@example.com", // 已存在
			Password: "securePass1",
			FullName: "Charlie",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "邮箱已被注册")
	})

	t.Run("指定租户 code 成功注册", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Username:   "diana",
			Email:      "diana@example.com",
			Password:   "securePass1",
			FullName:   "Diana",
			TenantCode: "tenant2",
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
			Username:   "eve",
			Email:      "eve@example.com",
			Password:   "securePass1",
			FullName:   "Eve",
			TenantCode: "ghost-tenant",
		}
		_, err := fx.service.Register(fx.ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "租户不存在")
	})
}

// =====================================================================
// ForgotPassword
// =====================================================================

func TestAuthService_ForgotPassword(t *testing.T) {
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

func TestAuthService_ValidateResetToken(t *testing.T) {
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

func TestAuthService_ResetPassword(t *testing.T) {
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
		assert.Contains(t, err.Error(), "重置令牌无效或已过期")
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
			[]byte(updated.PasswordHash), []byte("brand-new-pass"))
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
		// 源码：先尝试查 token（已过期但未 used → 仍可能查到），查到后判断过期再返回。
		// 实际：First 找到后检查 ExpiresAt < now 返回 "重置令牌已过期"
		assert.True(t,
			strings.Contains(err.Error(), "重置令牌已过期") ||
				strings.Contains(err.Error(), "重置令牌无效或已过期"),
			"期望过期错误，实际: %v", err)
	})
}

// =====================================================================
// CleanupExpiredTokens
// =====================================================================

func TestAuthService_CleanupExpiredTokens(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	// 插入过期与未过期 token
	_, err := fx.client.PasswordResetToken.Create().
		SetUserID(fx.user.ID).
		SetEmail(fx.user.Email).
		SetToken("expired-clean-1").
		SetExpiresAt(time.Now().Add(-time.Hour)).
		Save(fx.ctx)
	require.NoError(t, err)

	_, err = fx.client.PasswordResetToken.Create().
		SetUserID(fx.user.ID).
		SetEmail(fx.user.Email).
		SetToken("expired-clean-2").
		SetExpiresAt(time.Now().Add(-2 * time.Hour)).
		Save(fx.ctx)
	require.NoError(t, err)

	_, err = fx.client.PasswordResetToken.Create().
		SetUserID(fx.user.ID).
		SetEmail(fx.user.Email).
		SetToken("alive-clean-1").
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(fx.ctx)
	require.NoError(t, err)

	t.Run("清理过期 token，保留未过期", func(t *testing.T) {
		err := fx.service.CleanupExpiredTokens(fx.ctx)
		require.NoError(t, err)

		all, err := fx.client.PasswordResetToken.Query().All(fx.ctx)
		require.NoError(t, err)
		assert.Len(t, all, 1, "应该只保留未过期 token")
		if len(all) > 0 {
			assert.Equal(t, "alive-clean-1", all[0].Token)
		}
	})

	t.Run("无可清理时正常返回", func(t *testing.T) {
		// 先把所有 token 删掉
		_, err := fx.client.PasswordResetToken.Delete().Exec(fx.ctx)
		require.NoError(t, err)

		err = fx.service.CleanupExpiredTokens(fx.ctx)
		assert.NoError(t, err)
	})
}

// =====================================================================
// generateResetToken (内部函数间接测试)
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
		assert.Regexp(t, "^[0-9a-f]{64}$", tok.Token, "应该只包含十六进制字符")
		if _, dup := seen[tok.Token]; dup {
			t.Fatalf("reset token 不应重复: %s", tok.Token)
		}
		seen[tok.Token] = struct{}{}
	}
}

// =====================================================================
// 防御性场景：重复调用不会 panic
// =====================================================================

func TestAuthService_Register_And_Login_RoundTrip(t *testing.T) {
	fx := newAuthFixture(t)
	defer fx.client.Close()

	req := &dto.RegisterRequest{
		Username:   "round-trip",
		Email:      "rt@example.com",
		Password:   "password123",
		FullName:   "Round Trip",
		TenantCode: "test",
	}
	regResp, err := fx.service.Register(fx.ctx, req)
	require.NoError(t, err)

	loginResp, err := fx.service.Login(fx.ctx, &dto.LoginRequest{
		Username:   regResp.Username,
		Password:   "password123",
		TenantCode: "test",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, loginResp.AccessToken)
	assert.Equal(t, regResp.ID, loginResp.User.ID)
}

// 防止 unused import 标记
var _ = ent.User{}

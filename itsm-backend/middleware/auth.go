package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"itsm-backend/common"
	"itsm-backend/internal/authz"
)

type Claims struct {
	UserID    int    `json:"userId"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TenantID  int    `json:"tenantId"`
	TokenType string `json:"tokenType"` // "access" 或 "refresh"
	// Phase B（2026-11-17）主体标记：空 = human（存量 token 行为不变）；
	// service = 系统间服务账号。该字段随 JWT 一起签名，调用方无法自行篡改。
	// 刻意不放进 gin context 的裸键当作授权依据——授权仍以 authz.WithPrincipal
	// 注入的 request context 为准（ExecuteTool Gate 2.5 从那里读）。
	Principal string `json:"principal,omitempty"`
	jwt.RegisteredClaims
}

// ServicePrincipalRole 是 service 主体 JWT 使用的载体角色。
//
// 选 audit_admin 而非新增 service 角色的原因：
//   - audit_admin 是已存在的全租户只读码集，无任何写/审批权限（roles.go 注释锁定）；
//   - 它已在角色词表、DB seeder、前端选项里登记，不需要污染 users.role 枚举——
//     service 是进程身份，不是可登录的人类用户；
//   - Gate 2.5 对 service 主体仍按 DB 策略零信任判定，载体角色只决定
//     RequirePermission 这一层的只读边界，不构成提权通道。
const ServicePrincipalRole = "audit_admin"

// GenerateServiceToken 签发系统间服务账号的受限 access token（Phase B / G3b）。
//
// 与 GenerateAccessToken 的差异，除 Principal 标记外：
//   - UserID=0：service 不是人类用户，不进用户表；审计里以 0 + principal=service 区分；
//   - 不做吊销/MinIssuedAt 检查：服务凭证的撤销靠轮换签名密钥或更换 token，
//     不与人类账号的改密/停用生命周期挂钩（见 AuthMiddleware 的分支）。
//
// 调用方须把 tenantID 固定为部署环境注入的值，禁止按请求参数自报租户。
func GenerateServiceToken(username, role string, tenantID int, jwtSecret string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("service token ttl must be positive")
	}
	if tenantID <= 0 {
		return "", fmt.Errorf("service token requires a fixed tenant id")
	}
	if role != ServicePrincipalRole {
		return "", fmt.Errorf("service token role must be %q, got %q", ServicePrincipalRole, role)
	}
	claims := Claims{
		UserID:    0,
		Username:  username,
		Role:      role,
		TenantID:  tenantID,
		TokenType: "access",
		Principal: authz.PrincipalService,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        fmt.Sprintf("svc-%d", time.Now().UnixNano()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// 会话凭证的唯一定义：cookie 名称与两类 token 的生命周期。
// 设置 cookie、读取回退、Max-Age 和 expiresIn 都必须从这里派生；分散的
// 900/604800/“access_token” 字面量会让浏览器凭证和服务端校验窗口分别改动后悄悄分叉。
const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
	AccessTokenTTL     = 15 * time.Minute
	RefreshTokenTTL    = 7 * 24 * time.Hour
)

// ValidateAccessToken 验证 access token 并返回声明。
func ValidateAccessToken(tokenString, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != "access" {
		return nil, jwt.ErrInvalidKey
	}
	return claims, nil
}

// ValidateRefreshToken 验证refresh token
func ValidateRefreshToken(tokenString, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != "refresh" {
		return nil, jwt.ErrInvalidKey
	}

	return claims, nil
}

// 生成Access Token
func GenerateAccessToken(userID int, username, role string, tenantID int, jwtSecret string, expireTime time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		TenantID:  tenantID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// GenerateRefreshToken 生成 refresh token。
//
// claims 必须携带完整的 username/role/tenantId 而不只是 userID：续签 handler 依赖这些
// 字段解析租户上下文，缺失时只能降级为匿名。调用方须传入用户完整信息。
func GenerateRefreshToken(userID int, username, role string, tenantID int, jwtSecret string, expireTime time.Duration) (string, error) {
	// jti 用于 refresh token 黑名单唯一标识；带随机后缀避免同一秒重复
	jti := fmt.Sprintf("rt-%d-%d-%d", userID, time.Now().UnixNano(), randSeq6())
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		TenantID:  tenantID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// randSeq6 生成 6 位数值序列，仅用于 jti 增加熵，非密码学安全用途
func randSeq6() int64 {
	return time.Now().UnixNano() % 1000000
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")

		// 调试日志：记录收到的请求信息
		zap.S().Infow(
			"AuthMiddleware: received request",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"has_auth_header", authHeader != "",
			"auth_header_prefix", strings.HasPrefix(authHeader, "Bearer "),
		)

		// 如果没有 Authorization header，尝试从 cookie 中获取 (支持 httpOnly cookie)
		if authHeader == "" {
			if cookieToken, err := c.Cookie(AccessTokenCookie); err == nil && cookieToken != "" {
				authHeader = "Bearer " + cookieToken
				zap.S().Infow(
					"AuthMiddleware: using token from cookie",
					"path", c.Request.URL.Path,
				)
			}
		}

		if authHeader == "" {
			zap.S().Warnw(
				"AuthMiddleware: missing Authorization header",
				"path", c.Request.URL.Path,
				"ip", c.ClientIP(),
			)
			common.Fail(c, common.AuthFailedCode, "缺少认证token")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			zap.S().Warnw(
				"AuthMiddleware: invalid token format",
				"path", c.Request.URL.Path,
				"prefix", authHeader[:min(10, len(authHeader))],
			)
			common.Fail(c, common.AuthFailedCode, "token格式错误")
			c.Abort()
			return
		}

		// 提取token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			zap.S().Warnw(
				"AuthMiddleware: empty token",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "token不能为空")
			c.Abort()
			return
		}

		// 调试日志：记录token解析前的信息
		zap.S().Infow(
			"AuthMiddleware: parsing token",
			"path", c.Request.URL.Path,
			"token_length", len(tokenString),
		)

		// 解析JWT token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法，防止算法混淆攻击
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil {
			zap.S().Warnw(
				"AuthMiddleware: token parse failed",
				"path", c.Request.URL.Path,
				"error", err.Error(),
				"error_type", fmt.Sprintf("%T", err),
			)
			common.Fail(c, common.AuthFailedCode, "token无效")
			c.Abort()
			return
		}

		if !token.Valid {
			zap.S().Warnw(
				"AuthMiddleware: token invalid",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "token无效")
			c.Abort()
			return
		}

		// 提取用户信息
		if claims, ok := token.Claims.(*Claims); ok {
			// 检查TokenType，必须是access类型
			if claims.TokenType != "access" {
				zap.S().Warnw(
					"AuthMiddleware: invalid token type",
					"path", c.Request.URL.Path,
					"token_type", claims.TokenType,
				)
				common.Fail(c, common.AuthFailedCode, "无效的token类型，请使用access token")
				c.Abort()
				return
			}

			// Phase B（2026-11-17）：主体标记校验。principal 随 JWT 签名，
			// 外部无法篡改，但签发端仍可能误配（例如拿 service token 挂了
			// 高权限角色）。这里把签发约束在入站再验一次，fail-closed。
			principal := claims.Principal
			if principal == "" {
				principal = authz.PrincipalHuman // 存量 token 行为不变
			}
			principalKnown := false
			for _, p := range authz.AllPrincipals {
				if p == principal {
					principalKnown = true
					break
				}
			}
			if !principalKnown {
				zap.S().Warnw("AuthMiddleware: unknown principal rejected",
					"path", c.Request.URL.Path, "principal", principal)
				common.Fail(c, common.AuthFailedCode, "无效的主体标记")
				c.Abort()
				return
			}
			if principal == authz.PrincipalService &&
				(claims.Role != ServicePrincipalRole || claims.UserID != 0 || claims.TenantID <= 0) {
				zap.S().Warnw("AuthMiddleware: service principal with non-readonly claims rejected",
					"path", c.Request.URL.Path, "role", claims.Role,
					"user_id", claims.UserID, "tenant_id", claims.TenantID)
				common.Fail(c, common.AuthFailedCode, "服务主体凭证不合法")
				c.Abort()
				return
			}

			revoked, revocationErr := isAccessTokenRevoked(c.Request.Context(), tokenString)
			if revocationErr != nil {
				zap.S().Errorw("AuthMiddleware: token revocation check failed",
					"path", c.Request.URL.Path, "error", revocationErr)
				common.Fail(c, common.AuthFailedCode, "token状态验证失败")
				c.Abort()
				return
			}
			if revoked {
				zap.S().Warnw("AuthMiddleware: revoked token rejected",
					"path", c.Request.URL.Path, "user_id", claims.UserID)
				common.Fail(c, common.AuthFailedCode, "token已失效")
				c.Abort()
				return
			}

			// 用户角色变更/停用/改密后，其存量 access token 需立即失效。
			// 比对 token 签发时间与该用户的最低可接受签发时间（MinIssuedAt）。
			// service 主体的 UserID=0 不对应真实用户，没有「改密/停用」语义，
			// 其撤销靠轮换签名密钥或更换 token，故跳过这条人类账号专属检查。
			if principal == authz.PrincipalHuman {
				store := currentTokenRevocationStore()
				minIAT, minIATErr := store.MinIssuedAt(c.Request.Context(), claims.UserID)
				if minIATErr != nil {
					// 查询失败按安全默认拒绝（fail-closed），与吊销检查语义一致。
					zap.S().Errorw("AuthMiddleware: user token min issued-at check failed",
						"path", c.Request.URL.Path, "user_id", claims.UserID, "error", minIATErr)
					common.Fail(c, common.AuthFailedCode, "token状态验证失败")
					c.Abort()
					return
				}
				if !minIAT.IsZero() && claims.IssuedAt != nil && claims.IssuedAt.Time.Before(minIAT) {
					zap.S().Warnw("AuthMiddleware: token issued before user permission change, rejected",
						"path", c.Request.URL.Path, "user_id", claims.UserID,
						"issued_at", claims.IssuedAt.Time, "min_issued_at", minIAT)
					common.Fail(c, common.AuthFailedCode, "账号权限已变更，请重新登录")
					c.Abort()
					return
				}
			}

			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
			c.Set("tenant_id", claims.TenantID) // 添加租户ID
			c.Set("token", tokenString)
			c.Set("principal", principal)

			// ExecuteTool Gate 2.5 从 request context 读主体（authz.PrincipalFromContext）。
			// 只写 gin key 不够——handler 传给 service 的是 c.Request.Context()，
			// 必须把标记注入到那个 context 链上，零信任闸才看得见 service 主体。
			c.Request = c.Request.WithContext(authz.WithPrincipal(c.Request.Context(), principal))

			// 调试日志：认证成功
			zap.S().Infow(
				"AuthMiddleware: authentication successful",
				"path", c.Request.URL.Path,
				"user_id", claims.UserID,
				"username", claims.Username,
				"tenant_id", claims.TenantID,
				"principal", principal,
			)
		} else {
			zap.S().Warnw(
				"AuthMiddleware: failed to extract claims",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "token解析失败")
			c.Abort()
			return
		}

		c.Next()
	}
}

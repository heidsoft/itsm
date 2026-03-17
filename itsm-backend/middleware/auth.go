package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"itsm-backend/common"
	"go.uber.org/zap"
)

type Claims struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TenantID  int    `json:"tenant_id"`
	TokenType string `json:"token_type"` // "access" 或 "refresh"
	jwt.RegisteredClaims
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

// 生成Refresh Token
func GenerateRefreshToken(userID int, jwtSecret string, expireTime time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")

		// 调试日志：记录收到的请求信息
		zap.S().Infow("AuthMiddleware: received request",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"has_auth_header", authHeader != "",
			"auth_header_prefix", strings.HasPrefix(authHeader, "Bearer "),
		)

		if authHeader == "" {
			zap.S().Warnw("AuthMiddleware: missing Authorization header",
				"path", c.Request.URL.Path,
				"ip", c.ClientIP(),
			)
			common.Fail(c, common.AuthFailedCode, "缺少认证token")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			zap.S().Warnw("AuthMiddleware: invalid token format",
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
			zap.S().Warnw("AuthMiddleware: empty token",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "token不能为空")
			c.Abort()
			return
		}

		// 调试日志：记录token解析前的信息
		zap.S().Infow("AuthMiddleware: parsing token",
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
			zap.S().Warnw("AuthMiddleware: token parse failed",
				"path", c.Request.URL.Path,
				"error", err.Error(),
				"error_type", fmt.Sprintf("%T", err),
			)
			common.Fail(c, common.AuthFailedCode, "token无效")
			c.Abort()
			return
		}

		if !token.Valid {
			zap.S().Warnw("AuthMiddleware: token invalid",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "token无效")
			c.Abort()
			return
		}

		// 提取用户信息
		if claims, ok := token.Claims.(*Claims); ok {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
			c.Set("tenant_id", claims.TenantID) // 添加租户ID
			c.Set("token", tokenString)

			// 调试日志：认证成功
			zap.S().Infow("AuthMiddleware: authentication successful",
				"path", c.Request.URL.Path,
				"user_id", claims.UserID,
				"username", claims.Username,
				"tenant_id", claims.TenantID,
			)
		} else {
			zap.S().Warnw("AuthMiddleware: failed to extract claims",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "token解析失败")
			c.Abort()
			return
		}

		c.Next()
	}
}

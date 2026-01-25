package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// JWTClaims JWT Token Claims
type JWTClaims struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TenantID  int    `json:"tenant_id"`
	jwt.RegisteredClaims
}

// TokenBlacklistService Token黑名单服务
type TokenBlacklistService struct {
	redisClient *redis.Client
	logger      *zap.SugaredLogger
	prefix      string
}

// NewTokenBlacklistService 创建Token黑名单服务
func NewTokenBlacklistService(redisClient *redis.Client, logger *zap.SugaredLogger) *TokenBlacklistService {
	return &TokenBlacklistService{
		redisClient: redisClient,
		logger:      logger,
		prefix:      "jwt:blacklist:",
	}
}

// AddToBlacklist 将Token加入黑名单
func (s *TokenBlacklistService) AddToBlacklist(tokenString string, expiresAt time.Time) error {
	ctx := context.Background()

	// 解析token获取 claims
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	// 计算TTL
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Hour // 默认1小时
	}

	// 存储到Redis
	key := s.prefix + tokenString
	claimsJSON, _ := json.Marshal(claims)

	err = s.redisClient.Set(ctx, key, claimsJSON, ttl).Err()
	if err != nil {
		s.logger.Errorw("Failed to add token to blacklist", "error", err, "user_id", claims.UserID)
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	s.logger.Infow("Token added to blacklist", "user_id", claims.UserID, "expires_in", ttl)
	return nil
}

// IsBlacklisted 检查Token是否在黑名单中
func (s *TokenBlacklistService) IsBlacklisted(tokenString string) (bool, error) {
	ctx := context.Background()
	key := s.prefix + tokenString

	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		s.logger.Errorw("Failed to check token blacklist", "error", err)
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return exists > 0, nil
}

// RemoveFromBlacklist 将Token从黑名单移除
func (s *TokenBlacklistService) RemoveFromBlacklist(tokenString string) error {
	ctx := context.Background()
	key := s.prefix + tokenString

	err := s.redisClient.Del(ctx, key).Err()
	if err != nil {
		s.logger.Errorw("Failed to remove token from blacklist", "error", err)
		return fmt.Errorf("failed to remove from blacklist: %w", err)
	}

	s.logger.Infow("Token removed from blacklist", "token_prefix", tokenString[:20]+"...")
	return nil
}

// RevokeUserTokens 撤销用户的所有Token（通过user_id）
func (s *TokenBlacklistService) RevokeUserTokens(ctx context.Context, userID int) error {
	// 使用 SCAN 命令查找该用户的所有被撤销token
	pattern := s.prefix + "*"
	var cursor uint64
	var count int64 = 100

	for {
		keys, nextCursor, err := s.redisClient.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return fmt.Errorf("failed to scan tokens: %w", err)
		}

		for _, key := range keys {
			// 获取token字符串
			tokenString := strings.TrimPrefix(key, s.prefix)

			// 解析token获取user_id
			claims, err := s.parseToken(tokenString)
			if err != nil {
				continue
			}

			// 如果是目标用户的token，添加到黑名单
			if claims.UserID == userID {
				// 获取token的过期时间
				if claims.ExpiresAt != nil {
					expiresAt := claims.ExpiresAt.Time
					if time.Now().Before(expiresAt) {
						err = s.AddToBlacklist(tokenString, expiresAt)
						if err != nil {
							s.logger.Warnw("Failed to revoke token", "error", err, "token", tokenString[:20]+"...")
							continue
						}
						count++
					}
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	s.logger.Infow("User tokens revoked", "user_id", userID, "revoked_count", count)
	return nil
}

// GetBlacklistedClaims 获取黑名单中的claims
func (s *TokenBlacklistService) GetBlacklistedClaims(tokenString string) (*JWTClaims, error) {
	ctx := context.Background()
	key := s.prefix + tokenString

	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var claims JWTClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

// parseToken 解析并验证Token
func (s *TokenBlacklistService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("your-secret-key"), nil // 实际应该从配置获取
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// CleanExpiredTokens 清理过期的黑名单条目
// Redis 会自动处理过期，所以这个方法主要用于日志记录
func (s *TokenBlacklistService) CleanExpiredTokens(ctx context.Context) error {
	count, err := s.redisClient.DBSize(ctx).Result()
	if err != nil {
		return err
	}

	s.logger.Infow("Blacklist cleanup", "total_keys", count)
	return nil
}

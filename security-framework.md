# ITSM 系统安全框架设计

## 1. 安全架构概览

### 1.1 整体安全架构

```
┌─────────────────────────────────────────────────────────────┐
│                    外部安全层                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   WAF/CDN   │  │  DDoS防护   │  │  SSL/TLS    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│                    应用安全层                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  API网关    │  │  认证中心    │  │  授权服务    │          │
│  │  (Kong)     │  │  (Keycloak) │  │  (RBAC)     │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│                    业务安全层                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  数据加密    │  │  审计日志    │  │  权限控制    │          │
│  │  (AES-256)  │  │  (ELK)      │  │  (细粒度)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│                    基础设施安全层                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  网络隔离    │  │  容器安全    │  │  密钥管理    │          │
│  │  (VPC)      │  │  (Pod安全)  │  │  (Vault)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 安全原则

#### 零信任架构
- 永不信任，始终验证
- 最小权限原则
- 持续监控和验证

#### 深度防御
- 多层安全控制
- 冗余安全机制
- 故障安全设计

#### 数据保护
- 数据分类分级
- 端到端加密
- 数据生命周期管理

## 2. 认证授权系统

### 2.1 认证架构设计

```go
// auth/models.go
package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    Username    string    `json:"username" gorm:"uniqueIndex"`
    Email       string    `json:"email" gorm:"uniqueIndex"`
    PasswordHash string   `json:"-" gorm:"column:password_hash"`
    FirstName   string    `json:"first_name"`
    LastName    string    `json:"last_name"`
    Department  string    `json:"department"`
    Position    string    `json:"position"`
    Status      UserStatus `json:"status"`
    LastLoginAt *time.Time `json:"last_login_at"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    
    // 关联关系
    Roles       []Role     `json:"roles" gorm:"many2many:user_roles;"`
    Groups      []Group    `json:"groups" gorm:"many2many:user_groups;"`
    Sessions    []Session  `json:"-" gorm:"foreignKey:UserID"`
}

type UserStatus string

const (
    UserStatusActive    UserStatus = "active"
    UserStatusInactive  UserStatus = "inactive"
    UserStatusLocked    UserStatus = "locked"
    UserStatusSuspended UserStatus = "suspended"
)

// Role 角色模型
type Role struct {
    ID          string       `json:"id" gorm:"primaryKey"`
    Name        string       `json:"name" gorm:"uniqueIndex"`
    DisplayName string       `json:"display_name"`
    Description string       `json:"description"`
    Type        RoleType     `json:"type"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    
    // 关联关系
    Permissions []Permission `json:"permissions" gorm:"many2many:role_permissions;"`
    Users       []User       `json:"-" gorm:"many2many:user_roles;"`
}

type RoleType string

const (
    RoleTypeSystem   RoleType = "system"
    RoleTypeBusiness RoleType = "business"
    RoleTypeCustom   RoleType = "custom"
)

// Permission 权限模型
type Permission struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    Resource    string    `json:"resource"`    // 资源类型：ticket, incident, user等
    Action      string    `json:"action"`      // 操作类型：create, read, update, delete
    Scope       string    `json:"scope"`       // 权限范围：own, department, all
    Condition   string    `json:"condition"`   // 条件表达式
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    
    // 关联关系
    Roles []Role `json:"-" gorm:"many2many:role_permissions;"`
}

// Session 会话模型
type Session struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    UserID      string    `json:"user_id" gorm:"index"`
    Token       string    `json:"-" gorm:"uniqueIndex"`
    RefreshToken string   `json:"-" gorm:"uniqueIndex"`
    IPAddress   string    `json:"ip_address"`
    UserAgent   string    `json:"user_agent"`
    ExpiresAt   time.Time `json:"expires_at"`
    CreatedAt   time.Time `json:"created_at"`
    LastUsedAt  time.Time `json:"last_used_at"`
    
    // 关联关系
    User User `json:"user" gorm:"foreignKey:UserID"`
}

// JWT Claims
type JWTClaims struct {
    UserID      string   `json:"user_id"`
    Username    string   `json:"username"`
    Email       string   `json:"email"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    SessionID   string   `json:"session_id"`
    jwt.RegisteredClaims
}

// SetPassword 设置密码
func (u *User) SetPassword(password string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.PasswordHash = string(hash)
    return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
    return err == nil
}

// HasPermission 检查权限
func (u *User) HasPermission(resource, action, scope string) bool {
    for _, role := range u.Roles {
        for _, perm := range role.Permissions {
            if perm.Resource == resource && 
               perm.Action == action && 
               (perm.Scope == scope || perm.Scope == "all") {
                return true
            }
        }
    }
    return false
}
```

### 2.2 认证服务实现

```go
// auth/service.go
package auth

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

type AuthService struct {
    db          *gorm.DB
    redis       *redis.Client
    jwtSecret   []byte
    tokenExpiry time.Duration
}

type LoginRequest struct {
    Username string `json:"username" validate:"required"`
    Password string `json:"password" validate:"required"`
    Remember bool   `json:"remember"`
}

type LoginResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    User         User      `json:"user"`
}

func NewAuthService(db *gorm.DB, redis *redis.Client, jwtSecret string) *AuthService {
    return &AuthService{
        db:          db,
        redis:       redis,
        jwtSecret:   []byte(jwtSecret),
        tokenExpiry: time.Hour * 24, // 24小时
    }
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req LoginRequest, 
                           ipAddress, userAgent string) (*LoginResponse, error) {
    // 查找用户
    var user User
    err := s.db.WithContext(ctx).
        Preload("Roles.Permissions").
        Where("username = ? OR email = ?", req.Username, req.Username).
        First(&user).Error
    if err != nil {
        return nil, fmt.Errorf("用户不存在或密码错误")
    }
    
    // 检查用户状态
    if user.Status != UserStatusActive {
        return nil, fmt.Errorf("用户账户已被禁用")
    }
    
    // 验证密码
    if !user.CheckPassword(req.Password) {
        // 记录登录失败
        s.recordLoginAttempt(ctx, user.ID, ipAddress, false)
        return nil, fmt.Errorf("用户不存在或密码错误")
    }
    
    // 检查登录失败次数
    if s.isAccountLocked(ctx, user.ID) {
        return nil, fmt.Errorf("账户已被锁定，请稍后再试")
    }
    
    // 生成会话
    session, err := s.createSession(ctx, &user, ipAddress, userAgent, req.Remember)
    if err != nil {
        return nil, fmt.Errorf("创建会话失败: %v", err)
    }
    
    // 生成JWT令牌
    accessToken, err := s.generateAccessToken(&user, session.ID)
    if err != nil {
        return nil, fmt.Errorf("生成访问令牌失败: %v", err)
    }
    
    // 更新最后登录时间
    now := time.Now()
    user.LastLoginAt = &now
    s.db.WithContext(ctx).Save(&user)
    
    // 记录登录成功
    s.recordLoginAttempt(ctx, user.ID, ipAddress, true)
    
    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: session.RefreshToken,
        ExpiresAt:    session.ExpiresAt,
        User:         user,
    }, nil
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
    // 查找会话
    var session Session
    err := s.db.WithContext(ctx).
        Preload("User.Roles.Permissions").
        Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now()).
        First(&session).Error
    if err != nil {
        return nil, fmt.Errorf("无效的刷新令牌")
    }
    
    // 生成新的访问令牌
    accessToken, err := s.generateAccessToken(&session.User, session.ID)
    if err != nil {
        return nil, fmt.Errorf("生成访问令牌失败: %v", err)
    }
    
    // 更新会话最后使用时间
    session.LastUsedAt = time.Now()
    s.db.WithContext(ctx).Save(&session)
    
    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: session.RefreshToken,
        ExpiresAt:    session.ExpiresAt,
        User:         session.User,
    }, nil
}

// Logout 用户登出
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
    // 删除会话
    err := s.db.WithContext(ctx).Where("id = ?", sessionID).Delete(&Session{}).Error
    if err != nil {
        return fmt.Errorf("登出失败: %v", err)
    }
    
    // 将令牌加入黑名单
    s.redis.Set(ctx, fmt.Sprintf("blacklist:%s", sessionID), "1", s.tokenExpiry)
    
    return nil
}

// ValidateToken 验证令牌
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*JWTClaims, error) {
    // 解析JWT令牌
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.jwtSecret, nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("无效的令牌: %v", err)
    }
    
    claims, ok := token.Claims.(*JWTClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("无效的令牌声明")
    }
    
    // 检查令牌是否在黑名单中
    result := s.redis.Get(ctx, fmt.Sprintf("blacklist:%s", claims.SessionID))
    if result.Err() == nil {
        return nil, fmt.Errorf("令牌已失效")
    }
    
    // 检查会话是否存在
    var session Session
    err = s.db.WithContext(ctx).Where("id = ?", claims.SessionID).First(&session).Error
    if err != nil {
        return nil, fmt.Errorf("会话不存在")
    }
    
    return claims, nil
}

// createSession 创建会话
func (s *AuthService) createSession(ctx context.Context, user *User, 
                                   ipAddress, userAgent string, remember bool) (*Session, error) {
    // 生成会话ID和刷新令牌
    sessionID, err := s.generateRandomString(32)
    if err != nil {
        return nil, err
    }
    
    refreshToken, err := s.generateRandomString(64)
    if err != nil {
        return nil, err
    }
    
    // 设置过期时间
    expiresAt := time.Now().Add(s.tokenExpiry)
    if remember {
        expiresAt = time.Now().Add(time.Hour * 24 * 30) // 30天
    }
    
    session := &Session{
        ID:           sessionID,
        UserID:       user.ID,
        Token:        sessionID,
        RefreshToken: refreshToken,
        IPAddress:    ipAddress,
        UserAgent:    userAgent,
        ExpiresAt:    expiresAt,
        CreatedAt:    time.Now(),
        LastUsedAt:   time.Now(),
    }
    
    err = s.db.WithContext(ctx).Create(session).Error
    if err != nil {
        return nil, err
    }
    
    return session, nil
}

// generateAccessToken 生成访问令牌
func (s *AuthService) generateAccessToken(user *User, sessionID string) (string, error) {
    // 提取角色和权限
    roles := make([]string, len(user.Roles))
    permissions := make([]string, 0)
    
    for i, role := range user.Roles {
        roles[i] = role.Name
        for _, perm := range role.Permissions {
            permissions = append(permissions, fmt.Sprintf("%s:%s:%s", 
                perm.Resource, perm.Action, perm.Scope))
        }
    }
    
    // 创建JWT声明
    claims := &JWTClaims{
        UserID:      user.ID,
        Username:    user.Username,
        Email:       user.Email,
        Roles:       roles,
        Permissions: permissions,
        SessionID:   sessionID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2)), // 2小时
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "itsm-system",
            Subject:   user.ID,
        },
    }
    
    // 生成令牌
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

// generateRandomString 生成随机字符串
func (s *AuthService) generateRandomString(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// recordLoginAttempt 记录登录尝试
func (s *AuthService) recordLoginAttempt(ctx context.Context, userID, ipAddress string, success bool) {
    key := fmt.Sprintf("login_attempts:%s", userID)
    if success {
        s.redis.Del(ctx, key)
    } else {
        s.redis.Incr(ctx, key)
        s.redis.Expire(ctx, key, time.Hour) // 1小时后重置
    }
}

// isAccountLocked 检查账户是否被锁定
func (s *AuthService) isAccountLocked(ctx context.Context, userID string) bool {
    key := fmt.Sprintf("login_attempts:%s", userID)
    attempts, err := s.redis.Get(ctx, key).Int()
    if err != nil {
        return false
    }
    return attempts >= 5 // 5次失败后锁定
}
```

### 2.3 权限控制中间件

```go
// middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
    "your-project/auth"
)

type AuthMiddleware struct {
    authService *auth.AuthService
}

func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
    return &AuthMiddleware{
        authService: authService,
    }
}

// RequireAuth 要求认证
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取Authorization头
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
            c.Abort()
            return
        }
        
        // 提取Bearer令牌
        tokenParts := strings.Split(authHeader, " ")
        if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证格式"})
            c.Abort()
            return
        }
        
        // 验证令牌
        claims, err := m.authService.ValidateToken(c.Request.Context(), tokenParts[1])
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            c.Abort()
            return
        }
        
        // 将用户信息存储到上下文
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("roles", claims.Roles)
        c.Set("permissions", claims.Permissions)
        c.Set("session_id", claims.SessionID)
        
        c.Next()
    }
}

// RequirePermission 要求特定权限
func (m *AuthMiddleware) RequirePermission(resource, action, scope string) gin.HandlerFunc {
    return func(c *gin.Context) {
        permissions, exists := c.Get("permissions")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{"error": "权限信息不存在"})
            c.Abort()
            return
        }
        
        permList, ok := permissions.([]string)
        if !ok {
            c.JSON(http.StatusForbidden, gin.H{"error": "权限格式错误"})
            c.Abort()
            return
        }
        
        // 检查权限
        requiredPerm := fmt.Sprintf("%s:%s:%s", resource, action, scope)
        allPerm := fmt.Sprintf("%s:%s:all", resource, action)
        
        hasPermission := false
        for _, perm := range permList {
            if perm == requiredPerm || perm == allPerm {
                hasPermission = true
                break
            }
        }
        
        if !hasPermission {
            c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// RequireRole 要求特定角色
func (m *AuthMiddleware) RequireRole(requiredRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        roles, exists := c.Get("roles")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{"error": "角色信息不存在"})
            c.Abort()
            return
        }
        
        userRoles, ok := roles.([]string)
        if !ok {
            c.JSON(http.StatusForbidden, gin.H{"error": "角色格式错误"})
            c.Abort()
            return
        }
        
        // 检查角色
        hasRole := false
        for _, userRole := range userRoles {
            for _, requiredRole := range requiredRoles {
                if userRole == requiredRole {
                    hasRole = true
                    break
                }
            }
            if hasRole {
                break
            }
        }
        
        if !hasRole {
            c.JSON(http.StatusForbidden, gin.H{"error": "角色权限不足"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

## 3. 数据加密系统

### 3.1 加密服务设计

```go
// security/encryption.go
package security

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "io"
    
    "golang.org/x/crypto/pbkdf2"
)

type EncryptionService struct {
    masterKey []byte
}

type EncryptedData struct {
    Data      string `json:"data"`
    Salt      string `json:"salt"`
    Algorithm string `json:"algorithm"`
}

func NewEncryptionService(masterKey string) *EncryptionService {
    return &EncryptionService{
        masterKey: []byte(masterKey),
    }
}

// EncryptSensitiveData 加密敏感数据
func (s *EncryptionService) EncryptSensitiveData(plaintext string) (*EncryptedData, error) {
    // 生成随机盐
    salt := make([]byte, 16)
    if _, err := io.ReadFull(rand.Reader, salt); err != nil {
        return nil, fmt.Errorf("生成盐失败: %v", err)
    }
    
    // 使用PBKDF2派生密钥
    key := pbkdf2.Key(s.masterKey, salt, 10000, 32, sha256.New)
    
    // 创建AES加密器
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("创建加密器失败: %v", err)
    }
    
    // 使用GCM模式
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("创建GCM失败: %v", err)
    }
    
    // 生成随机nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("生成nonce失败: %v", err)
    }
    
    // 加密数据
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    
    return &EncryptedData{
        Data:      base64.StdEncoding.EncodeToString(ciphertext),
        Salt:      base64.StdEncoding.EncodeToString(salt),
        Algorithm: "AES-256-GCM",
    }, nil
}

// DecryptSensitiveData 解密敏感数据
func (s *EncryptionService) DecryptSensitiveData(encData *EncryptedData) (string, error) {
    // 解码数据
    ciphertext, err := base64.StdEncoding.DecodeString(encData.Data)
    if err != nil {
        return "", fmt.Errorf("解码密文失败: %v", err)
    }
    
    salt, err := base64.StdEncoding.DecodeString(encData.Salt)
    if err != nil {
        return "", fmt.Errorf("解码盐失败: %v", err)
    }
    
    // 派生密钥
    key := pbkdf2.Key(s.masterKey, salt, 10000, 32, sha256.New)
    
    // 创建AES解密器
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("创建解密器失败: %v", err)
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("创建GCM失败: %v", err)
    }
    
    // 提取nonce
    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", fmt.Errorf("密文长度不足")
    }
    
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    
    // 解密数据
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("解密失败: %v", err)
    }
    
    return string(plaintext), nil
}

// HashPassword 哈希密码
func (s *EncryptionService) HashPassword(password string) (string, error) {
    salt := make([]byte, 16)
    if _, err := io.ReadFull(rand.Reader, salt); err != nil {
        return "", err
    }
    
    hash := pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)
    
    // 组合盐和哈希
    combined := append(salt, hash...)
    return base64.StdEncoding.EncodeToString(combined), nil
}

// VerifyPassword 验证密码
func (s *EncryptionService) VerifyPassword(password, hashedPassword string) bool {
    combined, err := base64.StdEncoding.DecodeString(hashedPassword)
    if err != nil || len(combined) != 48 { // 16字节盐 + 32字节哈希
        return false
    }
    
    salt := combined[:16]
    hash := combined[16:]
    
    expectedHash := pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)
    
    // 常数时间比较
    return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}
```

### 3.2 数据库字段加密

```go
// models/encrypted_field.go
package models

import (
    "database/sql/driver"
    "fmt"
    "your-project/security"
)

// EncryptedString 加密字符串类型
type EncryptedString struct {
    Value     string
    encrypted *security.EncryptedData
    service   *security.EncryptionService
}

func NewEncryptedString(value string, service *security.EncryptionService) *EncryptedString {
    return &EncryptedString{
        Value:   value,
        service: service,
    }
}

// Scan 实现sql.Scanner接口
func (es *EncryptedString) Scan(value interface{}) error {
    if value == nil {
        es.Value = ""
        return nil
    }
    
    switch v := value.(type) {
    case string:
        // 假设数据库中存储的是JSON格式的加密数据
        var encData security.EncryptedData
        if err := json.Unmarshal([]byte(v), &encData); err != nil {
            return err
        }
        
        decrypted, err := es.service.DecryptSensitiveData(&encData)
        if err != nil {
            return err
        }
        
        es.Value = decrypted
        es.encrypted = &encData
        return nil
    case []byte:
        return es.Scan(string(v))
    default:
        return fmt.Errorf("无法扫描 %T 到 EncryptedString", value)
    }
}

// Value 实现driver.Valuer接口
func (es EncryptedString) Value() (driver.Value, error) {
    if es.Value == "" {
        return nil, nil
    }
    
    // 如果已经加密过，直接返回
    if es.encrypted != nil {
        data, err := json.Marshal(es.encrypted)
        if err != nil {
            return nil, err
        }
        return string(data), nil
    }
    
    // 加密数据
    encrypted, err := es.service.EncryptSensitiveData(es.Value)
    if err != nil {
        return nil, err
    }
    
    data, err := json.Marshal(encrypted)
    if err != nil {
        return nil, err
    }
    
    return string(data), nil
}

// 使用示例
type User struct {
    ID       string          `json:"id" gorm:"primaryKey"`
    Username string          `json:"username"`
    Email    EncryptedString `json:"email" gorm:"type:text"`    // 加密邮箱
    Phone    EncryptedString `json:"phone" gorm:"type:text"`    // 加密电话
    IDCard   EncryptedString `json:"id_card" gorm:"type:text"`  // 加密身份证
}
```

## 4. 审计日志系统

### 4.1 审计日志模型

```go
// audit/models.go
package audit

import (
    "encoding/json"
    "time"
)

// AuditLog 审计日志模型
type AuditLog struct {
    ID          string          `json:"id" gorm:"primaryKey"`
    UserID      string          `json:"user_id" gorm:"index"`
    Username    string          `json:"username"`
    Action      string          `json:"action"`      // 操作类型
    Resource    string          `json:"resource"`    // 资源类型
    ResourceID  string          `json:"resource_id"` // 资源ID
    Method      string          `json:"method"`      // HTTP方法
    Path        string          `json:"path"`        // 请求路径
    IPAddress   string          `json:"ip_address"`
    UserAgent   string          `json:"user_agent"`
    Status      AuditStatus     `json:"status"`
    Message     string          `json:"message"`
    Details     json.RawMessage `json:"details"`     // 详细信息
    Duration    int64           `json:"duration"`    // 执行时间(毫秒)
    CreatedAt   time.Time       `json:"created_at" gorm:"index"`
    
    // 安全相关字段
    RiskLevel   RiskLevel       `json:"risk_level"`
    Tags        []string        `json:"tags" gorm:"type:json"`
    SessionID   string          `json:"session_id"`
}

type AuditStatus string

const (
    AuditStatusSuccess AuditStatus = "success"
    AuditStatusFailed  AuditStatus = "failed"
    AuditStatusError   AuditStatus = "error"
)

type RiskLevel string

const (
    RiskLevelLow      RiskLevel = "low"
    RiskLevelMedium   RiskLevel = "medium"
    RiskLevelHigh     RiskLevel = "high"
    RiskLevelCritical RiskLevel = "critical"
)

// SecurityEvent 安全事件模型
type SecurityEvent struct {
    ID          string          `json:"id" gorm:"primaryKey"`
    Type        EventType       `json:"type"`
    Severity    Severity        `json:"severity"`
    Title       string          `json:"title"`
    Description string          `json:"description"`
    UserID      string          `json:"user_id"`
    IPAddress   string          `json:"ip_address"`
    Details     json.RawMessage `json:"details"`
    Status      EventStatus     `json:"status"`
    CreatedAt   time.Time       `json:"created_at" gorm:"index"`
    UpdatedAt   time.Time       `json:"updated_at"`
    
    // 关联的审计日志
    AuditLogs   []AuditLog      `json:"audit_logs" gorm:"foreignKey:ResourceID"`
}

type EventType string

const (
    EventTypeLoginFailure     EventType = "login_failure"
    EventTypeUnauthorizedAccess EventType = "unauthorized_access"
    EventTypeDataBreach       EventType = "data_breach"
    EventTypePrivilegeEscalation EventType = "privilege_escalation"
    EventTypeSuspiciousActivity EventType = "suspicious_activity"
)

type Severity string

const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityError    Severity = "error"
    SeverityCritical Severity = "critical"
)

type EventStatus string

const (
    EventStatusOpen       EventStatus = "open"
    EventStatusInvestigating EventStatus = "investigating"
    EventStatusResolved   EventStatus = "resolved"
    EventStatusFalsePositive EventStatus = "false_positive"
)
```

### 4.2 审计服务实现

```go
// audit/service.go
package audit

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "gorm.io/gorm"
    "github.com/gin-gonic/gin"
)

type AuditService struct {
    db *gorm.DB
}

type AuditContext struct {
    UserID    string
    Username  string
    SessionID string
    IPAddress string
    UserAgent string
}

func NewAuditService(db *gorm.DB) *AuditService {
    return &AuditService{db: db}
}

// LogAction 记录操作日志
func (s *AuditService) LogAction(ctx context.Context, auditCtx *AuditContext,
                                action, resource, resourceID string,
                                status AuditStatus, details interface{},
                                duration time.Duration) error {
    
    detailsJSON, _ := json.Marshal(details)
    
    auditLog := &AuditLog{
        ID:         generateID(),
        UserID:     auditCtx.UserID,
        Username:   auditCtx.Username,
        Action:     action,
        Resource:   resource,
        ResourceID: resourceID,
        IPAddress:  auditCtx.IPAddress,
        UserAgent:  auditCtx.UserAgent,
        Status:     status,
        Details:    detailsJSON,
        Duration:   duration.Milliseconds(),
        RiskLevel:  s.calculateRiskLevel(action, resource, status),
        Tags:       s.generateTags(action, resource, status),
        SessionID:  auditCtx.SessionID,
        CreatedAt:  time.Now(),
    }
    
    return s.db.WithContext(ctx).Create(auditLog).Error
}

// LogSecurityEvent 记录安全事件
func (s *AuditService) LogSecurityEvent(ctx context.Context, eventType EventType,
                                       severity Severity, title, description string,
                                       userID, ipAddress string, details interface{}) error {
    
    detailsJSON, _ := json.Marshal(details)
    
    event := &SecurityEvent{
        ID:          generateID(),
        Type:        eventType,
        Severity:    severity,
        Title:       title,
        Description: description,
        UserID:      userID,
        IPAddress:   ipAddress,
        Details:     detailsJSON,
        Status:      EventStatusOpen,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    return s.db.WithContext(ctx).Create(event).Error
}

// GetAuditLogs 获取审计日志
func (s *AuditService) GetAuditLogs(ctx context.Context, filter AuditFilter) ([]AuditLog, int64, error) {
    query := s.db.WithContext(ctx).Model(&AuditLog{})
    
    // 应用过滤条件
    if filter.UserID != "" {
        query = query.Where("user_id = ?", filter.UserID)
    }
    if filter.Action != "" {
        query = query.Where("action = ?", filter.Action)
    }
    if filter.Resource != "" {
        query = query.Where("resource = ?", filter.Resource)
    }
    if filter.Status != "" {
        query = query.Where("status = ?", filter.Status)
    }
    if filter.RiskLevel != "" {
        query = query.Where("risk_level = ?", filter.RiskLevel)
    }
    if !filter.StartTime.IsZero() {
        query = query.Where("created_at >= ?", filter.StartTime)
    }
    if !filter.EndTime.IsZero() {
        query = query.Where("created_at <= ?", filter.EndTime)
    }
    
    // 获取总数
    var total int64
    query.Count(&total)
    
    // 获取数据
    var logs []AuditLog
    err := query.Order("created_at DESC").
        Limit(filter.Limit).
        Offset(filter.Offset).
        Find(&logs).Error
    
    return logs, total, err
}

// DetectAnomalies 检测异常行为
func (s *AuditService) DetectAnomalies(ctx context.Context) error {
    // 检测登录失败异常
    if err := s.detectLoginFailures(ctx); err != nil {
        return err
    }
    
    // 检测权限提升异常
    if err := s.detectPrivilegeEscalation(ctx); err != nil {
        return err
    }
    
    // 检测异常访问模式
    if err := s.detectSuspiciousAccess(ctx); err != nil {
        return err
    }
    
    return nil
}

// detectLoginFailures 检测登录失败异常
func (s *AuditService) detectLoginFailures(ctx context.Context) error {
    // 查找最近1小时内登录失败超过5次的IP
    var results []struct {
        IPAddress string
        Count     int64
    }
    
    err := s.db.WithContext(ctx).
        Model(&AuditLog{}).
        Select("ip_address, COUNT(*) as count").
        Where("action = ? AND status = ? AND created_at > ?",
            "login", AuditStatusFailed, time.Now().Add(-time.Hour)).
        Group("ip_address").
        Having("COUNT(*) > ?", 5).
        Find(&results).Error
    
    if err != nil {
        return err
    }
    
    // 为每个异常IP创建安全事件
    for _, result := range results {
        s.LogSecurityEvent(ctx, EventTypeLoginFailure, SeverityWarning,
            "多次登录失败",
            fmt.Sprintf("IP %s 在1小时内登录失败 %d 次", result.IPAddress, result.Count),
            "", result.IPAddress, map[string]interface{}{
                "failure_count": result.Count,
                "time_window": "1hour",
            })
    }
    
    return nil
}

// calculateRiskLevel 计算风险等级
func (s *AuditService) calculateRiskLevel(action, resource string, status AuditStatus) RiskLevel {
    // 基于操作类型和资源类型计算风险等级
    riskScore := 0
    
    // 操作风险评分
    switch action {
    case "delete":
        riskScore += 3
    case "update", "create":
        riskScore += 2
    case "read":
        riskScore += 1
    }
    
    // 资源风险评分
    switch resource {
    case "user", "role", "permission":
        riskScore += 3
    case "ticket", "incident":
        riskScore += 2
    default:
        riskScore += 1
    }
    
    // 状态风险评分
    if status == AuditStatusFailed {
        riskScore += 2
    }
    
    // 确定风险等级
    switch {
    case riskScore >= 7:
        return RiskLevelCritical
    case riskScore >= 5:
        return RiskLevelHigh
    case riskScore >= 3:
        return RiskLevelMedium
    default:
        return RiskLevelLow
    }
}

// generateTags 生成标签
func (s *AuditService) generateTags(action, resource string, status AuditStatus) []string {
    tags := []string{action, resource}
    
    if status == AuditStatusFailed {
        tags = append(tags, "failed")
    }
    
    return tags
}

type AuditFilter struct {
    UserID    string
    Action    string
    Resource  string
    Status    AuditStatus
    RiskLevel RiskLevel
    StartTime time.Time
    EndTime   time.Time
    Limit     int
    Offset    int
}
```

### 4.3 审计中间件

```go
// middleware/audit.go
package middleware

import (
    "bytes"
    "io"
    "time"
    
    "github.com/gin-gonic/gin"
    "your-project/audit"
)

type AuditMiddleware struct {
    auditService *audit.AuditService
}

func NewAuditMiddleware(auditService *audit.AuditService) *AuditMiddleware {
    return &AuditMiddleware{
        auditService: auditService,
    }
}

// AuditLog 审计日志中间件
func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // 读取请求体
        var requestBody []byte
        if c.Request.Body != nil {
            requestBody, _ = io.ReadAll(c.Request.Body)
            c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
        }
        
        // 创建响应写入器
        writer := &responseWriter{
            ResponseWriter: c.Writer,
            body:          &bytes.Buffer{},
        }
        c.Writer = writer
        
        // 处理请求
        c.Next()
        
        // 计算执行时间
        duration := time.Since(start)
        
        // 获取用户信息
        userID, _ := c.Get("user_id")
        username, _ := c.Get("username")
        sessionID, _ := c.Get("session_id")
        
        // 构建审计上下文
        auditCtx := &audit.AuditContext{
            UserID:    getString(userID),
            Username:  getString(username),
            SessionID: getString(sessionID),
            IPAddress: c.ClientIP(),
            UserAgent: c.Request.UserAgent(),
        }
        
        // 确定操作状态
        status := audit.AuditStatusSuccess
        if c.Writer.Status() >= 400 {
            status = audit.AuditStatusFailed
        }
        
        // 构建详细信息
        details := map[string]interface{}{
            "method":       c.Request.Method,
            "path":         c.Request.URL.Path,
            "query":        c.Request.URL.RawQuery,
            "status_code":  c.Writer.Status(),
            "request_size": len(requestBody),
            "response_size": writer.body.Len(),
        }
        
        // 如果是敏感操作，记录更多详情
        if m.isSensitiveOperation(c.Request.Method, c.Request.URL.Path) {
            details["request_body"] = string(requestBody)
            details["response_body"] = writer.body.String()
        }
        
        // 记录审计日志
        action := m.extractAction(c.Request.Method, c.Request.URL.Path)
        resource := m.extractResource(c.Request.URL.Path)
        resourceID := m.extractResourceID(c.Request.URL.Path)
        
        m.auditService.LogAction(
            c.Request.Context(),
            auditCtx,
            action,
            resource,
            resourceID,
            status,
            details,
            duration,
        )
    }
}

// responseWriter 响应写入器
type responseWriter struct {
    gin.ResponseWriter
    body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
    w.body.Write(b)
    return w.ResponseWriter.Write(b)
}

// 辅助函数
func getString(v interface{}) string {
    if s, ok := v.(string); ok {
        return s
    }
    return ""
}

func (m *AuditMiddleware) isSensitiveOperation(method, path string) bool {
    // 定义敏感操作
    sensitivePatterns := []string{
        "/api/users",
        "/api/roles",
        "/api/permissions",
        "/api/auth",
    }
    
    for _, pattern := range sensitivePatterns {
        if strings.Contains(path, pattern) {
            return true
        }
    }
    
    return method == "DELETE" || method == "POST" || method == "PUT"
}

func (m *AuditMiddleware) extractAction(method, path string) string {
    switch method {
    case "GET":
        return "read"
    case "POST":
        return "create"
    case "PUT", "PATCH":
        return "update"
    case "DELETE":
        return "delete"
    default:
        return method
    }
}

func (m *AuditMiddleware) extractResource(path string) string {
    // 从路径中提取资源类型
    parts := strings.Split(strings.Trim(path, "/"), "/")
    if len(parts) >= 2 {
        return parts[1] // 假设格式为 /api/resource/id
    }
    return "unknown"
}

func (m *AuditMiddleware) extractResourceID(path string) string {
    // 从路径中提取资源ID
    parts := strings.Split(strings.Trim(path, "/"), "/")
    if len(parts) >= 3 {
        return parts[2]
    }
    return ""
}
```

## 5. 安全配置和部署

### 5.1 安全配置

```yaml
# config/security.yaml
security:
  # JWT配置
  jwt:
    secret: ${JWT_SECRET}
    expiry: 2h
    refresh_expiry: 720h # 30天
    
  # 加密配置
  encryption:
    master_key: ${ENCRYPTION_MASTER_KEY}
    algorithm: AES-256-GCM
    
  # 密码策略
  password:
    min_length: 8
    require_uppercase: true
    require_lowercase: true
    require_numbers: true
    require_symbols: true
    max_age_days: 90
    history_count: 5
    
  # 会话配置
  session:
    timeout: 30m
    max_concurrent: 3
    
  # 登录安全
  login:
    max_attempts: 5
    lockout_duration: 1h
    
  # CORS配置
  cors:
    allowed_origins:
      - "https://itsm.company.com"
      - "https://admin.itsm.company.com"
    allowed_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
    allowed_headers:
      - Authorization
      - Content-Type
      - X-Requested-With
    max_age: 86400
    
  # 安全头配置
  headers:
    x_frame_options: DENY
    x_content_type_options: nosniff
    x_xss_protection: "1; mode=block"
    strict_transport_security: "max-age=31536000; includeSubDomains"
    content_security_policy: "default-src 'self'; script-src 'self' 'unsafe-inline'"
    
  # 审计配置
  audit:
    enabled: true
    retention_days: 365
    sensitive_fields:
      - password
      - token
      - secret
      - key
```

### 5.2 Kubernetes 安全配置

```yaml
# k8s/security-policies.yaml
apiVersion: v1
kind: Secret
metadata:
  name: itsm-secrets
type: Opaque
data:
  jwt-secret: <base64-encoded-jwt-secret>
  encryption-key: <base64-encoded-encryption-key>
  db-password: <base64-encoded-db-password>

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: itsm-network-policy
spec:
  podSelector:
    matchLabels:
      app: itsm
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: nginx-ingress
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379

---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: itsm-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
```

这个安全框架设计提供了完整的认证授权、数据加密、审计日志等安全功能，通过多层防护和零信任架构，确保ITSM系统的安全性和合规性。
package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// EncryptionService 数据加密服务
type EncryptionService struct {
	key []byte
}

// NewEncryptionService 创建加密服务实例
func NewEncryptionService(secretKey string) *EncryptionService {
	// 使用SHA256生成32字节的密钥
	hash := sha256.Sum256([]byte(secretKey))
	return &EncryptionService{
		key: hash[:],
	}
}

// Encrypt 加密数据
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func (e *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// base64解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文长度不足")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// HashPassword 使用bcrypt哈希密码
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("密码不能为空")
	}

	// 使用bcrypt默认cost(10)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("密码哈希失败: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyPassword 验证密码
func VerifyPassword(hashedPassword, password string) bool {
	if hashedPassword == "" || password == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// EncryptSensitiveFields 加密敏感字段
func (e *EncryptionService) EncryptSensitiveFields(data map[string]interface{}) error {
	sensitiveFields := []string{"phone", "email", "id_card", "address", "bank_account"}
	
	for _, field := range sensitiveFields {
		if value, exists := data[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				encrypted, err := e.Encrypt(strValue)
				if err != nil {
					return fmt.Errorf("加密字段 %s 失败: %w", field, err)
				}
				data[field] = encrypted
			}
		}
	}
	
	return nil
}

// DecryptSensitiveFields 解密敏感字段
func (e *EncryptionService) DecryptSensitiveFields(data map[string]interface{}) error {
	sensitiveFields := []string{"phone", "email", "id_card", "address", "bank_account"}
	
	for _, field := range sensitiveFields {
		if value, exists := data[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				decrypted, err := e.Decrypt(strValue)
				if err != nil {
					return fmt.Errorf("解密字段 %s 失败: %w", field, err)
				}
				data[field] = decrypted
			}
		}
	}
	
	return nil
}

// GenerateSecureToken 生成安全令牌
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成随机令牌失败: %w", err)
	}
	
	return base64.URLEncoding.EncodeToString(bytes), nil
}
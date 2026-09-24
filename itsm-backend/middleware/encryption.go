package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/hkdf"
)

// currentKeyVersion 标识当前密文格式版本。
// 版本字节写入密文前缀，未来轮换密钥时递增版本号即可。
const currentKeyVersion byte = 1

// hkdfInfo 是 HKDF 的 info 参数，绑定用途防止跨用途密钥复用。
var hkdfInfo = []byte("itsm-encryption-v1")

// EncryptionService 数据加密服务
type EncryptionService struct {
	secretKey []byte
	// legacyKey 是旧版 SHA256(secretKey) 派生密钥，仅用于解密旧密文。
	legacyKey []byte
}

// NewEncryptionService 创建加密服务实例
func NewEncryptionService(secretKey string) *EncryptionService {
	hash := sha256.Sum256([]byte(secretKey))
	return &EncryptionService{
		secretKey: []byte(secretKey),
		legacyKey: hash[:],
	}
}

// deriveKey 使用 HKDF 从主密钥+盐派生 32 字节 AES 密钥。
func (e *EncryptionService) deriveKey(salt []byte) ([]byte, error) {
	key := make([]byte, 32)
	kdf := hkdf.New(sha256.New, e.secretKey, salt, hkdfInfo)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, fmt.Errorf("HKDF 密钥派生失败: %w", err)
	}
	return key, nil
}

// Encrypt 加密数据。
// 新密文格式: version(1) || salt_len(2) || salt(salt_len) || nonce(12) || ciphertext
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 生成随机盐（16 字节）
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("生成盐失败: %w", err)
	}

	derivedKey, err := e.deriveKey(salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

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
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// 组装: version(1) || salt_len(2) || salt || nonce || ciphertext
	saltLen := make([]byte, 2)
	binary.BigEndian.PutUint16(saltLen, uint16(len(salt)))

	output := make([]byte, 0, 1+2+len(salt)+len(nonce)+len(ciphertext))
	output = append(output, currentKeyVersion)
	output = append(output, saltLen...)
	output = append(output, salt...)
	output = append(output, nonce...)
	output = append(output, ciphertext...)

	return base64.StdEncoding.EncodeToString(output), nil
}

// Decrypt 解密数据。
// 自动识别新格式（version 前缀）和旧格式（无版本字节，SHA256 派生密钥）。
func (e *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// base64解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %w", err)
	}

	if len(data) > 0 && data[0] == currentKeyVersion {
		return e.decryptV1(data)
	}

	// 旧版 fallback: nonce || ciphertext，使用 legacyKey
	return e.decryptLegacy(data)
}

// decryptV1 解密新版密文: version(1) || salt_len(2) || salt || nonce(12) || ciphertext
func (e *EncryptionService) decryptV1(data []byte) (string, error) {
	if len(data) < 3 {
		return "", errors.New("密文长度不足: 缺少版本和盐长度")
	}

	saltLen := binary.BigEndian.Uint16(data[1:3])
	offset := 3 + int(saltLen)
	if len(data) < offset {
		return "", errors.New("密文长度不足: 盐数据不完整")
	}

	salt := data[3:offset]
	derivedKey, err := e.deriveKey(salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data[offset:]) < nonceSize {
		return "", errors.New("密文长度不足: nonce 不完整")
	}

	nonce := data[offset : offset+nonceSize]
	cipherData := data[offset+nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// decryptLegacy 解密旧版密文: nonce || ciphertext，使用 SHA256(secretKey) 派生密钥。
// 仅用于迁移期解密已有数据，不应在新加密路径中使用。
func (e *EncryptionService) decryptLegacy(data []byte) (string, error) {
	block, err := aes.NewCipher(e.legacyKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败(legacy): %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败(legacy): %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文长度不足(legacy)")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败(legacy): %w", err)
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
	sensitiveFields := []string{
		// PII
		"phone", "email", "id_card", "address", "bank_account",
		// Connector secrets
		"password", "secret", "api_key", "app_key", "app_secret", "client_secret",
		"access_token", "refresh_token", "bot_token", "encrypt_key",
		"signing_secret", "corp_secret", "agent_secret",
	}

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
	sensitiveFields := []string{
		// PII
		"phone", "email", "id_card", "address", "bank_account",
		// Connector secrets
		"password", "secret", "api_key", "app_key", "app_secret", "client_secret",
		"access_token", "refresh_token", "bot_token", "encrypt_key",
		"signing_secret", "corp_secret", "agent_secret",
	}

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

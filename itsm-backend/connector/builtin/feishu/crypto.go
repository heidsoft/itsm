package feishu

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// 飞书 Encrypt Key 加密策略（官方算法，与 larksuite/oapi-sdk-go event.EventDecrypt
// 及开放平台「接收回调」文档一致）：
//
//   - 密钥 = SHA256(encrypt_key) 的原始 32 字节
//   - 密文 = base64 解码后，前 16 字节为 IV，其余为 AES-256-CBC 密文
//   - 明文 = 16 字节随机 + 4 字节大端消息长度 + 消息体 + receive_id，PKCS#7 填充
//
// challenge 回包按官方 SDK 语义保持明文（{"challenge":"..."}），不做加密响应。

// decryptInboundPayload 解密 body 的 encrypt 信封，返回明文消息体。
// body 不含 encrypt 字段时原样返回；含 encrypt 字段但未配置 encrypt_key 时
// 失败关闭（加密回调不可静默降级成明文解析）。
func decryptInboundPayload(encryptKey string, body []byte) ([]byte, error) {
	var envelope struct {
		Encrypt string `json:"encrypt"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("feishu: parse inbound envelope: %w", err)
	}
	if envelope.Encrypt == "" {
		return body, nil
	}
	if encryptKey == "" {
		return nil, fmt.Errorf("feishu: encrypted callback requires encrypt_key to be configured")
	}
	return decryptMessage(encryptKey, envelope.Encrypt)
}

func decryptMessage(encryptKey, cipherB64 string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return nil, fmt.Errorf("feishu: invalid base64 ciphertext: %w", err)
	}
	if len(raw) < 2*aes.BlockSize || len(raw)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("feishu: ciphertext length %d is not a valid block chain", len(raw))
	}
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("feishu: new cipher: %w", err)
	}
	iv, ciphertext := raw[:aes.BlockSize], raw[aes.BlockSize:]
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)

	// 先按 PKCS#7 去填充再结构化解析；校验不过时退回官方 SDK 的 JSON 扫描法
	// （首个 '{' 到末个 '}'），明文的两种形态（纯 JSON / 带头结构）都能取消息。
	trimmed, err := pkcs7Unpad(plaintext)
	if err != nil {
		trimmed = plaintext
	}
	if msg, _, ok := parsePlaintextEnvelope(trimmed); ok {
		return msg, nil
	}
	return extractJSON(trimmed), nil
}

// parsePlaintextEnvelope 按「16 字节随机 + 4 字节大端长度 + 消息 + receive_id」解析。
func parsePlaintextEnvelope(plaintext []byte) (msg []byte, receiveID string, ok bool) {
	if len(plaintext) < 20 {
		return nil, "", false
	}
	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	end := 20 + int(msgLen)
	if msgLen == 0 || end > len(plaintext) {
		return nil, "", false
	}
	msg = plaintext[20:end]
	if msg[0] != '{' && msg[0] != '[' {
		return nil, "", false
	}
	return msg, string(plaintext[end:]), true
}

// extractJSON 官方 SDK 的兜底取消息法。
func extractJSON(plaintext []byte) []byte {
	start := bytes.IndexByte(plaintext, '{')
	end := bytes.LastIndexByte(plaintext, '}')
	if start < 0 || end <= start {
		return plaintext
	}
	return plaintext[start : end+1]
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("feishu: empty plaintext")
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > aes.BlockSize || pad > len(data) {
		return nil, fmt.Errorf("feishu: invalid pkcs7 padding")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, fmt.Errorf("feishu: corrupt pkcs7 padding")
		}
	}
	return data[:len(data)-pad], nil
}

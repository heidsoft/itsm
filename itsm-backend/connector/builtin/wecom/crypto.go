package wecom

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// 企业微信回调加解密（官方 WXBizMsgCrypt 算法）。
//
// EncodingAESKey 是 43 字符 base64（无填充），补 "=" 后解码得 32 字节 AES-256 密钥；
// CBC 的 IV 取密钥前 16 字节。明文结构为：
//
//	16 字节随机串 + 4 字节大端消息长度 + 消息体 + receiveid（企业 corpid）
//
// 填充用 PKCS#7，块大小 32（企微官方约定，非 AES 的 16）。

// decodeAESKey 把 EncodingAESKey 还原为 32 字节密钥。
func decodeAESKey(encodingAESKey string) ([]byte, error) {
	if encodingAESKey == "" {
		return nil, fmt.Errorf("wecom: encoding_aes_key not configured")
	}
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("wecom: invalid encoding_aes_key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("wecom: encoding_aes_key must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

// pkcs7Unpad 去除块大小 32 的 PKCS#7 填充。
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("wecom: empty plaintext")
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > 32 || pad > len(data) {
		return nil, fmt.Errorf("wecom: invalid pkcs7 padding")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, fmt.Errorf("wecom: corrupt pkcs7 padding")
		}
	}
	return data[:len(data)-pad], nil
}

// decryptMessage 解密 base64 密文，返回消息体与 receiveid（corpid）。
// 调用方负责校验 receiveid 与配置的 corpid 一致（防跨企业伪造）。
func decryptMessage(aesKey []byte, ciphertextB64 string) ([]byte, string, error) {
	if len(aesKey) != 32 {
		return nil, "", fmt.Errorf("wecom: aes key must be 32 bytes")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, "", fmt.Errorf("wecom: invalid base64 ciphertext: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, "", fmt.Errorf("wecom: ciphertext length not a multiple of block size")
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, "", fmt.Errorf("wecom: new cipher: %w", err)
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, aesKey[:aes.BlockSize]).CryptBlocks(plaintext, ciphertext)
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return nil, "", err
	}
	if len(plaintext) < 20 {
		return nil, "", fmt.Errorf("wecom: plaintext too short")
	}
	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	if uint64(20)+uint64(msgLen) > uint64(len(plaintext)) {
		return nil, "", fmt.Errorf("wecom: declared message length exceeds plaintext")
	}
	msg := plaintext[20 : 20+msgLen]
	receiveID := string(plaintext[20+msgLen:])
	return msg, receiveID, nil
}

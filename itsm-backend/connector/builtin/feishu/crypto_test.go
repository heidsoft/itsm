package feishu

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"
)

// encryptOfficial 按官方算法造一份密文，供解密测试往返验证：
// 明文 = 16 字节随机 + 4 字节大端长度 + msg + receive_id，PKCS#7 填充，
// 密文 = base64(IV + AES-256-CBC(key=sha256(encrypt_key)))
func encryptOfficial(t *testing.T, encryptKey, msg, receiveID string) string {
	t.Helper()
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	plain := make([]byte, 0, 20+len(msg)+len(receiveID)+aes.BlockSize)
	plain = append(plain, make([]byte, 16)...)
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(msg)))
	plain = append(plain, lenBuf[:]...)
	plain = append(plain, msg...)
	plain = append(plain, receiveID...)
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	for i := 0; i < pad; i++ {
		plain = append(plain, byte(pad))
	}
	out := make([]byte, aes.BlockSize+len(plain))
	iv := out[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		t.Fatal(err)
	}
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out[aes.BlockSize:], plain)
	return base64.StdEncoding.EncodeToString(out)
}

func envelope(cipherB64 string) []byte {
	b, _ := json.Marshal(map[string]string{"encrypt": cipherB64})
	return b
}

func TestDecryptInbound_OfficialRoundTrip(t *testing.T) {
	f := New()
	f.client = NewClient("", "app", "sec", "", "ek-roundtrip")
	msg := `{"type":"url_verification","challenge":"chal-enc","token":"tok"}`
	body := envelope(encryptOfficial(t, "ek-roundtrip", msg, "app-receiver"))

	plain, err := f.DecryptInbound(body)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != msg {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", plain, msg)
	}
}

// TestDecryptInbound_PureJSONPlaintextFallback 明文若不带 16+4 结构（纯 JSON +
// 填充），必须走官方 SDK 的 JSON 扫描兜底正确取出。
func TestDecryptInbound_PureJSONPlaintextFallback(t *testing.T) {
	msg := []byte(`{"type":"event_callback","event":{"k":"v"}}`)
	key := sha256.Sum256([]byte("ek-json"))
	block, _ := aes.NewCipher(key[:])
	padded := append([]byte{}, msg...)
	pad := aes.BlockSize - len(padded)%aes.BlockSize
	for i := 0; i < pad; i++ {
		padded = append(padded, byte(pad))
	}
	out := make([]byte, aes.BlockSize+len(padded))
	copy(out[:aes.BlockSize], []byte("0123456789abcdef"))
	cipher.NewCBCEncrypter(block, out[:aes.BlockSize]).CryptBlocks(out[aes.BlockSize:], padded)

	plain, err := decryptInboundPayload("ek-json", envelope(base64.StdEncoding.EncodeToString(out)))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != string(msg) {
		t.Fatalf("fallback mismatch:\n got %s\nwant %s", plain, msg)
	}
}

// TestDecryptInbound_PlaintextPassthrough 明文回调原样放行。
func TestDecryptInbound_PlaintextPassthrough(t *testing.T) {
	body := []byte(`{"type":"url_verification","challenge":"c1"}`)
	plain, err := decryptInboundPayload("ek-x", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(plain) != string(body) {
		t.Fatalf("plaintext must pass through unchanged")
	}
}

// TestDecryptInbound_EncryptedWithoutKeyFailsClosed 加密回调在未配置
// encrypt_key 时必须失败关闭，不得静默按明文解析。
func TestDecryptInbound_EncryptedWithoutKeyFailsClosed(t *testing.T) {
	body := envelope(encryptOfficial(t, "ek-any", `{"type":"event_callback"}`, ""))
	if _, err := decryptInboundPayload("", body); err == nil {
		t.Fatal("expected fail-closed error when encrypt_key is missing")
	}
}

// TestDecryptInbound_WrongKeyDoesNotYieldMessage 密钥错误不得还原出消息。
func TestDecryptInbound_WrongKeyDoesNotYieldMessage(t *testing.T) {
	msg := `{"type":"event_callback","challenge":"secret-chal"}`
	body := envelope(encryptOfficial(t, "ek-right", msg, "rcv"))
	plain, err := decryptInboundPayload("ek-wrong", body)
	if err == nil && string(plain) == msg {
		t.Fatal("wrong key must not decrypt to the original message")
	}
}

// TestParseInbound_TransparentDecrypt ParseInbound 必须透明解密 Encrypt 信封，
// 调用方无需先手动解密。
func TestParseInbound_TransparentDecrypt(t *testing.T) {
	f := New()
	f.client = NewClient("", "app", "sec", "tok", "ek-parse")
	msg := `{"type":"url_verification","challenge":"chal-2","token":"tok"}`
	body := envelope(encryptOfficial(t, "ek-parse", msg, "rcv"))

	inbound, err := f.ParseInbound(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if inbound.Type != "url_verification" || inbound.Content != "chal-2" {
		t.Fatalf("unexpected inbound: %+v", inbound)
	}
}

package wecom

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"

	"itsm-backend/connector"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 43 字符 EncodingAESKey（官方样例同款），补 "=" 后 base64 解码为 32 字节密钥。
const testEncodingAESKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

// encryptForTest 独立构造企微密文（不复用任何生产加密代码）：
// 16 字节随机 + 4 字节大端长度 + 消息体 + receiveid，PKCS#7（块 32）填充后 AES-256-CBC。
// 用它喂给生产解密路径，确保解密不是「自加密自解密」的同义反复。
func encryptForTest(t *testing.T, encodingAESKey, msg, receiveID string) string {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	require.NoError(t, err)
	require.Len(t, key, 32)

	raw := make([]byte, 0, 16+4+len(msg)+len(receiveID)+32)
	raw = append(raw, []byte("0123456789abcdef")...) // 16 字节随机串（测试用固定值）
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(msg)))
	raw = append(raw, lenBuf...)
	raw = append(raw, []byte(msg)...)
	raw = append(raw, []byte(receiveID)...)

	pad := 32 - len(raw)%32
	for i := 0; i < pad; i++ {
		raw = append(raw, byte(pad))
	}

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	ciphertext := make([]byte, len(raw))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(ciphertext, raw)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func TestWeComDecryptRoundTrip(t *testing.T) {
	key, err := decodeAESKey(testEncodingAESKey)
	require.NoError(t, err)

	const xml = `<xml><ToUserName><![CDATA[corp-1]]></ToUserName><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[你好]]></Content></xml>`
	ciphertext := encryptForTest(t, testEncodingAESKey, xml, "corp-1")

	msg, receiveID, err := decryptMessage(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, xml, string(msg))
	assert.Equal(t, "corp-1", receiveID)
}

func TestWeComDecodeAESKeyRejectsInvalid(t *testing.T) {
	_, err := decodeAESKey("")
	require.Error(t, err, "未配置必须报错（fail-closed）")

	_, err = decodeAESKey("short")
	require.Error(t, err, "非 32 字节密钥必须拒绝")
}

func TestWeComPKCS7UnpadRejectsCorrupt(t *testing.T) {
	_, err := pkcs7Unpad(nil)
	require.Error(t, err)
	// 末字节声明 5 字节填充，但填充内容不一致 → 拒绝。
	_, err = pkcs7Unpad([]byte{1, 2, 3, 5, 5, 5, 5, 9})
	require.Error(t, err)
	// 合法块 32 填充：末 4 字节均为 4。
	out, err := pkcs7Unpad([]byte{1, 2, 3, 4, 4, 4, 4})
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, out)
}

func newCryptoWeCom(t *testing.T, corpID string) *WeCom {
	t.Helper()
	w := &WeCom{}
	w.token = "token-1"
	w.aesKey = testEncodingAESKey
	w.client = NewClient(corpID, "corp-secret", "agent-id", BaseURL)
	w.cfg = connector.Config{
		TenantID:    1,
		Name:        "wecom",
		Credentials: map[string]string{"corp_id": corpID},
	}
	return w
}

// 关键回归（预览能力收口）：事件回调密文必须被解密成结构化消息。
// 此前 ParseInbound 返回 encrypted_event 密文、Handler 仅日志+ACK，企微入站形同虚设。
func TestWeComParseInboundDecryptsTextEvent(t *testing.T) {
	w := newCryptoWeCom(t, "corp-1")
	const xml = `<xml><ToUserName><![CDATA[corp-1]]></ToUserName>` +
		`<FromUserName><![CDATA[user-9]]></FromUserName>` +
		`<CreateTime>1700000000</CreateTime>` +
		`<MsgType><![CDATA[text]]></MsgType>` +
		`<Content><![CDATA[报修：打印机卡纸]]></Content>` +
		`<MsgId>1234567890</MsgId>` +
		`<AgentID>1000001</AgentID></xml>`
	body, err := json.Marshal(map[string]string{"Encrypt": encryptForTest(t, testEncodingAESKey, xml, "corp-1")})
	require.NoError(t, err)

	msg, err := w.ParseInbound(body)
	require.NoError(t, err)
	assert.Equal(t, "text", msg.Type)
	assert.Equal(t, "报修：打印机卡纸", msg.Content)
	assert.Equal(t, "user-9", msg.UserID)
	assert.Equal(t, "1234567890", msg.MessageID)
	assert.NotEqual(t, "encrypted_event", msg.Type, "不得再把密文当未解密事件放行")
}

func TestWeComParseInboundEventWithoutMsgIDSynthesizesDedupKey(t *testing.T) {
	w := newCryptoWeCom(t, "corp-1")
	const xml = `<xml><FromUserName><![CDATA[user-9]]></FromUserName>` +
		`<CreateTime>1700000000</CreateTime>` +
		`<MsgType><![CDATA[event]]></MsgType>` +
		`<Event><![CDATA[subscribe]]></Event></xml>`
	body, err := json.Marshal(map[string]string{"Encrypt": encryptForTest(t, testEncodingAESKey, xml, "corp-1")})
	require.NoError(t, err)

	msg, err := w.ParseInbound(body)
	require.NoError(t, err)
	assert.Equal(t, "event", msg.Type)
	assert.Equal(t, "subscribe", msg.Extras["event"])
	assert.NotEmpty(t, msg.MessageID, "事件无 MsgId 时须合成去重键，避免重放绕过")
}

func TestWeComParseInboundRejectsReceiveIDMismatch(t *testing.T) {
	w := newCryptoWeCom(t, "corp-1")
	const xml = `<xml><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[x]]></Content></xml>`
	// 密文由另一个企业（corp-evil）加密：receiveid 不符必须拒绝。
	body, err := json.Marshal(map[string]string{"Encrypt": encryptForTest(t, testEncodingAESKey, xml, "corp-evil")})
	require.NoError(t, err)

	_, err = w.ParseInbound(body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "receiveid")
}

func TestWeComParseInboundUnconfiguredAESKeyFailsClosed(t *testing.T) {
	w := newCryptoWeCom(t, "corp-1")
	w.aesKey = ""
	const xml = `<xml><MsgType><![CDATA[text]]></MsgType></xml>`
	body, err := json.Marshal(map[string]string{"Encrypt": encryptForTest(t, testEncodingAESKey, xml, "corp-1")})
	require.NoError(t, err)

	_, err = w.ParseInbound(body)
	require.Error(t, err, "未配置 encoding_aes_key 必须失败，不得放行密文")
}

func TestWeComParseInboundURLVerificationDecryptsEcho(t *testing.T) {
	w := newCryptoWeCom(t, "corp-1")
	// echostr 是密文，解密后才是企微期望原样返回的明文 echo。
	echoCipher := encryptForTest(t, testEncodingAESKey, "echo-plaintext-1", "corp-1")
	body, err := json.Marshal(map[string]string{"echostr": echoCipher})
	require.NoError(t, err)

	msg, err := w.ParseInbound(body)
	require.NoError(t, err)
	assert.Equal(t, "url_verification", msg.Type)
	assert.Equal(t, "echo-plaintext-1", msg.Content, "必须返回解密后的明文，而非密文")
	assert.NotEqual(t, echoCipher, msg.Content)
}

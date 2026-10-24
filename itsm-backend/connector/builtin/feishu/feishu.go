// Package feishu 是飞书/Lark 连接器入口（聚合 Client + Connector）
package feishu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"itsm-backend/connector"
)

// maxCallbackBody 事件回调请求体的读取上限。
//
// 飞书事件体通常几 KB，但回调端点必须假设「有人往这里打流量」：不设上限的
// io.ReadAll 会把内存交给对端控制。超限按 413 拒绝，不去解析。
const maxCallbackBody = 1 << 20 // 1 MiB

// Compile-time assertions
var (
	_ connector.Connector = (*Feishu)(nil)
	_ connector.Receiver  = (*Feishu)(nil)
)

// HTTPHandler 返回一个 http.Handler 用来接收飞书事件回调
// 自动处理 URL Verification 和签名校验
func (f *Feishu) HTTPHandler(r *connector.Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// 原实现是 `body := make([]byte, req.ContentLength); req.Body.Read(body)`，
		// 有两个硬伤：①chunked 请求 ContentLength 为 -1，make([]byte, -1) 直接
		// panic；②Read 不保证一次读满，大事件体只拿到前一段，JSON 解析失败。
		// 改为限额 ReadAll。
		body, err := io.ReadAll(io.LimitReader(req.Body, maxCallbackBody+1))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		if len(body) > maxCallbackBody {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		// URL Verification
		var quick struct {
			Type      string `json:"type"`
			Challenge string `json:"challenge"`
		}
		if err := json.Unmarshal(body, &quick); err == nil && quick.Type == "url_verification" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"challenge":"%s"}`, quick.Challenge)))
			return
		}
		// 签名校验
		headers := map[string]string{
			"X-Lark-Request-Timestamp": req.Header.Get("X-Lark-Request-Timestamp"),
			"X-Lark-Request-Nonce":     req.Header.Get("X-Lark-Request-Nonce"),
			"X-Lark-Signature":         req.Header.Get("X-Lark-Signature"),
		}
		if err := f.VerifySignature(headers, body); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		// 解析
		msg, err := f.ParseInbound(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = r.Dispatch(req.Context(), msg)
		w.WriteHeader(http.StatusOK)
	})
}

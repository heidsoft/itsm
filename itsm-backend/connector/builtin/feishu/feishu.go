// Package feishu 是飞书/Lark 连接器入口（聚合 Client + Connector）
package feishu

import (
	"encoding/json"
	"fmt"
	"net/http"

	"itsm-backend/connector"
)

// Compile-time assertions
var (
	_ connector.Connector = (*Feishu)(nil)
	_ connector.Receiver  = (*Feishu)(nil)
)

// HTTPHandler 返回一个 http.Handler 用来接收飞书事件回调
// 自动处理 URL Verification 和签名校验
func (f *Feishu) HTTPHandler(r *connector.Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body := make([]byte, req.ContentLength)
		if _, err := req.Body.Read(body); err != nil && err.Error() != "EOF" {
			http.Error(w, "read body", http.StatusBadRequest)
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

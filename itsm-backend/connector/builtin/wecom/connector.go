package wecom

import (
	"context"
	"fmt"
	"sync"
	"time"

	"itsm-backend/connector"
)

type WeCom struct {
	client    *Client
	cfg       connector.Config
	mu        sync.RWMutex
	startedAt time.Time
}

func init() {
	connector.MustRegister(func() connector.Connector { return &WeCom{} })
}

func New() *WeCom { return &WeCom{} }

func (w *WeCom) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:         "wecom",
		Version:      "1.0.0",
		Title:        "企业微信 WeCom",
		Provider:     "wecom",
		Type:         connector.TypeIM,
		Description:  "企业微信连接器：应用消息（精确到 UserID/部门/标签）+ 群机器人 Webhook。支持 text/markdown/textcard/news。",
		Capabilities: []connector.Capability{
			connector.CapSendMessage,
			connector.CapSendCard,
		},
		Tags:     []string{"im", "wecom", "wechat", "china"},
		Homepage: "https://developer.work.weixin.qq.com",
	}
}

func (w *WeCom) Init(_ context.Context, cfg connector.Config) error {
	corpID, _ := cfg.Credentials["corp_id"]
	corpSecret, _ := cfg.Credentials["corp_secret"]
	agentID, _ := cfg.Credentials["agent_id"]
	if corpID == "" || corpSecret == "" {
		return fmt.Errorf("wecom: credentials.corp_id and corp_secret are required")
	}
	baseURL, _ := cfg.Settings["base_url"].(string)
	w.client = NewClient(corpID, corpSecret, agentID, baseURL)
	w.cfg = cfg
	w.startedAt = time.Now()
	return nil
}

func (w *WeCom) Send(ctx context.Context, msg *connector.Message) error {
	if w.client == nil {
		return fmt.Errorf("wecom: connector not initialized")
	}
	return w.client.Send(ctx, msg)
}

func (w *WeCom) HealthCheck(ctx context.Context) connector.HealthStatus {
	if w.client == nil {
		return connector.HealthStatus{OK: false, Message: "not initialized"}
	}
	tok, err := w.client.Token(ctx)
	if err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	return connector.HealthStatus{
		OK:        tok != "",
		Message:   "access_token valid",
		CheckedAt: time.Now(),
		Extra:     map[string]interface{}{"started_at": w.startedAt, "uptime_s": int(time.Since(w.startedAt).Seconds())},
	}
}

func (w *WeCom) Close() error { return nil }

var _ connector.Connector = (*WeCom)(nil)

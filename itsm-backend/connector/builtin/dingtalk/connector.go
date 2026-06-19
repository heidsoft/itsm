package dingtalk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"itsm-backend/connector"
)

type DingTalk struct {
	client    *Client
	cfg       connector.Config
	mu        sync.RWMutex
	startedAt time.Time
}

func init() {
	connector.MustRegister(func() connector.Connector { return &DingTalk{} })
}

func New() *DingTalk { return &DingTalk{} }

func (d *DingTalk) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:        "dingtalk",
		Version:     "1.0.0",
		Title:       "钉钉 DingTalk",
		Provider:    "dingtalk",
		Type:        connector.TypeIM,
		Description: "钉钉开放平台连接器：工作通知 + 群机器人（加签模式），支持 text/markdown/actionCard。",
		Capabilities: []connector.Capability{
			connector.CapSendMessage,
			connector.CapSendCard,
			connector.CapReplyMessage,
		},
		Tags:     []string{"im", "dingtalk", "china"},
		Homepage: "https://open.dingtalk.com",
	}
}

func (d *DingTalk) Init(_ context.Context, cfg connector.Config) error {
	appKey, _ := cfg.Credentials["app_key"]
	appSecret, _ := cfg.Credentials["app_secret"]
	agentID, _ := cfg.Credentials["agent_id"]
	if appKey == "" || appSecret == "" {
		return fmt.Errorf("dingtalk: credentials.app_key and app_secret are required")
	}
	baseURL, _ := cfg.Settings["base_url"].(string)
	d.client = NewClient(appKey, appSecret, agentID, baseURL)
	d.cfg = cfg
	d.startedAt = time.Now()
	return nil
}

func (d *DingTalk) Send(ctx context.Context, msg *connector.Message) error {
	if d.client == nil {
		return fmt.Errorf("dingtalk: connector not initialized")
	}
	return d.client.Send(ctx, msg)
}

func (d *DingTalk) HealthCheck(ctx context.Context) connector.HealthStatus {
	if d.client == nil {
		return connector.HealthStatus{OK: false, Message: "not initialized"}
	}
	tok, err := d.client.Token(ctx)
	if err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	return connector.HealthStatus{
		OK:        tok != "",
		Message:   "access_token valid",
		CheckedAt: time.Now(),
		Extra:     map[string]interface{}{"started_at": d.startedAt, "uptime_s": int(time.Since(d.startedAt).Seconds())},
	}
}

func (d *DingTalk) Close() error { return nil }

var _ connector.Connector = (*DingTalk)(nil)

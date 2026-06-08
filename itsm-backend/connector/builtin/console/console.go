// Package console 提供一个"日志输出"连接器
// 用途：本地调试、单元测试、CLI 场景
package console

import (
	"context"
	"fmt"
	"sync"

	"itsm-backend/connector"
)

type Console struct {
	mu      sync.Mutex
	history []connector.Message
	cfg     connector.Config
}

func init() { connector.MustRegister(func() connector.Connector { return New() }) }

func New() *Console { return &Console{} }

func (c *Console) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:         "console",
		Version:      "1.0.0",
		Title:        "控制台日志",
		Provider:     "builtin",
		Type:         connector.TypeCustom,
		Description:  "把消息打印到标准输出，常用于本地开发与单元测试",
		Capabilities: []connector.Capability{connector.CapSendMessage},
		Tags:         []string{"debug", "log", "local"},
	}
}

func (c *Console) Init(_ context.Context, cfg connector.Config) error {
	c.cfg = cfg
	return nil
}

func (c *Console) Send(_ context.Context, msg *connector.Message) error {
	c.mu.Lock()
	c.history = append(c.history, *msg)
	c.mu.Unlock()
	fmt.Printf("[console-connector] -> channel=%s type=%s title=%q content=%q\n",
		msg.Channel, msg.Type, msg.Title, msg.Content)
	return nil
}

func (c *Console) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true, Message: "always healthy"}
}

func (c *Console) Close() error { return nil }

// History 返回最近 N 条（用于测试断言）
func (c *Console) History() []connector.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]connector.Message, len(c.history))
	copy(out, c.history)
	return out
}

package alicloud

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/infrastructure/cloud"
)

// StubProvider 仅用于测试与本地演示，不调用真实阿里云 API。
// 生产装配必须替换为 ECSProvider，否则交付会返回假的 request id。
type StubProvider struct{}

func NewStubProvider() *StubProvider { return &StubProvider{} }

func (p *StubProvider) Name() string { return "alicloud" }

func (p *StubProvider) RequiresCloudAccount() bool { return false }

func (p *StubProvider) Execute(ctx context.Context, req cloud.ExecuteRequest) (*cloud.ExecuteResult, error) {
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if v, ok := req.Payload["compliance_ack"]; ok {
		if b, ok2 := v.(bool); ok2 && !b {
			return nil, fmt.Errorf("Compliance terms not confirmed")
		}
	}

	return &cloud.ExecuteResult{
		Resources: map[string]string{
			"alicloud_request_id": fmt.Sprintf("req-%d", time.Now().UnixNano()),
		},
	}, nil
}

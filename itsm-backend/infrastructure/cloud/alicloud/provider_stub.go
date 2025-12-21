package alicloud

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/infrastructure/cloud"
)

// StubProvider：M2 骨架版本，不调用真实阿里云 API。
// 后续将替换为：RAM Role + STS + OpenAPI SDK 的真实实现。
type StubProvider struct{}

func NewStubProvider() *StubProvider { return &StubProvider{} }

func (p *StubProvider) Name() string { return "alicloud" }

func (p *StubProvider) Execute(ctx context.Context, payload map[string]any) (*cloud.ExecuteResult, error) {
	// 模拟耗时
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// 简单校验：必须合规确认
	if v, ok := payload["compliance_ack"]; ok {
		if b, ok2 := v.(bool); ok2 && !b {
			return nil, fmt.Errorf("未确认合规条款")
		}
	}

	// 返回一个模拟资源ID
	return &cloud.ExecuteResult{
		Resources: map[string]string{
			"alicloud_request_id": fmt.Sprintf("req-%d", time.Now().UnixNano()),
		},
	}, nil
}



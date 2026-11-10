package dto

// CMDB 发现/交付回写共用的值类型、sentinel 与窄契约。
// 2026-11-10 由 service/{ci_attribute_validator,provisioning_delivery}.go 下沉；
// service 侧保留同名别名，errors.Is 身份与 JSON 形状均不变化。

import (
	"context"
	"errors"
)

// ErrCIAttributeValidation 标记「属性值不符合该 CI 类型的定义」（缺必填、类型/枚举不通过、唯一冲突）。
// 调用方（如云发现回写 CI）据此区分「数据要人补录」和写入失败，不得靠匹配错误文案。
var ErrCIAttributeValidation = errors.New("ci attribute validation")

type CIAttributeRequirement struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	DataType     string   `json:"dataType"`
	EnumValues   []string `json:"enumValues"`
	DefaultValue string   `json:"defaultValue"`
	Placeholder  string   `json:"placeholder"`
	HelpText     string   `json:"helpText"`
}

const (
	ProvisioningCIProjectionCreated     = "created"
	ProvisioningCIProjectionUpdated     = "updated"
	ProvisioningCIProjectionDeferred    = "deferred"
	ProvisioningCIProjectionSkipped     = "skipped"
	ProvisioningCIProjectionFailed      = "failed"
	ProvisioningCIProjectionUnavailable = "unavailable"
)

type ProvisioningCIProjectionRequest struct {
	TenantID           int
	ServiceRequestID   int
	ProvisioningTaskID int
	ResourceType       string
	Provider           string
	Resources          map[string]string
	Payload            map[string]any
}

type ProvisioningCIProjectionResult struct {
	Status string
	CIID   int
	Reason string
}

type ProvisioningCIProjector interface {
	ProjectProvisionedResource(ctx context.Context, req ProvisioningCIProjectionRequest) ProvisioningCIProjectionResult
}

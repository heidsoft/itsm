package alicloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"

	"itsm-backend/infrastructure/cloud"
)

// ECSProvider 通过阿里云 ECS RunInstances 交付云主机。
//
// 只负责「怎么调用云 API」；选择哪个租户账号、解析哪份凭据、地域是否在
// 白名单内，都由应用层完成并通过 ExecuteRequest 传入。
type ECSProvider struct {
	logger *zap.SugaredLogger
}

func NewECSProvider(logger *zap.SugaredLogger) *ECSProvider {
	if logger == nil {
		l, _ := zap.NewDevelopment()
		logger = l.Sugar()
	}
	return &ECSProvider{logger: logger}
}

func (*ECSProvider) Name() string { return "aliyun" }

func (*ECSProvider) RequiresCloudAccount() bool { return true }

func (p *ECSProvider) Execute(ctx context.Context, req cloud.ExecuteRequest) (*cloud.ExecuteResult, error) {
	params, err := parseECSLaunchParams(req.Payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Region) == "" {
		return nil, fmt.Errorf("交付参数缺少 regionId")
	}
	if req.Credential.AccessKeyID == "" || req.Credential.AccessKeySecret == "" {
		return nil, fmt.Errorf("阿里云凭据不完整，拒绝发起交付")
	}
	if strings.TrimSpace(req.ClientToken) == "" {
		return nil, fmt.Errorf("交付任务缺少幂等键，拒绝发起交付")
	}

	client, err := newECSClient(req.Region, req.Credential)
	if err != nil {
		return nil, fmt.Errorf("创建阿里云 ECS 客户端失败: %w", err)
	}

	runReq := buildRunInstancesRequest(req, params)
	resp, err := client.RunInstancesWithOptions(runReq, runtimeOptions())
	if err != nil {
		suffix := describeAliyunError(err)
		// 业务拒绝（余额不足/配额/参数不存在）重试不会成功，标成永久错误让上层
		// 停止自动重投；网络抖动、限流、5xx 仍按可重试处理。
		if code := aliyunErrorCode(err); code != "" && isPermanentAliyunCode(code) {
			return nil, fmt.Errorf("%w: 阿里云 RunInstances 失败: %s", cloud.ErrPermanent, suffix)
		}
		return nil, fmt.Errorf("阿里云 RunInstances 失败: %s", suffix)
	}
	if resp == nil || resp.Body == nil || resp.Body.InstanceIdSets == nil {
		return nil, fmt.Errorf("阿里云 RunInstances 未返回实例集合")
	}

	instanceIDs := make([]string, 0, len(resp.Body.InstanceIdSets.InstanceIdSet))
	for _, id := range resp.Body.InstanceIdSets.InstanceIdSet {
		if v := tea.StringValue(id); v != "" {
			instanceIDs = append(instanceIDs, v)
		}
	}
	if len(instanceIDs) == 0 {
		return nil, fmt.Errorf("阿里云 RunInstances 返回成功但未返回实例 ID，拒绝标记交付成功")
	}

	result := map[string]string{
		"provider":     "aliyun",
		"account_id":   req.AccountID,
		"region_id":    req.Region,
		"instance_id":  instanceIDs[0],
		"instance_ids": strings.Join(instanceIDs, ","),
	}
	if requestID := tea.StringValue(resp.Body.RequestId); requestID != "" {
		result["request_id"] = requestID
	}
	if tradePrice := tea.Float32Value(resp.Body.TradePrice); tradePrice > 0 {
		result["trade_price"] = strconv.FormatFloat(float64(tradePrice), 'f', -1, 32)
	}

	p.logger.Infow("aliyun ecs instance provisioned",
		"tenantID", req.TenantID,
		"taskID", req.TaskID,
		"serviceRequestID", req.ServiceRequestID,
		"accountID", req.AccountID,
		"region", req.Region,
		"instanceIDs", instanceIDs)
	return &cloud.ExecuteResult{Resources: result}, nil
}

type ecsLaunchParams struct {
	ImageID                 string
	InstanceType            string
	SecurityGroupID         string
	VSwitchID               string
	ZoneID                  string
	InstanceName            string
	Description             string
	KeyPairName             string
	Password                string
	SystemDiskCategory      string
	SystemDiskSize          string
	InstanceChargeType      string
	Amount                  int32
	InternetMaxBandwidthOut int32
}

func parseECSLaunchParams(payload map[string]any) (*ecsLaunchParams, error) {
	if payload == nil {
		return nil, fmt.Errorf("交付参数为空")
	}
	if needsPublicIP, _ := payload["needsPublicIp"].(bool); needsPublicIP {
		// 公网交付必须先落实源 IP 白名单。当前 Provider 没有安全组规则管理权限，
		// 无法证明白名单已生效，因此明确拒绝，而不是开出一台全公网可达的机器。
		return nil, fmt.Errorf("当前交付未开放公网 IP：源 IP 白名单需要安全组规则管理权限，请先按内网实例交付或由运维预置受限安全组")
	}

	params := &ecsLaunchParams{
		ImageID:                 stringParam(payload, "imageId"),
		InstanceType:            stringParam(payload, "instanceType"),
		SecurityGroupID:         stringParam(payload, "securityGroupId"),
		VSwitchID:               stringParam(payload, "vSwitchId"),
		ZoneID:                  stringParam(payload, "zoneId"),
		InstanceName:            stringParam(payload, "instanceName"),
		Description:             stringParam(payload, "description"),
		KeyPairName:             stringParam(payload, "keyPairName"),
		Password:                stringParam(payload, "password"),
		SystemDiskCategory:      stringParam(payload, "systemDiskCategory"),
		InstanceChargeType:      strings.TrimSpace(stringParam(payload, "instanceChargeType")),
		Amount:                  1,
		InternetMaxBandwidthOut: int32Param(payload, "internetMaxBandwidthOut", 0),
	}

	var missing []string
	if params.ImageID == "" {
		missing = append(missing, "imageId")
	}
	if params.InstanceType == "" {
		missing = append(missing, "instanceType")
	}
	if params.SecurityGroupID == "" {
		missing = append(missing, "securityGroupId")
	}
	if params.VSwitchID == "" {
		missing = append(missing, "vSwitchId")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("交付参数缺少必填字段: %s", strings.Join(missing, ", "))
	}

	if size := stringParam(payload, "systemDiskSize"); size != "" {
		params.SystemDiskSize = size
	}
	if params.InstanceChargeType == "" {
		params.InstanceChargeType = "PostPaid"
	}
	if params.InstanceChargeType != "PostPaid" {
		return nil, fmt.Errorf("当前只支持按量付费（instanceChargeType=PostPaid），拒绝创建 %s 实例", params.InstanceChargeType)
	}
	if amount := int32Param(payload, "quantity", 1); amount > 1 && amount <= 100 {
		params.Amount = amount
	}
	if params.InternetMaxBandwidthOut < 0 {
		params.InternetMaxBandwidthOut = 0
	}
	return params, nil
}

func buildRunInstancesRequest(req cloud.ExecuteRequest, params *ecsLaunchParams) *ecs.RunInstancesRequest {
	instanceName := params.InstanceName
	if instanceName == "" {
		instanceName = fmt.Sprintf("itsm-sr-%d", req.ServiceRequestID)
	}
	runReq := &ecs.RunInstancesRequest{
		RegionId:                tea.String(req.Region),
		ImageId:                 tea.String(params.ImageID),
		InstanceType:            tea.String(params.InstanceType),
		SecurityGroupId:         tea.String(params.SecurityGroupID),
		VSwitchId:               tea.String(params.VSwitchID),
		InstanceName:            tea.String(truncate(instanceName, 128)),
		ClientToken:             tea.String(req.ClientToken),
		Amount:                  tea.Int32(params.Amount),
		InstanceChargeType:      tea.String(params.InstanceChargeType),
		InternetMaxBandwidthOut: tea.Int32(params.InternetMaxBandwidthOut),
		Description:             tea.String(truncate(params.Description, 256)),
		Tag: []*ecs.RunInstancesRequestTag{
			{Key: tea.String("itsm-tenant-id"), Value: tea.String(strconv.Itoa(req.TenantID))},
			{Key: tea.String("itsm-service-request-id"), Value: tea.String(strconv.Itoa(req.ServiceRequestID))},
			{Key: tea.String("itsm-provisioning-task-id"), Value: tea.String(strconv.Itoa(req.TaskID))},
		},
	}
	if params.ZoneID != "" {
		runReq.ZoneId = tea.String(params.ZoneID)
	}
	if params.KeyPairName != "" {
		runReq.KeyPairName = tea.String(params.KeyPairName)
	}
	if params.Password != "" {
		runReq.Password = tea.String(params.Password)
	}
	if params.SystemDiskCategory != "" || params.SystemDiskSize != "" {
		runReq.SystemDisk = &ecs.RunInstancesRequestSystemDisk{}
		if params.SystemDiskCategory != "" {
			runReq.SystemDisk.Category = tea.String(params.SystemDiskCategory)
		}
		if params.SystemDiskSize != "" {
			runReq.SystemDisk.Size = tea.String(params.SystemDiskSize)
		}
	}
	return runReq
}

func newECSClient(region string, cred cloud.Credential) (*ecs.Client, error) {
	cfg := &openapiutil.Config{}
	cfg.RegionId = tea.String(region)
	cfg.AccessKeyId = tea.String(cred.AccessKeyID)
	cfg.AccessKeySecret = tea.String(cred.AccessKeySecret)
	cfg.ConnectTimeout = tea.Int(5_000)
	cfg.ReadTimeout = tea.Int(30_000)
	if cred.SessionToken != "" {
		cfg.SecurityToken = tea.String(cred.SessionToken)
	}
	return ecs.NewClient(cfg)
}

func runtimeOptions() *dara.RuntimeOptions {
	return &dara.RuntimeOptions{
		ConnectTimeout: tea.Int(5_000),
		ReadTimeout:    tea.Int(30_000),
		Autoretry:      tea.Bool(false),
		MaxAttempts:    tea.Int(1),
	}
}

// describeAliyunError 只保留错误码、请求 ID 和截断后的消息，
// 避免把请求体（可能含密码）或完整响应写进任务错误和审计。
func describeAliyunError(err error) string {
	var sdkErr *tea.SDKError
	if errors.As(err, &sdkErr) {
		code := tea.StringValue(sdkErr.Code)
		message := sanitizeMessage(tea.StringValue(sdkErr.Message))
		requestID := requestIDFromSDKError(sdkErr)
		parts := make([]string, 0, 3)
		if code != "" {
			parts = append(parts, "code="+code)
		}
		if requestID != "" {
			parts = append(parts, "requestId="+requestID)
		}
		if message != "" {
			parts = append(parts, "message="+message)
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return sanitizeMessage(err.Error())
}

func requestIDFromSDKError(err *tea.SDKError) string {
	if err == nil || err.Data == nil {
		return ""
	}
	var payload struct {
		RequestID string `json:"RequestId"`
		RequestId string `json:"requestId"`
	}
	if json.Unmarshal([]byte(tea.StringValue(err.Data)), &payload) != nil {
		return ""
	}
	if payload.RequestID != "" {
		return payload.RequestID
	}
	return payload.RequestId
}

func sanitizeMessage(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	return truncate(message, 300)
}

func truncate(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func stringParam(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func int32Param(payload map[string]any, key string, defaultValue int32) int32 {
	raw := stringParam(payload, key)
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(parsed)
}

// aliyunErrorCode 从 SDK 错误里取业务错误码（网络错误没有 code，返回空）。
func aliyunErrorCode(err error) string {
	var sdkErr *tea.SDKError
	if errors.As(err, &sdkErr) {
		return tea.StringValue(sdkErr.Code)
	}
	return ""
}

// permanentAliyunCodePrefixes 是「重试不可能成功」的错误码前缀。
// 清单只收确定性的业务拒绝：账号状态、配额、资源不存在、参数非法、权限不足。
// 限流（Throttling.*）、ServiceUnavailable、InternalError、超时都不在列——那些
// 重试有意义，误判成永久错误会让一次网络抖动变成「交付永久失败」。
var permanentAliyunCodePrefixes = []string{
	"InvalidAccountStatus.",
	"QuotaExceeded.",
	"InvalidParameter.",
	"InvalidVSwitchId.",
	"InvalidSecurityGroupId.",
	"InvalidImageId.",
	"InvalidInstanceType.",
	"InvalidRegionId.",
	"Forbidden.",
	"OperationDenied.",
	"InvalidInstanceChargeType.",
	"InvalidSystemDisk",
}

func isPermanentAliyunCode(code string) bool {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return false
	}
	for _, prefix := range permanentAliyunCodePrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

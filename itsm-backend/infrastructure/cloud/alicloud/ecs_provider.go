package alicloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
		return nil, fmt.Errorf("Delivery parameter missing regionId")
	}
	if req.Credential.AccessKeyID == "" || req.Credential.AccessKeySecret == "" {
		return nil, fmt.Errorf("Alibaba Cloud credentials are incomplete and delivery is refused.")
	}
	if strings.TrimSpace(req.ClientToken) == "" {
		return nil, fmt.Errorf("The delivery task lacks an idempotent key and the delivery is refused.")
	}

	client, err := newECSClient(req.Region, req.Credential)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Alibaba Cloud ECS client: %w", err)
	}

	runReq := buildRunInstancesRequest(req, params)
	resp, err := client.RunInstancesWithOptions(runReq, runtimeOptions())
	if err != nil {
		suffix := describeAliyunError(err)
		// 业务拒绝（余额不足/配额/参数不存在）重试不会成功，标成永久错误让上层
		// 停止自动重投；网络抖动、限流、5xx 仍按可重试处理。
		if code := aliyunErrorCode(err); code != "" && isPermanentAliyunCode(code) {
			return nil, fmt.Errorf("%w: Alibaba Cloud RunInstances failed: %s", cloud.ErrPermanent, suffix)
		}
		return nil, fmt.Errorf("Alibaba Cloud RunInstances failed: %s", suffix)
	}
	if resp == nil || resp.Body == nil || resp.Body.InstanceIdSets == nil {
		return nil, fmt.Errorf("Alibaba Cloud RunInstances did not return the instance collection")
	}

	instanceIDs := make([]string, 0, len(resp.Body.InstanceIdSets.InstanceIdSet))
	for _, id := range resp.Body.InstanceIdSets.InstanceIdSet {
		if v := tea.StringValue(id); v != "" {
			instanceIDs = append(instanceIDs, v)
		}
	}
	if len(instanceIDs) == 0 {
		return nil, fmt.Errorf("Alibaba Cloud RunInstances returns successfully but does not return the instance ID, rejecting the mark delivery successfully")
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
		return nil, fmt.Errorf("Delivery parameter is empty")
	}
	if needsPublicIP, _ := payload["needsPublicIp"].(bool); needsPublicIP {
		// 公网交付必须先落实源 IP 白名单。当前 Provider 没有安全组规则管理权限，
		// 无法证明白名单已生效，因此明确拒绝，而不是开出一台全公网可达的机器。
		return nil, fmt.Errorf("The current delivery does not open public IP: the source IP whitelist requires security group rule management rights. Please first deliver it according to the intranet instance or preset the restricted security group by operation and maintenance.")
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
		return nil, fmt.Errorf("Delivery parameters are missing required fields: %s", strings.Join(missing, ", "))
	}

	if size := stringParam(payload, "systemDiskSize"); size != "" {
		params.SystemDiskSize = size
	}
	if params.InstanceChargeType == "" {
		params.InstanceChargeType = "PostPaid"
	}
	if params.InstanceChargeType != "PostPaid" {
		return nil, fmt.Errorf("Currently only pay-as-you-go (instanceChargeType=PostPaid) is supported, and the creation of %s instance is refused.", params.InstanceChargeType)
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

// BuildDryRunRequest 用生产交付的同一套参数解析/构造逻辑，生成一次
// RunInstances DryRun 校验请求。DryRun=true 时阿里云只检查必填参数、请求格式、
// RAM 权限、服务限制与可用 ECS 资源，不创建实例、不产生费用；校验通过时云侧
// 返回 DryRunOperation 错误码（这是成功信号，调用方需单独识别）。
//
// 单独导出是为了让「只读 API 验证」与「交付前预检」复用同一份构造逻辑，
// 避免验证跑通的参数与真正交付的参数在字段上漂移（例如 ImageId 失效、
// SystemDisk 漏传、Tag 谱系缺失）。Execute 里的前置校验在这里同样生效。
func BuildDryRunRequest(req cloud.ExecuteRequest) (*ecs.RunInstancesRequest, error) {
	params, err := parseECSLaunchParams(req.Payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Region) == "" {
		return nil, fmt.Errorf("Delivery parameter missing regionId")
	}
	if strings.TrimSpace(req.ClientToken) == "" {
		return nil, fmt.Errorf("The delivery task lacks an idempotent key and the verification is refused.")
	}
	runReq := buildRunInstancesRequest(req, params)
	runReq.DryRun = tea.Bool(true)
	return runReq, nil
}

// SDKErrorClass 是一次 SDK 错误的可行动分类结果，供交付失败处理、只读验证
// 工具与排障复用，避免各处重复实现错误码提取规则而漂移。
type SDKErrorClass struct {
	Code      string // 业务错误码；空表示不是 SDK 业务错误（多为网络/超时）
	Permanent bool   // 重试期望收益为零，调用方必须停止自动重投
	Detail    string // 脱敏后的对账文案：code/requestId/截断消息，不含请求体
}

// ClassifySDKError 把 SDK 错误收敛成三元组。Detail 复用交付失败写审计时
// 的同一套脱敏逻辑，保证验证工具看到的文案与线上任务错误一致。
//
// err 为 nil 是合法输入：只读校验里「非法参数却返回成功」正是要被记录的
// 结果，分类函数必须 fail-safe 返回空分类，不能让观测工具崩在错误处理上。
func ClassifySDKError(err error) SDKErrorClass {
	if err == nil {
		return SDKErrorClass{}
	}
	code := aliyunErrorCode(err)
	return SDKErrorClass{
		Code:      code,
		Permanent: isPermanentAliyunCode(code),
		Detail:    describeAliyunError(err),
	}
}

// describeAliyunError 只保留错误码、请求 ID 和截断后的消息，
// 避免把请求体（可能含密码）或完整响应写进任务错误和审计。
// nil 输入返回空串：错误处理函数本身不允许成为新的故障点。
func describeAliyunError(err error) string {
	if err == nil {
		return ""
	}
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

// sensitiveValuePattern 匹配云侧错误消息里可能回显的请求参数值。阿里云部分
// 业务错误会把触发错误的参数原样带回 message，交付请求里含 Password /
// AccessKeySecret 时，只做空白折叠会把凭据写进任务错误和审计日志。
//
// 只列确定高危的键名：code= / requestId= / message= 是对账证据，绝不能误脱敏；
// \b 边界避免把 secretId、tokenType 这类含敏感词根的业务字段一起吃掉。
// 键名后的 "? 与分隔符前的空白用来覆盖 JSON 形态 "Password":"xxx"——
// 云侧返回体是 JSON，只认 password=xxx 会漏掉最常见的泄露来源。
var sensitiveValuePattern = regexp.MustCompile(`(?i)\b(password|passwd|pwd|access[_-]?key[_-]?secret|secret|token|api[_-]?key|private[_-]?key|credential|authorization)\b"?\s*[=:]\s*"?([^\s,;"'}&]+)"?`)

func sanitizeMessage(message string) string {
	// 先脱敏再折叠/截断：若先截断，敏感值可能被切成半截反而绕过匹配。
	message = sensitiveValuePattern.ReplaceAllString(message, "$1=[REDACTED]")
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
//
// 2026-11-03 真实账号验证补入 InvalidOperation.NotSupportedEndpoint / RegionNotSupported：
// 地域配置错误时云侧返回 400「The specified endpoint can't operate this region」，
// 此前不在清单里会被判可重试，同一份错配置被重投 5 次。只列这两个确切前缀而不是
// 整个 InvalidOperation.*：该命名空间下还有「实例状态冲突、资源正在变更」这类
// 临时性拒绝，等状态变化后重试是有意义的。
var permanentAliyunCodePrefixes = []string{
	"InvalidAccountStatus.",
	"QuotaExceeded.",
	"InvalidParameter.",
	"InvalidVSwitchId.",
	"InvalidSecurityGroupId.",
	"InvalidImageId.",
	"InvalidInstanceType.",
	"InvalidRegionId.",
	"InvalidOperation.NotSupportedEndpoint",
	"InvalidOperation.RegionNotSupported",
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

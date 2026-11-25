package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"

	"itsm-backend/service/cloud"
)

// 只读探测工具（cmd/aliyun-connection-check、cmd/aliyun-sdk-api-verify 等）复用的导出包装。
// 不放进 ecs.go 是为了避开并行批次的同文件编辑；语义与适配器内部完全一致。

// NewECSClientForProbe 按 Region + 运行时凭据构造 ECS 客户端。
func NewECSClientForProbe(region string, cred *cloud.ResolvedCredential) (*ecs.Client, error) {
	return newECSClient(region, cred)
}

// ProbeRuntimeOptions 与发现链路同一套超时/不重试语义。
func ProbeRuntimeOptions() *dara.RuntimeOptions {
	return runtimeOptions()
}

// ===== 云安全中心（SAS）/ 云监控（CMS）只读探测 =====
//
// 这两条链路走的是手写 RPC 签名（HMAC-SHA1 v1.0），不像 ECS 那样有官方 SDK
// 兜底，因此签名实现、端点、版本、参数、权限、开通状态全靠真实调用验证。
//
// 探测入口做两件事：
//  1. 复用生产同一套签名与端点（newRPCClient），不是另写一份；
//  2. 用运行时只读动作白名单把写操作挡在门外——白名单是数据不是注释，
//     将来有人往探测命令里塞 OpenVulScan 之类的写动作会直接被拒。

// readOnlySecurityActions 云安全中心只读动作白名单。
//
// 收录范围：产品实际调用的 DescribeVulList（漏洞同步主路径），以及排查
// 开通状态/资产清单/安全事件所需的只读动作。全部以 Describe 开头且语义为读。
var readOnlySecurityActions = map[string]bool{
	"DescribeVulList":              true, // 产品在用：漏洞列表
	"DescribeVulDetails":           true, // 漏洞详情
	"DescribeVulNumStatistics":     true, // 漏洞统计
	"DescribeCloudCenterInstances": true, // 安全中心资产清单
	"DescribeAllGroups":            true, // 资产分组
	"DescribeWarningMachines":      true, // 安全事件告警机器（需 RiskId）
	"DescribeCheckWarningSummary":  true, // 基线检查风险项汇总（RiskId 的来源）
	"DescribeVersionConfig":        true, // 开通版本与配置
	"DescribeVpcList":              true, // VPC 清单
}

// readOnlyMonitorActions 云监控只读动作白名单。
var readOnlyMonitorActions = map[string]bool{
	"DescribeAlertLogList":       true, // 产品在用：报警历史
	"DescribeMetricList":         true, // 指标元数据
	"DescribeResourceMetricList": true, // 资源指标
}

// ReadOnlySecurityActions 返回云安全中心只读动作白名单的副本（排序后），
// 供探测工具输出与守卫测试使用。
func ReadOnlySecurityActions() []string { return sortedKeys(readOnlySecurityActions) }

// ReadOnlyMonitorActions 返回云监控只读动作白名单的副本（排序后）。
func ReadOnlyMonitorActions() []string { return sortedKeys(readOnlyMonitorActions) }

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// RPCProbeResult 是一次只读 RPC 探测的结构化结果。
//
// 关键点：Body 保留原始响应体。阿里云 RPC 的业务错误可能以 HTTP 200 +
// 响应体里的 Code 字段返回，只看 HTTP 状态码会把错误当成功——探测必须把
// 原始体带出来，由调用方同时判状态码与 Code。
type RPCProbeResult struct {
	HTTPStatus int             `json:"http_status"`
	Code       string          `json:"code,omitempty"`
	Message    string          `json:"message,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
	Success    bool            `json:"success"`
	Body       json.RawMessage `json:"-"`
}

// ErrNotReadOnlyAction 表示请求的动作不在只读白名单里，调用方不会向云侧发起请求。
type ErrNotReadOnlyAction struct {
	Action string
	Scope  string
}

func (e *ErrNotReadOnlyAction) Error() string {
	return fmt.Sprintf("action %q 不在 %s 只读动作白名单内，拒绝向云侧发起请求", e.Action, e.Scope)
}

// ProbeSecurityCenter 用生产同一套 RPC 签名与端点探测云安全中心只读动作。
//
// action 必须在只读白名单内，否则直接返回 ErrNotReadOnlyAction 且不发起任何请求。
func ProbeSecurityCenter(ctx context.Context, accessKeyID, accessKeySecret, securityToken,
	action string, params map[string]string,
) (*RPCProbeResult, error) {
	if !readOnlySecurityActions[action] {
		return nil, &ErrNotReadOnlyAction{Action: action, Scope: "云安全中心"}
	}
	return probeRPC(ctx, securityCenterEndpoint, securityCenterVersion, action,
		params, accessKeyID, accessKeySecret, securityToken)
}

// ProbeCloudMonitor 用生产同一套 RPC 签名与端点探测云监控只读动作。
func ProbeCloudMonitor(ctx context.Context, accessKeyID, accessKeySecret, securityToken,
	action string, params map[string]string,
) (*RPCProbeResult, error) {
	if !readOnlyMonitorActions[action] {
		return nil, &ErrNotReadOnlyAction{Action: action, Scope: "云监控"}
	}
	return probeRPC(ctx, cloudMonitorEndpoint, cloudMonitorVersion, action,
		params, accessKeyID, accessKeySecret, securityToken)
}

// probeRPC 发起一次 RPC 请求并把响应整理成结构化结果。
//
// 复用生产 rpcClient.callRaw：签名、编码、端点、超时全部是生产那一套，
// 探测跑通即证明生产签名可用。与生产 rpcClient.Call 的唯一差别是这里
// 不把「HTTP 200 + 响应体里带业务错误码」当成功——原始体一并返回，
// Code/Success 由调用方判定，避免探测工具复用同一处盲区。
func probeRPC(ctx context.Context, endpoint, version, action string, params map[string]string,
	accessKeyID, accessKeySecret, securityToken string,
) (*RPCProbeResult, error) {
	caller := newRPCClient(endpoint, version, accessKeyID, accessKeySecret, securityToken)

	raw, err := caller.callRaw(ctx, action, params)
	if err != nil {
		return nil, err
	}

	out := &RPCProbeResult{HTTPStatus: raw.Status, Body: raw.Body, RequestID: raw.RequestID}
	// 业务错误码可能在 200 响应体里（CMS 线上就出现过 Code 为字符串
	// "Throttling" 的错误分支），因此无论状态码都要尝试解出 Code。
	var envelope struct {
		Code      flexString `json:"Code"`
		Message   string     `json:"Message"`
		RequestID string     `json:"RequestId"`
		Success   *bool      `json:"Success"`
	}
	if json.Unmarshal(raw.Body, &envelope) == nil {
		out.Code = strings.TrimSpace(string(envelope.Code))
		out.Message = envelope.Message
		out.RequestID = firstNonEmpty(envelope.RequestID, out.RequestID)
		if envelope.Success != nil {
			out.Success = *envelope.Success
		}
	}
	// 业务错误码为空（或 CMS 成功时的数字 200）才视为成功。
	if out.Code == "" || out.Code == "200" {
		out.Success = true
		out.Code = ""
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

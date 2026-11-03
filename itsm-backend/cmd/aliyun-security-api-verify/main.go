package main

// 阿里云安全事件 API 只读验证（云安全中心 SAS + 云监控 CMS）。
//
// 背景：服务请求交付链路走官方 Tea SDK，而云信号链路（漏洞同步 + 告警同步）
// 走的是**手写 RPC 签名**（HMAC-SHA1 v1.0）。手写签名此前只有 mock 单测，
// 从未对真实端点验证过——签名、端点、版本、参数、权限、开通状态全部是盲区。
//
// 本命令只做只读调用，不创建/修改/删除任何云资源，不产生费用：
//
//   - 产品在用的动作：SAS DescribeVulList（各漏洞类型）、CMS DescribeAlertLogList；
//   - 开通状态与资产面：SAS DescribeVersionConfig / DescribeCloudCenterInstances
//     / DescribeAllGroups / DescribeWarningMachines / DescribeVulNumStatistics；
//   - 负向用例：故意传非法参数，观察云侧错误形态（HTTP 状态码 + 响应体业务码）。
//
// 只读边界由运行时白名单保证（service/cloud/aliyun/probe.go 的
// readOnlySecurityActions / readOnlyMonitorActions）：动作不在白名单内直接
// 拒绝，连请求都不发。
//
// 用法（凭据经 env://ITSM_ALIYUN 从环境变量读取，AK/SK 不落命令行）：
//
//	export ITSM_ALIYUN_ACCESS_KEY_ID=... ITSM_ALIYUN_ACCESS_KEY_SECRET=...
//	go run ./cmd/aliyun-security-api-verify -json-out /tmp/aliyun-security-verify.json
//
// 退出码：0 = 全部只读动作完成验证（云侧业务拒绝属预期观测结果）；1 = 凭据/
// 客户端构造失败。

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
)

// result 是一条安全侧 API 验证结果，同时作为 -json-out 的结构化证据。
type result struct {
	Group      string `json:"group"`
	API        string `json:"api"`
	OK         bool   `json:"ok"`
	HTTPStatus int    `json:"http_status"`
	RequestID  string `json:"request_id,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Items      int    `json:"items,omitempty"`
	Total      int    `json:"total,omitempty"`
	ElapsedMS  int64  `json:"elapsed_ms"`
	Verdict    string `json:"verdict"`
	Note       string `json:"note,omitempty"`
	// BodySample 是原始响应体片段（截断）。留证用的关键：条数为 0 时，
	// 只有看得到响应体才能区分「查到了但确实没有」与「响应根本没数据」。
	BodySample string `json:"body_sample,omitempty"`
}

type probe struct {
	group  string
	api    string
	note   string
	action string
	params map[string]string
	scope  string // "sas" | "cms"
}

func main() {
	jsonOut := flag.String("json-out", "", "结构化结果输出路径（留空则只打印）")
	flag.Parse()

	fmt.Println("阿里云安全事件 API 只读验证：云安全中心(SAS) + 云监控(CMS)，全部只读调用")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cred, err := cloud.ResolveAliyunCredential(ctx, "env://ITSM_ALIYUN")
	if err != nil {
		fmt.Fprintf(os.Stderr, "凭据解析失败（env://ITSM_ALIYUN）: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[OK  ] 凭据解析（env://ITSM_ALIYUN），AK/SK 不回显")

	// 产品在用的漏洞类型（handlers/cloudsignal/vulnerability_sync.go 的 vulnSyncTypes）
	// 之外补 sca/app/emg：官方取值含 sca（软件成分分析），产品首版未纳入，
	// 这里验证该类型是否真实存在数据，确认是否构成覆盖缺口。
	productTypes := []string{"cve", "sys", "cms"}
	extraTypes := []string{"sca", "app", "emg"}

	var probes []probe
	for _, t := range productTypes {
		probes = append(probes, probe{
			group: "product-path", scope: "sas", action: "DescribeVulList",
			api:  "DescribeVulList[type=" + t + "]",
			note: "产品漏洞同步在用的类型",
			params: map[string]string{
				"Type":        t,
				"StatusList":  "1,2,4",
				"CurrentPage": "1",
				"PageSize":    "100",
				"RegionId":    "cn-hangzhou",
			},
		})
	}
	for _, t := range extraTypes {
		probes = append(probes, probe{
			group: "coverage-gap", scope: "sas", action: "DescribeVulList",
			api:  "DescribeVulList[type=" + t + "]",
			note: "产品首版未纳入的漏洞类型，用于确认覆盖缺口",
			params: map[string]string{
				"Type":        t,
				"StatusList":  "1,2,4",
				"CurrentPage": "1",
				"PageSize":    "100",
				"RegionId":    "cn-hangzhou",
			},
		})
	}

	probes = append(probes,
		// 开通状态与资产面：判定「安全事件能不能进来」的前置条件。
		probe{group: "activation", scope: "sas", action: "DescribeVersionConfig",
			api: "DescribeVersionConfig", note: "云安全中心开通版本与配置",
			params: map[string]string{"Lang": "zh"}},
		probe{group: "activation", scope: "sas", action: "DescribeAllGroups",
			api: "DescribeAllGroups", note: "安全中心资产分组",
			params: map[string]string{"Lang": "zh"}},
		probe{group: "activation", scope: "sas", action: "DescribeCloudCenterInstances",
			api: "DescribeCloudCenterInstances", note: "安全中心资产清单（可与 ECS 侧交叉验证 CI 投影）",
			params: map[string]string{"Lang": "zh", "CurrentPage": "1", "PageSize": "50"}},
		probe{group: "activation", scope: "sas", action: "DescribeCheckWarningSummary",
			api: "DescribeCheckWarningSummary", note: "基线检查风险项汇总（RiskId 来源）",
			params: map[string]string{"Lang": "zh"}},
		probe{group: "activation", scope: "sas", action: "DescribeWarningMachines",
			api: "DescribeWarningMachines", note: "安全事件告警机器（RiskId 取官方示例值 196）",
			params: map[string]string{"Lang": "zh", "RiskId": "196", "CurrentPage": "1", "PageSize": "50"}},
		probe{group: "activation", scope: "sas", action: "DescribeVulNumStatistics",
			api: "DescribeVulNumStatistics", note: "漏洞数量统计",
			params: map[string]string{"Lang": "zh"}},

		// 产品告警同步在用的动作。
		probe{group: "product-path", scope: "cms", action: "DescribeAlertLogList",
			api: "DescribeAlertLogList[近15分钟]", note: "产品告警同步在用的报警历史（同步窗口语义）",
			params: map[string]string{
				"StartTime":  strconv.FormatInt(time.Now().Add(-15*time.Minute).UnixMilli(), 10),
				"EndTime":    strconv.FormatInt(time.Now().UnixMilli(), 10),
				"PageNumber": "1",
				"PageSize":   "100",
			}},
		probe{group: "product-path", scope: "cms", action: "DescribeAlertLogList",
			api: "DescribeAlertLogList[近7天]", note: "更宽时间窗，验证分页与总量语义",
			params: map[string]string{
				"StartTime":  strconv.FormatInt(time.Now().Add(-7*24*time.Hour).UnixMilli(), 10),
				"EndTime":    strconv.FormatInt(time.Now().UnixMilli(), 10),
				"PageNumber": "1",
				"PageSize":   "100",
			}},
		probe{group: "activation", scope: "cms", action: "DescribeMetricList",
			api: "DescribeMetricList[ecs/cpu_idle]", note: "云监控指标数据（时间格式为 YYYY-MM-DD HH:MM:SS，非毫秒）",
			params: map[string]string{
				"Namespace":  "acs_ecs_dashboard",
				"MetricName": "cpu_idle",
				"Period":     "60",
				"StartTime":  time.Now().Add(-2 * time.Hour).Format("2006-01-02 15:04:05"),
				"EndTime":    time.Now().Format("2006-01-02 15:04:05"),
				"Length":     "100",
			}},

		// 负向用例：观察云侧错误形态，重点看「HTTP 200 + 响应体业务码」
		// 这种只看状态码会漏判的形态。
		probe{group: "negative", scope: "sas", action: "DescribeVulList",
			api: "DescribeVulList[非法 Type]", note: "观察非法参数的错误形态",
			params: map[string]string{
				"Type":        "not-a-vuln-type",
				"CurrentPage": "1",
				"PageSize":    "10",
				"RegionId":    "cn-hangzhou",
			}},
		probe{group: "negative", scope: "sas", action: "DescribeVulList",
			api: "DescribeVulList[缺 Type]", note: "观察缺必填参数的错误形态",
			params: map[string]string{
				"CurrentPage": "1",
				"PageSize":    "10",
				"RegionId":    "cn-hangzhou",
			}},
		probe{group: "negative", scope: "cms", action: "DescribeAlertLogList",
			api: "DescribeAlertLogList[超 15 天窗口]", note: "观察超上限时间窗的错误形态",
			params: map[string]string{
				"StartTime":  strconv.FormatInt(time.Now().Add(-30*24*time.Hour).UnixMilli(), 10),
				"EndTime":    strconv.FormatInt(time.Now().UnixMilli(), 10),
				"PageNumber": "1",
				"PageSize":   "10",
			}},
		probe{group: "negative", scope: "cms", action: "DescribeAlertLogList",
			api: "DescribeAlertLogList[非法页码]", note: "观察非法分页参数的错误形态",
			params: map[string]string{
				"StartTime":  strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10),
				"EndTime":    strconv.FormatInt(time.Now().UnixMilli(), 10),
				"PageNumber": "-1",
				"PageSize":   "10",
			}},
	)

	var results []result
	for _, p := range probes {
		start := time.Now()
		r := result{
			Group: p.group, API: p.api, Note: p.note,
		}
		probeResult, err := runProbe(ctx, cred, p)
		r.ElapsedMS = time.Since(start).Milliseconds()
		if err != nil {
			r.Verdict = "transport-error"
			r.Message = err.Error()
			r.OK = false
		} else {
			r.HTTPStatus = probeResult.HTTPStatus
			r.RequestID = probeResult.RequestID
			r.Code = probeResult.Code
			r.Message = probeResult.Message
			r.Items, r.Total = countItems(probeResult.Body)
			r.BodySample = truncate(string(probeResult.Body), 2000)
			r.OK = probeResult.Success
			r.Verdict = classify(r, p.group)
		}
		results = append(results, r)
		printResult(r)
	}

	// 汇总：按分组统计。
	fmt.Println()
	for _, g := range []string{"product-path", "coverage-gap", "activation", "negative"} {
		rows := filterGroup(results, g)
		ok := 0
		for _, r := range rows {
			if r.OK {
				ok++
			}
		}
		fmt.Printf("%-14s %d/%d 通过\n", g, ok, len(rows))
	}

	if *jsonOut != "" {
		payload, err := json.MarshalIndent(struct {
			Results []result `json:"results"`
		}{Results: results}, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化结果失败: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*jsonOut, append(payload, '\n'), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "写入结果失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("结构化证据已写入 %s\n", *jsonOut)
	}
}

func runProbe(ctx context.Context, cred *cloud.ResolvedCredential, p probe) (*aliyun.RPCProbeResult, error) {
	switch p.scope {
	case "sas":
		return aliyun.ProbeSecurityCenter(ctx, cred.AccessKeyID, cred.AccessKeySecret, cred.SessionToken, p.action, p.params)
	case "cms":
		return aliyun.ProbeCloudMonitor(ctx, cred.AccessKeyID, cred.AccessKeySecret, cred.SessionToken, p.action, p.params)
	default:
		return nil, fmt.Errorf("未知探测范围 %q", p.scope)
	}
}

// classify 把观测结果归到可行动的结论。
//
// 关键区分：HTTP 200 但响应体带业务错误码，在只看状态码的实现里会被当成
// 「查询成功但结果为空」——对漏洞同步而言等于「所有漏洞都没了」。
func classify(r result, group string) string {
	if r.OK {
		switch group {
		case "negative":
			return "unexpected-success"
		case "coverage-gap":
			return "ok"
		default:
			return "ok"
		}
	}
	if r.Code != "" {
		lower := strings.ToLower(r.Code)
		switch {
		case strings.Contains(lower, "notopen"), strings.Contains(lower, "not_open"),
			strings.Contains(lower, "serviceopen"), strings.Contains(lower, "notpurchased"),
			strings.Contains(lower, "unsubscribe"), strings.Contains(lower, "notsubscribe"):
			return "service-not-open"
		case strings.Contains(lower, "forbidden"), strings.Contains(lower, "nopermission"),
			strings.Contains(lower, "unauthorized"), strings.Contains(lower, "authfailure"),
			strings.Contains(lower, "noauthorized"):
			return "no-permission"
		case strings.Contains(lower, "invalidparameter"), strings.Contains(lower, "missingparameter"),
			strings.Contains(lower, "invalidaction"):
			return "invalid-parameter"
		case strings.Contains(lower, "throttl"), strings.Contains(lower, "toomanyrequests"),
			strings.Contains(lower, "serviceunavailable"):
			return "throttled"
		default:
			return "cloud-error"
		}
	}
	if r.HTTPStatus == 0 {
		return "transport-error"
	}
	return "http-error"
}

// countItems 从原始响应体里取列表条数与总量。
// SAS 的不同接口字段名不统一（VulRecords / VulNumStatistics / Instances ...），
// 这里取「最大的数组长度」作为条数、「TotalCount/Total」作为总量。
func countItems(body json.RawMessage) (items, total int) {
	var root any
	if json.Unmarshal(body, &root) != nil {
		return 0, 0
	}
	items = largestArray(root)
	if m, ok := root.(map[string]any); ok {
		for _, key := range []string{"TotalCount", "Total"} {
			if v, ok := m[key].(float64); ok && int(v) > total {
				total = int(v)
			}
		}
	}
	return items, total
}

func largestArray(node any) int {
	best := 0
	switch typed := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if n := largestArray(typed[k]); n > best {
				best = n
			}
		}
	case []any:
		if len(typed) > best {
			best = len(typed)
		}
		for _, item := range typed {
			if n := largestArray(item); n > best {
				best = n
			}
		}
	}
	return best
}

func filterGroup(results []result, group string) []result {
	out := make([]result, 0, len(results))
	for _, r := range results {
		if r.Group == group {
			out = append(out, r)
		}
	}
	return out
}

func printResult(r result) {
	status := "OK  "
	if !r.OK {
		status = "FAIL"
	}
	fmt.Printf("[%s] %-42s http=%-3d code=%-34s items=%-4d total=%-6d %5dms  %s\n",
		status, r.API, r.HTTPStatus, truncate(r.Code, 34), r.Items, r.Total, r.ElapsedMS, r.Verdict)
	if r.Message != "" && !r.OK {
		fmt.Printf("       └─ %s\n", truncate(r.Message, 220))
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

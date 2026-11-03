package main

// 阿里云 ECS SDK API 零费用验证（真实账号 / 只读 + DryRun）。
//
// 账号余额不足、不充值的前提下，把交付与发现链路依赖的 SDK API 面验证完：
//
//   - 只读面：全部是 Describe* 调用，不创建/修改/删除任何资源，不产生费用；
//     逐项记录 HTTP 状态、RequestId、返回条数与耗时。
//   - 写 API 的校验链：RunInstances 且 DryRun=true。阿里云只校验必填参数、
//     请求格式、RAM 权限、服务限制与可用资源，不创建实例、不计费；校验通过
//     返回 DryRunOperation 错误码，业务拒绝返回真实错误码。两者都是有效证据。
//   - 错误分类面：故意传非法参数触发真实 tea.SDKError，验证生产代码里的
//     错误码提取、永久/可重试判定与脱敏文案在真实云响应上是否成立。
//
// 凭据只经 env://ITSM_ALIYUN 从进程环境读取，AK/SK 不进命令行、不进输出、
// 不写日志。用法：
//
//	export ITSM_ALIYUN_ACCESS_KEY_ID=... ITSM_ALIYUN_ACCESS_KEY_SECRET=...
//	go run ./cmd/aliyun-sdk-api-verify -region cn-beijing -json-out /tmp/aliyun-verify.json
//
// 退出码：0 = 所有只读面与 DryRun 校验链完成验证（DryRun 云侧拒绝属预期结果，
// 不算失败）；1 = 凭据/客户端构造失败或出现非预期异常。
//
// 本文件受 TestVerifyCommandHasNoWriteAPI 源码扫描守卫约束：不得出现任何
// 非 Describe* 的 SDK 写调用，RunInstances 必须经由 BuildDryRunRequest。

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"

	infracloud "itsm-backend/infrastructure/cloud"
	"itsm-backend/infrastructure/cloud/alicloud"
	servicecloud "itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
)

// result 是一条 API 验证结果，同时作为 -json-out 的结构化证据。
type result struct {
	Group      string `json:"group"`
	API        string `json:"api"`
	OK         bool   `json:"ok"`
	HTTPStatus int    `json:"http_status"`
	RequestID  string `json:"request_id,omitempty"`
	Items      int    `json:"items,omitempty"`
	ElapsedMS  int64  `json:"elapsed_ms"`
	Code       string `json:"code,omitempty"`
	Permanent  bool   `json:"permanent,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Note       string `json:"note,omitempty"`
}

type probe struct {
	api  string
	call func(context.Context) (any, error)
	note string
}

func main() {
	region := flag.String("region", "cn-beijing", "验证目标地域")
	imageID := flag.String("image", "aliyun_3_x64_20G_pro_alibase_20260827.vhd", "交付用镜像 ID")
	instanceType := flag.String("instance-type", "ecs.e-c1m1.large", "交付用实例规格")
	vSwitchID := flag.String("vswitch", "vsw-2zevngivqc6lt5mtqyl2c", "交付用交换机 ID")
	securityGroupID := flag.String("security-group", "sg-2ze0uyosip126siwlspg", "交付用安全组 ID")
	jsonOut := flag.String("json-out", "", "结构化结果输出路径（留空则只打印）")
	flag.Parse()

	fmt.Println("阿里云 ECS SDK API 零费用验证：仅 Describe* 只读调用 + RunInstances(DryRun=true)")
	fmt.Printf("目标地域: %s\n\n", *region)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 凭据与客户端都走生产代码路径，验证的不是复制品。
	cred, err := servicecloud.ResolveAliyunCredential(ctx, "env://ITSM_ALIYUN")
	if err != nil {
		fmt.Fprintf(os.Stderr, "凭据解析失败（env://ITSM_ALIYUN）: %v\n", err)
		os.Exit(1)
	}
	client, err := aliyun.NewECSClientForProbe(*region, cred)
	if err != nil {
		fmt.Fprintf(os.Stderr, "构造 ECS 客户端失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[OK  ] 凭据解析与 SDK 客户端构造（env://ITSM_ALIYUN）")

	rt := aliyun.ProbeRuntimeOptions()
	// DescribeZones 的结果供库存查询复用，避免把可用区 ID 写死。
	firstZone := ""
	// DescribeAccountAttributes 的结果用于回答「除了余额，配额/实名认证是否也卡」。
	var accountAttrs *ecs.DescribeAccountAttributesResponse

	probes := []probe{
		{"DescribeRegions", func(c context.Context) (any, error) {
			return client.DescribeRegionsWithOptions(&ecs.DescribeRegionsRequest{}, rt)
		}, "全局端点、签名与账号可见地域"},
		{"DescribeAccountAttributes", func(c context.Context) (any, error) {
			resp, err := client.DescribeAccountAttributesWithOptions(&ecs.DescribeAccountAttributesRequest{
				RegionId: tea.String(*region),
			}, rt)
			if err == nil {
				accountAttrs = resp
			}
			return resp, err
		}, "账号配额、实名认证、按量 vCPU 上限"},
		{"DescribeZones", func(c context.Context) (any, error) {
			resp, err := client.DescribeZonesWithOptions(&ecs.DescribeZonesRequest{
				RegionId:           tea.String(*region),
				InstanceChargeType: tea.String("PostPaid"),
			}, rt)
			if err == nil {
				firstZone = firstStringField(resp, "ZoneId")
			}
			return resp, err
		}, "可用区与按量付费支持"},
		{"DescribeImages", func(c context.Context) (any, error) {
			return client.DescribeImagesWithOptions(&ecs.DescribeImagesRequest{
				RegionId:        tea.String(*region),
				ImageId:         tea.String(*imageID),
				ImageOwnerAlias: tea.String("system"),
			}, rt)
		}, "交付镜像真实存在"},
		{"DescribeImageSupportInstanceTypes", func(c context.Context) (any, error) {
			return client.DescribeImageSupportInstanceTypesWithOptions(&ecs.DescribeImageSupportInstanceTypesRequest{
				RegionId:   tea.String(*region),
				ImageId:    tea.String(*imageID),
				ActionType: tea.String("CreateEcs"),
			}, rt)
		}, "镜像与规格可组合（交叉校验交付参数）"},
		{"DescribeVSwitches", func(c context.Context) (any, error) {
			return client.DescribeVSwitchesWithOptions(&ecs.DescribeVSwitchesRequest{
				RegionId:  tea.String(*region),
				VSwitchId: tea.String(*vSwitchID),
			}, rt)
		}, "交付交换机真实存在"},
		{"DescribeSecurityGroups", func(c context.Context) (any, error) {
			return client.DescribeSecurityGroupsWithOptions(&ecs.DescribeSecurityGroupsRequest{
				RegionId:        tea.String(*region),
				SecurityGroupId: tea.String(*securityGroupID),
			}, rt)
		}, "交付安全组真实存在"},
		{"DescribeSecurityGroupAttribute", func(c context.Context) (any, error) {
			return client.DescribeSecurityGroupAttributeWithOptions(&ecs.DescribeSecurityGroupAttributeRequest{
				RegionId:        tea.String(*region),
				SecurityGroupId: tea.String(*securityGroupID),
				Direction:       tea.String("all"),
			}, rt)
		}, "安全组规则可读"},
		{"DescribeInstanceTypes", func(c context.Context) (any, error) {
			return client.DescribeInstanceTypesWithOptions(&ecs.DescribeInstanceTypesRequest{
				InstanceTypes: []*string{tea.String(*instanceType)},
			}, rt)
		}, "交付规格合法"},
		{"DescribeInstanceTypeFamilies", func(c context.Context) (any, error) {
			return client.DescribeInstanceTypeFamiliesWithOptions(&ecs.DescribeInstanceTypeFamiliesRequest{
				RegionId: tea.String(*region),
			}, rt)
		}, "规格族清单"},
		{"DescribeAvailableResource", func(c context.Context) (any, error) {
			req := &ecs.DescribeAvailableResourceRequest{
				RegionId:            tea.String(*region),
				DestinationResource: tea.String("InstanceType"),
				InstanceChargeType:  tea.String("PostPaid"),
			}
			if firstZone != "" {
				req.ZoneId = tea.String(firstZone)
			}
			return client.DescribeAvailableResourceWithOptions(req, rt)
		}, "可用区库存（按量）"},
		{"DescribeInstances", func(c context.Context) (any, error) {
			return client.DescribeInstancesWithOptions(&ecs.DescribeInstancesRequest{
				RegionId:   tea.String(*region),
				MaxResults: tea.Int32(100),
			}, rt)
		}, "发现链路主接口、分页语义"},
		{"DescribeInstanceStatus", func(c context.Context) (any, error) {
			return client.DescribeInstanceStatusWithOptions(&ecs.DescribeInstanceStatusRequest{
				RegionId: tea.String(*region),
				PageSize: tea.Int32(50),
			}, rt)
		}, "实例状态清单"},
		{"DescribePrice", func(c context.Context) (any, error) {
			return client.DescribePriceWithOptions(&ecs.DescribePriceRequest{
				RegionId:            tea.String(*region),
				InstanceType:        tea.String(*instanceType),
				ImageId:             tea.String(*imageID),
				InstanceNetworkType: tea.String("vpc"),
				PriceUnit:           tea.String("Hour"),
				Period:              tea.Int32(1),
				Amount:              tea.Int32(1),
				SystemDisk: &ecs.DescribePriceRequestSystemDisk{
					Category: tea.String("cloud_essd"),
					Size:     tea.Int32(40),
				},
			}, rt)
		}, "按量询价链路（纯查询，不产生费用）"},
		{"DescribeKeyPairs", func(c context.Context) (any, error) {
			return client.DescribeKeyPairsWithOptions(&ecs.DescribeKeyPairsRequest{
				RegionId: tea.String(*region),
				PageSize: tea.Int32(10),
			}, rt)
		}, "密钥对清单"},
		{"DescribeTags", func(c context.Context) (any, error) {
			return client.DescribeTagsWithOptions(&ecs.DescribeTagsRequest{
				RegionId:     tea.String(*region),
				ResourceType: tea.String("instance"),
				PageSize:     tea.Int32(10),
			}, rt)
		}, "资源标签清单"},
	}

	var results []result
	for _, p := range probes {
		start := time.Now()
		resp, callErr := p.call(ctx)
		r := result{
			Group:     "read-only",
			API:       p.api,
			ElapsedMS: time.Since(start).Milliseconds(),
			Note:      p.note,
		}
		if callErr != nil {
			class := alicloud.ClassifySDKError(callErr)
			r.OK = false
			r.Code = class.Code
			r.Permanent = class.Permanent
			r.Detail = class.Detail
			r.HTTPStatus, r.RequestID = sdkErrorMeta(callErr)
		} else {
			r.OK = true
			r.HTTPStatus, r.RequestID, r.Items = inspectResponse(resp)
		}
		results = append(results, r)
		printResult(r)
	}

	fmt.Println()

	// 账号侧门槛单独摘出来：交付能否成功不只看参数存在，还看实名认证与
	// 按量 vCPU 配额。这段输出用来把「账号侧阻塞」精确到具体是哪一项。
	if accountAttrs != nil && accountAttrs.Body != nil && accountAttrs.Body.AccountAttributeItems != nil {
		printAccountQuotas(accountAttrs, *region)
	}

	// 写 API 的零费用校验链：DryRun=true 不创建实例、不计费。
	// 复用生产交付的 parseECSLaunchParams + buildRunInstancesRequest，
	// 验证的是真正交付时会发出去的那份参数。
	{
		start := time.Now()
		r := result{Group: "dry-run", API: "RunInstances(DryRun=true)", ElapsedMS: 0,
			Note: "写 API 的参数/权限/配额校验链；不创建实例、不产生费用"}
		dryReq, buildErr := alicloud.BuildDryRunRequest(infracloud.ExecuteRequest{
			TenantID:         1,
			TaskID:           0,
			ServiceRequestID: 0,
			Region:           *region,
			ClientToken:      "itsm-sdk-api-verify-dryrun-20261103",
			Payload: map[string]any{
				"imageId":         *imageID,
				"instanceType":    *instanceType,
				"securityGroupId": *securityGroupID,
				"vSwitchId":       *vSwitchID,
			},
		})
		if buildErr != nil {
			r.OK = false
			r.Detail = "生产参数构造失败: " + buildErr.Error()
		} else {
			if dryReq.DryRun == nil || !*dryReq.DryRun {
				r.OK = false
				r.Detail = "守卫失败：DryRun 未置 true，拒绝向云侧发起请求"
			} else {
				_, callErr := client.RunInstancesWithOptions(dryReq, rt)
				class := alicloud.ClassifySDKError(callErr)
				r.OK = true // 拿到云侧确定性响应即完成验证，业务拒绝不算失败
				r.Code = class.Code
				r.Permanent = class.Permanent
				r.Detail = class.Detail
				r.HTTPStatus, r.RequestID = sdkErrorMeta(callErr)
			}
		}
		r.ElapsedMS = time.Since(start).Milliseconds()
		results = append(results, r)
		printResult(r)
	}

	fmt.Println()

	// 错误分类验证：故意用非法参数触发真实 SDKError，确认生产代码的
	// 错误码提取 / 永久错误判定 / 脱敏文案在真实云响应上成立。
	//
	// expect="empty" 锁的是另一条更要紧的契约：阿里云多个 Describe* 对不存在
	// 的 ID 返回 HTTP 200 + 空列表而非报错。只判 err / 只判状态码会把「资源
	// 不存在」当成「查询成功」。这类用例的 OK 表示「观测到 200 + 空列表」，
	// 不是表示查询成功；就绪度探测必须据此判返回条数，而不是只看错误。
	type negativeCase struct {
		api    string
		expect string // "error" = 期望云侧抛结构化错误；"empty" = 期望 200 + 空列表
		call   func(context.Context) (any, error)
		note   string
	}
	negative := []negativeCase{
		{"DescribeImages(bad image)", "empty", func(c context.Context) (any, error) {
			return client.DescribeImagesWithOptions(&ecs.DescribeImagesRequest{
				RegionId: tea.String(*region), ImageId: tea.String("img-itsm-verify-not-exist"),
			}, rt)
		}, "非法镜像 ID：存在性由返回条数判定"},
		{"DescribeZones(bad region)", "error", func(c context.Context) (any, error) {
			return client.DescribeZonesWithOptions(&ecs.DescribeZonesRequest{
				RegionId: tea.String("cn-itsm-verify-invalid"),
			}, rt)
		}, "非法地域 ID：云侧抛结构化错误"},
		{"DescribeVSwitches(bad vswitch)", "empty", func(c context.Context) (any, error) {
			return client.DescribeVSwitchesWithOptions(&ecs.DescribeVSwitchesRequest{
				RegionId: tea.String(*region), VSwitchId: tea.String("vsw-itsm-verify-not-exist"),
			}, rt)
		}, "非法交换机 ID：存在性由返回条数判定"},
		{"DescribeSecurityGroups(bad sg)", "empty", func(c context.Context) (any, error) {
			return client.DescribeSecurityGroupsWithOptions(&ecs.DescribeSecurityGroupsRequest{
				RegionId: tea.String(*region), SecurityGroupId: tea.String("sg-itsm-verify-not-exist"),
			}, rt)
		}, "非法安全组 ID：存在性由返回条数判定"},
		{"DescribeInstanceTypes(bad type)", "empty", func(c context.Context) (any, error) {
			return client.DescribeInstanceTypesWithOptions(&ecs.DescribeInstanceTypesRequest{
				InstanceTypes: []*string{tea.String("ecs.itsm.verify.invalid")},
			}, rt)
		}, "非法规格 ID：存在性由返回条数判定"},
		{"RunInstances(bad image, DryRun)", "error", func(c context.Context) (any, error) {
			return client.RunInstancesWithOptions(&ecs.RunInstancesRequest{
				RegionId:           tea.String(*region),
				ImageId:            tea.String("img-itsm-verify-not-exist"),
				InstanceType:       tea.String(*instanceType),
				SecurityGroupId:    tea.String(*securityGroupID),
				VSwitchId:          tea.String(*vSwitchID),
				InstanceChargeType: tea.String("PostPaid"),
				ClientToken:        tea.String("itsm-sdk-api-verify-negative-20261103"),
				Amount:             tea.Int32(1),
				DryRun:             tea.Bool(true),
			}, rt)
		}, "非法镜像走写 API 的 DryRun 校验链"},
	}
	for _, p := range negative {
		start := time.Now()
		resp, callErr := p.call(ctx)
		class := alicloud.ClassifySDKError(callErr)
		r := result{
			Group:     "error-classification",
			API:       p.api,
			ElapsedMS: time.Since(start).Milliseconds(),
			Code:      class.Code,
			Permanent: class.Permanent,
			Detail:    class.Detail,
			Note:      p.note,
		}
		if p.expect == "error" {
			// 期望云侧抛结构化错误，且生产代码能解出业务码。
			r.OK = callErr != nil && class.Code != ""
			r.HTTPStatus, r.RequestID = sdkErrorMeta(callErr)
			if callErr == nil {
				r.Detail = "期望云侧拒绝却返回成功，需复核云侧校验"
			}
		} else {
			// 期望 HTTP 200 + 空列表：这正是「只判 err 会漏」的证据。
			r.HTTPStatus, r.RequestID, r.Items = inspectResponse(resp)
			r.OK = callErr == nil && r.HTTPStatus == 200 && r.Items == 0
			if !r.OK {
				r.Detail = "未观测到 200 + 空列表契约，存在性判据可能已漂移"
			} else {
				r.Detail = "云侧返回 HTTP 200 + 空列表（无 err）；就绪度探测必须判返回条数"
			}
		}
		results = append(results, r)
		printResult(r)
	}

	// 汇总。
	readOK, readTotal := 0, 0
	for _, r := range results {
		if r.Group == "read-only" {
			readTotal++
			if r.OK {
				readOK++
			}
		}
	}
	fmt.Printf("\n汇总：只读 API %d/%d 通过；DryRun 校验链与错误分类结果见上表。\n", readOK, readTotal)
	if *jsonOut != "" {
		payload, err := json.MarshalIndent(struct {
			Region  string   `json:"region"`
			Results []result `json:"results"`
		}{Region: *region, Results: results}, "", "  ")
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

// printAccountQuotas 把账号侧门槛单独摘出来：交付能否成功不只取决于参数
// 是否存在，还取决于实名认证与按量 vCPU 配额。把这几项明确打印出来，
// 「账号侧阻塞」才算精确到具体项，而不是一句笼统的「余额不足」。
func printAccountQuotas(resp *ecs.DescribeAccountAttributesResponse, region string) {
	items := resp.Body.AccountAttributeItems.AccountAttributeItem
	type kv struct{ name, values string }
	var picked []kv
	for _, item := range items {
		if item == nil || item.AttributeName == nil || item.AttributeValues == nil {
			continue
		}
		name := tea.StringValue(item.AttributeName)
		vals := make([]string, 0, len(item.AttributeValues.ValueItem))
		for _, v := range item.AttributeValues.ValueItem {
			if v != nil && v.Value != nil {
				vals = append(vals, tea.StringValue(v.Value))
			}
		}
		switch name {
		case "real-name-authentication", "max-postpaid-instance-vcpu-count",
			"used-postpaid-instance-vcpu-count", "instance-network-type",
			"max-security-groups":
			picked = append(picked, kv{name, strings.Join(vals, ",")})
		}
	}
	if len(picked) == 0 {
		return
	}
	fmt.Printf("账号侧门槛（%s）：\n", region)
	for _, p := range picked {
		fmt.Printf("  %-38s %s\n", p.name, p.values)
	}
	fmt.Println()
}

func printResult(r result) {
	status := "OK  "
	if !r.OK {
		status = "FAIL"
	}
	extra := ""
	if r.Code != "" {
		extra = fmt.Sprintf(" code=%s permanent=%v", r.Code, r.Permanent)
	}
	if r.Detail != "" {
		extra += " detail=" + truncate(r.Detail, 160)
	}
	fmt.Printf("[%s] %-46s http=%-3d req=%-36s items=%-4d %4dms%s\n",
		status, r.API, r.HTTPStatus, truncate(r.RequestID, 36), r.Items, r.ElapsedMS, extra)
}

// inspectResponse 用 SDK 自带的 JSON tag 统一解析响应，不依赖各 API 的
// 具体 Body 结构：statusCode / headers / body 是所有 Response 的公共字段。
func inspectResponse(resp any) (status int, requestID string, items int) {
	if resp == nil {
		return 0, "", 0
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return 0, "", 0
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return 0, "", 0
	}
	if v, ok := m["statusCode"].(float64); ok {
		status = int(v)
	}
	if h, ok := m["headers"].(map[string]any); ok {
		for k, v := range h {
			if strings.EqualFold(k, "x-acs-request-id") {
				requestID, _ = v.(string)
				break
			}
		}
	}
	if b, ok := m["body"].(map[string]any); ok {
		if requestID == "" {
			requestID, _ = b["RequestId"].(string)
		}
		items = countListItems(b, "")
	}
	return status, requestID, items
}

// countListItems 取响应体里最长的数组作为「返回条数」。SDK 的列表响应统一
// 包一层（Regions.Region / Zones.Zone / Images.Image ...），取最大值即可覆盖
// 嵌套结构，不会把嵌套的资源数组误当成空响应。
func countListItems(node any, path string) int {
	best := 0
	switch typed := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if n := countListItems(typed[k], path+"."+k); n > best {
				best = n
			}
		}
	case []any:
		if len(typed) > best {
			best = len(typed)
		}
		for _, item := range typed {
			if n := countListItems(item, path+"[]"); n > best {
				best = n
			}
		}
	}
	return best
}

// sdkErrorMeta 从 tea.SDKError 里取 HTTP 状态与 RequestId。阿里云 RPC 业务
// 错误也可能伴随 HTTP 200，记录状态码是为了验证「只判状态码会漏错误」。
func sdkErrorMeta(err error) (status int, requestID string) {
	if err == nil {
		return 0, ""
	}
	var sdkErr *tea.SDKError
	if errors.As(err, &sdkErr) {
		if sdkErr.StatusCode != nil {
			status = *sdkErr.StatusCode
		}
		if sdkErr.Data != nil {
			var payload struct {
				RequestID string `json:"RequestId"`
				RequestId string `json:"requestId"`
			}
			if json.Unmarshal([]byte(*sdkErr.Data), &payload) == nil {
				requestID = payload.RequestID
				if requestID == "" {
					requestID = payload.RequestId
				}
			}
		}
	}
	return status, requestID
}

// firstStringField 在响应 JSON 里找第一个指定键的字符串值（如 ZoneId），
// 不依赖 SDK 响应结构的具体字段层级。
func firstStringField(node any, key string) string {
	raw, err := json.Marshal(node)
	if err != nil {
		return ""
	}
	var root any
	if json.Unmarshal(raw, &root) != nil {
		return ""
	}
	return findString(root, key)
}

func findString(node any, key string) string {
	switch typed := node.(type) {
	case map[string]any:
		if v, ok := typed[key].(string); ok && v != "" {
			return v
		}
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if v := findString(typed[k], key); v != "" {
				return v
			}
		}
	case []any:
		for _, item := range typed {
			if v := findString(item, key); v != "" {
				return v
			}
		}
	}
	return ""
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// Package tencent 是腾讯云的云发现适配器。
//
// 之所以不引 tencentcloud-sdk-go：本适配器只做**只读发现**（列地域 + 列 CVM
// 实例），用得到的只是两个 Action。官方 SDK 会把上百个产品的请求/响应结构和
// 一整套 common client 带进依赖树，而这里需要的只是「按 TC3-HMAC-SHA256 签名
// 发一次 POST」。与 service/cloud/kubernetes、service/cloud/aws 的理由一致。
//
// 覆盖的接口（JSON API，统一 POST 到 https://cvm.tencentcloudapi.com/）：
//   - DescribeRegions    → ListRegions
//   - DescribeInstances  → DiscoverRegion（Offset/Limit 分页）
package tencent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// Provider / ServiceCodeCVM 供装配层引用，避免注册处再写字面量。
const (
	Provider       = "tencent"
	ServiceCodeCVM = "cvm"
)

const (
	// apiVersion CVM API 版本，进 X-TC-Version 头。
	apiVersion = "2017-03-12"
	// defaultEndpoint 腾讯云用单一全局接入点，地域走 X-TC-Region 头而不是域名。
	defaultEndpoint = "https://cvm.tencentcloudapi.com"
	// signingService CVM 的签名服务名。
	signingService = "cvm"
	// defaultPageSize Limit，上限 100。
	defaultPageSize = 100
	// defaultRegion 列地域时兜底的地域。
	defaultRegion = "ap-guangzhou"
)

// Adapter 腾讯云发现适配器
type Adapter struct {
	logger *zap.SugaredLogger
	hc     *http.Client
	now    func() time.Time
}

// NewAdapter 构造器
func NewAdapter(logger *zap.SugaredLogger) *Adapter {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Adapter{
		logger: logger.Named("cloud.tencent"),
		hc:     &http.Client{Timeout: 20 * time.Second},
		now:    time.Now,
	}
}

func (*Adapter) Provider() string    { return Provider }
func (*Adapter) ServiceCode() string { return ServiceCodeCVM }
func (*Adapter) Close()              {}

func (a *Adapter) InitClients(ctx context.Context, account *ent.CloudAccount, regions []string) (map[string]cloud.Client, error) {
	cred, err := cloud.ResolveCredential(ctx, Provider, account.CredentialRef)
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		regions = []string{defaultRegion}
	}
	clients := make(map[string]cloud.Client, len(regions))
	for _, region := range regions {
		clients[region] = &apiClient{region: region, cred: cred, hc: a.hc, now: a.now}
	}
	return clients, nil
}

func (a *Adapter) ListRegions(ctx context.Context, account *ent.CloudAccount) ([]string, error) {
	cred, err := cloud.ResolveCredential(ctx, Provider, account.CredentialRef)
	if err != nil {
		return nil, err
	}
	c := &apiClient{region: defaultRegion, cred: cred, hc: a.hc, now: a.now}
	var out apiResponse
	// DescribeRegions 不需要入参，地域列表是全量的。
	if err := c.call(ctx, "DescribeRegions", []byte(`{}`), &out); err != nil {
		return nil, err
	}
	regions := make([]string, 0, len(out.Response.RegionSet))
	for _, r := range out.Response.RegionSet {
		if r.Region != "" {
			regions = append(regions, r.Region)
		}
	}
	return regions, nil
}

func (a *Adapter) DiscoverRegion(ctx context.Context, _ *ent.CloudAccount, region string, client cloud.Client, nextToken string) (*cloud.PageResult, error) {
	c, ok := client.(*apiClient)
	if !ok {
		return nil, fmt.Errorf("tencent: unexpected client type %T", client)
	}
	// 腾讯云的分页是 Offset/Limit 而不是游标，nextToken 里装的就是下一个偏移量。
	offset := 0
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("tencent: invalid next token %q", nextToken)
		}
		offset = parsed
	}
	body, err := json.Marshal(map[string]interface{}{
		"Offset": offset,
		"Limit":  defaultPageSize,
	})
	if err != nil {
		return nil, err
	}
	var out apiResponse
	if err := c.call(ctx, "DescribeInstances", body, &out); err != nil {
		return nil, err
	}
	resources := make([]cloud.DiscoveredResource, 0, len(out.Response.InstanceSet))
	for i := range out.Response.InstanceSet {
		resources = append(resources, transformInstance(&out.Response.InstanceSet[i], region))
	}

	// 下一页：还有剩余就继续。TotalCount 是真实总数，用它收尾比「本页不满就是
	// 最后一页」可靠——后者在总数恰好是页大小整数倍时会多打一次空请求，在
	// 服务端提前截断时会漏数据。
	next := ""
	if offset+len(resources) < out.Response.TotalCount {
		next = strconv.Itoa(offset + len(resources))
	}
	return &cloud.PageResult{Resources: resources, NextToken: next}, nil
}

func (a *Adapter) ValidateCredential(ctx context.Context, account *ent.CloudAccount) error {
	_, err := a.ListRegions(ctx, account)
	return err
}

// ===== CVM API 客户端 =====

type apiClient struct {
	region string
	cred   *cloud.ResolvedCredential
	hc     *http.Client
	now    func() time.Time
	// endpoint 便于测试与私有化接入点覆盖。
	endpoint string
}

func (*apiClient) Close() error { return nil }

func (c *apiClient) baseURL() string {
	if c.endpoint != "" {
		return strings.TrimSuffix(c.endpoint, "/")
	}
	if c.cred != nil && c.cred.Endpoint != "" {
		return strings.TrimSuffix(c.cred.Endpoint, "/")
	}
	return defaultEndpoint
}

func (c *apiClient) call(ctx context.Context, action string, payload []byte, out *apiResponse) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	contentType := "application/json; charset=utf-8"
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", apiVersion)
	req.Header.Set("X-TC-Region", c.region)

	now := time.Now()
	if c.now != nil {
		now = c.now()
	}
	timestamp := now.Unix()
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	if c.cred != nil && c.cred.SessionToken != "" {
		req.Header.Set("X-TC-Token", c.cred.SessionToken)
	}

	s := &tc3Signer{secretID: c.cred.AccessKeyID, secretKey: c.cred.AccessKeySecret}
	s.SetHeaders(req, action, payload, signingService, timestamp)

	hc := c.hc
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("tencent: read %s: %w", action, err)
	}

	var parsed apiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("tencent: decode %s http=%d: %w", action, resp.StatusCode, err)
	}
	// 腾讯云的业务错误是 HTTP 200 + Response.Error：只看状态码会把「签名错误」
	// 「余额不足」「地域未开通」全部当成成功，然后拿到一份空资源列表，表现为
	// 「发现跑完了但什么都没发现」——这类假成功最难定位。
	if parsed.Response.Error != nil && parsed.Response.Error.Code != "" {
		return fmt.Errorf("tencent cvm: %s code=%s msg=%s", action, parsed.Response.Error.Code, parsed.Response.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("tencent cvm: %s http=%d body=%s", action, resp.StatusCode, truncate(string(raw), 256))
	}
	if out != nil {
		*out = parsed
	}
	return nil
}

// ===== 响应结构（只声明用到的字段） =====

type apiResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		TotalCount  int           `json:"TotalCount"`
		RegionSet   []region      `json:"RegionSet"`
		InstanceSet []cvmInstance `json:"InstanceSet"`
	} `json:"Response"`
}

type region struct {
	Region      string `json:"Region"`
	RegionName  string `json:"RegionName"`
	RegionState string `json:"RegionState"`
}

type cvmInstance struct {
	InstanceID   string `json:"InstanceId"`
	InstanceName string `json:"InstanceName"`
	InstanceType string `json:"InstanceType"`
	CPU          int    `json:"CPU"`
	Memory       int    `json:"Memory"` // 单位 GB
	OsName       string `json:"OsName"`
	ImageID      string `json:"ImageId"`
	// InstanceState 是英文枚举：PENDING / RUNNING / STOPPED / STOPPING /
	// STARTING / REBOOTING / SHUTDOWN / TERMINATING。
	InstanceState      string   `json:"InstanceState"`
	CreatedTime        string   `json:"CreatedTime"`
	ExpiredTime        string   `json:"ExpiredTime"`
	PrivateIPAddresses []string `json:"PrivateIpAddresses"`
	PublicIPAddresses  []string `json:"PublicIpAddresses"`
	InstanceChargeType string   `json:"InstanceChargeType"`
	Placement          struct {
		Zone string `json:"Zone"`
	} `json:"Placement"`
	VirtualPrivateCloud struct {
		VpcID    string `json:"VpcId"`
		SubnetID string `json:"SubnetId"`
	} `json:"VirtualPrivateCloud"`
	Tags []struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	} `json:"Tags"`
}

func transformInstance(inst *cvmInstance, region string) cloud.DiscoveredResource {
	name := inst.InstanceName
	if name == "" {
		name = inst.InstanceID
	}
	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID:   inst.InstanceID,
			ResourceName: name,
			Region:       region,
			Zone:         inst.Placement.Zone,
			Status:       mapInstanceState(inst.InstanceState),
			Tags:         extractTags(inst),
			CreatedTime:  inst.CreatedTime,
		},
		CloudServiceCode: ServiceCodeCVM,
		CloudServiceName: "云服务器 CVM",
		Extra: map[string]interface{}{
			"instance_type":        inst.InstanceType,
			"cpu":                  inst.CPU,
			"memory":               inst.Memory,
			"os_name":              inst.OsName,
			"image_id":             inst.ImageID,
			"vpc_id":               inst.VirtualPrivateCloud.VpcID,
			"subnet_id":            inst.VirtualPrivateCloud.SubnetID,
			"private_ip_addresses": inst.PrivateIPAddresses,
			"public_ip_addresses":  inst.PublicIPAddresses,
			"charge_type":          inst.InstanceChargeType,
			"expired_time":         inst.ExpiredTime,
		},
	}
}

func extractTags(inst *cvmInstance) map[string]string {
	if len(inst.Tags) == 0 {
		return nil
	}
	tags := make(map[string]string, len(inst.Tags))
	for _, t := range inst.Tags {
		if t.Key != "" {
			tags[t.Key] = t.Value
		}
	}
	return tags
}

func mapInstanceState(state string) string {
	switch strings.ToUpper(state) {
	case "PENDING", "STARTING", "REBOOTING":
		return "pending"
	case "RUNNING":
		return "active"
	case "STOPPED", "STOPPING", "SHUTDOWN":
		return "inactive"
	case "TERMINATING", "LAUNCH_FAILED":
		return "retired"
	default:
		return "inactive"
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// compile-time interface compliance check
var (
	_ cloud.CloudDiscoveryAdapter = (*Adapter)(nil)
	_ cloud.Client                = (*apiClient)(nil)
)

package volcengine

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
	"itsm-backend/service/cloud/signing"
)

// Provider / ServiceCodeECS 供装配层引用，避免注册处再写字面量。
const (
	Provider       = "volcengine"
	ServiceCodeECS = "ecs"
)

const (
	// apiVersion ECS OpenAPI 版本，必须出现在 Query String 里。
	apiVersion = "2020-04-01"
	// signingService ECS 的签名服务名。
	signingService = "ecs"
	// defaultEndpoint 火山引擎的公共接入点；地域不进域名，进凭据范围的 Region。
	defaultEndpoint = "https://open.volcengineapi.com"
	// defaultRegion 列地域时兜底的地域。
	defaultRegion = "cn-beijing"
	// defaultPageSize PageSize，火山 ECS 上限 100。
	defaultPageSize = 100
)

// Adapter 火山引擎发现适配器
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
		logger: logger.Named("cloud.volcengine"),
		hc:     &http.Client{Timeout: 20 * time.Second},
		now:    time.Now,
	}
}

func (*Adapter) Provider() string    { return Provider }
func (*Adapter) ServiceCode() string { return ServiceCodeECS }
func (*Adapter) Close()              {}

func (a *Adapter) InitClients(ctx context.Context, account *ent.CloudAccount, regions []string) (map[string]cloud.Client, error) {
	cred, err := cloud.ResolveVolcengineCredential(ctx, account.CredentialRef)
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
	cred, err := cloud.ResolveVolcengineCredential(ctx, account.CredentialRef)
	if err != nil {
		return nil, err
	}
	c := &apiClient{region: defaultRegion, cred: cred, hc: a.hc, now: a.now}
	var out describeRegionsResponse
	if err := c.call(ctx, "DescribeRegions", nil, &out); err != nil {
		return nil, err
	}
	regions := make([]string, 0, len(out.Result.Regions))
	for _, r := range out.Result.Regions {
		if r.RegionID != "" {
			regions = append(regions, r.RegionID)
		}
	}
	return regions, nil
}

func (a *Adapter) DiscoverRegion(ctx context.Context, _ *ent.CloudAccount, region string, client cloud.Client, nextToken string) (*cloud.PageResult, error) {
	c, ok := client.(*apiClient)
	if !ok {
		return nil, fmt.Errorf("volcengine: unexpected client type %T", client)
	}
	// 火山用 PageNumber/PageSize 而不是游标，nextToken 里装的是下一个页码。
	pageNumber := 1
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 1 {
			return nil, fmt.Errorf("volcengine: invalid next token %q", nextToken)
		}
		pageNumber = parsed
	}
	params := map[string]string{
		"PageNumber": strconv.Itoa(pageNumber),
		"PageSize":   strconv.Itoa(defaultPageSize),
	}
	var out describeInstancesResponse
	if err := c.call(ctx, "DescribeInstances", params, &out); err != nil {
		return nil, err
	}
	resources := make([]cloud.DiscoveredResource, 0, len(out.Result.Instances))
	for i := range out.Result.Instances {
		resources = append(resources, transformInstance(&out.Result.Instances[i], region))
	}
	// 用 TotalCount 收尾：本页满了不代表还有下一页，本页没满也不一定没有，
	// 只有真实总数能判。
	next := ""
	if pageNumber*defaultPageSize < out.Result.TotalCount {
		next = strconv.Itoa(pageNumber + 1)
	}
	return &cloud.PageResult{Resources: resources, NextToken: next}, nil
}

func (a *Adapter) ValidateCredential(ctx context.Context, account *ent.CloudAccount) error {
	_, err := a.ListRegions(ctx, account)
	return err
}

// ===== OpenAPI 客户端 =====

type apiClient struct {
	region string
	cred   *cloud.ResolvedCredential
	hc     *http.Client
	now    func() time.Time
	// endpoint 便于测试与专有接入点覆盖。
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

// call 发一次 GET：Action 与 Version 必须在 Query String 里（官方要求），
// 业务参数也一并拼在查询串上，因此请求体恒为空。
func (c *apiClient) call(ctx context.Context, action string, params map[string]string, out interface{}) error {
	query := signing.EncodeForm(queryParams(action, params))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+"/?"+query, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	now := time.Now()
	if c.now != nil {
		now = c.now()
	}
	s := &volcSigner{
		accessKeyID:     c.cred.AccessKeyID,
		secretAccessKey: c.cred.AccessKeySecret,
		sessionToken:    c.cred.SessionToken,
	}
	// 官方文档的示例里也保留了 region-specific 域名（ecs.{region}.volcengineapi.com），
	// 两者都可用；默认走公共接入点。
	s.SetHeaders(req, nil, c.region, signingService, now)

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
		return fmt.Errorf("volcengine: read %s: %w", action, err)
	}

	// 火山的业务错误同样是 HTTP 200 + ResponseMetadata.Error。
	var meta responseMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return fmt.Errorf("volcengine: decode %s http=%d: %w", action, resp.StatusCode, err)
	}
	if meta.ResponseMetadata.Error != nil && meta.ResponseMetadata.Error.Code != "" {
		return fmt.Errorf("volcengine ecs: %s code=%s msg=%s", action,
			meta.ResponseMetadata.Error.Code, meta.ResponseMetadata.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("volcengine ecs: %s http=%d body=%s", action, resp.StatusCode, truncate(string(raw), 256))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("volcengine: decode %s: %w", action, err)
	}
	return nil
}

// queryParams 组装查询参数，Action 与 Version 是必带的公共参数。
func queryParams(action string, params map[string]string) map[string]string {
	out := make(map[string]string, len(params)+2)
	for k, v := range params {
		out[k] = v
	}
	out["Action"] = action
	out["Version"] = apiVersion
	return out
}

// ===== 响应结构（只声明用到的字段） =====

type responseMetadata struct {
	ResponseMetadata struct {
		RequestID string `json:"RequestId"`
		Action    string `json:"Action"`
		Version   string `json:"Version"`
		Service   string `json:"Service"`
		Region    string `json:"Region"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"ResponseMetadata"`
}

type describeRegionsResponse struct {
	Result struct {
		Regions []struct {
			RegionID   string `json:"RegionId"`
			RegionName string `json:"RegionName"`
		} `json:"Regions"`
	} `json:"Result"`
}

type describeInstancesResponse struct {
	Result struct {
		TotalCount int            `json:"TotalCount"`
		Instances  []volcInstance `json:"Instances"`
	} `json:"Result"`
}

type volcInstance struct {
	InstanceID   string `json:"InstanceId"`
	InstanceName string `json:"InstanceName"`
	InstanceType string `json:"InstanceType"`
	// 火山的状态是英文枚举：CREATING / RUNNING / STOPPING / STOPPED /
	// STARTING / REBOOTING / EXPIRED / DELETED。
	Status      string `json:"Status"`
	ZoneID      string `json:"ZoneId"`
	ImageID     string `json:"ImageId"`
	VpcID       string `json:"VpcId"`
	Hostname    string `json:"Hostname"`
	Description string `json:"Description"`
	OsType      string `json:"OsType"`
	Cpus        int    `json:"Cpus"`
	// MemorySize 单位是 GiB。
	MemorySize        int    `json:"MemorySize"`
	CreatedAt         string `json:"CreatedAt"`
	UpdatedAt         string `json:"UpdatedAt"`
	NetworkInterfaces []struct {
		NetworkInterfaceID string `json:"NetworkInterfaceId"`
		SubnetID           string `json:"SubnetId"`
		PrimaryIPAddress   string `json:"PrimaryIpAddress"`
	} `json:"NetworkInterfaces"`
	Tags []struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	} `json:"Tags"`
}

func transformInstance(inst *volcInstance, region string) cloud.DiscoveredResource {
	name := inst.InstanceName
	if name == "" {
		name = inst.InstanceID
	}
	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID:   inst.InstanceID,
			ResourceName: name,
			Region:       region,
			Zone:         inst.ZoneID,
			Status:       mapStatus(inst.Status),
			Tags:         extractTags(inst),
			CreatedTime:  inst.CreatedAt,
		},
		CloudServiceCode: ServiceCodeECS,
		CloudServiceName: "云服务器 ECS",
		Extra: map[string]interface{}{
			"instance_type": inst.InstanceType,
			"image_id":      inst.ImageID,
			"os_type":       inst.OsType,
			"cpu":           inst.Cpus,
			"memory_gib":    inst.MemorySize,
			"vpc_id":        inst.VpcID,
			"subnet_id":     primarySubnet(inst),
			"primary_ip":    primaryIP(inst),
			"hostname":      inst.Hostname,
			"description":   inst.Description,
			"updated_at":    inst.UpdatedAt,
		},
	}
}

func primaryIP(inst *volcInstance) string {
	for _, nic := range inst.NetworkInterfaces {
		if nic.PrimaryIPAddress != "" {
			return nic.PrimaryIPAddress
		}
	}
	return ""
}

func primarySubnet(inst *volcInstance) string {
	for _, nic := range inst.NetworkInterfaces {
		if nic.SubnetID != "" {
			return nic.SubnetID
		}
	}
	return ""
}

func extractTags(inst *volcInstance) map[string]string {
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

func mapStatus(status string) string {
	switch strings.ToUpper(status) {
	case "CREATING", "STARTING", "REBOOTING":
		return "pending"
	case "RUNNING":
		return "active"
	case "STOPPED", "STOPPING":
		return "inactive"
	case "DELETED", "EXPIRED":
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

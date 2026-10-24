// Package aws 是 AWS 的云发现适配器。
//
// 之所以不引 aws-sdk-go-v2：本适配器只做**只读发现**（列 region + 列 EC2
// 实例），用得到的只是 EC2 Query API 的两个 Action。aws-sdk-go-v2 会带进
// 一整套 endpoint 解析、重试、凭证链、序列化框架和数十个间接依赖，而这里需要
// 的只是「按 SigV4 签名发一次 POST」。边界小、可测（httptest 即可断言签名与
// 解析），与 service/cloud/kubernetes 的理由一致。
//
// 覆盖的接口（EC2 Query API，响应是 XML）：
//   - DescribeRegions    → ListRegions
//   - DescribeInstances  → DiscoverRegion（NextToken 分页）
package aws

import (
	"context"
	"encoding/xml"
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

// Provider / ServiceCodeEC2 供装配层引用，避免注册处再写字面量。
const (
	Provider       = "aws"
	ServiceCodeEC2 = "ec2"
)

const (
	// apiVersion EC2 Query API 版本，签名与解析都按这个版本。
	apiVersion = "2016-11-15"
	// defaultPageSize DescribeInstances 的 MaxResults，上限 1000，取 100 与
	// 其余厂商适配器的页大小保持一致。
	defaultPageSize = 100
	// defaultRegion 列 region 时用的地域：DescribeRegions 在任一地域都能返回
	// 全量 region，但总得挑一个地域发请求。
	defaultRegion = "us-east-1"
	// signingService EC2 的签名服务名（不是 "aws"，写错就是 403）。
	signingService = "ec2"
)

// Adapter AWS 发现适配器
type Adapter struct {
	logger *zap.SugaredLogger
	hc     *http.Client
	// now 注入时钟，测试里固定时间才能断言签名。
	now func() time.Time
}

// NewAdapter 构造器
func NewAdapter(logger *zap.SugaredLogger) *Adapter {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Adapter{
		logger: logger.Named("cloud.aws"),
		hc:     &http.Client{Timeout: 20 * time.Second},
		now:    time.Now,
	}
}

func (*Adapter) Provider() string    { return Provider }
func (*Adapter) ServiceCode() string { return ServiceCodeEC2 }
func (*Adapter) Close()              {}

func (a *Adapter) InitClients(ctx context.Context, account *ent.CloudAccount, regions []string) (map[string]cloud.Client, error) {
	cred, err := cloud.ResolveAWSCredential(ctx, account.CredentialRef)
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
	cred, err := cloud.ResolveAWSCredential(ctx, account.CredentialRef)
	if err != nil {
		return nil, err
	}
	c := &apiClient{region: defaultRegion, cred: cred, hc: a.hc, now: a.now}
	var out describeRegionsResponse
	if err := c.call(ctx, "DescribeRegions", nil, &out); err != nil {
		return nil, err
	}
	regions := make([]string, 0, len(out.RegionInfo.Items))
	for _, item := range out.RegionInfo.Items {
		if item.RegionName != "" {
			regions = append(regions, item.RegionName)
		}
	}
	return regions, nil
}

func (a *Adapter) DiscoverRegion(ctx context.Context, _ *ent.CloudAccount, region string, client cloud.Client, nextToken string) (*cloud.PageResult, error) {
	c, ok := client.(*apiClient)
	if !ok {
		return nil, fmt.Errorf("aws: unexpected client type %T", client)
	}
	params := map[string]string{"MaxResults": strconv.Itoa(defaultPageSize)}
	if nextToken != "" {
		params["NextToken"] = nextToken
	}
	var out describeInstancesResponse
	if err := c.call(ctx, "DescribeInstances", params, &out); err != nil {
		return nil, err
	}
	resources := make([]cloud.DiscoveredResource, 0, defaultPageSize)
	for _, r := range out.ReservationSet.Items {
		for i := range r.InstancesSet.Items {
			resources = append(resources, transformInstance(&r.InstancesSet.Items[i], region))
		}
	}
	return &cloud.PageResult{Resources: resources, NextToken: out.NextToken}, nil
}

func (a *Adapter) ValidateCredential(ctx context.Context, account *ent.CloudAccount) error {
	_, err := a.ListRegions(ctx, account)
	return err
}

// ===== EC2 Query API 客户端 =====

type apiClient struct {
	region string
	cred   *cloud.ResolvedCredential
	hc     *http.Client
	now    func() time.Time
}

func (*apiClient) Close() error { return nil }

// baseURL 推导地域端点。
//
// 中国区（cn-north-1 / cn-northwest-1）的域名后缀是 amazonaws.com.cn，与其他
// 地域不同——写死 .com 会让中国区账号全部解析失败。凭据里显式给了 endpoint 时
// 优先用（VPC endpoint / 本地栈等特殊部署）。
func (c *apiClient) baseURL() string {
	if c.cred != nil && c.cred.Endpoint != "" {
		return strings.TrimSuffix(c.cred.Endpoint, "/")
	}
	suffix := "amazonaws.com"
	if strings.HasPrefix(c.region, "cn-") {
		suffix = "amazonaws.com.cn"
	}
	return "https://ec2." + c.region + "." + suffix
}

func (c *apiClient) call(ctx context.Context, action string, params map[string]string, out interface{}) error {
	body := signing.EncodeForm(queryParams(action, params))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/", strings.NewReader(body))
	if err != nil {
		return err
	}
	contentType := "application/x-www-form-urlencoded; charset=utf-8"
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/xml")

	s := &sigv4Signer{
		accessKeyID:     c.cred.AccessKeyID,
		secretAccessKey: c.cred.AccessKeySecret,
		sessionToken:    c.cred.SessionToken,
	}
	now := time.Now()
	if c.now != nil {
		now = c.now()
	}
	// 表单体本身就是签名的一部分，所以 canonical query 留空、参数只出现在 body。
	s.SetHeaders(req, [][2]string{{"content-type", contentType}}, []byte(body), c.region, signingService, now)

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
		return fmt.Errorf("aws: read %s: %w", action, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// EC2 的错误体是 XML，Code 才是判断依据（AuthFailure / UnauthorizedOperation
		// / OptInRequired 的处置完全不同），只报状态码等于让运维重新猜一遍。
		var er ec2ErrorResponse
		_ = xml.Unmarshal(raw, &er)
		if er.Errors.Error.Code != "" {
			return fmt.Errorf("aws ec2: %s http=%d code=%s msg=%s", action, resp.StatusCode, er.Errors.Error.Code, er.Errors.Error.Message)
		}
		return fmt.Errorf("aws ec2: %s http=%d body=%s", action, resp.StatusCode, truncate(string(raw), 256))
	}
	if out == nil {
		return nil
	}
	if err := xml.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("aws ec2: decode %s: %w", action, err)
	}
	return nil
}

// queryParams 组装请求参数，Action 与 Version 是每个请求必带的公共参数。
func queryParams(action string, params map[string]string) map[string]string {
	out := make(map[string]string, len(params)+2)
	for k, v := range params {
		out[k] = v
	}
	out["Action"] = action
	out["Version"] = apiVersion
	return out
}

// ===== EC2 响应结构（只声明用到的字段） =====

type ec2ErrorResponse struct {
	Errors struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	} `xml:"Errors"`
}

type describeRegionsResponse struct {
	RegionInfo struct {
		Items []struct {
			RegionName  string `xml:"regionName"`
			Endpoint    string `xml:"endpoint"`
			OptInStatus string `xml:"optInStatus"`
		} `xml:"item"`
	} `xml:"regionInfo"`
	NextToken string `xml:"nextToken"`
}

type describeInstancesResponse struct {
	ReservationSet struct {
		Items []struct {
			InstancesSet struct {
				Items []ec2Instance `xml:"item"`
			} `xml:"instancesSet"`
		} `xml:"item"`
	} `xml:"reservationSet"`
	NextToken string `xml:"nextToken"`
}

type ec2Instance struct {
	InstanceID       string `xml:"instanceId"`
	ImageID          string `xml:"imageId"`
	InstanceType     string `xml:"instanceType"`
	KeyName          string `xml:"keyName"`
	PrivateIPAddress string `xml:"privateIpAddress"`
	PublicIPAddress  string `xml:"publicIpAddress"`
	VpcID            string `xml:"vpcId"`
	SubnetID         string `xml:"subnetId"`
	Architecture     string `xml:"architecture"`
	LaunchTime       string `xml:"launchTime"`
	InstanceState    struct {
		Code int    `xml:"code"`
		Name string `xml:"name"`
	} `xml:"instanceState"`
	Placement struct {
		AvailabilityZone string `xml:"availabilityZone"`
	} `xml:"placement"`
	TagSet struct {
		Items []struct {
			Key   string `xml:"key"`
			Value string `xml:"value"`
		} `xml:"item"`
	} `xml:"tagSet"`
}

func transformInstance(inst *ec2Instance, region string) cloud.DiscoveredResource {
	// EC2 的响应里没有「实例名」字段，控制台显示的是 Name 标签——这里沿用同一
	// 口径，取不到就回落实例 ID，避免 CI 表里一片空白。
	name := ""
	tags := make(map[string]string, len(inst.TagSet.Items))
	for _, tag := range inst.TagSet.Items {
		if tag.Key == "" {
			continue
		}
		tags[tag.Key] = tag.Value
		if tag.Key == "Name" {
			name = tag.Value
		}
	}
	if name == "" {
		name = inst.InstanceID
	}
	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID:   inst.InstanceID,
			ResourceName: name,
			Region:       region,
			Zone:         inst.Placement.AvailabilityZone,
			Status:       mapInstanceState(inst.InstanceState.Name),
			Tags:         tags,
			CreatedTime:  inst.LaunchTime,
		},
		CloudServiceCode: ServiceCodeEC2,
		CloudServiceName: "EC2 实例",
		Extra: map[string]interface{}{
			"instance_type":     inst.InstanceType,
			"image_id":          inst.ImageID,
			"architecture":      inst.Architecture,
			"key_name":          inst.KeyName,
			"vpc_id":            inst.VpcID,
			"subnet_id":         inst.SubnetID,
			"private_ip":        inst.PrivateIPAddress,
			"public_ip":         inst.PublicIPAddress,
			"state_code":        inst.InstanceState.Code,
			"availability_zone": inst.Placement.AvailabilityZone,
		},
	}
}

// mapInstanceState 对齐其余厂商适配器的状态口径：
// pending / active / inactive / retired。
func mapInstanceState(state string) string {
	switch strings.ToLower(state) {
	case "pending":
		return "pending"
	case "running":
		return "active"
	case "stopping", "stopped", "shutting-down":
		return "inactive"
	case "terminated":
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

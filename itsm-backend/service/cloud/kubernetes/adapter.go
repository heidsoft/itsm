// Package kubernetes 是 Kubernetes 集群的云发现适配器。
//
// 之所以不引 client-go：本适配器只做**只读发现**（列命名空间 + 列 Pod），
// 用得到的是 APIServer 的 REST 接口，而不是 client-go 的 Informer / 缓存 /
// 本地 kubeconfig 加载那一整套。引 client-go 会带进 ~40 个间接依赖和一套
// 与本产品隔离模型无关的缓存生命周期，收益却是零。这里用 net/http +
// Bearer token 直接调 APIServer，边界更小、也更好测（httptest 即可断言）。
//
// 覆盖的接口：
//   - GET  /api/v1/namespaces                        → ListRegions（把 namespace 当作 region）
//   - GET  /api/v1/namespaces/{ns}/pods[?continue=]  → DiscoverRegion（分页用 continue）
//
// region 语义映射说明：云厂商的 region 是地域，K8s 没有地域概念，与之最接近
// 的**隔离边界**是 namespace——发现任务按 namespace 分片、失败可单 namespace
// 重试，语义上和「按 region 分片」一致，所以这里复用 region 字段承载 namespace。
package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// Provider 与注册中心归一后的厂商标识保持一致（cloud.NormalizeProvider）。
const Provider = "k8s"

// ServiceCode 本适配器发现的资源类型。
const ServiceCode = "pod"

// defaultPageSize 单次 list 请求的页大小。APIServer 的 limit 缺省是 500，
// 显式声明避免依赖服务端默认值。
const defaultPageSize = 200

// Adapter Kubernetes 发现适配器
type Adapter struct {
	logger *zap.SugaredLogger
	hc     *http.Client
}

// NewAdapter 构造器
func NewAdapter(logger *zap.SugaredLogger) *Adapter {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Adapter{
		logger: logger.Named("cloud.k8s"),
		hc:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (*Adapter) Provider() string    { return Provider }
func (*Adapter) ServiceCode() string { return ServiceCode }
func (*Adapter) Close()              {}

func (a *Adapter) InitClients(_ context.Context, account *ent.CloudAccount, regions []string) (map[string]cloud.Client, error) {
	cred, err := cloud.ResolveKubernetesCredential(context.Background(), account.CredentialRef)
	if err != nil {
		return nil, err
	}
	// K8s 的客户端不分地域：所有 namespace 共用同一个 APIServer 连接。这里
	// 仍然按 region 逐个返回，好让执行器按统一契约取用。
	clients := make(map[string]cloud.Client, len(regions))
	for _, region := range regions {
		clients[region] = &apiClient{baseURL: cred.Endpoint, token: cred.BearerToken, hc: a.hc}
	}
	if len(regions) == 0 {
		clients[""] = &apiClient{baseURL: cred.Endpoint, token: cred.BearerToken, hc: a.hc}
	}
	return clients, nil
}

// ListRegions 返回集群的 namespace 列表。
//
// 若 ServiceAccount 只有某个 namespace 的权限（namespace-scoped），列全集群
// namespace 会 403；这时退化为只发现默认 namespace，而不是让整个 job 失败。
func (a *Adapter) ListRegions(ctx context.Context, account *ent.CloudAccount) ([]string, error) {
	cred, err := cloud.ResolveKubernetesCredential(ctx, account.CredentialRef)
	if err != nil {
		return nil, err
	}
	c := &apiClient{baseURL: cred.Endpoint, token: cred.BearerToken, hc: a.hc}
	var out namespaceList
	if err := c.getJSON(ctx, "/api/v1/namespaces", &out); err != nil {
		if isForbidden(err) {
			a.logger.Warnw("无集群级 namespace 列举权限，退化为默认 namespace", "error", err)
			return []string{"default"}, nil
		}
		return nil, err
	}
	regions := make([]string, 0, len(out.Items))
	for _, ns := range out.Items {
		if ns.Metadata.Name != "" {
			regions = append(regions, ns.Metadata.Name)
		}
	}
	return regions, nil
}

func (a *Adapter) DiscoverRegion(ctx context.Context, _ *ent.CloudAccount, region string, client cloud.Client, nextToken string) (*cloud.PageResult, error) {
	c, ok := client.(*apiClient)
	if !ok {
		return nil, fmt.Errorf("k8s: unexpected client type %T", client)
	}
	path := "/api/v1/pods"
	if region != "" {
		path = "/api/v1/namespaces/" + url.PathEscape(region) + "/pods"
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	path += sep + "limit=" + strconv.Itoa(defaultPageSize)
	if nextToken != "" {
		path += "&continue=" + url.QueryEscape(nextToken)
	}

	var out podList
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	resources := make([]cloud.DiscoveredResource, 0, len(out.Items))
	for i := range out.Items {
		resources = append(resources, transformPod(&out.Items[i], region))
	}
	return &cloud.PageResult{
		Resources: resources,
		// APIServer 用 continue 表示「还有下一页」，空串即结束。
		NextToken: out.Metadata.Continue,
	}, nil
}

func (a *Adapter) ValidateCredential(ctx context.Context, account *ent.CloudAccount) error {
	cred, err := cloud.ResolveKubernetesCredential(ctx, account.CredentialRef)
	if err != nil {
		return err
	}
	c := &apiClient{baseURL: cred.Endpoint, token: cred.BearerToken, hc: a.hc}
	var out namespaceList
	return c.getJSON(ctx, "/api/v1/namespaces", &out)
}

// ===== APIServer 客户端 =====

type apiClient struct {
	baseURL string
	token   string
	hc      *http.Client
}

func (c *apiClient) Close() error { return nil }

func (c *apiClient) getJSON(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(c.baseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("k8s: read %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return &forbiddenError{path: path, body: truncate(raw, 256)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// APIServer 的错误体是 Status 结构，带上 reason 才能区分 401（token 失效）
		// 与 404（namespace 不存在）——两者处置完全不同。
		var st status
		_ = json.Unmarshal(raw, &st)
		return fmt.Errorf("k8s: GET %s http=%d reason=%s msg=%s", path, resp.StatusCode, st.Reason, st.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("k8s: decode %s: %w", path, err)
	}
	return nil
}

// ===== APIServer 响应结构（只声明用到的字段） =====

type status struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type objectMeta struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels"`
}

type namespaceList struct {
	Items []struct {
		Metadata objectMeta `json:"metadata"`
	} `json:"items"`
}

type podList struct {
	Metadata struct {
		Continue string `json:"continue"`
	} `json:"metadata"`
	Items []pod `json:"items"`
}

type pod struct {
	Metadata objectMeta `json:"metadata"`
	Spec     struct {
		NodeName string `json:"nodeName"`
	} `json:"spec"`
	Status struct {
		Phase     string `json:"phase"`
		PodIP     string `json:"podIP"`
		HostIP    string `json:"hostIP"`
		StartTime string `json:"startTime"`
	} `json:"status"`
}

func transformPod(p *pod, region string) cloud.DiscoveredResource {
	// Pod 没有稳定的「实例 ID」语义，用 UID：删除重建后 UID 必变，重建会被
	// 识别为新资源，而不是把旧 CI 原地覆盖掉。
	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID:   p.Metadata.UID,
			ResourceName: p.Metadata.Name,
			Region:       region,
			// K8s 没有可用区概念，用 nodeName 兜底承载调度位置信息。
			Zone:        p.Spec.NodeName,
			Status:      p.Status.Phase,
			Tags:        p.Metadata.Labels,
			CreatedTime: p.Metadata.CreationTimestamp,
		},
		CloudServiceCode: ServiceCode,
		CloudServiceName: "Pod",
		Extra: map[string]interface{}{
			"namespace": p.Metadata.Namespace,
			"node":      p.Spec.NodeName,
			"pod_ip":    p.Status.PodIP,
			"host_ip":   p.Status.HostIP,
		},
	}
}

// ===== 辅助 =====

type forbiddenError struct {
	path string
	body string
}

func (e *forbiddenError) Error() string {
	return fmt.Sprintf("k8s: GET %s forbidden (cluster-scoped permission missing): %s", e.path, e.body)
}

func isForbidden(err error) bool {
	var fe *forbiddenError
	return errors.As(err, &fe)
}

func truncate(raw []byte, n int) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

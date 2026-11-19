package cmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/dto"
)

// GET /api/v1/cmdb/discovery/results 的过滤参数契约。
//
// 该端点是「逐资源差异（diff 待人确认）」的读面，服务层对 jobID 的约定是
// 「<=0 表示不过滤，列出本租户全部差异」。因此 query 参数错误有两种截然不同的
// 后果，必须分开锁住：
//
//   - jobId 非法（abc / 0 / 负数）时，旧实现调用 ParsePositiveIDFromQuery 写下
//     400 并 Abort，但**丢弃了返回的 ok 继续往下跑** —— 服务层收到 jobID=0 当作
//     「不过滤」，于是把全租户差异都查了出来。且 gin 在 Abort 之后又写一次 200
//     body。客户端拿到的是「400 的语义 + 全量数据的 payload」的半截响应。
//     这与 router/cmdb_cloud_list_route_test.go 已为云资源侧修掉的是同类缺陷
//     （serviceId 非法值必须 400/1001，不得「解析失败当 0 继续查」）。
//
//   - jobId 合法但对应任务不存在时必须返回空列表而不是报错：差异读面按任务下钻，
//     任务被清理与尚未产出结果同义，都表现为空集。
//
// 参数名已硬切为 camelCase jobId；响应从裸数组改为分页信封（items/total/page/pageSize），
// 与同域 ListDiscoveryJobs 同口径。

type discoveryResultsFixture struct {
	router *gin.Engine
	// seen 记录服务层收到的 jobID 序列，用于区分「参数非法时提前中止」与
	// 「静默当 0 继续查全量」这两种实现。
	// 必须是指针：闭包与断言读的是同一个切片头，值接收者会让 append 写进副本，
	// 断言恒看到一个空切片（这本身就是一个假绿来源）。
	seen *[]int
}

func newDiscoveryResultsFixture(t *testing.T, results []*DiscoveryResult) *discoveryResultsFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	seen := make([]int, 0, 4)
	f := &discoveryResultsFixture{seen: &seen}
	repo := &mockRepository{
		listDiscoveryResultsFn: func(ctx context.Context, tenantID int, jobID int, page, pageSize int) ([]*DiscoveryResult, int, error) {
			// seen 是被闭包捕获的变量，f.seen 指向同一个变量；
			// 必须 append 到捕获变量本身（写成 *seen 会因 seen 是切片而编译失败）。
			seen = append(seen, jobID)
			if jobID <= 0 {
				return results, len(results), nil
			}
			kept := make([]*DiscoveryResult, 0, len(results))
			for _, r := range results {
				if r.JobID == jobID {
					kept = append(kept, r)
				}
			}
			return kept, len(kept), nil
		},
	}
	svc := NewService(repo, nil, zap.NewNop().Sugar())
	handler := NewHandler(svc)

	f.router = gin.New()
	group := f.router.Group("/api/v1/cmdb", withTestTenant(1))
	group.GET("/discovery/results", handler.ListDiscoveryResults)
	return f
}

func (f *discoveryResultsFixture) get(t *testing.T, url string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec
}

func (f *discoveryResultsFixture) reset() {
	*f.seen = (*f.seen)[:0]
}

// decodeDiscoveryResults 解出分页信封里的 items（data.items/total/page/pageSize）。
func decodeDiscoveryResults(t *testing.T, rec *httptest.ResponseRecorder) []dto.DiscoveryResultResponse {
	t.Helper()
	require.NotZero(t, rec.Body.Len(), "响应体不得为空")
	// 成功路径是 {code,message,data:[...]}，失败路径没有 data。
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "响应不是标准包封: %s", rec.Body.String())
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	var paged struct {
		Items    []dto.DiscoveryResultResponse `json:"items"`
		Total    int                           `json:"total"`
		Page     int                           `json:"page"`
		PageSize int                           `json:"pageSize"`
	}
	require.NoError(t, json.Unmarshal(envelope.Data, &paged), "data 不是分页信封: %s", rec.Body.String())
	return paged.Items
}

// 分页信封键必须齐全：裸数组时代 data 直接是 [...]。现改为
// {items,total,page,pageSize}，任何键缺失都会让 itsm-cli 的分页消费失效。
func TestListDiscoveryResults_ResponseIsPaginationEnvelope(t *testing.T) {
	f := newDiscoveryResultsFixture(t, []*DiscoveryResult{
		{ID: 1, JobID: 7, Action: "update", Status: "pending"},
	})
	rec := f.get(t, "/api/v1/cmdb/discovery/results?page=2&pageSize=5")
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var envelope struct {
		Data struct {
			Items    json.RawMessage `json:"items"`
			Total    int             `json:"total"`
			Page     int             `json:"page"`
			PageSize int             `json:"pageSize"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "body=%s", rec.Body.String())
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(envelope.Data.Items, &items), "items 必须是数组: %s", rec.Body.String())
	assert.Len(t, items, 1, "items 必须带出数据: %s", rec.Body.String())
	assert.Equal(t, 2, envelope.Data.Page, "page 必须回显请求值")
	assert.Equal(t, 5, envelope.Data.PageSize, "pageSize 必须回显请求值")
	assert.Equal(t, 1, envelope.Data.Total, "total 必须是计数而非省略")
}

// decodeSingleEnvelope 断言响应体是**恰好一个** JSON 文档并解出业务码。
//
// 单独抽出来的理由：旧实现在参数非法时先写 400 再继续跑完、又写了一次 200，
// 于是 body 变成 `{"code":1001,...}{"code":0,...}` 两个对象首尾相接。这种响应
// 任何按标准 JSON 解析的客户端（包括 SDK、Agent、浏览器 fetch+json()）都会失败，
// 所以「body 必须能被单次解析」本身就是一条要锁的契约，不能只断言状态码。
func decodeSingleEnvelope(t *testing.T, rec *httptest.ResponseRecorder) (int, string) {
	t.Helper()
	body := strings.TrimSpace(rec.Body.String())
	require.NotEmpty(t, body, "响应体不得为空")

	dec := json.NewDecoder(strings.NewReader(body))
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, dec.Decode(&envelope), "响应体不是合法 JSON: %s", body)
	// 再解一次必须有错：body 里不能有第二个 JSON 文档。
	// 旧实现在参数非法时先写 400、再继续跑完写 200，body 就成了
	// `{"code":1001,...}{"code":0,...}` —— 任何按标准解析的客户端都会失败。
	if err := dec.Decode(&json.RawMessage{}); err == nil {
		t.Fatalf("响应体在首个 JSON 对象之后还有多余内容（先写 400 再继续跑完又写 200）: %s", body)
	}
	return envelope.Code, envelope.Message
}

func TestListDiscoveryResults_InvalidJobIDIsBadRequestNotFullScan(t *testing.T) {
	results := []*DiscoveryResult{
		{ID: 1, JobID: 7, Action: "update", Status: "pending"},
		{ID: 2, JobID: 8, Action: "update", Status: "pending"},
	}
	f := newDiscoveryResultsFixture(t, results)

	for _, raw := range []string{"abc", "0", "-3", "1.5"} {
		t.Run("jobId="+raw, func(t *testing.T) {
			f.reset()
			rec := f.get(t, "/api/v1/cmdb/discovery/results?jobId="+raw)

			// 1) HTTP 层必须是参数错误，且不能是「先 Abort 400 再补写 200」。
			assert.Equal(t, http.StatusBadRequest, rec.Code,
				"非法 jobId 必须是 400，实际 body=%s", rec.Body.String())
			code, _ := decodeSingleEnvelope(t, rec)
			assert.Equal(t, common.ParamErrorCode, code,
				"非法 jobId 的业务码必须是 1001（请求参数错误），body=%s", rec.Body.String())

			// 2) 最关键的一条：不得把「解析失败」当成「不过滤」继续查全量。
			assert.Empty(t, *f.seen,
				"参数非法时不得再调用服务层：旧实现会把 jobID=0 当不过滤，返回全租户差异")
		})
	}
}

func TestListDiscoveryResults_ValidJobIDFiltersByJob(t *testing.T) {
	results := []*DiscoveryResult{
		{ID: 1, JobID: 7, Action: "update", Status: "pending"},
		{ID: 2, JobID: 8, Action: "update", Status: "pending"},
		{ID: 3, JobID: 7, Action: "create", Status: "pending"},
	}
	f := newDiscoveryResultsFixture(t, results)

	rec := f.get(t, "/api/v1/cmdb/discovery/results?jobId=7")
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, []int{7}, *f.seen, "合法 jobId 必须原样透传给服务层")

	items := decodeDiscoveryResults(t, rec)
	assert.Len(t, items, 2, "必须只返回该任务的差异: body=%s", rec.Body.String())
	for _, it := range items {
		assert.Equal(t, 7, it.JobID, "返回项的 jobId 必须与过滤条件一致: body=%s", rec.Body.String())
	}
}

func TestListDiscoveryResults_MissingJobIDMeansAllOfTenant(t *testing.T) {
	// 缺省过滤 = 列出本租户全部差异（服务层约定 jobID<=0 不过滤）。
	results := []*DiscoveryResult{
		{ID: 1, JobID: 7, Action: "update", Status: "pending"},
		{ID: 2, JobID: 8, Action: "update", Status: "pending"},
	}
	f := newDiscoveryResultsFixture(t, results)

	rec := f.get(t, "/api/v1/cmdb/discovery/results")
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, []int{0}, *f.seen, "缺省时服务层应收到 0（不过滤）")
	assert.Len(t, decodeDiscoveryResults(t, rec), 2, "body=%s", rec.Body.String())
}

func TestListDiscoveryResults_UnknownJobIDIsEmptyListNot404(t *testing.T) {
	f := newDiscoveryResultsFixture(t, []*DiscoveryResult{
		{ID: 1, JobID: 7, Action: "update", Status: "pending"},
	})

	rec := f.get(t, "/api/v1/cmdb/discovery/results?jobId=999")
	require.Equal(t, http.StatusOK, rec.Code,
		"任务不存在是空集而非 404：差异读面按任务下钻，任务被清理与尚未产出同义: body=%s", rec.Body.String())
	assert.Empty(t, decodeDiscoveryResults(t, rec), "body=%s", rec.Body.String())
}

func TestListDiscoveryResults_RequiresTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepository{
		listDiscoveryResultsFn: func(ctx context.Context, tenantID int, jobID int, page, pageSize int) ([]*DiscoveryResult, int, error) {
			t.Fatalf("无租户上下文时不得进入服务层（仓储会按 tenant_id 过滤，但缺上下文连租户都解析不出来）")
			return nil, 0, nil
		},
	}
	handler := NewHandler(NewService(repo, nil, zap.NewNop().Sugar()))
	r := gin.New()
	r.GET("/api/v1/cmdb/discovery/results", handler.ListDiscoveryResults)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/results?jobId=7", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
}

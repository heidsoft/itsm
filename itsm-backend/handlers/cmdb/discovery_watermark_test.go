package cmdb

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/cloud"

	_ "github.com/mattn/go-sqlite3"
)

// watermarkAdapter 按脚本分页/失败的适配器，记录每次 DiscoverRegion 收到的
// nextToken——「断点续扫必须从水位线恢复而不是从零全量」这类断言靠它。
type watermarkAdapter struct {
	*fakeDiscoveryAdapter
	pages    map[string]cloud.PageResult
	failAt   map[string]error
	received []string
}

func (a *watermarkAdapter) DiscoverRegion(_ context.Context, _ *ent.CloudAccount, _ string, _ cloud.Client, nextToken string) (*cloud.PageResult, error) {
	a.received = append(a.received, nextToken)
	if err, ok := a.failAt[nextToken]; ok && err != nil {
		return nil, err
	}
	page, ok := a.pages[nextToken]
	if !ok {
		return &cloud.PageResult{}, nil
	}
	return &page, nil
}

type watermarkHarness struct {
	t         *testing.T
	ctx       context.Context
	repo      Repository
	svc       *Service
	executor  *DiscoveryJobExecutor
	states    ScopeStateRepository
	adapter   *watermarkAdapter
	tenantID  int
	accountID int
	sourceID  string
}

func newWatermarkHarness(t *testing.T, name string) *watermarkHarness {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:discovery_wm_%s?mode=memory&cache=shared&_fk=1", name))
	t.Cleanup(func() { client.Close() })
	repo := NewEntRepository(client)
	tenantID, accountID, _ := seedDiscoveryEnv(t, repo, "wm-"+name)
	states, _ := newScopeStateTestRepo(t, name)

	adapter := &watermarkAdapter{
		fakeDiscoveryAdapter: &fakeDiscoveryAdapter{
			provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		},
		pages:  map[string]cloud.PageResult{},
		failAt: map[string]error{},
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, nil, nopLogger())
	executor.SetScopeStateRepository(states)
	return &watermarkHarness{
		t: t, ctx: context.Background(), repo: repo,
		svc: NewService(repo, nil, nopLogger()), executor: executor,
		states: states, adapter: adapter,
		tenantID: tenantID, accountID: accountID, sourceID: "ds-wm-" + name,
	}
}

func (h *watermarkHarness) runIncremental() *DiscoveryJob {
	h.t.Helper()
	job, err := h.svc.CreateScheduledDiscoveryJob(h.ctx, h.tenantID, &dto.DiscoveryJobRequest{SourceID: h.sourceID})
	require.NoError(h.t, err)
	require.NoError(h.t, h.executor.Execute(h.ctx, h.tenantID, job.ID))
	got, err := h.repo.GetDiscoveryJob(h.ctx, h.tenantID, job.ID)
	require.NoError(h.t, err)
	return got
}

func (h *watermarkHarness) runFull() *DiscoveryJob {
	h.t.Helper()
	job, err := h.svc.CreateDiscoveryJob(h.ctx, h.tenantID, 9, &dto.DiscoveryJobRequest{SourceID: h.sourceID})
	require.NoError(h.t, err)
	require.NoError(h.t, h.executor.Execute(h.ctx, h.tenantID, job.ID))
	got, err := h.repo.GetDiscoveryJob(h.ctx, h.tenantID, job.ID)
	require.NoError(h.t, err)
	return got
}

func wmResource(id string) cloud.DiscoveredResource {
	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID: id, ResourceName: id, Region: "cn-hangzhou", Status: "running",
		},
	}
}

func (h *watermarkHarness) scopeState() *DiscoveryScopeState {
	h.t.Helper()
	st, err := h.states.Load(h.ctx, h.tenantID, h.sourceID, "ecs", "cn-hangzhou")
	require.NoError(h.t, err)
	return st
}

func (h *watermarkHarness) resource(t *testing.T, id string) *CloudResource {
	t.Helper()
	list, err := h.repo.ListCloudResourcesByAccountService(h.ctx, h.tenantID, h.accountID, "ecs")
	require.NoError(t, err)
	for _, r := range list {
		if r.ResourceID == id {
			return r
		}
	}
	t.Fatalf("resource %s not found", id)
	return nil
}

// TestDiscoveryWatermark_ResumesFromCheckpoint 断点续扫：第一页 checkpoint 后
// 第二页失败，下一轮必须从水位线 token 续扫，而不是从零全量列举；封窗后
// 水位线清空、完整覆盖证明落位。
func TestDiscoveryWatermark_ResumesFromCheckpoint(t *testing.T) {
	h := newWatermarkHarness(t, "resume")
	h.adapter.pages[""] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-a")}, NextToken: "token-2"}
	h.adapter.failAt["token-2"] = errors.New("provider 500")

	job1 := h.runIncremental()
	assert.Equal(t, DiscoveryJobFailed, job1.Status)
	assert.Equal(t, []string{"", "token-2"}, h.adapter.received, "首轮：首页 + checkpoint 后失败")

	st := h.scopeState()
	assert.Equal(t, "token-2", st.ProviderWatermark, "失败必须保留水位线")
	assert.NotNil(t, st.CoverageStart, "失败窗口必须保留 coverage_start")
	assert.False(t, st.PaginationCompleted)
	assert.Equal(t, 1, st.ConsecutiveFailures)

	// 服务恢复：从 token-2 续扫完成。
	delete(h.adapter.failAt, "token-2")
	h.adapter.pages["token-2"] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-b")}}

	job2 := h.runIncremental()
	assert.Equal(t, DiscoveryJobSucceeded, job2.Status)
	assert.Equal(t, []string{"token-2"}, h.adapter.received[2:], "第二轮必须从水位线续扫，不得回到空 token")

	st = h.scopeState()
	assert.Equal(t, "", st.ProviderWatermark, "封窗后水位线清空")
	assert.True(t, st.PaginationCompleted, "翻页自然耗尽 = 完整覆盖证明")
	assert.NotNil(t, st.LastFullCompletedAt)
	assert.Equal(t, 0, st.ConsecutiveFailures)
	assert.Equal(t, 0, h.resource(t, "i-a").MissingCount, "续窗不得误判首页资源缺失")
	assert.Equal(t, 0, h.resource(t, "i-b").MissingCount)
}

// TestDiscoveryWatermark_CatchUpDoesNotRetireSeenResources 是蓝图 §3.2-17 的
// 缺陷锁：失败 scope 的补采（同一覆盖窗口的断点续扫）不得把上一轮已见资源
// 判成缺失并推进退役；只有窗口起点以来从未露面的资源才计 missing。
func TestDiscoveryWatermark_CatchUpDoesNotRetireSeenResources(t *testing.T) {
	h := newWatermarkHarness(t, "catchup")
	now := time.Now()

	// 存量资源 i-c：窗口开始前最后一次出现（应判缺失）。
	oldSeen := now.Add(-time.Hour)
	hash := resourceIdentityHash(h.tenantID, "aliyun", "acc-wm-catchup", "cn-hangzhou", "ecs", "instance", "i-c")
	_, err := h.repo.CreateCloudResource(h.ctx, &CloudResource{
		CloudAccountID: h.accountID, ServiceID: 1, ResourceID: "i-c", IdentityVersion: 1,
		Provider: "aliyun", Partition: "public", CanonicalAccountID: "acc-wm-catchup",
		ResourceScope: "zone", ServiceCode: "ecs", ResourceType: "instance", IdentityHash: hash,
		SourceID: h.sourceID, SourceFingerprint: "fp-c", MissingCount: 0,
		ResourceName: "i-c", Region: "cn-hangzhou", Status: "running",
		FirstSeenAt: &oldSeen, LastSeenAt: &oldSeen, LifecycleState: "active",
		TenantID: h.tenantID,
	})
	require.NoError(t, err)

	h.adapter.pages[""] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-a")}, NextToken: "t2"}
	h.adapter.failAt["t2"] = errors.New("provider 500")
	job1 := h.runIncremental()
	assert.Equal(t, DiscoveryJobFailed, job1.Status)

	delete(h.adapter.failAt, "t2")
	h.adapter.pages["t2"] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-b")}}
	job2 := h.runIncremental()
	assert.Equal(t, DiscoveryJobSucceeded, job2.Status)

	assert.Equal(t, 0, h.resource(t, "i-a").MissingCount,
		"补采不得把窗口内已见资源误判缺失（蓝图：失败 scope 的补采不得触发退役）")
	assert.Equal(t, 0, h.resource(t, "i-b").MissingCount)
	assert.Equal(t, 1, h.resource(t, "i-c").MissingCount,
		"窗口起点以来从未露面的资源才计 missing")
}

// TestDiscoveryWatermark_FullOperationResetsWaterline 手动全量（full_discovery）
// 必须重置水位线从零开新窗——操作员要的是「重扫一遍」，不是续采。
func TestDiscoveryWatermark_FullOperationResetsWaterline(t *testing.T) {
	h := newWatermarkHarness(t, "fullreset")
	h.adapter.pages[""] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-a")}, NextToken: "t2"}
	h.adapter.failAt["t2"] = errors.New("provider 500")
	job1 := h.runIncremental()
	assert.Equal(t, DiscoveryJobFailed, job1.Status)
	require.Equal(t, "t2", h.scopeState().ProviderWatermark)

	delete(h.adapter.failAt, "t2")
	h.adapter.pages["t2"] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-b")}}

	job2 := h.runFull()
	assert.Equal(t, DiscoveryJobSucceeded, job2.Status)
	assert.Equal(t, []string{"", "t2"}, h.adapter.received[2:],
		"全量必须从空 token 从零列举（随后正常翻页），忽略旧水位线")
	assert.Equal(t, 0, h.resource(t, "i-a").MissingCount, "全量重扫见到 i-a，不得判缺失")
}

// TestDiscoveryWatermark_ConsecutiveFailuresReset 连续失败达上限必须重置水位线：
// provider 游标可能已失效，续扫同一死路径会永远失败，从零开窗才能自愈。
func TestDiscoveryWatermark_ConsecutiveFailuresReset(t *testing.T) {
	h := newWatermarkHarness(t, "reset")
	h.adapter.pages[""] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-a")}, NextToken: "t2"}
	h.adapter.failAt["t2"] = errors.New("provider 500")

	h.runIncremental()
	assert.Equal(t, "t2", h.scopeState().ProviderWatermark)
	assert.Equal(t, 1, h.scopeState().ConsecutiveFailures)

	h.runIncremental() // 续扫 t2 立刻失败
	assert.Equal(t, 2, h.scopeState().ConsecutiveFailures)

	h.runIncremental() // 第 3 次失败 → 重置
	st := h.scopeState()
	assert.Equal(t, "", st.ProviderWatermark, "连续失败达上限必须重置水位线")
	assert.Nil(t, st.CoverageStart)
	assert.Equal(t, 0, st.ConsecutiveFailures)

	// 重置后从零开窗：适配器收到空 token（而不是继续撞死掉的 t2）。
	delete(h.adapter.failAt, "t2")
	h.adapter.pages["t2"] = cloud.PageResult{Resources: []cloud.DiscoveredResource{wmResource("i-b")}}
	job := h.runIncremental()
	assert.Equal(t, DiscoveryJobSucceeded, job.Status)
	assert.Equal(t, []string{"", "t2"}, h.adapter.received[4:],
		"重置后必须从空 token 从零列举（随后正常翻页）")
}

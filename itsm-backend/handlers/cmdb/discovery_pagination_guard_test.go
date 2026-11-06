package cmdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/cloud"

	_ "github.com/mattn/go-sqlite3"
)

// pagingAdapter 模拟两种病态分页：
//   - stalled：每次回显同一个非空 NextToken（腾讯 CVM 空页回显旧游标即此形态）
//   - endless：每页都给一个新 token，永远不结束
//
// 修复前两者都会让 runScope 的 for{} 永不退出：计费 API 被放大、worker 卡死。
type pagingAdapter struct {
	*fakeDiscoveryAdapter
	mode  string
	calls int
}

func (p *pagingAdapter) DiscoverRegion(_ context.Context, _ *ent.CloudAccount, _ string, _ cloud.Client, nextToken string) (*cloud.PageResult, error) {
	p.calls++
	switch p.mode {
	case "stalled":
		// 首轮给一个 token，之后原样回显；守卫必须在第 2 次调用时就断掉。
		return &cloud.PageResult{NextToken: "stuck-token"}, nil
	case "endless":
		return &cloud.PageResult{NextToken: fmt.Sprintf("page-%d", p.calls)}, nil
	default:
		return &cloud.PageResult{}, nil
	}
}

// missingScanSpyRepo 记录缺失扫描是否真的执行了写回。
// 分页异常若被静默 break，循环后的缺失扫描会把没扫到的存量资源当作
// "本轮未出现"推进 missing_count —— 这是比死循环更隐蔽的数据破坏。
type missingScanSpyRepo struct {
	Repository
	updateCalls int
}

func (r *missingScanSpyRepo) UpdateDiscoveredCloudResource(ctx context.Context, cr *CloudResource) error {
	r.updateCalls++
	return r.Repository.UpdateDiscoveredCloudResource(ctx, cr)
}

func runPagingGuardJob(t *testing.T, mode string) (*pagingAdapter, *missingScanSpyRepo, *DiscoveryJob) {
	t.Helper()
	ctx := context.Background()
	dbName := fmt.Sprintf("file:discovery_paging_%s?mode=memory&cache=shared&_fk=1", mode)
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })
	baseRepo := NewEntRepository(client)
	tenantID, _, _ := seedDiscoveryEnv(t, baseRepo, "paging-"+mode)

	svc := NewService(baseRepo, nil, nopLogger())
	job, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-paging-" + mode})
	require.NoError(t, err)

	adapter := &pagingAdapter{
		fakeDiscoveryAdapter: &fakeDiscoveryAdapter{
			provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		},
		mode: mode,
	}
	spy := &missingScanSpyRepo{Repository: baseRepo}
	executor := NewDiscoveryJobExecutor(spy, &fakeRegistry{adapter: adapter}, &fakeResolver{}, nil, nopLogger())
	require.NoError(t, executor.Execute(ctx, tenantID, job.ID))

	got, err := spy.GetDiscoveryJob(ctx, tenantID, job.ID)
	require.NoError(t, err)
	return adapter, spy, got
}

func TestDiscoveryPagination_StalledTokenFailsClosedWithoutMissingScan(t *testing.T) {
	adapter, spy, job := runPagingGuardJob(t, "stalled")

	assert.LessOrEqual(t, adapter.calls, 2, "回显不变 token 时第 2 次调用就该终止")
	assert.Equal(t, []string{"ecs/cn-hangzhou"}, job.FailedScopes)
	details, ok := job.Summary["failedScopeDetails"].([]interface{})
	require.True(t, ok)
	require.Len(t, details, 1)
	assert.Equal(t, "discover_page_token_stalled", details[0].(map[string]interface{})["errorCode"])
	assert.Equal(t, 0, spy.updateCalls, "分页异常绝不能触发缺失扫描写回")
}

func TestDiscoveryPagination_PageLimitFailsClosedWithoutMissingScan(t *testing.T) {
	adapter, spy, job := runPagingGuardJob(t, "endless")

	assert.Equal(t, maxDiscoveryPages, adapter.calls, "翻页必须硬停在 maxDiscoveryPages")
	assert.Equal(t, []string{"ecs/cn-hangzhou"}, job.FailedScopes)
	details, ok := job.Summary["failedScopeDetails"].([]interface{})
	require.True(t, ok)
	require.Len(t, details, 1)
	assert.Equal(t, "discover_page_limit_exceeded", details[0].(map[string]interface{})["errorCode"])
	assert.Equal(t, 0, spy.updateCalls, "翻页超限绝不能触发缺失扫描写回")
}

// 正控制：正常单页收尾仍成功，不能因为加了上限把正常路径判成失败。
func TestDiscoveryPagination_NormalSinglePageStillSucceeds(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:discovery_paging_ok?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	repo := NewEntRepository(client)
	tenantID, _, _ := seedDiscoveryEnv(t, repo, "paging-ok")

	svc := NewService(repo, nil, nopLogger())
	created, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-paging-ok"})
	require.NoError(t, err)

	adapter := &pagingAdapter{
		fakeDiscoveryAdapter: &fakeDiscoveryAdapter{
			provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		},
		mode: "normal",
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, nil, nopLogger())
	require.NoError(t, executor.Execute(ctx, tenantID, created.ID))

	got, err := repo.GetDiscoveryJob(ctx, tenantID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, DiscoveryJobSucceeded, got.Status, "NextToken 为空是正常收尾，不是翻页超限")
	assert.Equal(t, 1, adapter.calls)
}

package knowledge

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 本文件锁定「发布必须过评审」这条产品契约。
//
// 背景：api-reference 与 capability contract 都承诺「发布必须过知识评审流
// （submit→approve）」，但 /publish 端点在加门禁前直接写 is_published=true，
// 等于给评审流开了一条旁路——作者可以绕过防自审（applyReview 拒绝作者批
// 自己的稿）把自己写的草稿直接放进 RAG。空壳内容进语料正是 422 质量门要
// 防的事，门禁必须在两条入口上一致。

const publishTestTenant = 7

func newPublishTestService(t *testing.T, dsn string) (*Service, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	svc := NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar())
	svc.SetEntClient(client)
	// rag 非 nil 才启用生产路径的向量同步入箱，用于断言「未发布不产生索引任务」。
	svc.SetRAG(service.NewRAGService(nil, nil, nil, zaptest.NewLogger(t).Sugar(), service.RAGConfig{}))
	return svc, client
}

func countVectorCommands(t *testing.T, client *ent.Client, articleID int) int {
	t.Helper()
	n, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(publishTestTenant),
		operationalcommand.AggregateIDEQ(articleID),
		operationalcommand.CommandTypeEQ(commandbus.CommandSyncKnowledgeVector),
	).Count(context.Background())
	require.NoError(t, err)
	return n
}

// TestPublishRequiresApprovedReview 未通过评审的三个状态一律拒绝发布，
// 且不留下任何索引任务（拒绝要彻底，不能「先发布再回滚」）。
func TestPublishRequiresApprovedReview(t *testing.T) {
	ctx := context.Background()
	for _, status := range []string{
		ReviewStatusDraft,
		ReviewStatusUnderReview,
		ReviewStatusRejected,
	} {
		t.Run(status, func(t *testing.T) {
			svc, client := newPublishTestService(t,
				fmt.Sprintf("file:publish_gate_%s?mode=memory&cache=shared&_fk=1", status))

			created, err := svc.CreateArticle(ctx, &Article{
				Title: "runbook", Content: "重启前先摘流量", Category: "运维",
				AuthorID: 1, TenantID: publishTestTenant,
			})
			require.NoError(t, err)
			require.False(t, created.IsPublished)

			if status != ReviewStatusDraft {
				// draft 是建库默认值；其余状态直接落库，模拟流转后的真实数据。
				_, err = client.KnowledgeArticle.UpdateOneID(created.ID).
					SetReviewStatus(knowledgearticle.ReviewStatus(status)).
					Save(ctx)
				require.NoError(t, err)
			}

			_, err = svc.SetArticlePublished(ctx, created.ID, publishTestTenant, true)
			require.Error(t, err, "未过评审的文章不允许发布")

			var be *common.BusinessError
			require.True(t, errors.As(err, &be), "应是业务错误而非裸错误: %v", err)
			require.Equal(t, common.ConflictCode, be.Code)
			require.Contains(t, be.Message, "评审")
			require.NotEmpty(t, be.Detail, "拒绝必须给出可行动指引（先 submit→approve）")

			got, err := svc.GetArticle(ctx, created.ID, publishTestTenant)
			require.NoError(t, err)
			require.False(t, got.IsPublished, "被拒的发布不能把 is_published 改掉")
			require.Zero(t, countVectorCommands(t, client, created.ID), "被拒的发布不能产生索引任务")
		})
	}
}

// TestApprovePublishesAndRepublishIsIdempotent 评审通过即对外可见（同一事务
// 入索引队列）；已发布再点发布是幂等操作，不应重复制造索引任务。
func TestApprovePublishesAndRepublishIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t,
		"file:publish_gate_approve?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "连接池耗尽", Content: "连接池未设上限", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)

	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	approved, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	require.Equal(t, ReviewStatusApproved, NormalizeReviewStatus(approved.ReviewStatus))
	require.True(t, approved.IsPublished, "评审通过 = 对外可见")
	require.Equal(t, 1, countVectorCommands(t, client, created.ID))

	// 幂等重发：不新增索引任务
	again, err := svc.SetArticlePublished(ctx, created.ID, publishTestTenant, true)
	require.NoError(t, err)
	require.True(t, again.IsPublished)
	require.Equal(t, 1, countVectorCommands(t, client, created.ID), "重复发布不应重复入箱")
}

// TestUnpublishAlwaysAllowedThenRepublish 撤回不设门槛（撤下内容永远是安全
// 方向），撤回后可经 publish 恢复；两次操作各自入箱（delete → sync）。
func TestUnpublishAlwaysAllowedThenRepublish(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t,
		"file:publish_gate_unpublish?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "变更回滚", Content: "回滚前确认备份", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	_, err = svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)

	unpublished, err := svc.SetArticlePublished(ctx, created.ID, publishTestTenant, false)
	require.NoError(t, err, "撤回不得要求评审状态")
	require.False(t, unpublished.IsPublished)

	republished, err := svc.SetArticlePublished(ctx, created.ID, publishTestTenant, true)
	require.NoError(t, err, "已通过评审的内容允许恢复可见")
	require.True(t, republished.IsPublished)

	// approve(sync) + unpublish(delete) + republish(sync)
	require.Equal(t, 3, countVectorCommands(t, client, created.ID))
}

// TestDraftNotRetrievableUntilApproved 端到端验收：草稿不进 RAG 检索，
// 审批通过后立即可检索，撤回后再次消失。
//
// 这正是 capability contract 记的缺口——「沉淀→评审→发布→RAG 可检索的
// 端到端验收仍缺（手工链路已通）」。此处用关键字检索路径（不依赖向量库
// 与 embedding），断言的是可见性契约本身而非 provider 行为。
func TestDraftNotRetrievableUntilApproved(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:publish_gate_rag?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	rag := service.NewRAGService(client, nil, nil, logger, service.RAGConfig{UseKeyword: true})
	svc := NewService(NewEntRepository(client), logger)
	svc.SetEntClient(client)
	svc.SetRAG(rag)

	created, err := svc.CreateArticle(ctx, &Article{
		Title:    "连接池耗尽导致服务雪崩",
		Content:  "根因：连接池未设上限；处置：限制并发并加超时",
		Category: "故障排查", AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)

	hits, err := rag.Ask(ctx, publishTestTenant, "连接池", 10)
	require.NoError(t, err)
	require.Empty(t, hits, "草稿不得进入 RAG 检索结果")

	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	hits, err = rag.Ask(ctx, publishTestTenant, "连接池", 10)
	require.NoError(t, err)
	require.Empty(t, hits, "评审中的内容仍不得进入检索")

	_, err = svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	hits, err = rag.Ask(ctx, publishTestTenant, "连接池", 10)
	require.NoError(t, err)
	require.Len(t, hits, 1, "审批通过后必须可检索（visible 与索引一致）")

	_, err = svc.SetArticlePublished(ctx, created.ID, publishTestTenant, false)
	require.NoError(t, err)
	hits, err = rag.Ask(ctx, publishTestTenant, "连接池", 10)
	require.NoError(t, err)
	require.Empty(t, hits, "撤回后必须立刻检索不到")
}

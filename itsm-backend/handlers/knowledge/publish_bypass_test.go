package knowledge

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"github.com/stretchr/testify/require"
)

// 本文件锁定两条「发布必须过评审」的旁路：
//
//  1. PUT /articles/:id 的 status=published 字段。门禁原本只在
//     SetArticlePublished（/publish 端点）上，UpdateArticle 直接写
//     is_published，等于给评审流开了第二条后门。
//  2. 版本恢复。RestoreVersion 覆写正文却保留 approved + is_published，
//     旧内容未经重新评审就重新进入 RAG，且不清理旧索引。
//
// 两条入口都必须与 /publish 一致：未过评审不得发布；恢复是内容变更，
// 必须回到 draft 并撤销可见性 + 清理索引。

// TestUpdateArticleCannotPublishUnapproved 通过 UpdateArticle 的 status 字段
// 发布未过评审的文章必须被拒，且不留任何索引任务。
func TestUpdateArticleCannotPublishUnapproved(t *testing.T) {
	ctx := context.Background()
	for _, status := range []string{
		ReviewStatusDraft,
		ReviewStatusUnderReview,
		ReviewStatusRejected,
	} {
		t.Run(status, func(t *testing.T) {
			svc, client := newPublishTestService(t,
				fmt.Sprintf("file:publish_bypass_update_%s?mode=memory&cache=shared&_fk=1", status))

			created, err := svc.CreateArticle(ctx, &Article{
				Title: "runbook", Content: "重启前先摘流量", Category: "运维",
				AuthorID: 1, TenantID: publishTestTenant,
			})
			require.NoError(t, err)

			if status != ReviewStatusDraft {
				_, err = client.KnowledgeArticle.UpdateOneID(created.ID).
					SetReviewStatus(knowledgearticle.ReviewStatus(status)).
					Save(ctx)
				require.NoError(t, err)
			}

			existing, err := svc.GetArticle(ctx, created.ID, publishTestTenant)
			require.NoError(t, err)
			existing.IsPublished = true

			_, err = svc.UpdateArticle(ctx, existing)
			require.Error(t, err, "未过评审不得经 UpdateArticle 发布")

			var be *common.BusinessError
			require.True(t, errors.As(err, &be), "应是业务错误而非裸错误: %v", err)
			require.Equal(t, common.ConflictCode, be.Code)
			require.Contains(t, be.Message, "评审")

			got, err := svc.GetArticle(ctx, created.ID, publishTestTenant)
			require.NoError(t, err)
			require.False(t, got.IsPublished, "被拒的发布不能改掉 is_published")
			require.Zero(t, countVectorCommands(t, client, created.ID), "被拒的发布不能产生索引任务")
		})
	}
}

// TestUpdateArticleApprovedStaysPublished 已通过评审的文章正常编辑/发布不受门禁影响。
func TestUpdateArticleApprovedStaysPublished(t *testing.T) {
	ctx := context.Background()
	svc, _ := newPublishTestService(t,
		"file:publish_bypass_approved?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "连接池耗尽", Content: "连接池未设上限", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	_, err = svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)

	existing, err := svc.GetArticle(ctx, created.ID, publishTestTenant)
	require.NoError(t, err)
	require.True(t, existing.IsPublished)

	existing.Content = "连接池未设上限；处置：限制并发并加超时"
	existing.IsPublished = true
	updated, err := svc.UpdateArticle(ctx, existing)
	require.NoError(t, err, "已通过评审的内容允许编辑并保持发布")
	require.True(t, updated.IsPublished)
	require.Equal(t, ReviewStatusApproved, NormalizeReviewStatus(updated.ReviewStatus))
}

// TestRestoreVersionResetsReviewAndInvalidatesIndex 恢复旧版本是内容变更：
// 必须回到 draft、撤销可见性，并在同事务入向量删除任务清理旧索引。
func TestRestoreVersionResetsReviewAndInvalidatesIndex(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t,
		"file:publish_bypass_restore?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "变更回滚", Content: "回滚前确认备份（旧版本）", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)

	// 种一个历史版本快照供恢复（生产里由恢复动作自身创建）。
	_, err = client.KnowledgeArticleVersion.Create().
		SetArticleID(created.ID).
		SetVersion(1).
		SetTitle("变更回滚").
		SetContent("回滚前确认备份（旧版本）").
		SetCategory("运维").
		SetTags("").
		SetAuthorID(1).
		SetChangeSummary("初始版本").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	_, err = svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)

	// 改动正文到「新版本」，再恢复回旧版本。
	existing, err := svc.GetArticle(ctx, created.ID, publishTestTenant)
	require.NoError(t, err)
	existing.Content = "回滚前确认备份（新版本）"
	_, err = svc.UpdateArticle(ctx, existing)
	require.NoError(t, err)

	beforeCommands := countVectorCommands(t, client, created.ID)

	restored, err := svc.RestoreVersion(ctx, created.ID, 1, publishTestTenant, 2)
	require.NoError(t, err)
	require.Contains(t, restored.Content, "旧版本", "正文应恢复到指定版本")
	require.False(t, restored.IsPublished, "恢复是内容变更，必须撤销可见性待重新评审")
	require.Equal(t, ReviewStatusDraft, NormalizeReviewStatus(restored.ReviewStatus), "恢复后必须回到 draft")

	got, err := svc.GetArticle(ctx, created.ID, publishTestTenant)
	require.NoError(t, err)
	require.False(t, got.IsPublished)
	require.Equal(t, ReviewStatusDraft, NormalizeReviewStatus(got.ReviewStatus))

	// 恢复必须在同事务入向量删除任务，清理旧索引。
	afterCommands := countVectorCommands(t, client, created.ID)
	require.Equal(t, beforeCommands+1, afterCommands, "恢复应新增一条索引任务")

	last, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(publishTestTenant),
		operationalcommand.AggregateIDEQ(created.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandSyncKnowledgeVector),
	).Order(ent.Desc(operationalcommand.FieldID)).First(ctx)
	require.NoError(t, err)
	require.Equal(t, vectorIndexDelete, last.Payload["action"], "恢复必须入删除任务而非同步")
}

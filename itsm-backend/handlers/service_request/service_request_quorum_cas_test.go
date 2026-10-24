package service_request

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entapproval "itsm-backend/ent/servicerequestapproval"
)

// 会签投票是对 Node JSON 的整块读改写。没有 version 谓词时，两个并发事务
// 都读到旧的 approved_ids 后先后覆盖，先提交者的票被静默丢弃，quorum 永不达标。
// version CAS 让 stale 快照的写 0 行命中并返回 ErrApprovalWriteConflict，
// service 层据此重读最新票数重新合并。

func quorumCASTestDB(t *testing.T) (*EntRepository, *ent.Client, context.Context, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_quorum_cas.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName(fmt.Sprintf("SRQ-%d", time.Now().UnixNano())).
		SetCode(fmt.Sprintf("SRQ-%d", time.Now().UnixNano())).
		SetDomain("sr-quorum.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return NewEntRepository(client), client, ctx, tenant.ID
}

func quorumCASTestApproval(t *testing.T, client *ent.Client, ctx context.Context, tenantID int) *ServiceRequestApproval {
	t.Helper()
	row, err := client.ServiceRequestApproval.Create().
		SetTenantID(tenantID).
		SetServiceRequestID(1).
		SetLevel(1).
		SetStatus(ApprovalStatusPending).
		SetTimeoutHours(24).
		SetNode(map[string]interface{}{
			"approver_ids":     []interface{}{101.0, 102.0},
			"quorum_threshold": 2.0,
			"approved_ids":     []interface{}{},
		}).
		Save(ctx)
	require.NoError(t, err)
	repo := NewEntRepository(client)
	return repo.toDomainApproval(row)
}

func quorumApprovedIDs(t *testing.T, client *ent.Client, ctx context.Context, approvalID int) []interface{} {
	t.Helper()
	row, err := client.ServiceRequestApproval.Query().
		Where(entapproval.ID(approvalID)).
		First(ctx)
	require.NoError(t, err)
	ids, _ := row.Node["approved_ids"].([]interface{})
	return ids
}

// appendVote 模拟 service 层的读-合并片段：基于 fresh 行追加 actor 并返回新快照。
func appendVote(fresh *ServiceRequestApproval, actorID int) *ServiceRequestApproval {
	approved := []interface{}{}
	if existing, ok := fresh.Node["approved_ids"].([]interface{}); ok {
		approved = append(approved, existing...)
	}
	approved = append(approved, float64(actorID))
	fresh.Node["approved_ids"] = approved
	return fresh
}

func TestUpdateApprovalWithClient_StaleVersionRejected_FreshRetrySucceeds(t *testing.T) {
	repo, client, ctx, tenantID := quorumCASTestDB(t)
	approval := quorumCASTestApproval(t, client, ctx, tenantID)
	require.Equal(t, 1, approval.Version)

	// 第一票：基于 version 1 快照写成功，版本推进到 2。
	voteA := appendVote(approval, 101)
	require.NoError(t, repo.UpdateApprovalWithClient(ctx, client, voteA))
	require.ElementsMatch(t, []interface{}{101.0}, quorumApprovedIDs(t, client, ctx, approval.ID))

	// 第二票并发场景：仍持 version 1 旧快照 → CAS 拒绝，票不得覆盖。
	stale := &ServiceRequestApproval{ID: approval.ID, TenantID: tenantID, Version: 1, Node: map[string]interface{}{
		"approver_ids":     []interface{}{101.0, 102.0},
		"quorum_threshold": 2.0,
		"approved_ids":     []interface{}{102.0},
	}}
	err := repo.UpdateApprovalWithClient(ctx, client, stale)
	require.True(t, errors.Is(err, ErrApprovalWriteConflict), "expected ErrApprovalWriteConflict, got %v", err)
	// 先到者的票不被后到者的 stale 写覆盖。
	require.ElementsMatch(t, []interface{}{101.0}, quorumApprovedIDs(t, client, ctx, approval.ID))

	// service 层重试路径：重读最新行（version 2）→ 合并第二票 → CAS 命中。
	fresh, err := repo.GetApprovalWithClient(ctx, client, 1, 1, tenantID)
	require.NoError(t, err)
	require.Equal(t, 2, fresh.Version)
	require.NoError(t, repo.UpdateApprovalWithClient(ctx, client, appendVote(fresh, 102)))
	require.ElementsMatch(t, []interface{}{101.0, 102.0}, quorumApprovedIDs(t, client, ctx, approval.ID), "两票必须共存，quorum 才可能达标")

	final, err := repo.GetApprovalWithClient(ctx, client, 1, 1, tenantID)
	require.NoError(t, err)
	require.Equal(t, 3, final.Version)
}

func TestUpdateApprovalWithClient_AlreadyTerminalStatusRejected(t *testing.T) {
	repo, client, ctx, tenantID := quorumCASTestDB(t)
	approval := quorumCASTestApproval(t, client, ctx, tenantID)

	// 先终审：status 离开 pending。
	vote := appendVote(approval, 101)
	vote.Status = ApprovalStatusApproved
	vote.Action = "approve"
	vote.ApproverID = &voteActioner
	require.NoError(t, repo.UpdateApprovalWithClient(ctx, client, vote))

	// 终审后的迟到写（即使 version 匹配新值也不该落库——status 谓词拦截）。
	late := appendVote(vote, 102)
	err := repo.UpdateApprovalWithClient(ctx, client, late)
	require.True(t, errors.Is(err, ErrApprovalWriteConflict), "expected ErrApprovalWriteConflict, got %v", err)
}

func TestUpdateRequestAndApprovalWithClient_RequestVersionConflictMapped(t *testing.T) {
	repo, client, ctx, tenantID := quorumCASTestDB(t)
	approval := quorumCASTestApproval(t, client, ctx, tenantID)

	// request 行 version 被外部推进（模拟并发终审推进了请求状态）。
	_, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(1).
		SetRequesterID(1).
		SetStatus(SRStatusSubmitted).
		SetTitle("quorum req").
		SetReason("testing").
		SetCurrentLevel(1).
		SetTotalLevels(2).
		Save(ctx)
	require.NoError(t, err)
	reqRow, err := client.ServiceRequest.Query().First(ctx)
	require.NoError(t, err)
	require.NoError(t, client.ServiceRequest.UpdateOneID(reqRow.ID).AddVersion(5).Exec(ctx))

	staleReq := &ServiceRequest{ID: reqRow.ID, TenantID: tenantID, Version: reqRow.Version, Status: SRStatusManagerApproved, CurrentLevel: 2}
	err = repo.UpdateRequestAndApprovalWithClient(ctx, client, staleReq, approval)
	require.True(t, errors.Is(err, ErrRequestWriteConflict), "expected ErrRequestWriteConflict, got %v", err)
}

var voteActioner = 101

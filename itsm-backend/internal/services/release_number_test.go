package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 收敛锁：release_number 由自增 ID 同事务派生（REL-YYYYMMDD-NNNNNN），
// 替代旧 math/rand 随机发号（可静默重号且库中无唯一约束）。
func TestReleaseNumberIsIDDerivedAndUnique(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Release Number Tenant").SetCode("relnum").SetDomain("relnum.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	creator, err := client.User.Create().
		SetUsername("relnum-user").SetEmail("relnum@test.local").SetName("RelNum").
		SetPasswordHash("hashed").SetRole("admin").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	numberPattern := regexp.MustCompile(`^REL-\d{8}-(\d{6})$`)
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		resp, err := svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
			Title: fmt.Sprintf("发布 %d", i), Type: "minor", Environment: "staging", Severity: "medium",
		}, creator.ID, tenant.ID)
		require.NoError(t, err)

		matches := numberPattern.FindStringSubmatch(resp.ReleaseNumber)
		require.NotNil(t, matches, "编号必须是 REL-YYYYMMDD-NNNNNN 数字后缀，got %s", resp.ReleaseNumber)
		require.False(t, strings.Contains(resp.ReleaseNumber, "PENDING"), "占位编号不得泄漏到事务外")

		suffix, err := strconv.Atoi(matches[1])
		require.NoError(t, err)
		require.Equal(t, resp.ID, suffix, "编号后缀必须等于发布自增 ID（同事务派生）")

		persisted, err := client.Release.Get(ctx, resp.ID)
		require.NoError(t, err)
		require.Equal(t, resp.ReleaseNumber, persisted.ReleaseNumber, "响应与落库编号必须一致")

		require.False(t, seen[resp.ReleaseNumber], "编号重复：%s", resp.ReleaseNumber)
		seen[resp.ReleaseNumber] = true
	}
}

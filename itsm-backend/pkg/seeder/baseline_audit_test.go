package seeder

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 审计是前滚修复方案的只读输入：必须一个字节都不写，并如实区分
// "基线齐备"与"存在缺口"的租户。
func TestAuditTenantBaselinesIsReadOnlyAndScoped(t *testing.T) {
	s, db, ctx := newColdVerifyFixture(t)
	empty, err := s.client.Tenant.Create().
		SetCode("audit-empty").SetName("Audit Empty").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	require.Positive(t, empty.ID)

	var before, after int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT total_changes()").Scan(&before))
	_, err = db.ExecContext(ctx, "PRAGMA query_only = ON")
	require.NoError(t, err)
	writeAttempts := 0
	s.client.Use(func(ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(context.Context, ent.Mutation) (ent.Value, error) {
			writeAttempts++
			return nil, errors.New("baseline audit must not write")
		})
	})

	audits, err := s.AuditTenantBaselines(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 2)

	byCode := make(map[string]TenantBaselineAudit, len(audits))
	for _, audit := range audits {
		byCode[audit.TenantCode] = audit
	}
	installed, exists := byCode["default"]
	require.True(t, exists, "审计必须覆盖平台租户")
	assert.True(t, installed.Ready, "已安装基线的租户应判为 ready")
	require.Len(t, installed.Components, len(ProductionComponentNames))
	for _, component := range installed.Components {
		assert.True(t, component.Verified, component.Component)
	}

	missing, exists := byCode["audit-empty"]
	require.True(t, exists, "审计必须覆盖未安装基线的租户")
	assert.False(t, missing.Ready, "空租户必须如实报告未就绪")
	require.NotEmpty(t, missing.Components)
	assert.False(t, missing.Components[0].Verified)
	assert.NotEmpty(t, missing.Components[0].Error, "未就绪必须给出可定位的缺口")

	require.Zero(t, writeAttempts, "审计试图写入，即使错误被吞也不允许")
	require.NoError(t, db.QueryRowContext(ctx, "SELECT total_changes()").Scan(&after))
	require.Equal(t, before, after, "审计改变了数据库")
}

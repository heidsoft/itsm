package service

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestCreateTenant_DuplicateCodeReturnsConflict 修复回归：查重原在事务外，
// 并发同 code 时另一请求插入后 Save 撞 tenants_code_key 唯一索引被 handler
// 分类成 500。现查重移入事务 + ConstraintError 显式映射 409 conflict。
func TestCreateTenant_DuplicateCodeReturnsConflict(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewTenantService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	req := &dto.CreateTenantRequest{
		Name: "Dup Code Tenant",
		Code: "dup-code",
		Type: "standard",
	}

	created, err := svc.CreateTenant(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "dup-code", created.Code)

	_, err = svc.CreateTenant(ctx, req)
	require.Error(t, err, "同 code 二次创建必须被拒绝")
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "应返回业务 conflict 错误，实际: %v", err)
	require.Equal(t, common.ConflictCode, bizErr.Code)
}

package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestUpdateUser_PatchPointerClearsOptionalFields 修复回归：Department/Phone 曾是
// omitempty 非指针字段，PATCH 传空串被当「未传」，字段永远清不掉（违反 AGENTS.md
// PATCH 指针规则）。指针化后非 nil 即提交、空串即清空。
func TestUpdateUser_PatchPointerClearsOptionalFields(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Patch Tenant").
		SetCode("patch-ptr").
		SetDomain("patch-ptr.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	dept := "研发部"
	phone := "13800000000"
	created, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		Username:   "patchptr",
		Email:      "patchptr@example.com",
		Name:       "Patch Ptr",
		Password:   "Str0ngP@ssw0rd!",
		Department: dept,
		Phone:      phone,
	}, tenant.ID)
	require.NoError(t, err)
	require.Equal(t, dept, created.Department)
	require.Equal(t, phone, created.Phone)

	empty := ""
	updated, err := svc.UpdateUser(ctx, created.ID, &dto.UpdateUserRequest{
		Department: &empty,
		Phone:      &empty,
	}, tenant.ID)
	require.NoError(t, err)
	require.Empty(t, updated.Department, "传空串必须清空 department（修复前是 no-op）")
	require.Empty(t, updated.Phone, "传空串必须清空 phone（修复前是 no-op）")

	// 「未传」仍不得触碰字段
	value := "新部门"
	_, err = svc.UpdateUser(ctx, created.ID, &dto.UpdateUserRequest{
		Department: &value,
	}, tenant.ID)
	require.NoError(t, err)
	finalUser, err := svc.GetUserByID(ctx, created.ID, tenant.ID)
	require.NoError(t, err)
	require.Equal(t, value, finalUser.Department)
	require.Empty(t, finalUser.Phone, "未传 phone 不得被触碰")
}

package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepartmentService_CreateDepartment(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	deptService := NewDepartmentService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		nameVal       string
		codeVal       string
		description   string
		managerID     int
		parentID      int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功创建根部门",
			nameVal:       "IT部门",
			codeVal:       "IT",
			description:   "信息技术部门",
			managerID:     0,
			parentID:      0,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "重复部门代码",
			nameVal:       "另一个IT部门",
			codeVal:       "IT",
			description:   "重复代码",
			managerID:     0,
			parentID:      0,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dept, err := deptService.CreateDepartment(ctx, tt.nameVal, tt.codeVal, tt.description, tt.managerID, tt.parentID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, dept)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dept)
				assert.Equal(t, tt.nameVal, dept.Name)
				assert.Equal(t, tt.codeVal, dept.Code)
			}
		})
	}
}

func TestDepartmentService_GetDepartmentTree(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	deptService := NewDepartmentService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建部门结构
	rootDept, err := client.Department.Create().
		SetName("总公司").
		SetCode("ROOT").
		SetDescription("总公司").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Department.Create().
		SetName("研发部").
		SetCode("RD").
		SetDescription("研发部门").
		SetTenantID(testTenant.ID).
		SetParentID(rootDept.ID).
		Save(ctx)
	require.NoError(t, err)

	// 测试获取部门树
	tree, err := deptService.GetDepartmentTree(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Len(t, tree, 1)
	assert.Equal(t, "总公司", tree[0].Name)
}

package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectService_CreateProject(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	projectService := NewProjectService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试部门
	testDept, err := client.Department.Create().
		SetName("IT部门").
		SetCode("IT").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		nameVal       string
		codeVal       string
		deptID        int
		managerID     int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功创建项目",
			nameVal:       "IT项目",
			codeVal:       "IT-PROJ",
			deptID:        testDept.ID,
			managerID:     0,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "重复项目代码",
			nameVal:       "另一个项目",
			codeVal:       "IT-PROJ",
			deptID:        testDept.ID,
			managerID:     0,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := projectService.CreateProject(ctx, tt.nameVal, tt.codeVal, tt.deptID, tt.managerID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, project)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				assert.Equal(t, tt.nameVal, project.Name)
				assert.Equal(t, tt.codeVal, project.Code)
			}
		})
	}
}

func TestProjectService_ListProjects(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	projectService := NewProjectService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建部门
	testDept, err := client.Department.Create().
		SetName("IT部门").
		SetCode("IT").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 为租户创建项目
	_, err = client.Project.Create().
		SetName("项目A").
		SetCode("PROJ-A").
		SetTenantID(testTenant.ID).
		SetDepartmentID(testDept.ID).
		SetStartDate(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Project.Create().
		SetName("项目B").
		SetCode("PROJ-B").
		SetTenantID(testTenant.ID).
		SetDepartmentID(testDept.ID).
		SetStartDate(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试获取项目列表
	projects, err := projectService.ListProjects(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestProjectService_UpdateProject(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	projectService := NewProjectService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建部门
	testDept, err := client.Department.Create().
		SetName("IT部门").
		SetCode("IT").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建项目
	testProject, err := client.Project.Create().
		SetName("原始项目").
		SetCode("ORIGINAL").
		SetTenantID(testTenant.ID).
		SetDepartmentID(testDept.ID).
		SetStartDate(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试更新
	newName := "新项目名"
	project, err := projectService.UpdateProject(ctx, testProject.ID, &newName, nil, nil, nil, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, newName, project.Name)
}

func TestProjectService_DeleteProject(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	projectService := NewProjectService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建部门
	testDept, err := client.Department.Create().
		SetName("IT部门").
		SetCode("IT").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建项目
	testProject, err := client.Project.Create().
		SetName("待删除项目").
		SetCode("DELETE-ME").
		SetTenantID(testTenant.ID).
		SetDepartmentID(testDept.ID).
		SetStartDate(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 删除项目
	err = projectService.DeleteProject(ctx, testProject.ID, testTenant.ID)
	assert.NoError(t, err)
}

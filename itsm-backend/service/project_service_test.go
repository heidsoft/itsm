package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectService_CreateProject(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
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
	client := enttest.Open(t, "sqlite3", testDSN())
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
	client := enttest.Open(t, "sqlite3", testDSN())
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

func TestProjectService_UpdateProject_RejectsCrossTenantReferences(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	projectService := NewProjectService(client)
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().
		SetName("Tenant A").SetCode("PA").SetDomain("pa.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("Tenant B").SetCode("PB").SetDomain("pb.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	deptA, err := client.Department.Create().
		SetName("Dept A").SetCode("DA").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	deptB, err := client.Department.Create().
		SetName("Dept B").SetCode("DB").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)
	managerA, err := client.User.Create().
		SetUsername("mgr-a").SetEmail("mgr-a@pa.test").SetName("Manager A").
		SetPasswordHash("x").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	managerB, err := client.User.Create().
		SetUsername("mgr-b").SetEmail("mgr-b@pb.test").SetName("Manager B").
		SetPasswordHash("x").SetRole("agent").SetActive(true).SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	target, err := client.Project.Create().
		SetName("Target").SetCode("TARGET").SetTenantID(tenantA.ID).
		SetDepartmentID(deptA.ID).SetManagerID(managerA.ID).SetStartDate(time.Now()).Save(ctx)
	require.NoError(t, err)

	// 跨租户部门 / 经理：参数错误，项目保持原值
	_, err = projectService.UpdateProject(ctx, target.ID, nil, nil, &deptB.ID, nil, tenantA.ID)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.ParamErrorCode, bizErr.Code)

	_, err = projectService.UpdateProject(ctx, target.ID, nil, nil, nil, &managerB.ID, tenantA.ID)
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.ParamErrorCode, bizErr.Code)

	fresh, err := client.Project.Get(ctx, target.ID)
	require.NoError(t, err)
	assert.Equal(t, deptA.ID, fresh.DepartmentID)
	assert.Equal(t, managerA.ID, fresh.ManagerID)

	// 跨租户更新目标：查不到，不得写入
	newName := "hijacked"
	_, err = projectService.UpdateProject(ctx, target.ID, &newName, nil, nil, nil, tenantB.ID)
	require.Error(t, err)
	require.True(t, ent.IsNotFound(err))

	// 同租户更新成功；0 表示清空可选引用
	cleared := 0
	updated, err := projectService.UpdateProject(ctx, target.ID, &newName, nil, &cleared, &cleared, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Zero(t, updated.DepartmentID)
	assert.Zero(t, updated.ManagerID)
}

func TestProjectService_DeleteProject(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
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

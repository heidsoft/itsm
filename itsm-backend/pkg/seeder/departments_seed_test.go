package seeder

import (
	"testing"

	"itsm-backend/ent/department"
	"itsm-backend/ent/tenant"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedDepartmentsBuildsConfiguredHierarchy(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	require.NoError(t, seeder.seedDepartments(ctx))
	require.NoError(t, seeder.seedDepartments(ctx), "repeat run must not duplicate rows")

	all, err := seeder.client.Department.Query().Where(department.TenantIDEQ(root.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(seeder.config.Departments))

	idByCode := make(map[string]int, len(all))
	parentByCode := make(map[string]int, len(all))
	for _, d := range all {
		idByCode[d.Code] = d.ID
		parentByCode[d.Code] = d.ParentID
	}
	children := 0
	for _, spec := range seeder.config.Departments {
		if spec.ParentCode == "" {
			assert.Zero(t, parentByCode[spec.Code], "root %s must stay parentless", spec.Code)
			continue
		}
		children++
		require.Contains(t, idByCode, spec.ParentCode)
		assert.Equal(t, idByCode[spec.ParentCode], parentByCode[spec.Code], "child %s parent", spec.Code)
	}
	require.NotZero(t, children, "the baseline must actually exercise parent links")
}

// An installed tenant has a flat department tree from the previous
// implementation. Reconciling must backfill parents without recreating rows
// or overwriting customer edits.
func TestSeedDepartmentsRepairsFlatTreeWithoutOverwritingCustomerEdits(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	var renamedID int
	for _, spec := range seeder.config.Departments {
		saved, err := seeder.client.Department.Create().
			SetName("Customer " + spec.Name).
			SetCode(spec.Code).
			SetDescription("customer owned").
			SetTenantID(root.ID).
			Save(ctx)
		require.NoError(t, err)
		if spec.Code == "IT-INFRA" {
			renamedID = saved.ID
		}
	}
	require.NotZero(t, renamedID)

	require.NoError(t, seeder.seedDepartments(ctx))

	all, err := seeder.client.Department.Query().Where(department.TenantIDEQ(root.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(seeder.config.Departments), "repair must not create duplicates")

	idByCode := make(map[string]int, len(all))
	parentByCode := make(map[string]int, len(all))
	for _, d := range all {
		idByCode[d.Code] = d.ID
		parentByCode[d.Code] = d.ParentID
		if d.Code == "IT-INFRA" {
			assert.Equal(t, "Customer IT基础架构", d.Name, "customer rename must survive")
			assert.Equal(t, renamedID, d.ID)
		}
	}
	assert.Equal(t, idByCode["IT"], parentByCode["IT-INFRA"], "flat parent link must be backfilled")
}

func TestSeedDepartmentsRejectsUnknownParentCode(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	seeder.config.Departments = []DepartmentSeed{
		{Name: "Orphan", Code: "ORPHAN", ParentCode: "DOES-NOT-EXIST"},
	}

	err := seeder.seedDepartments(ctx)
	require.ErrorContains(t, err, "unknown parent_code DOES-NOT-EXIST")
}

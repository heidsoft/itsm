package service

import (
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/enttest"
)

func TestListTemplates_DevApprovalOpsFlow(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:ent?mode=memory&_fk=1")
	defer client.Close()

	svc := NewBPMNTemplateService(client)
	templates, err := svc.listTemplates()
	require.NoError(t, err)

	var found bool
	for _, tpl := range templates {
		if tpl.ID == "dev_approval_ops_flow" {
			found = true
			assert.Equal(t, "开发审批运维流程", tpl.Name)
			assert.Equal(t, "ticket", tpl.Category)
			assert.Equal(t, "approval", tpl.SubCategory)
			assert.Equal(t, "开发提交→主管审批→运维操作三级流程", tpl.Description)
			break
		}
	}
	assert.True(t, found, "dev_approval_ops_flow template should be listed")
}

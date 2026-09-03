package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Exercise the registered decision handler with the production engine and database.
// Numeric route IDs must identify the same primary key during validation and completion.
func TestSubmitTaskDecision_NumericIdentityCollision(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:decision_identity?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	deployment := client.ProcessDeployment.Create().SetDeploymentID("decision-deploy").SetDeploymentName("test").SetDeploymentTime(time.Now()).SetDeployedBy("test").SetTenantID(1).SaveX(ctx)
	definition := client.ProcessDefinition.Create().SetKey("decision-process").SetName("test").SetVersion("1").SetBpmnXML([]byte("<definitions/>")).SetDeploymentID(deployment.ID).SetDeployedAt(time.Now()).SetTenantID(1).SaveX(ctx)
	instance := client.ProcessInstance.Create().SetProcessInstanceID("decision-instance").SetProcessDefinitionKey(definition.Key).SetProcessDefinitionID(definition.ID).SetTenantID(1).SaveX(ctx)
	target := client.ProcessTask.Create().SetTaskID("target-task").SetTaskDefinitionKey("approval").SetTaskName("Target").SetProcessDefinitionKey(definition.Key).SetProcessInstanceID(instance.ID).SetTenantID(1).SetTaskVariables(map[string]interface{}{"commentRequiredOnReject": true}).SaveX(ctx)
	decoy := client.ProcessTask.Create().SetTaskID(strconv.Itoa(target.ID)).SetTaskDefinitionKey("decoy").SetTaskName("Decoy").SetProcessDefinitionKey(definition.Key).SetProcessInstanceID(instance.ID).SetTenantID(1).SaveX(ctx)
	handler := NewWorkflowHandler(service.NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()), nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		tenant := 1
		if c.GetHeader("X-Test-Tenant") == "2" {
			tenant = 2
		}
		c.Set("tenant_id", tenant)
		c.Set("user_id", 7)
	})
	handler.RegisterRoutes(r.Group("/api/v1"))
	for _, tenant := range []int{1, 2} {
		w := doAuthedRequest(t, r, http.MethodPost, "/api/v1/bpmn/tasks/"+strconv.Itoa(target.ID)+"/decisions", map[string]interface{}{"action": "reject"}, tenant, 7)
		status, code := http.StatusBadRequest, 1001
		if tenant == 2 {
			status, code = http.StatusNotFound, 4004
		}
		require.Equal(t, status, w.Code, w.Body.String())
		var response struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, code, response.Code)
		if tenant == 1 {
			assert.Equal(t, "该审批节点要求拒绝时填写意见", response.Message)
		}
	}
	assert.Equal(t, target.Status, client.ProcessTask.GetX(ctx, target.ID).Status)
	assert.Equal(t, decoy.Status, client.ProcessTask.GetX(ctx, decoy.ID).Status)
	assert.Zero(t, client.ProcessApprovalDecision.Query().CountX(ctx))
}

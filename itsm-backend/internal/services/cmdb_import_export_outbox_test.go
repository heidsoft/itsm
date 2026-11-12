package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cmdbexporttask"
	"itsm-backend/ent/cmdbimporttask"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func cmdbJobFixture(t *testing.T) (*ent.Client, context.Context, *CMDBImportExportService, int, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	ctx := context.Background()
	tenantEntity, err := client.Tenant.Create().SetName("CMDB Tenant").SetCode("cmdb-job").
		SetDomain("cmdb-job.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	operator, err := client.User.Create().SetUsername("cmdb-operator").SetEmail("cmdb@example.com").
		SetName("CMDB Operator").SetPasswordHash("hash").SetRole("admin").SetTenantID(tenantEntity.ID).Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenantEntity.ID).Save(ctx)
	require.NoError(t, err)
	logger := zap.NewNop().Sugar()
	history := NewCIHistoryService(client, logger)
	tags := NewCITagService(client, logger)
	ciService := NewConfigurationItemService(client, logger, history, tags)
	return client, ctx, NewCMDBImportExportService(client, logger, ciService, tags), tenantEntity.ID, operator.ID, ciType.ID
}

func TestCreateImportTaskAndWorkerProcessAreDurable(t *testing.T) {
	client, ctx, service, tenantID, operatorID, ciTypeID := cmdbJobFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/csv")
		_, _ = fmt.Fprintf(response, "Name,CI Type ID,Status,Asset Tag\napi-server,%d,active,asset-001\n", ciTypeID)
	}))
	defer server.Close()

	created, err := service.CreateImportTask(ctx, &dto.ImportCIRequest{
		FileURL: server.URL, FileType: "csv", UpdateMode: "skip",
	}, tenantID, operatorID, "CMDB Operator")
	require.NoError(t, err)
	require.Equal(t, "pending", created.Status)
	ciCount, err := client.ConfigurationItem.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ciCount)
	command, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandProcessCMDBImport),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "cmdb_import_task", command.AggregateType)

	registry := commandbus.NewRegistry()
	require.NoError(t, registry.Register(commandbus.CommandProcessCMDBImport, service.HandleImportCommand))
	worker := commandbus.NewWorker(client, registry, zap.NewNop().Sugar(), "cmdb-worker")
	processed, err := worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	ciCount, err = client.ConfigurationItem.Query().Where(
		configurationitem.TenantIDEQ(tenantID), configurationitem.AssetTagEQ("asset-001"),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, ciCount)
	task, err := client.CMDBImportTask.Query().Where(
		cmdbimporttask.TaskIDEQ(created.TaskID), cmdbimporttask.TenantIDEQ(tenantID),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.Equal(t, 1, task.SuccessCount)
	storedCommand, err := client.OperationalCommand.Get(ctx, command.ID)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusSucceeded, storedCommand.Status)

	// Simulate recovery after an ambiguous worker outcome. The stable asset key
	// reconciles the row instead of creating a duplicate CI.
	_, err = client.CMDBImportTask.UpdateOneID(task.ID).SetStatus("pending").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, service.HandleImportCommand(ctx, command))
	ciCount, err = client.ConfigurationItem.Query().Where(
		configurationitem.TenantIDEQ(tenantID), configurationitem.AssetTagEQ("asset-001"),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, ciCount)
}

func TestCreateExportTaskSchedulesCommandAndCompletedJobIsIdempotent(t *testing.T) {
	client, ctx, service, tenantID, operatorID, _ := cmdbJobFixture(t)
	created, err := service.CreateExportTask(ctx, &dto.ExportCIRequest{ExportType: "csv"}, tenantID, operatorID, "CMDB Operator")
	require.NoError(t, err)
	command, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandProcessCMDBExport),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "cmdb_export_task", command.AggregateType)
	_, err = client.CMDBExportTask.Update().Where(
		cmdbexporttask.TaskIDEQ(created.TaskID), cmdbexporttask.TenantIDEQ(tenantID),
	).SetStatus("completed").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, service.HandleExportCommand(ctx, command))
}

func TestCMDBJobCommandRejectsCrossTenantAggregate(t *testing.T) {
	client, ctx, service, tenantID, operatorID, _ := cmdbJobFixture(t)
	created, err := service.CreateImportTask(ctx, &dto.ImportCIRequest{
		FileURL: "https://example.invalid/import.csv", FileType: "csv", UpdateMode: "skip",
	}, tenantID, operatorID, "CMDB Operator")
	require.NoError(t, err)
	task, err := client.CMDBImportTask.Query().Where(cmdbimporttask.TaskIDEQ(created.TaskID)).Only(ctx)
	require.NoError(t, err)
	command := &ent.OperationalCommand{
		TenantID: tenantID + 1, CommandType: commandbus.CommandProcessCMDBImport,
		AggregateType: "cmdb_import_task", AggregateID: task.ID,
		Payload: map[string]interface{}{"taskId": task.TaskID},
	}
	require.Error(t, service.HandleImportCommand(ctx, command))
	task, err = client.CMDBImportTask.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", task.Status)
}

func TestParseCIRowRequiresStableReconciliationIdentity(t *testing.T) {
	service := &CMDBImportExportService{}
	request, fieldErrors := service.parseCIRow(
		map[string]string{"Name": "api-server", "CI Type ID": "42"},
		service.buildFieldMap(), map[string]int{"42": 42},
	)
	require.Equal(t, "api-server", request.Name)
	require.Contains(t, fieldErrors, &FieldError{
		Field:   "资产标签/序列号/云资源标识",
		Message: "至少提供资产标签、序列号，或云厂商与云资源ID，以支持幂等导入",
	})
}

func TestSafeCMDBTaskErrorDoesNotPersistProviderDetails(t *testing.T) {
	require.Equal(t, "下载文件失败", safeCMDBTaskError(
		"下载文件失败: Get https://storage.example/file?token=secret: timeout",
	))
	require.Equal(t, "任务执行失败", safeCMDBTaskError("  "))
}

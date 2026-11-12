// test-coverage-guard: skip
package service

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/cmdbexporttask"
	"itsm-backend/ent/cmdbimporttask"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/internal/commandbus"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

const (
	cmdbTaskTimeout       = 30 * time.Minute
	maxCMDBImportFileSize = 50 << 20 // 50 MiB
	cmdbExportMaxRecords  = 10000    // 单次导出上限，DTO 同步下发供前端与页面 COUNT 对账
	cmdbExportRetention   = 7 * 24 * time.Hour
)

var cmdbImportHTTPClient = &http.Client{Timeout: 2 * time.Minute}

// CMDBImportExportService handles asynchronous CMDB import/export tasks.
type CMDBImportExportService struct {
	client     *ent.Client
	logger     *zap.SugaredLogger
	ciService  *ConfigurationItemService
	tagService *CITagService
}

// HandleImportCommand reloads the authoritative Job under the command tenant.
// Completed jobs are idempotent no-ops; failed/interrupted jobs are resumed by
// the same command identity and remain visible through the Job API.
func (s *CMDBImportExportService) HandleImportCommand(ctx context.Context, command *ent.OperationalCommand) error {
	if command == nil || command.CommandType != commandbus.CommandProcessCMDBImport || command.AggregateType != "cmdb_import_task" {
		return fmt.Errorf("invalid CMDB import command")
	}
	task, err := s.client.CMDBImportTask.Query().Where(
		cmdbimporttask.IDEQ(command.AggregateID), cmdbimporttask.TenantIDEQ(command.TenantID),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("load CMDB import job: %w", err)
	}
	if payloadTaskID, _ := command.Payload["taskId"].(string); payloadTaskID != task.TaskID {
		return fmt.Errorf("CMDB import command identity mismatch")
	}
	if task.Status == "completed" {
		return nil
	}
	request := &dto.ImportCIRequest{
		FileURL: task.FileURL, FileType: task.FileType, UpdateMode: task.UpdateMode, SheetName: task.SheetName,
	}
	s.processImportTask(ctx, task.TaskID, request, task.TenantID, task.OperatorID, task.OperatorName)
	status, err := s.client.CMDBImportTask.Query().Where(
		cmdbimporttask.IDEQ(task.ID), cmdbimporttask.TenantIDEQ(command.TenantID),
	).Select(cmdbimporttask.FieldStatus).String(ctx)
	if err != nil {
		return fmt.Errorf("reload CMDB import job: %w", err)
	}
	if status != "completed" {
		return fmt.Errorf("CMDB import job did not complete")
	}
	return nil
}

func (s *CMDBImportExportService) HandleExportCommand(ctx context.Context, command *ent.OperationalCommand) error {
	if command == nil || command.CommandType != commandbus.CommandProcessCMDBExport || command.AggregateType != "cmdb_export_task" {
		return fmt.Errorf("invalid CMDB export command")
	}
	task, err := s.client.CMDBExportTask.Query().Where(
		cmdbexporttask.IDEQ(command.AggregateID), cmdbexporttask.TenantIDEQ(command.TenantID),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("load CMDB export job: %w", err)
	}
	if payloadTaskID, _ := command.Payload["taskId"].(string); payloadTaskID != task.TaskID {
		return fmt.Errorf("CMDB export command identity mismatch")
	}
	if task.Status == "completed" {
		return nil
	}
	filterJSON, err := json.Marshal(task.Filters)
	if err != nil {
		return fmt.Errorf("decode CMDB export filters")
	}
	var filters dto.CISearchFilter
	if err := json.Unmarshal(filterJSON, &filters); err != nil {
		return fmt.Errorf("decode CMDB export filters")
	}
	request := &dto.ExportCIRequest{Filters: &filters, ExportType: task.ExportType, ExportFields: task.ExportFields}
	s.processExportTask(ctx, task.TaskID, request, task.TenantID)
	status, err := s.client.CMDBExportTask.Query().Where(
		cmdbexporttask.IDEQ(task.ID), cmdbexporttask.TenantIDEQ(command.TenantID),
	).Select(cmdbexporttask.FieldStatus).String(ctx)
	if err != nil {
		return fmt.Errorf("reload CMDB export job: %w", err)
	}
	if status != "completed" {
		return fmt.Errorf("CMDB export job did not complete")
	}
	return nil
}

// NewCMDBImportExportService 创建CMDB导入导出服务
func NewCMDBImportExportService(client *ent.Client, logger *zap.SugaredLogger, ciService *ConfigurationItemService, tagService *CITagService) *CMDBImportExportService {
	return &CMDBImportExportService{
		client:     client,
		logger:     logger,
		ciService:  ciService,
		tagService: tagService,
	}
}

// CreateImportTask 创建导入任务
func (s *CMDBImportExportService) CreateImportTask(ctx context.Context, req *dto.ImportCIRequest, tenantID int, operatorID int, operatorName string) (*dto.ImportCIResult, error) {
	taskID := uuid.NewString()

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建导入任务事务失败: %w", err)
	}
	defer tx.Rollback()
	// Job 和调度命令必须在同一事务提交。
	task, err := tx.CMDBImportTask.Create().
		SetTaskID(taskID).
		SetFileURL(req.FileURL).
		SetFileType(req.FileType).
		SetUpdateMode(req.UpdateMode).
		SetNillableSheetName(&req.SheetName).
		SetTenantID(tenantID).
		SetOperatorID(operatorID).
		SetOperatorName(operatorName).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create import task", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("创建导入任务失败: %w", err)
	}
	if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandProcessCMDBImport,
		AggregateType: "cmdb_import_task", AggregateID: task.ID,
		IdempotencyKey: fmt.Sprintf("cmdb-import:%s:process", taskID),
		Payload:        map[string]interface{}{"taskId": taskID}, MaxAttempts: 8,
	}); err != nil {
		return nil, fmt.Errorf("创建导入任务调度命令失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交导入任务失败: %w", err)
	}

	return s.convertToImportDTO(task), nil
}

// GetImportTaskStatus 获取导入任务状态
func (s *CMDBImportExportService) GetImportTaskStatus(ctx context.Context, taskID string, tenantID int) (*dto.ImportCIResult, error) {
	task, err := s.client.CMDBImportTask.Query().
		Where(
			cmdbimporttask.TaskID(taskID),
			cmdbimporttask.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("导入任务不存在")
		}
		s.logger.Errorw("Failed to get import task", "error", err, "task_id", taskID)
		return nil, fmt.Errorf("获取导入任务失败: %w", err)
	}

	return s.convertToImportDTO(task), nil
}

// ListImportTasks 获取导入任务列表
func (s *CMDBImportExportService) ListImportTasks(ctx context.Context, tenantID int, page, pageSize int) (*dto.ListResponse[dto.ImportCIResult], error) {
	page, pageSize = common.ValidatePagination(page, pageSize)

	query := s.client.CMDBImportTask.Query().
		Where(cmdbimporttask.TenantID(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count import tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("统计导入任务失败: %w", err)
	}

	tasks, err := query.
		Offset((page-1)*pageSize).
		Limit(pageSize).
		Order(ent.Desc(cmdbimporttask.FieldCreatedAt), ent.Asc(cmdbimporttask.FieldID)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list import tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("获取导入任务列表失败: %w", err)
	}

	items := make([]dto.ImportCIResult, len(tasks))
	for i, task := range tasks {
		items[i] = *s.convertToImportDTO(task)
	}

	pagination := common.NewPaginationResponse(page, pageSize, int64(total))
	return &dto.ListResponse[dto.ImportCIResult]{
		Items:      items,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}

// processImportTask 处理导入任务
func (s *CMDBImportExportService) processImportTask(parent context.Context, taskID string, req *dto.ImportCIRequest, tenantID int, operatorID int, operatorName string) {
	ctx, cancel := context.WithTimeout(parent, cmdbTaskTimeout)
	defer cancel()

	// 更新任务状态为处理中
	_, err := s.client.CMDBImportTask.Update().
		Where(cmdbimporttask.TaskID(taskID), cmdbimporttask.TenantID(tenantID)).
		SetStatus("processing").
		SetStartedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update import task status to processing", "error", err, "task_id", taskID)
		return
	}

	var records [][]string
	var headers []string

	// 下载文件
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.FileURL, nil)
	if err != nil {
		s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("创建下载请求失败: %v", err))
		return
	}
	resp, err := cmdbImportHTTPClient.Do(httpReq)
	if err != nil {
		s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("下载文件失败: %v", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("下载文件失败: HTTP %d", resp.StatusCode))
		return
	}
	if resp.ContentLength > maxCMDBImportFileSize {
		s.failImportTask(ctx, taskID, tenantID, "导入文件超过 50 MiB 限制")
		return
	}
	limitedBody := http.MaxBytesReader(nil, resp.Body, maxCMDBImportFileSize)

	// 解析文件
	if req.FileType == "csv" {
		reader := csv.NewReader(limitedBody)
		records, err = reader.ReadAll()
		if err != nil {
			s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("解析CSV文件失败: %v", err))
			return
		}
	} else if req.FileType == "xlsx" {
		// 先保存到临时文件
		tmpFile, err := os.CreateTemp("", "cmdb-import-*.xlsx")
		if err != nil {
			s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("创建临时文件失败: %v", err))
			return
		}
		defer os.Remove(tmpFile.Name())

		_, err = io.Copy(tmpFile, limitedBody)
		if err != nil {
			tmpFile.Close()
			s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("保存Excel文件失败: %v", err))
			return
		}
		tmpFile.Close()

		// 读取Excel
		rows, readErr := safeExcelizeGetRows(tmpFile.Name(), req.SheetName)
		if readErr != nil {
			s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("读取Excel Sheet失败: %v", readErr))
			return
		}
		records = rows
	}

	if len(records) < 2 {
		s.failImportTask(ctx, taskID, tenantID, "文件中没有有效数据")
		return
	}

	// 第一行是表头
	headers = records[0]
	dataRows := records[1:]

	successCount := 0
	failedCount := 0
	var errors []dto.CIImportError

	// 字段映射：表头名称到CI字段名
	fieldMap := s.buildFieldMap()

	// 预处理CI类型映射
	ciTypeMap, err := s.buildCITypeMap(ctx, tenantID)
	if err != nil {
		s.failImportTask(ctx, taskID, tenantID, fmt.Sprintf("获取CI类型失败: %v", err))
		return
	}

	// 逐行处理数据
	for rowIdx, row := range dataRows {
		rowNumber := rowIdx + 2 // 行号从2开始算，因为第一行是表头
		ciData := make(map[string]string)

		// 构建行数据
		for colIdx, header := range headers {
			if colIdx < len(row) {
				ciData[strings.TrimSpace(header)] = strings.TrimSpace(row[colIdx])
			}
		}

		// 转换为创建请求
		createReq, parseErrs := s.parseCIRow(ciData, fieldMap, ciTypeMap)
		if len(parseErrs) > 0 {
			failedCount++
			for _, e := range parseErrs {
				errors = append(errors, dto.CIImportError{
					RowNumber: rowNumber,
					FieldName: e.Field,
					Message:   e.Message,
				})
			}
			continue
		}

		// 检查CI是否已存在（根据资产标签或序列号）
		existingCI, err := s.findExistingCI(ctx, createReq, tenantID)
		if err != nil {
			failedCount++
			errors = append(errors, dto.CIImportError{
				RowNumber: rowNumber,
				FieldName: "",
				Message:   fmt.Sprintf("检查CI是否存在失败: %v", err),
			})
			continue
		}

		if existingCI != nil {
			// CI已存在，根据更新模式处理
			switch req.UpdateMode {
			case "skip":
				// 跳过
				continue
			case "overwrite":
				// 覆盖更新
				updateReq := s.convertCreateToUpdateReq(createReq)
				_, err = s.ciService.UpdateCI(ctx, existingCI.ID, tenantID, updateReq)
			case "merge":
				// 合并更新，只更新非空字段
				updateReq := s.convertCreateToUpdateReqMerge(createReq, existingCI)
				_, err = s.ciService.UpdateCI(ctx, existingCI.ID, tenantID, updateReq)
			}
		} else {
			// 创建新CI
			_, err = s.ciService.CreateCI(ctx, createReq, tenantID)
		}

		if err != nil {
			failedCount++
			errors = append(errors, dto.CIImportError{
				RowNumber: rowNumber,
				FieldName: "",
				Message:   fmt.Sprintf("保存CI失败: %v", err),
			})
			continue
		}

		successCount++
	}

	// 更新任务结果
	errMsg := ""
	if len(errors) > 0 {
		errJSON, _ := json.Marshal(errors)
		errMsg = string(errJSON)
	}
	_ = errMsg // errMsg may be used in future error reporting

	errorList := make([]map[string]interface{}, len(errors))
	for i, e := range errors {
		errorList[i] = map[string]interface{}{
			"row_number": e.RowNumber,
			"field_name": e.FieldName,
			"message":    e.Message,
		}
	}

	_, err = s.client.CMDBImportTask.Update().
		Where(cmdbimporttask.TaskID(taskID), cmdbimporttask.TenantID(tenantID)).
		SetStatus("completed").
		SetTotalCount(len(dataRows)).
		SetSuccessCount(successCount).
		SetFailedCount(failedCount).
		SetErrors(errorList).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update import task result", "error", err, "task_id", taskID)
	}

	s.logger.Infow("Import task completed", "task_id", taskID, "total", len(dataRows), "success", successCount, "failed", failedCount)
}

// CreateExportTask 创建导出任务
func (s *CMDBImportExportService) CreateExportTask(ctx context.Context, req *dto.ExportCIRequest, tenantID int, operatorID int, operatorName string) (*dto.ExportCIResult, error) {
	// R2：expires_at 写入后原本全仓无比较点，临时文件只增不减。
	// 创建新任务时顺带回收本租户已过期文件（best-effort，失败不阻塞导出）。
	s.cleanupExpiredExportFiles(ctx, tenantID)
	taskID := uuid.NewString()

	filtersJSON, err := json.Marshal(req.Filters)
	if err != nil {
		s.logger.Errorw("Failed to marshal export filters", "error", err)
		return nil, fmt.Errorf("序列化过滤条件失败: %w", err)
	}

	var filtersMap map[string]interface{}
	err = json.Unmarshal(filtersJSON, &filtersMap)
	if err != nil {
		s.logger.Errorw("Failed to unmarshal export filters", "error", err)
		return nil, fmt.Errorf("反序列化过滤条件失败: %w", err)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建导出任务事务失败: %w", err)
	}
	defer tx.Rollback()
	// Job 和调度命令必须在同一事务提交。
	task, err := tx.CMDBExportTask.Create().
		SetTaskID(taskID).
		SetFilters(filtersMap).
		SetExportFields(req.ExportFields).
		SetExportType(req.ExportType).
		SetTenantID(tenantID).
		SetOperatorID(operatorID).
		SetOperatorName(operatorName).
		SetExpiresAt(time.Now().Add(cmdbExportRetention)). // 7天后过期
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create export task", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("创建导出任务失败: %w", err)
	}
	if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandProcessCMDBExport,
		AggregateType: "cmdb_export_task", AggregateID: task.ID,
		IdempotencyKey: fmt.Sprintf("cmdb-export:%s:process", taskID),
		Payload:        map[string]interface{}{"taskId": taskID}, MaxAttempts: 8,
	}); err != nil {
		return nil, fmt.Errorf("创建导出任务调度命令失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交导出任务失败: %w", err)
	}

	return s.convertToExportDTO(task), nil
}

// GetExportTaskStatus 获取导出任务状态
func (s *CMDBImportExportService) GetExportTaskStatus(ctx context.Context, taskID string, tenantID int) (*dto.ExportCIResult, error) {
	task, err := s.client.CMDBExportTask.Query().
		Where(
			cmdbexporttask.TaskID(taskID),
			cmdbexporttask.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("导出任务不存在")
		}
		s.logger.Errorw("Failed to get export task", "error", err, "task_id", taskID)
		return nil, fmt.Errorf("获取导出任务失败: %w", err)
	}

	return s.convertToExportDTO(task), nil
}

// ListExportTasks 获取导出任务列表
func (s *CMDBImportExportService) ListExportTasks(ctx context.Context, tenantID int, page, pageSize int) (*dto.ListResponse[dto.ExportCIResult], error) {
	page, pageSize = common.ValidatePagination(page, pageSize)

	query := s.client.CMDBExportTask.Query().
		Where(cmdbexporttask.TenantID(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count export tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("统计导出任务失败: %w", err)
	}

	tasks, err := query.
		Offset((page-1)*pageSize).
		Limit(pageSize).
		Order(ent.Desc(cmdbexporttask.FieldCreatedAt), ent.Asc(cmdbexporttask.FieldID)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list export tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("获取导出任务列表失败: %w", err)
	}

	items := make([]dto.ExportCIResult, len(tasks))
	for i, task := range tasks {
		items[i] = *s.convertToExportDTO(task)
	}

	pagination := common.NewPaginationResponse(page, pageSize, int64(total))
	return &dto.ListResponse[dto.ExportCIResult]{
		Items:      items,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}

// processExportTask 处理导出任务
func (s *CMDBImportExportService) processExportTask(parent context.Context, taskID string, req *dto.ExportCIRequest, tenantID int) {
	ctx, cancel := context.WithTimeout(parent, cmdbTaskTimeout)
	defer cancel()

	// 更新任务状态为处理中
	_, err := s.client.CMDBExportTask.Update().
		Where(cmdbexporttask.TaskID(taskID), cmdbexporttask.TenantID(tenantID)).
		SetStatus("processing").
		SetStartedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update export task status to processing", "error", err, "task_id", taskID)
		return
	}

	// 查询所有符合条件的CI
	filters := dto.CISearchFilter{}
	if req.Filters != nil {
		filters = *req.Filters
	}

	// 导出上限 1 万条，但页长归平台单一所有者夹紧（MaxPageSize=100），不能一次性索要 1 万：
	// 按页读到上限。把 PageSize: 10000 直接下传会被服务层静默夹紧，
	// 导出结果远少于用户勾选的范围。
	items := make([]dto.CIResponse, 0, common.MaxPageSize)
	availableTotal := 0
	for page := 1; ; page++ {
		searchReq := &dto.CISearchRequest{
			Filters:   filters,
			Page:      page,
			PageSize:  common.MaxPageSize,
			SortBy:    "id",
			SortOrder: "asc",
		}
		pageRes, err := s.ciService.SearchCI(ctx, tenantID, searchReq)
		if err != nil {
			s.failExportTask(ctx, taskID, tenantID, fmt.Sprintf("查询CI数据失败: %v", err))
			return
		}
		if page == 1 {
			availableTotal = pageRes.Total
		}
		items = append(items, pageRes.Items...)
		if len(pageRes.Items) < pageRes.PageSize || len(items) >= cmdbExportMaxRecords {
			break
		}
	}
	if len(items) > cmdbExportMaxRecords {
		items = items[:cmdbExportMaxRecords]
	}
	if availableTotal > len(items) && s.logger != nil {
		s.logger.Warnw("CMDB export truncated by per-task limit",
			"task_id", taskID, "tenant_id", tenantID,
			"available_total", availableTotal, "exported", len(items), "limit", cmdbExportMaxRecords)
	}

	if len(items) == 0 {
		s.failExportTask(ctx, taskID, tenantID, "没有符合条件的CI数据")
		return
	}

	// 确定导出字段
	exportFields := req.ExportFields
	if len(exportFields) == 0 {
		exportFields = s.getDefaultExportFields()
	}

	var fileURL string
	var fileSize int64

	// 生成导出文件
	// Convert []dto.CIResponse to []*dto.CIResponse for export functions
	ciPtrs := make([]*dto.CIResponse, len(items))
	for i := range items {
		ciPtrs[i] = &items[i]
	}
	if req.ExportType == "csv" {
		fileURL, fileSize, err = s.generateCSVExport(ciPtrs, exportFields, tenantID, taskID)
	} else {
		fileURL, fileSize, err = s.generateExcelExport(ciPtrs, exportFields, tenantID, taskID)
	}

	if err != nil {
		s.failExportTask(ctx, taskID, tenantID, fmt.Sprintf("生成导出文件失败: %v", err))
		return
	}

	// 更新任务结果
	_, err = s.client.CMDBExportTask.Update().
		Where(cmdbexporttask.TaskID(taskID), cmdbexporttask.TenantID(tenantID)).
		SetStatus("completed").
		SetTotalCount(len(items)).
		// DB 只存物理文件名（basename）：绝对路径绝不出库，
		// 下载端点再按 tenantID/operatorID/expires_at 校验后拼回隔离目录。
		SetFileURL(filepath.Base(fileURL)).
		SetFileSize(fileSize).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update export task result", "error", err, "task_id", taskID)
	}

	s.logger.Infow("Export task completed", "task_id", taskID, "count", len(items), "file_size", fileSize)
}

// failImportTask 标记导入任务失败
func (s *CMDBImportExportService) failImportTask(ctx context.Context, taskID string, tenantID int, errMsg string) {
	safeMessage := safeCMDBTaskError(errMsg)
	_, err := s.client.CMDBImportTask.Update().
		Where(cmdbimporttask.TaskID(taskID), cmdbimporttask.TenantIDEQ(tenantID)).
		SetStatus("failed").
		SetErrorMessage(safeMessage).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to mark import task as failed", "error", err, "task_id", taskID)
	}
	s.logger.Errorw("Import task failed", "task_id", taskID, "error", safeMessage)
}

// failExportTask 标记导出任务失败
func (s *CMDBImportExportService) failExportTask(ctx context.Context, taskID string, tenantID int, errMsg string) {
	safeMessage := safeCMDBTaskError(errMsg)
	_, err := s.client.CMDBExportTask.Update().
		Where(cmdbexporttask.TaskID(taskID), cmdbexporttask.TenantIDEQ(tenantID)).
		SetStatus("failed").
		SetErrorMessage(safeMessage).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to mark export task as failed", "error", err, "task_id", taskID)
	}
	s.logger.Errorw("Export task failed", "task_id", taskID, "error", safeMessage)
}

func safeCMDBTaskError(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "任务执行失败"
	}
	if prefix, _, found := strings.Cut(message, ":"); found {
		message = prefix
	}
	const maxMessageLength = 200
	runes := []rune(message)
	if len(runes) > maxMessageLength {
		message = string(runes[:maxMessageLength])
	}
	return message
}

// buildFieldMap 构建表头到字段的映射
func (s *CMDBImportExportService) buildFieldMap() map[string]string {
	return map[string]string{
		"名称":     "name",
		"CI类型ID": "ci_type_id",
		"CI类型":   "ci_type",
		"状态":     "status",
		"环境":     "environment",
		"重要级别":   "criticality",
		"资产标签":   "asset_tag",
		"序列号":    "serial_number",
		"型号":     "model",
		"厂商":     "vendor",
		"位置":     "location",
		"负责人":    "assigned_to",
		"归属人":    "owned_by",
		"发现源":    "discovery_source",
		"来源":     "source",
		"云厂商":    "cloud_provider",
		"云账号ID":  "cloud_account_id",
		"云区域":    "cloud_region",
		"云可用区":   "cloud_zone",
		"云资源ID":  "cloud_resource_id",
		"云资源类型":  "cloud_resource_type",
		"生命周期状态": "lifecycle_status",
		"生效时间":   "effective_at",
		"失效时间":   "expire_at",
		// 英文别名
		"Name":                "name",
		"CI Type ID":          "ci_type_id",
		"CI Type":             "ci_type",
		"Status":              "status",
		"Environment":         "environment",
		"Criticality":         "criticality",
		"Asset Tag":           "asset_tag",
		"Serial Number":       "serial_number",
		"Model":               "model",
		"Vendor":              "vendor",
		"Location":            "location",
		"Assigned To":         "assigned_to",
		"Owned By":            "owned_by",
		"Discovery Source":    "discovery_source",
		"Source":              "source",
		"Cloud Provider":      "cloud_provider",
		"Cloud Account ID":    "cloud_account_id",
		"Cloud Region":        "cloud_region",
		"Cloud Zone":          "cloud_zone",
		"Cloud Resource ID":   "cloud_resource_id",
		"Cloud Resource Type": "cloud_resource_type",
		"Lifecycle Status":    "lifecycle_status",
		"Effective At":        "effective_at",
		"Expire At":           "expire_at",
	}
}

// buildCITypeMap 构建CI类型名称到ID的映射
func (s *CMDBImportExportService) buildCITypeMap(ctx context.Context, tenantID int) (map[string]int, error) {
	ciTypes, err := s.client.CIType.Query().
		Where(citype.TenantID(tenantID), citype.IsActive(true)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	typeMap := make(map[string]int)
	for _, ct := range ciTypes {
		typeMap[strings.ToLower(ct.Name)] = ct.ID
		// 也支持ID直接映射
		typeMap[strconv.Itoa(ct.ID)] = ct.ID
	}

	return typeMap, nil
}

// parseCIRow 解析CI行数据
func (s *CMDBImportExportService) parseCIRow(row map[string]string, fieldMap map[string]string, ciTypeMap map[string]int) (*dto.CreateCIRequest, []*FieldError) {
	var errors []*FieldError
	req := &dto.CreateCIRequest{}

	for header, value := range row {
		field, ok := fieldMap[header]
		if !ok {
			// 尝试忽略大小写匹配
			for h, f := range fieldMap {
				if strings.EqualFold(h, header) {
					field = f
					ok = true
					break
				}
			}
			if !ok {
				// 未知字段，忽略
				continue
			}
		}

		if value == "" {
			continue
		}

		switch field {
		case "name":
			req.Name = value
		case "ci_type_id":
			// 尝试转换为ID
			id, err := strconv.Atoi(value)
			if err == nil {
				req.CITypeID = id
			} else {
				// 尝试按名称查找
				id, ok := ciTypeMap[strings.ToLower(value)]
				if ok {
					req.CITypeID = id
				} else {
					errors = append(errors, &FieldError{
						Field:   header,
						Message: fmt.Sprintf("无效的CI类型: %s", value),
					})
				}
			}
		case "status":
			req.Status = value
		case "environment":
			req.Environment = value
		case "criticality":
			req.Criticality = value
		case "asset_tag":
			req.AssetTag = value
		case "serial_number":
			req.SerialNumber = value
		case "model":
			req.Model = value
		case "vendor":
			req.Vendor = value
		case "location":
			req.Location = value
		case "assigned_to":
			req.AssignedTo = value
		case "owned_by":
			req.OwnedBy = value
		case "discovery_source":
			req.DiscoverySource = value
		case "source":
			req.Source = value
		case "cloud_provider":
			req.CloudProvider = value
		case "cloud_account_id":
			req.CloudAccountID = value
		case "cloud_region":
			req.CloudRegion = value
		case "cloud_zone":
			req.CloudZone = value
		case "cloud_resource_id":
			req.CloudResourceID = value
		case "cloud_resource_type":
			req.CloudResourceType = value
		case "lifecycle_status":
			req.LifecycleStatus = value
		case "effective_at":
			t, err := time.Parse(time.RFC3339, value)
			if err == nil {
				req.EffectiveAt = &t
			} else {
				// 尝试其他常见格式
				t, err = time.Parse("2006-01-02", value)
				if err == nil {
					req.EffectiveAt = &t
				}
			}
		case "expire_at":
			t, err := time.Parse(time.RFC3339, value)
			if err == nil {
				req.ExpireAt = &t
			} else {
				t, err = time.Parse("2006-01-02", value)
				if err == nil {
					req.ExpireAt = &t
				}
			}
		}
	}

	// 校验必填字段
	if req.Name == "" {
		errors = append(errors, &FieldError{
			Field:   "名称",
			Message: "CI名称不能为空",
		})
	}
	if req.CITypeID == 0 {
		errors = append(errors, &FieldError{
			Field:   "CI类型ID/CI类型",
			Message: "CI类型不能为空",
		})
	}
	if req.Status == "" {
		req.Status = "active" // 默认状态
	}
	// Durable retries must reconcile against a stable, tenant-scoped identity.
	// Name is mutable and cannot safely prevent duplicate CIs after a worker
	// crash between the CI write and the Job progress update.
	if req.AssetTag == "" && req.SerialNumber == "" && (req.CloudProvider == "" || req.CloudResourceID == "") {
		errors = append(errors, &FieldError{
			Field:   "资产标签/序列号/云资源标识",
			Message: "至少提供资产标签、序列号，或云厂商与云资源ID，以支持幂等导入",
		})
	}

	return req, errors
}

// findExistingCI 查找已存在的CI
func (s *CMDBImportExportService) findExistingCI(ctx context.Context, req *dto.CreateCIRequest, tenantID int) (*ent.ConfigurationItem, error) {
	// 优先按资产标签查找
	if req.AssetTag != "" {
		ci, err := s.client.ConfigurationItem.Query().
			Where(
				configurationitem.AssetTag(req.AssetTag),
				configurationitem.TenantID(tenantID),
			).
			First(ctx)
		if err == nil {
			return ci, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}

	// 其次按序列号查找
	if req.SerialNumber != "" {
		ci, err := s.client.ConfigurationItem.Query().
			Where(
				configurationitem.SerialNumber(req.SerialNumber),
				configurationitem.TenantID(tenantID),
			).
			First(ctx)
		if err == nil {
			return ci, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}

	// 最后按云资源ID查找
	if req.CloudResourceID != "" && req.CloudProvider != "" {
		ci, err := s.client.ConfigurationItem.Query().
			Where(
				configurationitem.CloudResourceID(req.CloudResourceID),
				configurationitem.CloudProvider(req.CloudProvider),
				configurationitem.TenantID(tenantID),
			).
			First(ctx)
		if err == nil {
			return ci, nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}

	return nil, nil
}

// convertCreateToUpdateReq 将创建请求转换为更新请求
func (s *CMDBImportExportService) convertCreateToUpdateReq(createReq *dto.CreateCIRequest) *dto.UpdateCIRequest {
	return &dto.UpdateCIRequest{
		CITypeID:           createReq.CITypeID,
		Name:               createReq.Name,
		Status:             createReq.Status,
		Environment:        createReq.Environment,
		Criticality:        createReq.Criticality,
		AssetTag:           createReq.AssetTag,
		SerialNumber:       createReq.SerialNumber,
		Model:              createReq.Model,
		Vendor:             createReq.Vendor,
		Location:           createReq.Location,
		AssignedTo:         createReq.AssignedTo,
		OwnedBy:            createReq.OwnedBy,
		DiscoverySource:    createReq.DiscoverySource,
		Source:             createReq.Source,
		Attributes:         createReq.Attributes,
		CloudProvider:      createReq.CloudProvider,
		CloudAccountID:     createReq.CloudAccountID,
		CloudRegion:        createReq.CloudRegion,
		CloudZone:          createReq.CloudZone,
		CloudResourceID:    createReq.CloudResourceID,
		CloudResourceType:  createReq.CloudResourceType,
		CloudMetadata:      createReq.CloudMetadata,
		CloudTags:          createReq.CloudTags,
		CloudMetrics:       createReq.CloudMetrics,
		CloudSyncTime:      createReq.CloudSyncTime,
		CloudSyncStatus:    createReq.CloudSyncStatus,
		CloudResourceRefID: createReq.CloudResourceRefID,
		LifecycleStatus:    createReq.LifecycleStatus,
		EffectiveAt:        createReq.EffectiveAt,
		ExpireAt:           createReq.ExpireAt,
	}
}

// convertCreateToUpdateReqMerge 将创建请求转换为合并更新请求，只更新非空字段
func (s *CMDBImportExportService) convertCreateToUpdateReqMerge(createReq *dto.CreateCIRequest, existingCI *ent.ConfigurationItem) *dto.UpdateCIRequest {
	updateReq := &dto.UpdateCIRequest{}

	if createReq.CITypeID != 0 {
		updateReq.CITypeID = createReq.CITypeID
	}
	if createReq.Name != "" {
		updateReq.Name = createReq.Name
	}
	if createReq.Status != "" {
		updateReq.Status = createReq.Status
	}
	if createReq.Environment != "" {
		updateReq.Environment = createReq.Environment
	}
	if createReq.Criticality != "" {
		updateReq.Criticality = createReq.Criticality
	}
	if createReq.AssetTag != "" {
		updateReq.AssetTag = createReq.AssetTag
	}
	if createReq.SerialNumber != "" {
		updateReq.SerialNumber = createReq.SerialNumber
	}
	if createReq.Model != "" {
		updateReq.Model = createReq.Model
	}
	if createReq.Vendor != "" {
		updateReq.Vendor = createReq.Vendor
	}
	if createReq.Location != "" {
		updateReq.Location = createReq.Location
	}
	if createReq.AssignedTo != "" {
		updateReq.AssignedTo = createReq.AssignedTo
	}
	if createReq.OwnedBy != "" {
		updateReq.OwnedBy = createReq.OwnedBy
	}
	if createReq.DiscoverySource != "" {
		updateReq.DiscoverySource = createReq.DiscoverySource
	}
	if createReq.Source != "" {
		updateReq.Source = createReq.Source
	}
	if createReq.Attributes != nil {
		// 合并属性
		mergedAttrs := make(map[string]interface{})
		for k, v := range existingCI.Attributes {
			mergedAttrs[k] = v
		}
		for k, v := range createReq.Attributes {
			mergedAttrs[k] = v
		}
		updateReq.Attributes = mergedAttrs
	}
	if createReq.CloudProvider != "" {
		updateReq.CloudProvider = createReq.CloudProvider
	}
	if createReq.CloudAccountID != "" {
		updateReq.CloudAccountID = createReq.CloudAccountID
	}
	if createReq.CloudRegion != "" {
		updateReq.CloudRegion = createReq.CloudRegion
	}
	if createReq.CloudZone != "" {
		updateReq.CloudZone = createReq.CloudZone
	}
	if createReq.CloudResourceID != "" {
		updateReq.CloudResourceID = createReq.CloudResourceID
	}
	if createReq.CloudResourceType != "" {
		updateReq.CloudResourceType = createReq.CloudResourceType
	}
	if createReq.CloudMetadata != nil {
		updateReq.CloudMetadata = createReq.CloudMetadata
	}
	if createReq.CloudTags != nil {
		updateReq.CloudTags = createReq.CloudTags
	}
	if createReq.CloudMetrics != nil {
		updateReq.CloudMetrics = createReq.CloudMetrics
	}
	if createReq.CloudSyncTime != nil {
		updateReq.CloudSyncTime = createReq.CloudSyncTime
	}
	if createReq.CloudSyncStatus != "" {
		updateReq.CloudSyncStatus = createReq.CloudSyncStatus
	}
	if createReq.CloudResourceRefID != 0 {
		updateReq.CloudResourceRefID = createReq.CloudResourceRefID
	}
	if createReq.LifecycleStatus != "" {
		updateReq.LifecycleStatus = createReq.LifecycleStatus
	}
	if createReq.EffectiveAt != nil {
		updateReq.EffectiveAt = createReq.EffectiveAt
	}
	if createReq.ExpireAt != nil {
		updateReq.ExpireAt = createReq.ExpireAt
	}

	return updateReq
}

// getDefaultExportFields 获取默认导出字段
func (s *CMDBImportExportService) getDefaultExportFields() []string {
	return []string{
		"id", "name", "ci_type", "status", "environment", "criticality",
		"asset_tag", "serial_number", "model", "vendor", "location",
		"assigned_to", "owned_by", "discovery_source", "source",
		"cloud_provider", "cloud_region", "cloud_resource_id",
		"lifecycle_status", "effective_at", "expire_at",
		"created_at", "updated_at",
	}
}

// sanitizeCMDBTaskID 把任务号收敛成文件名安全字符，防止 `../` 路径穿越。
func sanitizeCMDBTaskID(taskID string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, taskID)
	if len(safe) > 64 {
		safe = safe[:64]
	}
	if safe == "" {
		safe = "unknown"
	}
	return safe
}

// cmdbExportDir 每个租户一个 0700 私有目录：即便 DB 里的文件名被搏改，
// 也只能落到当前租户自己的目录，无法跨租户读文件。
func cmdbExportDir(tenantID int) (string, error) {
	if tenantID <= 0 {
		return "", fmt.Errorf("invalid CMDB export tenant id")
	}
	dir := filepath.Join(os.TempDir(), "itsm-exports", fmt.Sprintf("t%d", tenantID))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create CMDB export directory: %w", err)
	}
	return dir, nil
}

// cmdbExportFilePath 生成租户隔离、任务唯一、不可预测的导出文件路径。
//
// R3：旧文件名只有秒级时间戳 `cmdb-export-20060102-150405.csv`，两个任务
// （可不同租户/不同过滤条件）同秒生成时后写直接 os.Create 截断先写，
// 前一个任务的 file_url 会指向后一个任务的内容。文件名带 tenantID + taskID
// + 8 字节随机数，落盘用 O_EXCL 独占创建；taskID 逐字符白名单化。
func cmdbExportFilePath(tenantID int, taskID, ext string) (string, error) {
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate export file nonce: %w", err)
	}
	dir, err := cmdbExportDir(tenantID)
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("cmdb-export-t%d-%s-%s.%s", tenantID, sanitizeCMDBTaskID(taskID), hex.EncodeToString(nonce), ext)
	return filepath.Join(dir, fileName), nil
}

// resolveCMDBExportFilePath 把 DB 里存的 basename 还原成物理路径。
// 只接受本租户目录下的普通文件名：不接受路径分隔符、不接受非白名单扩展名，
// 因此下载端点无法被用来读任意服务器文件。
func resolveCMDBExportFilePath(tenantID int, storedName string) (string, error) {
	if storedName == "" || storedName != filepath.Base(storedName) || strings.ContainsAny(storedName, `/\`) {
		return "", common.NewBadRequestError("导出文件名非法", nil)
	}
	if !strings.HasPrefix(storedName, fmt.Sprintf("cmdb-export-t%d-", tenantID)) {
		return "", common.NewBadRequestError("导出文件不属于当前租户", nil)
	}
	if ext := filepath.Ext(storedName); ext != ".csv" && ext != ".xlsx" {
		return "", common.NewBadRequestError("导出文件类型非法", nil)
	}
	dir, err := cmdbExportDir(tenantID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, storedName), nil
}

// generateCSVExport 生成CSV导出文件
func (s *CMDBImportExportService) generateCSVExport(items []*dto.CIResponse, fields []string, tenantID int, taskID string) (string, int64, error) {
	filePath, err := cmdbExportFilePath(tenantID, taskID, "csv")
	if err != nil {
		return "", 0, err
	}
	// O_EXCL：即便随机数撞车也拒绝覆盖既有任务文件，而不是静默复用。
	// 0600：导出内容含 CI 资产清单，不应对同机其他用户可读。
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	// UTF-8 BOM：Excel 直接打开中文表头不乱码（与工单导出同一口径）。
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return "", 0, err
	}
	writer := csv.NewWriter(file)

	// 写入表头
	headers := s.convertFieldsToHeaders(fields)
	if err := writer.Write(headers); err != nil {
		return "", 0, err
	}

	// 写入数据
	for _, item := range items {
		row := sanitizeSpreadsheetRow(s.convertCIToRow(item, fields))
		if err := writer.Write(row); err != nil {
			return "", 0, err
		}
	}

	// csv.Writer 自带缓冲：旧实现把 Flush 放在 defer 里却在它之前 Stat，
	// 于是 file_size 恒为 0，任务看着 completed、大小却对不上。必须先显式
	// Flush 并检查错误，再取文件大小。
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", 0, err
	}
	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	// 返回实际落盘路径（os.TempDir() 不保证是 /tmp，旧写法硬编 /tmp/ 会指错）。
	// 调用方只把 basename 入库，对外 URL 由下载端点生成。
	return filePath, stat.Size(), nil
}

// generateExcelExport 生成Excel导出文件
func (s *CMDBImportExportService) generateExcelExport(items []*dto.CIResponse, fields []string, tenantID int, taskID string) (string, int64, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "CI数据"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", 0, err
	}
	f.SetActiveSheet(index)

	// 写入表头
	headers := s.convertFieldsToHeaders(fields)
	for colIdx, header := range headers {
		cellName, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cellName, header)
	}

	// 写入数据
	for rowIdx, item := range items {
		row := sanitizeSpreadsheetRow(s.convertCIToRow(item, fields))
		for colIdx, value := range row {
			cellName, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cellName, value)
		}
	}

	// 保存文件：与 CSV 同一套唯一命名 + O_EXCL 占位，避免 SaveAs 覆盖同名文件。
	filePath, err := cmdbExportFilePath(tenantID, taskID, "xlsx")
	if err != nil {
		return "", 0, err
	}
	placeholder, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", 0, err
	}
	if err := placeholder.Close(); err != nil {
		return "", 0, err
	}
	if err := f.SaveAs(filePath); err != nil {
		_ = os.Remove(filePath)
		return "", 0, err
	}

	// 获取文件大小
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", 0, err
	}

	// 同 CSV：返回实际落盘路径，不硬编 /tmp/。
	return filePath, stat.Size(), nil
}

// convertFieldsToHeaders 转换字段名为中文表头
func (s *CMDBImportExportService) convertFieldsToHeaders(fields []string) []string {
	fieldToHeader := map[string]string{
		"id":                  "ID",
		"name":                "名称",
		"ci_type":             "CI类型",
		"ci_type_id":          "CI类型ID",
		"status":              "状态",
		"environment":         "环境",
		"criticality":         "重要级别",
		"asset_tag":           "资产标签",
		"serial_number":       "序列号",
		"model":               "型号",
		"vendor":              "厂商",
		"location":            "位置",
		"assigned_to":         "负责人",
		"owned_by":            "归属人",
		"discovery_source":    "发现源",
		"source":              "来源",
		"cloud_provider":      "云厂商",
		"cloud_account_id":    "云账号ID",
		"cloud_region":        "云区域",
		"cloud_zone":          "云可用区",
		"cloud_resource_id":   "云资源ID",
		"cloud_resource_type": "云资源类型",
		"lifecycle_status":    "生命周期状态",
		"effective_at":        "生效时间",
		"expire_at":           "失效时间",
		"created_at":          "创建时间",
		"updated_at":          "更新时间",
	}

	headers := make([]string, len(fields))
	for i, field := range fields {
		if header, ok := fieldToHeader[field]; ok {
			headers[i] = header
		} else {
			headers[i] = field
		}
	}

	return headers
}

// convertCIToRow 将CI转换为导出行
func (s *CMDBImportExportService) convertCIToRow(ci *dto.CIResponse, fields []string) []string {
	row := make([]string, len(fields))

	ciValue := reflect.ValueOf(ci).Elem()
	ciType := ciValue.Type()

	for i, field := range fields {
		// 先尝试查找json tag
		var fieldValue reflect.Value
		found := false
		for j := 0; j < ciType.NumField(); j++ {
			structField := ciType.Field(j)
			tag := structField.Tag.Get("json")
			if strings.Split(tag, ",")[0] == field {
				fieldValue = ciValue.Field(j)
				found = true
				break
			}
		}

		if !found {
			row[i] = ""
			continue
		}

		switch fieldValue.Kind() {
		case reflect.String:
			row[i] = fieldValue.String()
		case reflect.Int, reflect.Int32, reflect.Int64:
			row[i] = strconv.FormatInt(fieldValue.Int(), 10)
		case reflect.Bool:
			row[i] = strconv.FormatBool(fieldValue.Bool())
		case reflect.Ptr:
			if !fieldValue.IsNil() {
				switch t := fieldValue.Elem().Interface().(type) {
				case time.Time:
					row[i] = t.Format(time.RFC3339)
				default:
					row[i] = fmt.Sprintf("%v", t)
				}
			} else {
				row[i] = ""
			}
		case reflect.Struct:
			if t, ok := fieldValue.Interface().(time.Time); ok {
				row[i] = t.Format(time.RFC3339)
			} else {
				row[i] = fmt.Sprintf("%v", fieldValue.Interface())
			}
		default:
			row[i] = fmt.Sprintf("%v", fieldValue.Interface())
		}
	}

	return row
}

// convertToImportDTO 转换为导入DTO
func (s *CMDBImportExportService) convertToImportDTO(task *ent.CMDBImportTask) *dto.ImportCIResult {
	res := &dto.ImportCIResult{
		TaskID:       task.TaskID,
		Status:       task.Status,
		TotalCount:   task.TotalCount,
		SuccessCount: task.SuccessCount,
		FailedCount:  task.FailedCount,
		CreatedAt:    task.CreatedAt,
		CompletedAt:  &task.CompletedAt,
	}

	// 转换错误信息
	if len(task.Errors) > 0 {
		errors := make([]dto.CIImportError, len(task.Errors))
		for i, e := range task.Errors {
			errors[i] = dto.CIImportError{
				RowNumber: int(e["row_number"].(float64)),
				FieldName: e["field_name"].(string),
				Message:   e["message"].(string),
			}
		}
		res.Errors = errors
	}

	return res
}

// convertToExportDTO 转换为导出DTO。
// R2：绝不把服务器绝对路径（旧实现 `/tmp/xxx.csv`）回给前端，
// completed 任务改为回下载端点 URL，由端点做归属与过期校验。
func (s *CMDBImportExportService) convertToExportDTO(task *ent.CMDBExportTask) *dto.ExportCIResult {
	res := &dto.ExportCIResult{
		TaskID:      task.TaskID,
		Status:      task.Status,
		TotalCount:  task.TotalCount,
		FileSize:    task.FileSize,
		CreatedAt:   task.CreatedAt,
		CompletedAt: &task.CompletedAt,
		ExpiresAt:   task.ExpiresAt,
	}
	if task.Status == "completed" && task.FileURL != "" {
		res.FileURL = fmt.Sprintf("/api/v1/cmdb/export/%s/download", url.PathEscape(task.TaskID))
		res.ExportLimit = cmdbExportMaxRecords
		// 保守提示：达到上限即标记可能截断（恰好等于上限时也提示），
		// 避免用户把截断后的 totalCount 当成页面 COUNT 的全量。
		res.Truncated = task.TotalCount >= cmdbExportMaxRecords
	}
	return res
}

// ResolveExportDownload 校验下载资格并还原物理路径。
// 四重校验：任务必须属于当前租户 → 必须是当前操作人创建 → 已完成且未过期
// → 文件名必须能还原到本租户隔离目录且文件真实存在。
func (s *CMDBImportExportService) ResolveExportDownload(ctx context.Context, taskID string, tenantID, operatorID int) (string, string, error) {
	task, err := s.client.CMDBExportTask.Query().Where(
		cmdbexporttask.TaskID(taskID),
		cmdbexporttask.TenantID(tenantID),
	).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", "", common.NewNotFoundError("导出任务")
		}
		return "", "", fmt.Errorf("查询导出任务失败: %w", err)
	}
	if task.OperatorID != operatorID {
		return "", "", common.NewForbiddenError("无权下载其他操作人创建的导出文件")
	}
	if task.Status != "completed" {
		return "", "", common.NewStateConflictError(
			fmt.Sprintf("导出任务尚未完成（当前 %s），无法下载", task.Status), task.Status)
	}
	if !task.ExpiresAt.IsZero() && time.Now().After(task.ExpiresAt) {
		return "", "", common.NewForbiddenError("导出文件已过期，请重新导出")
	}
	filePath, err := resolveCMDBExportFilePath(tenantID, task.FileURL)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return "", "", common.NewNotFoundError("导出文件")
	}
	downloadName := fmt.Sprintf("cmdb-export-%s%s", sanitizeCMDBTaskID(taskID), filepath.Ext(filePath))
	return filePath, downloadName, nil
}

// cleanupExpiredExportFiles 回收本租户已过期导出文件。
// 只删除能成功还原到本租户私有目录的文件，避免误删或路径穿越。
func (s *CMDBImportExportService) cleanupExpiredExportFiles(ctx context.Context, tenantID int) {
	if s.client == nil || tenantID <= 0 {
		return
	}
	dir, err := cmdbExportDir(tenantID)
	if err != nil {
		return
	}
	tasks, err := s.client.CMDBExportTask.Query().Where(
		cmdbexporttask.TenantID(tenantID),
	).All(ctx)
	if err != nil {
		return
	}
	for _, task := range tasks {
		if task.FileURL == "" || task.ExpiresAt.IsZero() || time.Now().Before(task.ExpiresAt) {
			continue
		}
		path, err := resolveCMDBExportFilePath(tenantID, task.FileURL)
		if err != nil || filepath.Dir(path) != dir {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) && s.logger != nil {
			s.logger.Warnw("failed to remove expired CMDB export file",
				"task_id", task.TaskID, "tenant_id", tenantID, "error", err)
		}
	}
}

// FieldError 字段错误
type FieldError struct {
	Field   string
	Message string
}

// safeExcelizeGetRows wraps excelize OpenFile + GetRows with panic recovery.
// GO-2026-6452: excelize v2.11.0 can panic on malformed shared-string indices;
// this converts the panic into a returned error.
func safeExcelizeGetRows(filePath, sheetName string) (rows [][]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			rows = nil
			err = fmt.Errorf("excelize panic: %v", r)
		}
	}()

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开Excel文件失败: %w", err)
	}
	defer f.Close()

	if sheetName == "" {
		sheetName = f.GetSheetName(0)
	}

	rows, err = f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("读取Sheet失败: %w", err)
	}
	return rows, nil
}

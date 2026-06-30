package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/cmdbexporttask"
	"itsm-backend/ent/cmdbimporttask"
	"itsm-backend/ent/configurationitem"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// CMDBImportExportService handles asynchronous CMDB import/export tasks.
type CMDBImportExportService struct {
	client     *ent.Client
	logger     *zap.SugaredLogger
	ciService  *ConfigurationItemService
	tagService *CITagService
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

	// 创建导入任务记录
	task, err := s.client.CMDBImportTask.Create().
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
		s.logger.Errorw("Failed to create import task", "error", err, "tenant_id", tenantID, "file_url", req.FileURL)
		return nil, fmt.Errorf("创建导入任务失败: %w", err)
	}

	// 异步执行导入
	go s.processImportTask(taskID, req, tenantID, operatorID, operatorName)

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
	query := s.client.CMDBImportTask.Query().
		Where(cmdbimporttask.TenantID(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count import tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("统计导入任务失败: %w", err)
	}

	tasks, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(cmdbimporttask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list import tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("获取导入任务列表失败: %w", err)
	}

	items := make([]dto.ImportCIResult, len(tasks))
	for i, task := range tasks {
		items[i] = *s.convertToImportDTO(task)
	}

	return &dto.ListResponse[dto.ImportCIResult]{
		Items: items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// processImportTask 处理导入任务
func (s *CMDBImportExportService) processImportTask(taskID string, req *dto.ImportCIRequest, tenantID int, operatorID int, operatorName string) {
	ctx := context.Background()

	// 更新任务状态为处理中
	_, err := s.client.CMDBImportTask.Update().
		Where(cmdbimporttask.TaskID(taskID)).
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
	resp, err := http.Get(req.FileURL)
	if err != nil {
		s.failImportTask(ctx, taskID, fmt.Sprintf("下载文件失败: %v", err))
		return
	}
	defer resp.Body.Close()

	// 解析文件
	if req.FileType == "csv" {
		reader := csv.NewReader(resp.Body)
		records, err = reader.ReadAll()
		if err != nil {
			s.failImportTask(ctx, taskID, fmt.Sprintf("解析CSV文件失败: %v", err))
			return
		}
	} else if req.FileType == "xlsx" {
		// 先保存到临时文件
		tmpFile, err := os.CreateTemp("", "cmdb-import-*.xlsx")
		if err != nil {
			s.failImportTask(ctx, taskID, fmt.Sprintf("创建临时文件失败: %v", err))
			return
		}
		defer os.Remove(tmpFile.Name())

		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			tmpFile.Close()
			s.failImportTask(ctx, taskID, fmt.Sprintf("保存Excel文件失败: %v", err))
			return
		}
		tmpFile.Close()

		// 读取Excel
		f, err := excelize.OpenFile(tmpFile.Name())
		if err != nil {
			s.failImportTask(ctx, taskID, fmt.Sprintf("打开Excel文件失败: %v", err))
			return
		}
		defer f.Close()

		sheetName := req.SheetName
		if sheetName == "" {
			sheetName = f.GetSheetName(0)
		}

		rows, err := f.GetRows(sheetName)
		if err != nil {
			s.failImportTask(ctx, taskID, fmt.Sprintf("读取Excel Sheet失败: %v", err))
			return
		}
		records = rows
	}

	if len(records) < 2 {
		s.failImportTask(ctx, taskID, "文件中没有有效数据")
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
		s.failImportTask(ctx, taskID, fmt.Sprintf("获取CI类型失败: %v", err))
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

		var ci *dto.CIResponse

		if existingCI != nil {
			// CI已存在，根据更新模式处理
			switch req.UpdateMode {
			case "skip":
				// 跳过
				_ = ci
				continue
			case "overwrite":
				// 覆盖更新
				updateReq := s.convertCreateToUpdateReq(createReq)
				ci, err = s.ciService.UpdateCI(ctx, existingCI.ID, tenantID, updateReq)
			case "merge":
				// 合并更新，只更新非空字段
				updateReq := s.convertCreateToUpdateReqMerge(createReq, existingCI)
				ci, err = s.ciService.UpdateCI(ctx, existingCI.ID, tenantID, updateReq)
			}
		} else {
			// 创建新CI
			ci, err = s.ciService.CreateCI(ctx, createReq, tenantID)
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
		Where(cmdbimporttask.TaskID(taskID)).
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

	// 创建导出任务记录
	task, err := s.client.CMDBExportTask.Create().
		SetTaskID(taskID).
		SetFilters(filtersMap).
		SetExportFields(req.ExportFields).
		SetExportType(req.ExportType).
		SetTenantID(tenantID).
		SetOperatorID(operatorID).
		SetOperatorName(operatorName).
		SetExpiresAt(time.Now().Add(7 * 24 * time.Hour)). // 7天后过期
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create export task", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("创建导出任务失败: %w", err)
	}

	// 异步执行导出
	go s.processExportTask(taskID, req, tenantID)

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
	query := s.client.CMDBExportTask.Query().
		Where(cmdbexporttask.TenantID(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count export tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("统计导出任务失败: %w", err)
	}

	tasks, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(cmdbexporttask.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list export tasks", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("获取导出任务列表失败: %w", err)
	}

	items := make([]dto.ExportCIResult, len(tasks))
	for i, task := range tasks {
		items[i] = *s.convertToExportDTO(task)
	}

	return &dto.ListResponse[dto.ExportCIResult]{
		Items: items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// processExportTask 处理导出任务
func (s *CMDBImportExportService) processExportTask(taskID string, req *dto.ExportCIRequest, tenantID int) {
	ctx := context.Background()

	// 更新任务状态为处理中
	_, err := s.client.CMDBExportTask.Update().
		Where(cmdbexporttask.TaskID(taskID)).
		SetStatus("processing").
		SetStartedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update export task status to processing", "error", err, "task_id", taskID)
		return
	}

	// 查询所有符合条件的CI
	searchReq := &dto.CISearchRequest{
		Filters:   *req.Filters,
		Page:      1,
		PageSize:  10000, // 最多导出1万条
		SortBy:    "id",
		SortOrder: "asc",
	}

	searchResult, err := s.ciService.SearchCI(ctx, tenantID, searchReq)
	if err != nil {
		s.failExportTask(ctx, taskID, fmt.Sprintf("查询CI数据失败: %v", err))
		return
	}

	if len(searchResult.Items) == 0 {
		s.failExportTask(ctx, taskID, "没有符合条件的CI数据")
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
	ciPtrs := make([]*dto.CIResponse, len(searchResult.Items))
	for i := range searchResult.Items {
		ciPtrs[i] = &searchResult.Items[i]
	}
	if req.ExportType == "csv" {
		fileURL, fileSize, err = s.generateCSVExport(ciPtrs, exportFields)
	} else {
		fileURL, fileSize, err = s.generateExcelExport(ciPtrs, exportFields)
	}

	if err != nil {
		s.failExportTask(ctx, taskID, fmt.Sprintf("生成导出文件失败: %v", err))
		return
	}

	// 更新任务结果
	_, err = s.client.CMDBExportTask.Update().
		Where(cmdbexporttask.TaskID(taskID)).
		SetStatus("completed").
		SetTotalCount(len(searchResult.Items)).
		SetFileURL(fileURL).
		SetFileSize(fileSize).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update export task result", "error", err, "task_id", taskID)
	}

	s.logger.Infow("Export task completed", "task_id", taskID, "count", len(searchResult.Items), "file_size", fileSize)
}

// failImportTask 标记导入任务失败
func (s *CMDBImportExportService) failImportTask(ctx context.Context, taskID string, errMsg string) {
	_, err := s.client.CMDBImportTask.Update().
		Where(cmdbimporttask.TaskID(taskID)).
		SetStatus("failed").
		SetErrorMessage(errMsg).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to mark import task as failed", "error", err, "task_id", taskID)
	}
	s.logger.Errorw("Import task failed", "task_id", taskID, "error", errMsg)
}

// failExportTask 标记导出任务失败
func (s *CMDBImportExportService) failExportTask(ctx context.Context, taskID string, errMsg string) {
	_, err := s.client.CMDBExportTask.Update().
		Where(cmdbexporttask.TaskID(taskID)).
		SetStatus("failed").
		SetErrorMessage(errMsg).
		SetCompletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to mark export task as failed", "error", err, "task_id", taskID)
	}
	s.logger.Errorw("Export task failed", "task_id", taskID, "error", errMsg)
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
		req.Status = "operational" // 默认状态
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

// generateCSVExport 生成CSV导出文件
func (s *CMDBImportExportService) generateCSVExport(items []*dto.CIResponse, fields []string) (string, int64, error) {
	// 创建临时文件
	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("cmdb-export-%s.csv", time.Now().Format("20060102-150405"))
	filePath := filepath.Join(tmpDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := s.convertFieldsToHeaders(fields)
	if err := writer.Write(headers); err != nil {
		return "", 0, err
	}

	// 写入数据
	for _, item := range items {
		row := s.convertCIToRow(item, fields)
		if err := writer.Write(row); err != nil {
			return "", 0, err
		}
	}

	// 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	// 这里应该返回可访问的URL，暂时返回文件路径
	// 实际生产环境应该上传到对象存储
	return fmt.Sprintf("/tmp/%s", fileName), stat.Size(), nil
}

// generateExcelExport 生成Excel导出文件
func (s *CMDBImportExportService) generateExcelExport(items []*dto.CIResponse, fields []string) (string, int64, error) {
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
		row := s.convertCIToRow(item, fields)
		for colIdx, value := range row {
			cellName, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cellName, value)
		}
	}

	// 保存文件
	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("cmdb-export-%s.xlsx", time.Now().Format("20060102-150405"))
	filePath := filepath.Join(tmpDir, fileName)

	if err := f.SaveAs(filePath); err != nil {
		return "", 0, err
	}

	// 获取文件大小
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", 0, err
	}

	// 这里应该返回可访问的URL，暂时返回文件路径
	return fmt.Sprintf("/tmp/%s", fileName), stat.Size(), nil
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

// convertToExportDTO 转换为导出DTO
func (s *CMDBImportExportService) convertToExportDTO(task *ent.CMDBExportTask) *dto.ExportCIResult {
	return &dto.ExportCIResult{
		TaskID:      task.TaskID,
		Status:      task.Status,
		FileURL:     task.FileURL,
		TotalCount:  task.TotalCount,
		FileSize:    task.FileSize,
		CreatedAt:   task.CreatedAt,
		CompletedAt: &task.CompletedAt,
		ExpiresAt:   task.ExpiresAt,
	}
}

// FieldError 字段错误
type FieldError struct {
	Field   string
	Message string
}

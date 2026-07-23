package service

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/configurationitemhistory"

	"go.uber.org/zap"
)

// CIHistoryService CI历史服务
type CIHistoryService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCIHistoryService 创建CI历史服务
func NewCIHistoryService(client *ent.Client, logger *zap.SugaredLogger) *CIHistoryService {
	return &CIHistoryService{
		client: client,
		logger: logger,
	}
}

// RecordCIHistory 记录CI变更历史
func (s *CIHistoryService) RecordCIHistory(
	ctx context.Context,
	ciID, tenantID, operatorID int,
	operatorName, operation, remark string,
	before, after *ent.ConfigurationItem,
) error {
	// 获取当前最大版本号
	// 使用 Scan + sql.NullInt64 兜底：MAX 在无记录时返回 NULL，直接 Int(ctx) 会 "converting NULL to int" 500
	var aggResult []struct {
		Max sql.NullInt64 `json:"max"`
	}
	err := s.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.TenantIDEQ(tenantID),
		).
		Aggregate(ent.Max(configurationitemhistory.FieldVersion)).
		Scan(ctx, &aggResult)
	if err != nil {
		s.logger.Errorw("Failed to get max CI version", "error", err, "ci_id", ciID)
		return fmt.Errorf("failed to get max version: %w", err)
	}
	version := 1
	if len(aggResult) > 0 && aggResult[0].Max.Valid && aggResult[0].Max.Int64 > 0 {
		version = int(aggResult[0].Max.Int64) + 1
	}

	// 转换before和after为map
	beforeMap := s.ciToMap(before)
	afterMap := s.ciToMap(after)

	// 计算变更的字段
	changedFields := s.getChangedFields(beforeMap, afterMap)

	_, err = s.client.ConfigurationItemHistory.Create().
		SetCiID(ciID).
		SetVersion(version).
		SetOperation(operation).
		SetBefore(beforeMap).
		SetAfter(afterMap).
		SetChangedFields(changedFields).
		SetOperatorID(operatorID).
		SetOperatorName(operatorName).
		SetRemark(remark).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to record CI history", "error", err, "ci_id", ciID, "operation", operation)
		return fmt.Errorf("failed to record history: %w", err)
	}

	return nil
}

// GetCIHistory 获取CI历史记录列表
func (s *CIHistoryService) GetCIHistory(ctx context.Context, ciID, tenantID int, page, pageSize int) (*dto.CIHistoryListResponse, error) {
	// 检查CI是否存在
	exists, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.IDEQ(ciID),
			configurationitem.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check CI existence", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to check CI existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("CI not found")
	}

	query := s.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.TenantIDEQ(tenantID),
		)

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI history", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to count history: %w", err)
	}

	histories, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(configurationitemhistory.FieldVersion)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI history", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to list history: %w", err)
	}

	return &dto.CIHistoryListResponse{
		Items: dto.ToCIHistoryResponseList(histories),
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// RevertCIVersion 回滚CI到指定版本
func (s *CIHistoryService) RevertCIVersion(ctx context.Context, ciID, tenantID, operatorID int, operatorName string, req *dto.RevertCIVersionRequest) (*dto.CIResponse, error) {
	// 获取指定版本的历史记录
	history, err := s.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.VersionEQ(req.Version),
			configurationitemhistory.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("version %d not found", req.Version)
		}
		s.logger.Errorw("Failed to get CI history version", "error", err, "ci_id", ciID, "version", req.Version)
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// 获取当前CI
	currentCI, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.IDEQ(ciID),
			configurationitem.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get current CI", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to get current CI: %w", err)
	}

	// 从历史记录的after字段恢复数据
	after := history.After
	update := s.client.ConfigurationItem.UpdateOneID(ciID)

	// 恢复字段
	if name, ok := after["name"].(string); ok {
		update.SetName(name)
	}
	if ciTypeID, ok := after["ci_type_id"].(int); ok {
		update.SetCiTypeID(ciTypeID)
	}
	if ciType, ok := after["ci_type"].(string); ok {
		update.SetCiType(ciType)
	}
	if status, ok := after["status"].(string); ok {
		update.SetStatus(status)
	}
	if environment, ok := after["environment"].(string); ok {
		update.SetEnvironment(environment)
	}
	if criticality, ok := after["criticality"].(string); ok {
		update.SetCriticality(criticality)
	}
	if assetTag, ok := after["asset_tag"].(string); ok {
		update.SetAssetTag(assetTag)
	}
	if serialNumber, ok := after["serial_number"].(string); ok {
		update.SetSerialNumber(serialNumber)
	}
	if model, ok := after["model"].(string); ok {
		update.SetModel(model)
	}
	if vendor, ok := after["vendor"].(string); ok {
		update.SetVendor(vendor)
	}
	if location, ok := after["location"].(string); ok {
		update.SetLocation(location)
	}
	if assignedTo, ok := after["assigned_to"].(string); ok {
		update.SetAssignedTo(assignedTo)
	}
	if ownedBy, ok := after["owned_by"].(string); ok {
		update.SetOwnedBy(ownedBy)
	}
	if discoverySource, ok := after["discovery_source"].(string); ok {
		update.SetDiscoverySource(discoverySource)
	}
	if source, ok := after["source"].(string); ok {
		update.SetSource(source)
	}
	if attributes, ok := after["attributes"].(map[string]interface{}); ok {
		update.SetAttributes(attributes)
	}
	if cloudProvider, ok := after["cloud_provider"].(string); ok {
		update.SetCloudProvider(cloudProvider)
	}
	if cloudAccountID, ok := after["cloud_account_id"].(string); ok {
		update.SetCloudAccountID(cloudAccountID)
	}
	if cloudRegion, ok := after["cloud_region"].(string); ok {
		update.SetCloudRegion(cloudRegion)
	}
	if cloudZone, ok := after["cloud_zone"].(string); ok {
		update.SetCloudZone(cloudZone)
	}
	if cloudResourceID, ok := after["cloud_resource_id"].(string); ok {
		update.SetCloudResourceID(cloudResourceID)
	}
	if cloudResourceType, ok := after["cloud_resource_type"].(string); ok {
		update.SetCloudResourceType(cloudResourceType)
	}
	if cloudMetadata, ok := after["cloud_metadata"].(map[string]interface{}); ok {
		update.SetCloudMetadata(cloudMetadata)
	}
	if cloudTags, ok := after["cloud_tags"].(map[string]interface{}); ok {
		update.SetCloudTags(cloudTags)
	}
	if cloudMetrics, ok := after["cloud_metrics"].(map[string]interface{}); ok {
		update.SetCloudMetrics(cloudMetrics)
	}
	if cloudSyncStatus, ok := after["cloud_sync_status"].(string); ok {
		update.SetCloudSyncStatus(cloudSyncStatus)
	}
	if cloudResourceRefID, ok := after["cloud_resource_ref_id"].(int); ok {
		update.SetCloudResourceRefID(cloudResourceRefID)
	}
	// tags字段在ConfigurationItem entity中不存在，跳过

	// 保存更新
	updatedCI, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to revert CI version", "error", err, "ci_id", ciID, "version", req.Version)
		return nil, fmt.Errorf("failed to revert version: %w", err)
	}

	// 记录回滚历史
	err = s.RecordCIHistory(
		ctx,
		ciID,
		tenantID,
		operatorID,
		operatorName,
		"revert",
		fmt.Sprintf("Reverted to version %d: %s", req.Version, req.Remark),
		currentCI,
		updatedCI,
	)
	if err != nil {
		s.logger.Warnw("Failed to record revert history", "error", err, "ci_id", ciID)
	}

	// 重新加载CI数据
	updatedCI, err = s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID)).
		WithOutgoingRelations().
		WithIncomingRelations().
		WithTags().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to reload reverted CI", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to reload CI: %w", err)
	}

	s.logger.Infow("CI reverted successfully", "ci_id", ciID, "version", req.Version, "tenant_id", tenantID)
	return dto.ToCIResponseWithRelations(updatedCI), nil
}

// ciToMap 将CI实体转换为map
func (s *CIHistoryService) ciToMap(ci *ent.ConfigurationItem) map[string]interface{} {
	if ci == nil {
		return nil
	}

	v := reflect.ValueOf(ci).Elem()
	t := v.Type()

	result := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 跳过私有字段和Edges字段
		if field.PkgPath != "" || field.Name == "Edges" {
			continue
		}
		value := v.Field(i).Interface()
		// 忽略nil和零值时间
		if str, ok := value.(string); ok && str == "" {
			continue
		}
		if tm, ok := value.(time.Time); ok && tm.IsZero() {
			continue
		}
		// 将camelCase字段名转换为snake_case（和Ent schema保持一致）
		snakeName := s.toSnakeCase(field.Name)
		result[snakeName] = value
	}

	return result
}

// getChangedFields 比较两个map，返回变更的字段列表
func (s *CIHistoryService) getChangedFields(before, after map[string]interface{}) []string {
	if before == nil {
		// 创建操作，所有字段都是变更的
		fields := make([]string, 0, len(after))
		for k := range after {
			fields = append(fields, k)
		}
		return fields
	}
	if after == nil {
		// 删除操作
		return []string{"all"}
	}

	changed := make([]string, 0)
	// 检查before中的字段是否在after中变更
	for k, vBefore := range before {
		vAfter, exists := after[k]
		if !exists || !reflect.DeepEqual(vBefore, vAfter) {
			changed = append(changed, k)
		}
	}
	// 检查after中新增的字段
	for k := range after {
		if _, exists := before[k]; !exists {
			changed = append(changed, k)
		}
	}

	return changed
}

// toSnakeCase 将驼峰命名转换为下划线命名
func (s *CIHistoryService) toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, []rune(strings.ToLower(string(r)))[0])
	}
	return string(result)
}

package service

import (
	"encoding/json"
	"fmt"

	"itsm-backend/dto"
)

// nodesToMaps 将强类型 []dto.ApprovalNodeConfig 转换为 Ent 存储所需的 []map[string]interface{}
// 这是在 Ent schema 尚未更新为强类型前的边界转换层
func nodesToMaps(configs []dto.ApprovalNodeConfig) ([]map[string]interface{}, error) {
	if len(configs) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 先序列化为 JSON，再反序列化为 map，确保类型一致性
	data, err := json.Marshal(configs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal approval nodes: %w", err)
	}

	var maps []map[string]interface{}
	if err := json.Unmarshal(data, &maps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approval nodes to maps: %w", err)
	}

	return maps, nil
}

// mapsToNodes 将 Ent 存储的 []map[string]interface{} 转换回强类型 []dto.ApprovalNodeConfig
// 所有类型断言集中在此函数，消费端无需再做任何断言
func mapsToNodes(maps []map[string]interface{}) ([]dto.ApprovalNodeConfig, error) {
	if len(maps) == 0 {
		return []dto.ApprovalNodeConfig{}, nil
	}

	// 先序列化为 JSON，再反序列化为强类型 struct
	data, err := json.Marshal(maps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal approval node maps: %w", err)
	}

	var configs []dto.ApprovalNodeConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approval nodes to configs: %w", err)
	}

	return configs, nil
}

// mapsToNodesUnsafe 与 mapsToNodes 相同，但出错时返回空切片而非 error
// 用于可以容忍解析失败的场景（如 canPerformAction 检查）
func mapsToNodesUnsafe(maps []map[string]interface{}) []dto.ApprovalNodeConfig {
	configs, err := mapsToNodes(maps)
	if err != nil {
		return nil
	}
	return configs
}

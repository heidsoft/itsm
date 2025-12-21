/**
 * CMDB 资产管理类型定义
 */

import { CIStatus } from "./constants";

// 配置项实体
export interface ConfigurationItem {
    id: number;
    name: string;
    description: string;
    type: string; // 冗余字段或分类
    status: CIStatus;
    location?: string;
    serial_number?: string;
    model?: string;
    vendor?: string;
    ci_type_id: number;
    tenant_id: number;
    attributes?: Record<string, any>;
    created_at: string;
    updated_at: string;
}

// CI 类型实体
export interface CIType {
    id: number;
    name: string;
    description: string;
    icon?: string;
    color?: string;
    attribute_schema?: string;
    is_active: boolean;
    tenant_id: number;
}

// CI 关系实体
export interface CIRelationship {
    id: number;
    source_ci_id: number;
    target_ci_id: number;
    relationship_type_id: number;
    description?: string;
    tenant_id: number;
}

// 统计信息
export interface CMDBStats {
    total_count: number;
    active_count: number;
    inactive_count: number;
    maintenance_count: number;
    type_distribution: Record<string, number>;
}

// 列表响应
export interface CIListResponse {
    items: ConfigurationItem[];
    total: number;
    page: number;
    size: number;
}

// 查询参数
export interface CIQuery {
    page?: number;
    page_size?: number;
    status?: string;
    ci_type_id?: number;
    search?: string;
}

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
    environment?: string;
    criticality?: string;
    asset_tag?: string;
    location?: string;
    serial_number?: string;
    model?: string;
    vendor?: string;
    ci_type_id: number;
    tenant_id: number;
    assigned_to?: string;
    owned_by?: string;
    discovery_source?: string;
    source?: string;
    cloud_provider?: string;
    cloud_account_id?: string;
    cloud_region?: string;
    cloud_zone?: string;
    cloud_resource_id?: string;
    cloud_resource_type?: string;
    cloud_resource_ref_id?: number;
    cloud_metadata?: Record<string, any>;
    cloud_tags?: Record<string, any>;
    cloud_metrics?: Record<string, any>;
    cloud_sync_time?: string;
    cloud_sync_status?: string;
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

export interface CloudService {
    id: number;
    parent_id?: number;
    provider: string;
    category?: string;
    service_code: string;
    service_name: string;
    resource_type_code: string;
    resource_type_name: string;
    api_version?: string;
    attribute_schema?: Record<string, any>;
    is_system?: boolean;
    is_active: boolean;
    tenant_id: number;
}

export interface CloudAccount {
    id: number;
    provider: string;
    account_id: string;
    account_name: string;
    credential_ref?: string;
    region_whitelist?: string[];
    is_active: boolean;
    tenant_id: number;
}

export interface CloudResource {
    id: number;
    cloud_account_id: number;
    service_id: number;
    resource_id: string;
    resource_name?: string;
    region?: string;
    zone?: string;
    status?: string;
    tags?: Record<string, string>;
    metadata?: Record<string, any>;
    first_seen_at?: string;
    last_seen_at?: string;
    lifecycle_state?: string;
    tenant_id: number;
}

export interface RelationshipType {
    id: number;
    name: string;
    directional: boolean;
    reverse_name?: string;
    description?: string;
    tenant_id: number;
}

export interface DiscoverySource {
    id: string;
    name: string;
    source_type: string;
    provider?: string;
    is_active: boolean;
    description?: string;
    tenant_id: number;
}

export interface DiscoveryJob {
    id: number;
    source_id: string;
    status: string;
    started_at?: string;
    finished_at?: string;
    summary?: Record<string, any>;
    tenant_id: number;
}

export interface DiscoveryResult {
    id: number;
    job_id: number;
    ci_id?: number;
    action: string;
    resource_type?: string;
    resource_id?: string;
    diff?: Record<string, any>;
    status: string;
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

export interface ReconciliationSummary {
    resource_total: number;
    bound_resource_count: number;
    unbound_resource_count: number;
    orphan_ci_count: number;
    unlinked_ci_count: number;
}

export interface ReconciliationResponse {
    summary: ReconciliationSummary;
    unbound_resources: CloudResource[];
    orphan_cis: ConfigurationItem[];
    unlinked_cis: ConfigurationItem[];
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

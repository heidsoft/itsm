/**
 * MSP (Managed Service Provider) 类型定义
 */

// 租户类型
export type TenantType = 'internal' | 'customer' | 'partner' | 'standard';

// MSP 角色
export type MSPRole = 'msp_tech' | 'msp_specialist' | 'msp_manager' | 'msp_viewer';

// 分配角色
export type AllocationRole = 'primary' | 'backup' | 'specialist';

// MSP 分配 DTO
export interface MSPAllocation {
  id: number;
  msp_user_id: number;
  msp_username?: string;
  customer_tenant_id: number;
  customer_name?: string;
  role: AllocationRole;
  assigned_at: string;
  deassigned_at?: string;
}

// 创建分配请求
export interface CreateAllocationRequest {
  msp_user_id: number;
  customer_tenant_id: number;
  role?: AllocationRole;
}

// 工单的 MSP 信息
export interface TicketMSPInfo {
  is_managed_by_msp: boolean;
  msp_provider_id?: number;
  msp_provider_name?: string;
  managed_by_user_id?: number;
  managed_by_username?: string;
  msp_ticket_id?: string;
}

// 工作流节点的 MSP 配置
export interface WorkflowNodeMSPConfig {
  enable_msp_support: boolean;
  allowed_msp_roles?: MSPRole[];
  require_msp_approval: boolean;
}

// MSP 上下文（用于中间件）
export interface MSPContext {
  is_msp: boolean;
  msp_user_id: number;
  customer_tenant_id?: number;
  role?: string;
  allowed_customers: number[];
}

// MSP 报表数据
export interface MSPCustomerReport {
  customer_tenant_id: number;
  customer_name: string;
  period: string;
  total_tickets: number;
  resolved_tickets: number;
  msp_handling_time_avg: number; // 小时
  sla_compliance_rate: number; // 0-1
}

// MSP 分配历史
export interface MSPAllocationHistory {
  id: number;
  msp_user_id: number;
  msp_username: string;
  customer_tenant_id: number;
  customer_name: string;
  role: AllocationRole;
  assigned_at: string;
  deassigned_at?: string;
  deallocation_reason?: string;
  created_by: number;
  created_by_name?: string;
}

// API 响应类型
export interface MSPAllocationListResponse {
  allocations: MSPAllocation[];
  total: number;
}

export interface MSPCustomersResponse {
  customers: {
    id: number;
    code: string;
    name: string;
  }[];
  total: number;
}

export interface MSPCustomerTicketsResponse {
  tickets: TicketMSPInfo & {
    id: number;
    title: string;
    status: string;
    assignee_id?: number;
    assignee_name?: string;
    created_at: string;
    tenant: {
      id: number;
      code: string;
      name: string;
    };
  };
  total: number;
}

// 扩展 Tenant 类型
export interface Tenant extends Record<string, unknown> {
  id: number;
  name: string;
  code: string;
  domain?: string;
  type: TenantType;
  parent_tenant_id?: number;
  msp_provider_id?: number;
  status: 'active' | 'suspended' | 'expired';
  expires_at?: string;
  created_at: string;
  updated_at: string;

  // 关联关系
  parent?: Tenant;
  children?: Tenant[];
  msp_provider?: Tenant;
  msp_customers?: Tenant[];
}

// 扩展 User 类型
export interface User extends Record<string, unknown> {
  id: number;
  username: string;
  email: string;
  tenant_id: number;
  msp_role?: MSPRole;
  assigned_by_msp_id?: number;
  created_at: string;
  updated_at: string;

  // 关联关系
  tenant?: Tenant;
  msp_allocations?: MSPAllocation[];
}

// 扩展 Ticket 类型
export interface Ticket extends Record<string, unknown> {
  id: number;
  tenant_id: number;
  title: string;
  description?: string;
  status: string;
  assignee_id?: number;
  is_managed_by_msp: boolean;
  msp_provider_id?: number;
  managed_by_user_id?: number;
  msp_ticket_id?: string;
  created_at: string;
  updated_at: string;

  // 关联关系
  tenant?: Tenant;
  assignee?: User;
  msp_provider?: Tenant;
  msp_user?: User;
  msp_info?: TicketMSPInfo;
}

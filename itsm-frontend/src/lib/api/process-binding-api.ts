import { httpClient } from './http-client';

export interface ProcessBinding {
  id: number;
  business_type: string;
  business_sub_type?: string;
  process_definition_key: string;
  process_version?: number;
  is_default?: boolean;
  priority: number;
  is_active: boolean;
  department_id?: number;
  team_id?: number;
  scenario?: string;
  category?: string;
  conditions?: Record<string, unknown>;
  approval_chain_id?: string;
  sla_policy_id?: string;
  overrides?: Record<string, unknown>;
  tenant_id?: number;
  created_at?: string;
  updated_at?: string;
}

export type ProcessBindingPayload = Omit<
  ProcessBinding,
  'id' | 'tenant_id' | 'created_at' | 'updated_at'
>;

type RawProcessBinding = Partial<ProcessBinding> & {
  businessType?: string;
  businessSubType?: string;
  processDefinitionKey?: string;
  processVersion?: number;
  isDefault?: boolean;
  isActive?: boolean;
  departmentId?: number;
  department_name?: string;
  departmentName?: string;
  teamId?: number;
  team_name?: string;
  teamName?: string;
  approvalChainId?: string;
  slaPolicyId?: string;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
};

export interface ProcessBindingQuery {
  business_type?: string;
  business_sub_type?: string;
  department_id?: number;
  team_id?: number;
  scenario?: string;
  category?: string;
  is_active?: boolean;
}

function normalizeBinding(binding: RawProcessBinding): ProcessBinding {
  return {
    ...binding,
    id: binding.id || 0,
    business_type: binding.business_type || binding.businessType || '',
    business_sub_type: binding.business_sub_type || binding.businessSubType,
    process_definition_key: binding.process_definition_key || binding.processDefinitionKey || '',
    process_version: binding.process_version ?? binding.processVersion,
    is_default: binding.is_default ?? binding.isDefault ?? false,
    priority: binding.priority ?? 0,
    is_active: binding.is_active ?? binding.isActive ?? false,
    department_id: binding.department_id ?? binding.departmentId,
    team_id: binding.team_id ?? binding.teamId,
    approval_chain_id: binding.approval_chain_id ?? binding.approvalChainId,
    sla_policy_id: binding.sla_policy_id ?? binding.slaPolicyId,
    tenant_id: binding.tenant_id ?? binding.tenantId,
    created_at: binding.created_at ?? binding.createdAt,
    updated_at: binding.updated_at ?? binding.updatedAt,
  };
}

function cleanPayload(payload: ProcessBindingPayload | Partial<ProcessBindingPayload>) {
  const next = { ...payload };
  Object.entries(next).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') {
      delete next[key as keyof typeof next];
    }
  });
  return next;
}

export class ProcessBindingApi {
  static async list(query?: ProcessBindingQuery): Promise<ProcessBinding[]> {
    const response = await httpClient.get<RawProcessBinding[]>('/api/v1/process-bindings', query);
    return (response || []).map(normalizeBinding);
  }

  static async get(id: number): Promise<ProcessBinding> {
    const response = await httpClient.get<RawProcessBinding>(`/api/v1/process-bindings/${id}`);
    return normalizeBinding(response);
  }

  static async create(payload: ProcessBindingPayload): Promise<ProcessBinding> {
    const response = await httpClient.post<RawProcessBinding>(
      '/api/v1/process-bindings',
      cleanPayload(payload)
    );
    return normalizeBinding(response);
  }

  static async update(id: number, payload: Partial<ProcessBindingPayload>): Promise<ProcessBinding> {
    const response = await httpClient.put<RawProcessBinding>(
      `/api/v1/process-bindings/${id}`,
      cleanPayload(payload)
    );
    return normalizeBinding(response);
  }

  static async delete(id: number): Promise<void> {
    await httpClient.delete(`/api/v1/process-bindings/${id}`);
  }

  static async listDepartmentProcesses(departmentId: number): Promise<ProcessBinding[]> {
    const response = await httpClient.get<RawProcessBinding[]>(
      `/api/v1/departments/${departmentId}/processes`
    );
    return (response || []).map(normalizeBinding);
  }

  static async initDepartmentProcesses(
    departmentId: number,
    departmentType: string
  ): Promise<void> {
    await httpClient.post(`/api/v1/departments/${departmentId}/init-processes`, {
      department_type: departmentType,
    });
  }
}

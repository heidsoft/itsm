import { httpClient } from './http-client';

export interface ProcessBinding {
  id: number;
  businessType: string;
  businessSubType?: string;
  processDefinitionKey: string;
  processVersion?: number;
  isDefault?: boolean;
  priority: number;
  isActive: boolean;
  departmentId?: number;
  teamId?: number;
  scenario?: string;
  category?: string;
  conditions?: Record<string, unknown>;
  approvalChainId?: string;
  slaPolicyId?: string;
  overrides?: Record<string, unknown>;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
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
  departmentName?: string;
  teamId?: number;
  teamName?: string;
  approvalChainId?: string;
  slaPolicyId?: string;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
};

export interface ProcessBindingQuery {
  businessType?: string;
  businessSubType?: string;
  departmentId?: number;
  teamId?: number;
  scenario?: string;
  category?: string;
  isActive?: boolean;
}

function normalizeBinding(binding: RawProcessBinding): ProcessBinding {
  return {
    ...binding,
    id: binding.id || 0,
    businessType: binding.businessType || '',
    businessSubType: binding.businessSubType,
    processDefinitionKey: binding.processDefinitionKey || '',
    processVersion: binding.processVersion,
    isDefault: binding.isDefault ?? false,
    priority: binding.priority ?? 0,
    isActive: binding.isActive ?? false,
    departmentId: binding.departmentId,
    teamId: binding.teamId,
    approvalChainId: binding.approvalChainId,
    slaPolicyId: binding.slaPolicyId,
    tenantId: binding.tenantId,
    createdAt: binding.createdAt,
    updatedAt: binding.updatedAt,
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
      departmentType: departmentType,
    });
  }
}

import { httpClient } from './http-client';

export type WorkflowTemplateStatus = 'draft' | 'published' | 'archived';

export interface WorkflowTemplate {
  id: number;
  key: string;
  name: string;
  description: string;
  domain: string;
  formSchema: Record<string, unknown>;
  approvalPolicy: Record<string, unknown>;
  ontologyBindings: Record<string, unknown>;
  slaConfig: Record<string, unknown>;
  bpmnXml?: string;
  version: string;
  status: WorkflowTemplateStatus | string;
  isPublic: boolean;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkflowTemplateListResponse {
  items: WorkflowTemplate[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CreateWorkflowTemplateRequest {
  key: string;
  name: string;
  description?: string;
  domain: string;
  formSchema?: Record<string, unknown>;
  approvalPolicy?: Record<string, unknown>;
  ontologyBindings?: Record<string, unknown>;
  slaConfig?: Record<string, unknown>;
  bpmnXml: string;
  isPublic: boolean;
}

export type UpdateWorkflowTemplateRequest = Partial<Omit<CreateWorkflowTemplateRequest, 'key'>>;

const TEMPLATES_PATH = '/api/v1/bpmn/ai/templates';

export class BPMNWorkflowTemplateApi {
  static list(params: { keyword?: string; domain?: string; status?: string; page?: number; pageSize?: number } = {}) {
    return httpClient.get<WorkflowTemplateListResponse>(TEMPLATES_PATH, params);
  }

  static get(key: string, version?: string) {
    return httpClient.get<WorkflowTemplate>(`${TEMPLATES_PATH}/${encodeURIComponent(key)}`, version ? { version } : undefined);
  }

  static create(request: CreateWorkflowTemplateRequest) {
    return httpClient.post<WorkflowTemplate>(TEMPLATES_PATH, request);
  }

  static update(key: string, request: UpdateWorkflowTemplateRequest) {
    return httpClient.put<WorkflowTemplate>(`${TEMPLATES_PATH}/${encodeURIComponent(key)}`, request);
  }

  static publish(key: string) {
    return httpClient.post<WorkflowTemplate>(`${TEMPLATES_PATH}/${encodeURIComponent(key)}/publish`);
  }

  static archive(key: string) {
    return httpClient.post<{ key: string; status: string }>(`${TEMPLATES_PATH}/${encodeURIComponent(key)}/archive`);
  }

  static versions(key: string) {
    return httpClient.get<WorkflowTemplate[]>(`${TEMPLATES_PATH}/${encodeURIComponent(key)}/versions`);
  }
}

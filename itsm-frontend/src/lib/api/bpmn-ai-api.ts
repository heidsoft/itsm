import { httpClient } from './http-client';

export type BPMNProcessType = 'incident' | 'change' | 'problem' | 'service_request' | 'custom';
export type BPMNEnterpriseType = 'cn_enterprise' | 'international' | 'startup' | 'government';

export interface GenerateBPMNRequest {
  requirement: string;
  processType: BPMNProcessType;
  enterpriseType: BPMNEnterpriseType;
  includeSla: boolean;
  includeNotifications: boolean;
  includeApprovals: boolean;
  tenantId?: number;
}

export interface GenerateBPMNResponse {
  bpmnXml: string;
  processId: string;
  processName: string;
  processDescription: string;
  version: string;
  nodeCount: number;
  complexity: 'low' | 'medium' | 'high' | string;
  explanation: string;
  deploymentId?: string;
  processDefinitionId?: number;
}

export interface PreviewBPMNRequest {
  requirement: string;
  processType: BPMNProcessType;
  enterpriseType: BPMNEnterpriseType;
}

export interface BPMNNodePreview {
  id: string;
  name: string;
  type: string;
  description: string;
  assigneeRole?: string;
  slaMinutes?: number;
}

export interface PreviewBPMNResponse {
  structureDescription: string;
  nodes: BPMNNodePreview[];
  complexity: 'low' | 'medium' | 'high' | string;
  estimatedNodeCount: number;
  useCases: string;
  suggestions: string[];
}

export interface BPMNTemplateSuggestion {
  id?: string;
  name?: string;
  description?: string;
  processType?: BPMNProcessType | string;
  score?: number;
}

export class BPMNAIApi {
  private static readonly baseUrl = '/api/v1/bpmn/ai';

  static async generateBPMN(
    request: GenerateBPMNRequest,
    options?: { autoDeploy?: boolean }
  ): Promise<GenerateBPMNResponse> {
    const tenantId = request.tenantId ?? httpClient.getTenantId() ?? 1;
    const endpoint = `${this.baseUrl}/generate${options?.autoDeploy ? '?auto_deploy=true' : ''}`;

    return httpClient.post<GenerateBPMNResponse>(endpoint, {
      ...request,
      tenantId,
    });
  }

  static async previewBPMN(request: PreviewBPMNRequest): Promise<PreviewBPMNResponse> {
    return httpClient.post<PreviewBPMNResponse>(`${this.baseUrl}/preview`, request);
  }

  static async getTemplateSuggestions(params: {
    keyword: string;
    processType?: BPMNProcessType | string;
  }): Promise<BPMNTemplateSuggestion[]> {
    return httpClient.get<BPMNTemplateSuggestion[]>(`${this.baseUrl}/templates/suggestions`, params);
  }
}

import { httpClient } from './http-client';

export interface Workflow {
  id: number;
  name: string;
  description: string;
  category: string;
  version: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  bpmn_xml: string;
  process_variables: Record<string, any>;
  created_at: string;
  updated_at: string;
  instances_count: number;
  running_instances: number;
  created_by: string;
}

export interface WorkflowInstance {
  id: number;
  workflow_id: number;
  instance_id: string;
  business_key: string;
  status: 'running' | 'completed' | 'suspended' | 'terminated';
  current_activity: string;
  variables: Record<string, any>;
  started_by: string;
  completed_by?: string;
  started_at: string;
  completed_at?: string;
  due_date?: string;
  priority: string;
}

export interface WorkflowTask {
  id: number;
  instance_id: number;
  task_id: string;
  activity_id: string;
  name: string;
  type: string;
  assignee?: string;
  candidate_users?: string;
  candidate_groups?: string;
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled';
  priority: string;
  created_at: string;
  due_date?: string;
  completed_at?: string;
  completed_by?: string;
  form_data?: Record<string, any>;
  variables?: Record<string, any>;
  comment?: string;
}

export interface CreateWorkflowRequest {
  name: string;
  description?: string;
  category: string;
  bpmn_xml: string;
  process_variables?: Record<string, any>;
  is_template?: boolean;
  template_category?: string;
  metadata?: Record<string, any>;
}

export interface UpdateWorkflowRequest {
  name?: string;
  description?: string;
  category?: string;
  bpmn_xml?: string;
  process_variables?: Record<string, any>;
  status?: string;
  metadata?: Record<string, any>;
}

export interface ListWorkflowsRequest {
  page?: number;
  page_size?: number;
  category?: string;
  status?: string;
  is_template?: boolean;
  keyword?: string;
  sort_by?: string;
  sort_order?: string;
}

export interface ListWorkflowsResponse {
  workflows: Workflow[];
  total: number;
  page: number;
  page_size: number;
}

export interface StartWorkflowRequest {
  workflow_id: number;
  business_key?: string;
  variables?: Record<string, any>;
  priority?: string;
  due_date?: string;
  assignee?: string;
  comment?: string;
}

export interface StartWorkflowResponse {
  instance_id: string;
  business_key: string;
  status: string;
  current_task?: WorkflowTask;
  variables: Record<string, any>;
  started_at: string;
}

export interface CompleteTaskRequest {
  task_id: string;
  variables?: Record<string, any>;
  form_data?: Record<string, any>;
  comment?: string;
  outcome?: string;
}

export interface ListWorkflowInstancesRequest {
  page?: number;
  page_size?: number;
  workflow_id?: number;
  status?: string;
  business_key?: string;
  started_by?: string;
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: string;
}

export interface ListWorkflowInstancesResponse {
  instances: WorkflowInstance[];
  total: number;
  page: number;
  page_size: number;
}

export interface ListWorkflowTasksRequest {
  page?: number;
  page_size?: number;
  instance_id?: number;
  assignee?: string;
  status?: string;
  priority?: string;
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: string;
}

export interface ListWorkflowTasksResponse {
  tasks: WorkflowTask[];
  total: number;
  page: number;
  page_size: number;
}

export interface ValidateBPMNRequest {
  bpmn_xml: string;
}

export interface ValidateBPMNResponse {
  is_valid: boolean;
  errors: string[];
  warnings: string[];
  info: string[];
}

export interface WorkflowStatistics {
  total_workflows: number;
  active_workflows: number;
  running_instances: number;
  completed_instances: number;
  pending_tasks: number;
  overdue_tasks: number;
  avg_completion_time: number;
}

// BPMN流程定义相关接口
export interface ProcessDefinition {
  key: string;
  name: string;
  description?: string;
  version: number;
  category?: string;
  deployment_id: string;
  resource_name: string;
  dgrm_resource_name?: string;
  tenant_id: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateProcessDefinitionRequest {
  name: string;
  description?: string;
  category?: string;
  bpmn_xml: string;
  tenant_id: number;
}

export interface UpdateProcessDefinitionRequest {
  name?: string;
  description?: string;
  category?: string;
  bpmn_xml?: string;
}

export interface ListProcessDefinitionsRequest {
  page?: number;
  page_size?: number;
  category?: string;
  is_active?: boolean;
  keyword?: string;
  sort_by?: string;
  sort_order?: string;
}

export interface ListProcessDefinitionsResponse {
  definitions: ProcessDefinition[];
  total: number;
  page: number;
  page_size: number;
}

// BPMN流程实例相关接口
export interface ProcessInstance {
  id: string;
  process_definition_key: string;
  business_key?: string;
  status: 'running' | 'completed' | 'suspended' | 'terminated';
  variables: Record<string, any>;
  started_by: string;
  started_at: string;
  completed_at?: string;
  tenant_id: number;
}

export interface StartProcessRequest {
  process_definition_key: string;
  business_key?: string;
  variables?: Record<string, any>;
  tenant_id: number;
}

export interface StartProcessResponse {
  instance_id: string;
  business_key?: string;
  status: string;
  variables: Record<string, any>;
  started_at: string;
}

export interface ListProcessInstancesRequest {
  page?: number;
  page_size?: number;
  process_definition_key?: string;
  status?: string;
  business_key?: string;
  started_by?: string;
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: string;
}

export interface ListProcessInstancesResponse {
  instances: ProcessInstance[];
  total: number;
  page: number;
  page_size: number;
}

// BPMN任务相关接口
export interface BPMNTask {
  id: string;
  name: string;
  description?: string;
  assignee?: string;
  candidate_users?: string[];
  candidate_groups?: string[];
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled';
  priority: number;
  created_at: string;
  due_date?: string;
  completed_at?: string;
  variables: Record<string, any>;
  process_instance_id: string;
  process_definition_key: string;
  tenant_id: number;
}

export interface ListBPMNTasksRequest {
  page?: number;
  page_size?: number;
  assignee?: string;
  status?: string;
  priority?: string;
  process_instance_id?: string;
  process_definition_key?: string;
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: string;
}

export interface ListBPMNTasksResponse {
  tasks: BPMNTask[];
  total: number;
  page: number;
  page_size: number;
}

export interface CompleteBPMNTaskRequest {
  task_id: string;
  variables?: Record<string, any>;
  outcome?: string;
  comment?: string;
}

export class WorkflowAPI {
  // 工作流管理 - 使用新的BPMN API端点
  static async listWorkflows(params: ListWorkflowsRequest = {}): Promise<ListWorkflowsResponse> {
    try {
      const response = await httpClient.get<ListWorkflowsResponse>('/api/v1/bpmn/process-definitions', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflows error:', error);
      throw error;
    }
  }

  static async createWorkflow(data: CreateWorkflowRequest): Promise<Workflow> {
    try {
      const response = await httpClient.post<Workflow>('/api/v1/bpmn/process-definitions', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.createWorkflow error:', error);
      throw error;
    }
  }

  static async getWorkflow(id: number): Promise<Workflow> {
    try {
      const response = await httpClient.get<Workflow>(`/api/v1/bpmn/process-definitions/${id}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getWorkflow error:', error);
      throw error;
    }
  }

  static async updateWorkflow(id: number, data: UpdateWorkflowRequest): Promise<Workflow> {
    try {
      const response = await httpClient.put<Workflow>(`/api/v1/bpmn/process-definitions/${id}`, data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.updateWorkflow error:', error);
      throw error;
    }
  }

  static async deleteWorkflow(id: number): Promise<void> {
    try {
      await httpClient.delete(`/api/v1/bpmn/process-definitions/${id}`);
    } catch (error) {
      console.error('WorkflowAPI.deleteWorkflow error:', error);
      throw error;
    }
  }

  static async getWorkflowStatistics(): Promise<WorkflowStatistics> {
    try {
      const response = await httpClient.get<WorkflowStatistics>('/api/v1/bpmn/process-definitions/statistics');
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getWorkflowStatistics error:', error);
      throw error;
    }
  }

  // 工作流实例管理 - 使用新的BPMN API端点
  static async startWorkflow(data: StartWorkflowRequest): Promise<StartWorkflowResponse> {
    try {
      const response = await httpClient.post<StartWorkflowResponse>('/api/v1/bpmn/process-instances', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.startWorkflow error:', error);
      throw error;
    }
  }

  static async listWorkflowInstances(params: ListWorkflowInstancesRequest = {}): Promise<ListWorkflowInstancesResponse> {
    try {
      const response = await httpClient.get<ListWorkflowInstancesResponse>('/api/v1/bpmn/process-instances', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflowInstances error:', error);
      throw error;
    }
  }

  static async getWorkflowInstance(id: number): Promise<WorkflowInstance> {
    try {
      const response = await httpClient.get<WorkflowInstance>(`/api/v1/bpmn/process-instances/${id}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getWorkflowInstance error:', error);
      throw error;
    }
  }

  static async suspendWorkflow(instanceId: string, reason?: string): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/suspend`, {
        reason,
      });
    } catch (error) {
      console.error('WorkflowAPI.suspendWorkflow error:', error);
      throw error;
    }
  }

  static async resumeWorkflow(instanceId: string, comment?: string): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/resume`, {
        comment,
      });
    } catch (error) {
      console.error('WorkflowAPI.resumeWorkflow error:', error);
      throw error;
    }
  }

  static async terminateWorkflow(instanceId: string, reason?: string): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/terminate`, {
        reason,
      });
    } catch (error) {
      console.error('WorkflowAPI.terminateWorkflow error:', error);
      throw error;
    }
  }

  // 工作流任务管理 - 使用新的BPMN API端点
  static async listWorkflowTasks(params: ListWorkflowTasksRequest = {}): Promise<ListWorkflowTasksResponse> {
    try {
      const response = await httpClient.get<ListWorkflowTasksResponse>('/api/v1/bpmn/tasks', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflowTasks error:', error);
      throw error;
    }
  }

  static async completeTask(data: CompleteTaskRequest): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/tasks/${data.task_id}/complete`, data);
    } catch (error) {
      console.error('WorkflowAPI.completeTask error:', error);
      throw error;
    }
  }

  // BPMN验证
  static async validateBPMN(data: ValidateBPMNRequest): Promise<ValidateBPMNResponse> {
    try {
      const response = await httpClient.post<ValidateBPMNResponse>('/api/v1/bpmn/process-definitions/validate', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.validateBPMN error:', error);
      throw error;
    }
  }

  // 导出/导入
  static async exportWorkflow(workflowId: number, format: string): Promise<Blob> {
    try {
      const response = await httpClient.post('/api/v1/bpmn/process-definitions/export', {
        workflow_id: workflowId,
        format,
      }, { responseType: 'blob' });
      return response;
    } catch (error) {
      console.error('WorkflowAPI.exportWorkflow error:', error);
      throw error;
    }
  }

  static async importWorkflow(data: CreateWorkflowRequest): Promise<Workflow> {
    try {
      const response = await httpClient.post<Workflow>('/api/v1/bpmn/process-definitions/import', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.importWorkflow error:', error);
      throw error;
    }
  }

  // 工作流部署
  static async deployWorkflow(workflowId: number): Promise<void> {
    try {
      await httpClient.post('/api/v1/bpmn/process-definitions/deploy', {
        workflow_id: workflowId,
      });
    } catch (error) {
      console.error('WorkflowAPI.deployWorkflow error:', error);
      throw error;
    }
  }

  // 工作流激活/停用
  static async activateWorkflow(workflowId: number): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/process-definitions/${workflowId}/active`);
    } catch (error) {
      console.error('WorkflowAPI.activateWorkflow error:', error);
      throw error;
    }
  }

  static async deactivateWorkflow(workflowId: number): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/process-definitions/${workflowId}/deactivate`);
    } catch (error) {
      console.error('WorkflowAPI.deactivateWorkflow error:', error);
      throw error;
    }
  }

  // 工作流版本管理
  static async createWorkflowVersion(workflowId: number, data: CreateWorkflowRequest): Promise<Workflow> {
    try {
      const response = await httpClient.post<Workflow>(`/api/v1/bpmn/process-definitions/${workflowId}/versions`, data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.createWorkflowVersion error:', error);
      throw error;
    }
  }

  static async listWorkflowVersions(workflowId: number): Promise<Workflow[]> {
    try {
      const response = await httpClient.get<Workflow[]>(`/api/v1/bpmn/process-definitions/${workflowId}/versions`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflowVersions error:', error);
      throw error;
    }
  }

  // 新增：BPMN流程定义管理
  static async createProcessDefinition(data: CreateProcessDefinitionRequest): Promise<ProcessDefinition> {
    try {
      const response = await httpClient.post<ProcessDefinition>('/api/v1/bpmn/process-definitions', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.createProcessDefinition error:', error);
      throw error;
    }
  }

  static async getProcessDefinition(key: string): Promise<ProcessDefinition> {
    try {
      const response = await httpClient.get<ProcessDefinition>(`/api/v1/bpmn/process-definitions/${key}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getProcessDefinition error:', error);
      throw error;
    }
  }

  static async updateProcessDefinition(key: string, data: UpdateProcessDefinitionRequest): Promise<ProcessDefinition> {
    try {
      const response = await httpClient.put<ProcessDefinition>(`/api/v1/bpmn/process-definitions/${key}`, data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.updateProcessDefinition error:', error);
      throw error;
    }
  }

  static async deleteProcessDefinition(key: string): Promise<void> {
    try {
      await httpClient.delete(`/api/v1/bpmn/process-definitions/${key}`);
    } catch (error) {
      console.error('WorkflowAPI.deleteProcessDefinition error:', error);
      throw error;
    }
  }

  static async setProcessDefinitionActive(key: string, active: boolean): Promise<void> {
    try {
      if (active) {
        await httpClient.put(`/api/v1/bpmn/process-definitions/${key}/active`);
      } else {
        await httpClient.put(`/api/v1/bpmn/process-definitions/${key}/deactivate`);
      }
    } catch (error) {
      console.error('WorkflowAPI.setProcessDefinitionActive error:', error);
      throw error;
    }
  }

  // 新增：BPMN流程实例管理
  static async startProcess(data: StartProcessRequest): Promise<StartProcessResponse> {
    try {
      const response = await httpClient.post<StartProcessResponse>('/api/v1/bpmn/process-instances', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.startProcess error:', error);
      throw error;
    }
  }

  static async listProcessInstances(params: ListProcessInstancesRequest = {}): Promise<ListProcessInstancesResponse> {
    try {
      const response = await httpClient.get<ListProcessInstancesResponse>('/api/v1/bpmn/process-instances', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listProcessInstances error:', error);
      throw error;
    }
  }

  static async getProcessInstance(id: string): Promise<ProcessInstance> {
    try {
      const response = await httpClient.get<ProcessInstance>(`/api/v1/bpmn/process-instances/${id}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getProcessInstance error:', error);
      throw error;
    }
  }

  static async setProcessInstanceVariables(id: string, variables: Record<string, any>): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/process-instances/${id}/variables`, { variables });
    } catch (error) {
      console.error('WorkflowAPI.setProcessInstanceVariables error:', error);
      throw error;
    }
  }

  // 新增：BPMN任务管理
  static async listBPMNTasks(params: ListBPMNTasksRequest = {}): Promise<ListBPMNTasksResponse> {
    try {
      const response = await httpClient.get<ListBPMNTasksResponse>('/api/v1/bpmn/tasks', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listBPMNTasks error:', error);
      throw error;
    }
  }

  static async getBPMNTask(id: string): Promise<BPMNTask> {
    try {
      const response = await httpClient.get<BPMNTask>(`/api/v1/bpmn/tasks/${id}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getBPMNTask error:', error);
      throw error;
    }
  }

  static async assignBPMNTask(id: string, assignee: string): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/tasks/${id}/assign`, { assignee });
    } catch (error) {
      console.error('WorkflowAPI.assignBPMNTask error:', error);
      throw error;
    }
  }

  static async completeBPMNTask(id: string, data: CompleteBPMNTaskRequest): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/tasks/${id}/complete`, data);
    } catch (error) {
      console.error('WorkflowAPI.completeBPMNTask error:', error);
      throw error;
    }
  }

  static async cancelBPMNTask(id: string, reason?: string): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/tasks/${id}/cancel`, { reason });
    } catch (error) {
      console.error('WorkflowAPI.cancelBPMNTask error:', error);
      throw error;
    }
  }

  static async setBPMNTaskVariables(id: string, variables: Record<string, any>): Promise<void> {
    try {
      await httpClient.put(`/api/v1/bpmn/tasks/${id}/variables`, { variables });
    } catch (error) {
      console.error('WorkflowAPI.setBPMNTaskVariables error:', error);
      throw error;
    }
  }
} 
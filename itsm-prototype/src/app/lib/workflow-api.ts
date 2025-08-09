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

export class WorkflowAPI {
  // 工作流管理
  static async listWorkflows(params: ListWorkflowsRequest = {}): Promise<ListWorkflowsResponse> {
    try {
      const response = await httpClient.get<ListWorkflowsResponse>('/api/v1/workflows', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflows error:', error);
      throw error;
    }
  }

  static async createWorkflow(data: CreateWorkflowRequest): Promise<Workflow> {
    try {
      const response = await httpClient.post<Workflow>('/api/v1/workflows', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.createWorkflow error:', error);
      throw error;
    }
  }

  static async getWorkflow(id: number): Promise<Workflow> {
    try {
      const response = await httpClient.get<Workflow>(`/api/v1/workflows/${id}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getWorkflow error:', error);
      throw error;
    }
  }

  static async updateWorkflow(id: number, data: UpdateWorkflowRequest): Promise<Workflow> {
    try {
      const response = await httpClient.put<Workflow>(`/api/v1/workflows/${id}`, data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.updateWorkflow error:', error);
      throw error;
    }
  }

  static async deleteWorkflow(id: number): Promise<void> {
    try {
      await httpClient.delete(`/api/v1/workflows/${id}`);
    } catch (error) {
      console.error('WorkflowAPI.deleteWorkflow error:', error);
      throw error;
    }
  }

  static async getWorkflowStatistics(): Promise<WorkflowStatistics> {
    try {
      const response = await httpClient.get<WorkflowStatistics>('/api/v1/workflows/statistics');
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getWorkflowStatistics error:', error);
      throw error;
    }
  }

  // 工作流实例管理
  static async startWorkflow(data: StartWorkflowRequest): Promise<StartWorkflowResponse> {
    try {
      const response = await httpClient.post<StartWorkflowResponse>('/api/v1/workflow-instances/start', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.startWorkflow error:', error);
      throw error;
    }
  }

  static async listWorkflowInstances(params: ListWorkflowInstancesRequest = {}): Promise<ListWorkflowInstancesResponse> {
    try {
      const response = await httpClient.get<ListWorkflowInstancesResponse>('/api/v1/workflow-instances', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflowInstances error:', error);
      throw error;
    }
  }

  static async getWorkflowInstance(id: number): Promise<WorkflowInstance> {
    try {
      const response = await httpClient.get<WorkflowInstance>(`/api/v1/workflow-instances/${id}`);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.getWorkflowInstance error:', error);
      throw error;
    }
  }

  static async suspendWorkflow(instanceId: string, reason?: string): Promise<void> {
    try {
      await httpClient.post('/api/v1/workflow-instances/suspend', {
        instance_id: instanceId,
        reason,
      });
    } catch (error) {
      console.error('WorkflowAPI.suspendWorkflow error:', error);
      throw error;
    }
  }

  static async resumeWorkflow(instanceId: string, comment?: string): Promise<void> {
    try {
      await httpClient.post('/api/v1/workflow-instances/resume', {
        instance_id: instanceId,
        comment,
      });
    } catch (error) {
      console.error('WorkflowAPI.resumeWorkflow error:', error);
      throw error;
    }
  }

  static async terminateWorkflow(instanceId: string, reason?: string): Promise<void> {
    try {
      await httpClient.post('/api/v1/workflow-instances/terminate', {
        instance_id: instanceId,
        reason,
      });
    } catch (error) {
      console.error('WorkflowAPI.terminateWorkflow error:', error);
      throw error;
    }
  }

  // 工作流任务管理
  static async listWorkflowTasks(params: ListWorkflowTasksRequest = {}): Promise<ListWorkflowTasksResponse> {
    try {
      const response = await httpClient.get<ListWorkflowTasksResponse>('/api/v1/workflow-tasks', params);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.listWorkflowTasks error:', error);
      throw error;
    }
  }

  static async completeTask(data: CompleteTaskRequest): Promise<void> {
    try {
      await httpClient.post('/api/v1/workflow-tasks/complete', data);
    } catch (error) {
      console.error('WorkflowAPI.completeTask error:', error);
      throw error;
    }
  }

  // BPMN验证
  static async validateBPMN(data: ValidateBPMNRequest): Promise<ValidateBPMNResponse> {
    try {
      const response = await httpClient.post<ValidateBPMNResponse>('/api/v1/workflows/validate-bpmn', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.validateBPMN error:', error);
      throw error;
    }
  }

  // 导出/导入
  static async exportWorkflow(workflowId: number, format: string): Promise<Blob> {
    try {
      const response = await httpClient.post('/api/v1/workflows/export', {
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
      const response = await httpClient.post<Workflow>('/api/v1/workflows/import', data);
      return response;
    } catch (error) {
      console.error('WorkflowAPI.importWorkflow error:', error);
      throw error;
    }
  }
} 
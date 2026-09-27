import { httpClient } from './http-client';

export interface WorkbenchItem {
  id: number;
  recordType: 'incident' | 'change' | 'problem' | 'ticket';
  title: string;
  description?: string;
  priority: string;
  status: string;
  phase: string;
  assigneeId?: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkbenchQuery {
  assigneeId?: number;
  phase?: string[];
  priority?: string[];
  recordType?: string[];
  page?: number;
  pageSize?: number;
}

export interface WorkbenchResponse {
  items: WorkbenchItem[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export async function queryWorkbench(params: WorkbenchQuery): Promise<WorkbenchResponse> {
  const query = new URLSearchParams();
  if (params.assigneeId !== undefined) query.set('assigneeId', String(params.assigneeId));
  if (params.phase?.length) query.set('phase', params.phase.join(','));
  if (params.priority?.length) query.set('priority', params.priority.join(','));
  if (params.recordType?.length) query.set('recordType', params.recordType.join(','));
  if (params.page) query.set('page', String(params.page));
  if (params.pageSize) query.set('pageSize', String(params.pageSize));

  const endpoint = `/api/v1/workbench${query.toString() ? `?${query.toString()}` : ''}`;
  return await httpClient.get<WorkbenchResponse>(endpoint);
}

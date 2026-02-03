import { IncidentAPI } from "@/lib/api/incident-api";

// 重新导出事件管理 API
export const fetchIncidents = IncidentAPI.listIncidents;
export const getIncident = IncidentAPI.getIncident;
export const createIncident = IncidentAPI.createIncident;
export const updateIncident = IncidentAPI.updateIncident;
export const updateIncidentStatus = IncidentAPI.updateIncidentStatus;
export const getIncidentMetrics = IncidentAPI.getIncidentMetrics;

// 导出常量和类型
export const API_URLS = {
  INCIDENTS: '/api/v1/incidents',
  METRICS: '/api/v1/incidents/metrics',
};

export interface StandardPaginationParams {
  page: number;
  pageSize: number;
  search?: string;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}
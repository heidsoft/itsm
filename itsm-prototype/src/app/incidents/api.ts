import { IncidentAPI } from "@/app/lib/incident-api";

// 重新导出事件管理 API
export const fetchIncidents = IncidentAPI.listIncidents;
export const getIncident = IncidentAPI.getIncident;
export const createIncident = IncidentAPI.createIncident;
export const updateIncident = IncidentAPI.updateIncident;
export const updateIncidentStatus = IncidentAPI.updateIncidentStatus;
export const getIncidentMetrics = IncidentAPI.getIncidentMetrics;

// 模拟数据（当后端 API 不可用时使用）
export const mockIncidents = [
  {
    id: 1,
    incident_number: "INC-2024-001",
    title: "System Login Issue",
    description: "Users report unable to login to system normally",
    status: "open",
    priority: "high",
    source: "email",
    type: "System Issue",
    category: "Authentication",
    reporter: { id: 1, name: "Li Si" },
    assignee: { id: 2, name: "Zhang San" },
    is_major_incident: false,
    created_at: "2024-01-15T10:30:00Z",
    updated_at: "2024-01-15T14:20:00Z",
    resolved_at: null,
    closed_at: null,
    resolution: null,
    form_fields: {}
  },
  {
    id: 2,
    incident_number: "INC-2024-002",
    title: "Printer Malfunction",
    description: "Office printer not working properly",
    status: "in_progress",
    priority: "medium",
    source: "phone",
    type: "Hardware Issue",
    category: "Equipment",
    reporter: { id: 3, name: "Zhao Liu" },
    assignee: { id: 4, name: "Wang Wu" },
    is_major_incident: false,
    created_at: "2024-01-15T09:15:00Z",
    updated_at: "2024-01-15T11:45:00Z",
    resolved_at: null,
    closed_at: null,
    resolution: null,
    form_fields: {}
  },
  {
    id: 3,
    incident_number: "INC-2024-003",
    title: "Slow Network Connection",
    description: "Network response speed significantly slower",
    status: "resolved",
    priority: "low",
    source: "web",
    type: "Network Issue",
    category: "Performance",
    reporter: { id: 5, name: "Sun Ba" },
    assignee: { id: 2, name: "Zhang San" },
    is_major_incident: false,
    created_at: "2024-01-14T16:20:00Z",
    updated_at: "2024-01-15T08:30:00Z",
    resolved_at: "2024-01-15T08:30:00Z",
    closed_at: null,
    resolution: "Network configuration optimized",
    form_fields: {}
  }
];

export const mockMetrics = {
  total_incidents: 3,
  critical_incidents: 1,
  major_incidents: 0,
  avg_resolution_time: 2.5,
  mtta: 0.5,
  mttr: 2.0,
};
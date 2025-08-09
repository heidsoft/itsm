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
    title: "系统登录异常",
    description: "用户反馈无法正常登录系统",
    status: "open",
    priority: "high",
    source: "manual",
    type: "系统问题",
    is_major_incident: false,
    reporter: { id: 1, name: "李四" },
    assignee: { id: 2, name: "张三" },
    detected_at: "2024-01-15T10:30:00Z",
    created_at: "2024-01-15T10:30:00Z",
    updated_at: "2024-01-15T10:30:00Z",
  },
  {
    id: 2,
    incident_number: "INC-2024-002",
    title: "打印机故障",
    description: "办公室打印机无法正常工作",
    status: "in_progress",
    priority: "medium",
    source: "manual",
    type: "硬件问题",
    is_major_incident: false,
    reporter: { id: 3, name: "赵六" },
    assignee: { id: 4, name: "王五" },
    detected_at: "2024-01-15T09:15:00Z",
    created_at: "2024-01-15T09:15:00Z",
    updated_at: "2024-01-15T09:15:00Z",
  },
  {
    id: 3,
    incident_number: "INC-2024-003",
    title: "网络连接缓慢",
    description: "网络响应速度明显变慢",
    status: "resolved",
    priority: "high",
    source: "manual",
    type: "网络问题",
    is_major_incident: false,
    reporter: { id: 5, name: "孙八" },
    assignee: { id: 2, name: "张三" },
    detected_at: "2024-01-14T16:45:00Z",
    resolved_at: "2024-01-15T08:30:00Z",
    created_at: "2024-01-14T16:45:00Z",
    updated_at: "2024-01-15T08:30:00Z",
  },
];

export const mockMetrics = {
  total_incidents: 3,
  critical_incidents: 1,
  major_incidents: 0,
  avg_resolution_time: 2.5,
  mtta: 0.5,
  mttr: 2.0,
}; 
export const INCIDENT_STATUS_LABEL: Record<string, string> = {
  new: '待处理',
  in_progress: '处理中',
  waiting_customer: '等待用户',
  resolved: '已解决',
  closed: '已关闭',
};

export const INCIDENT_PRIORITY_LABEL: Record<string, string> = {
  urgent: '严重',
  high: '高',
  medium: '中',
  low: '低',
};

export const TICKET_STATUS_LABEL: Record<string, string> = {
  submitted: '已提交',
  assigned: '已分配',
  in_progress: '处理中',
  resolved: '已解决',
  closed: '已关闭',
};

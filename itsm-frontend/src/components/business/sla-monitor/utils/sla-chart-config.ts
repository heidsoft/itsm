/**
 * SLA Monitor 图表配置
 */

export const SLA_VIOLATION_SEVERITY_COLORS = {
  critical: '#ff4d4f',
  high: '#fa8c16',
  medium: '#faad14',
  low: '#52c41a',
};

export const SLA_VIOLATION_STATUS_COLORS = {
  open: '#ff4d4f',
  acknowledged: '#fa8c16',
  resolved: '#52c41a',
};

export const CHART_COLORS = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1'];

export const DEFAULT_CHART_HEIGHT = 300;

export const TIME_RANGE_OPTIONS = [
  { label: '今日', value: 'today' },
  { label: '本周', value: 'week' },
  { label: '本月', value: 'month' },
  { label: '自定义', value: 'custom' },
];

export const SEVERITY_OPTIONS = [
  { label: '全部', value: '' },
  { label: '严重', value: 'critical' },
  { label: '高', value: 'high' },
  { label: '中', value: 'medium' },
  { label: '低', value: 'low' },
];

export const STATUS_OPTIONS = [
  { label: '全部', value: '' },
  { label: '待处理', value: 'open' },
  { label: '已确认', value: 'acknowledged' },
  { label: '已解决', value: 'resolved' },
];

export const TYPE_OPTIONS = [
  { label: '全部', value: '' },
  { label: '响应时间', value: 'response_time' },
  { label: '解决时间', value: 'resolution_time' },
  { label: '可用性', value: 'availability' },
];

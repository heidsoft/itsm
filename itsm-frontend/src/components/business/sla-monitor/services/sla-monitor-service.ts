/**
 * SLA Monitor 服务层
 * 封装与 SLA 违规监控相关的 API 调用
 */

import SLAApi from '@/lib/api/sla-api';
import type { SLAViolation, SLAAlertRule } from '../types';

/**
 * 获取 SLA 违规列表
 */
export const fetchSLAViolations = async (params?: {
  status?: string;
  severity?: string;
  sla_type?: string;
  start_date?: string;
  end_date?: string;
  search?: string;
}): Promise<SLAViolation[]> => {
  const response = await SLAApi.getSLAViolations({
    ...params,
    page: 1,
    size: 100, // 获取足够多数据
  });
  return response.items;
};

/**
 * 获取单个违规详情（通过列表查找）
 */
export const fetchSLAViolation = async (id: number): Promise<SLAViolation> => {
  const violations = await fetchSLAViolations();
  const violation = violations.find(v => v.id === id);
  if (!violation) throw new Error('Violation not found');
  return violation;
};

/**
 * 解决 SLA 违规
 */
export const resolveSLAViolation = async (violationId: number): Promise<void> => {
  await SLAApi.updateSLAViolationStatus(violationId, true, '手动解决');
};

/**
 * 确认 SLA 违规
 */
export const acknowledgeSLAViolation = async (violationId: number): Promise<void> => {
  // 确认操作可能是增加备注或设置状态，这里用更新备注的方式模拟
  await SLAApi.updateSLAViolationStatus(violationId, false, '已确认违规');
};

/**
 * 获取告警规则列表
 */
export const fetchAlertRules = async (): Promise<SLAAlertRule[]> => {
  try {
    const response = await SLAApi.getAlertRules();
    return Array.isArray(response) ? response : [];
  } catch (error) {
    console.error('Failed to fetch alert rules:', error);
    return [];
  }
};

/**
 * 创建告警规则
 */
export const createAlertRule = async (rule: Partial<SLAAlertRule>): Promise<SLAAlertRule> => {
  const response = await SLAApi.createAlertRule({
    name: rule.name || 'New Rule',
    sla_definition_id: rule.sla_definition_id || 0,
    alert_level: rule.alert_level || 'warning',
    threshold_percentage: rule.threshold_percentage || 80,
    notification_channels: rule.notification_channels || ['email'],
    is_active: rule.is_active ?? true,
  });
  return response;
};

/**
 * 更新告警规则
 */
export const updateAlertRule = async (ruleId: number, rule: Partial<SLAAlertRule>): Promise<SLAAlertRule> => {
  const response = await SLAApi.updateAlertRule(ruleId, {
    name: rule.name,
    alert_level: rule.alert_level,
    threshold_percentage: rule.threshold_percentage,
    notification_channels: rule.notification_channels,
    is_active: rule.is_active,
  });
  return response;
};

/**
 * 删除告警规则
 */
export const deleteAlertRule = async (ruleId: number): Promise<void> => {
  await SLAApi.deleteAlertRule(ruleId);
};

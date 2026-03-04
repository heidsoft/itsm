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
  // TODO: 实现 API 调用
  // return await SLAApi.getAlertRules();
  return [];
};

/**
 * 创建告警规则
 */
export const createAlertRule = async (rule: Partial<SLAAlertRule>): Promise<SLAAlertRule> => {
  // TODO: 实现 API 调用
  // return await SLAApi.createAlertRule(rule);
  throw new Error('Not implemented');
};

/**
 * 更新告警规则
 */
export const updateAlertRule = async (ruleId: number, rule: Partial<SLAAlertRule>): Promise<SLAAlertRule> => {
  // TODO: 实现 API 调用
  // return await SLAApi.updateAlertRule(ruleId, rule);
  throw new Error('Not implemented');
};

/**
 * 删除告警规则
 */
export const deleteAlertRule = async (ruleId: number): Promise<void> => {
  // TODO: 实现 API 调用
  // await SLAApi.deleteAlertRule(ruleId);
};

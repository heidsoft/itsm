/**
 * SLA Monitor 过滤工具函数
 */

import type { SLAViolation, SLAFilters } from '../types';
import dayjs from 'dayjs';

/**
 * 过滤 SLA 违规
 */
export const filterSLAViolations = (
  violations: SLAViolation[],
  filters: SLAFilters
): SLAViolation[] => {
  return violations.filter(violation => {
    // 状态过滤
    if (filters.status && violation.status !== filters.status) {
      return false;
    }

    // 严重程度过滤
    if (filters.severity && violation.severity !== filters.severity) {
      return false;
    }

    // 类型过滤
    if (filters.type && violation.violation_type !== filters.type) {
      return false;
    }

    // 日期范围过滤
    if (filters.dateRange && filters.dateRange[0] && filters.dateRange[1]) {
      const violationDate = dayjs(violation.created_at);
      if (
        violationDate.isBefore(filters.dateRange[0], 'day') ||
        violationDate.isAfter(filters.dateRange[1], 'day')
      ) {
        return false;
      }
    }

    // 搜索过滤
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      const matchesDescription = violation.description?.toLowerCase().includes(searchLower);
      const matchesTicketId = String(violation.ticket_id).includes(searchLower);
      if (!matchesDescription && !matchesTicketId) {
        return false;
      }
    }

    return true;
  });
};

/**
 * 重置过滤器
 */
export const resetFilters = (): SLAFilters => ({
  status: '',
  severity: '',
  type: '',
  dateRange: null,
  search: '',
});

/**
 * 获取过滤后的违规数量
 */
export const getFilteredCount = (violations: SLAViolation[], filters: SLAFilters): number => {
  return filterSLAViolations(violations, filters).length;
};

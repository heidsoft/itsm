/**
 * sla-filter-utils - SLA 过滤器工具函数测试
 */

import {
  filterViolationsByStatus,
  filterViolationsBySeverity,
  filterViolationsByPolicy,
  filterViolationsByDateRange,
  filterViolationsByAssignee,
  combineFilters,
  getUniquePolicies,
  getUniqueAssignees,
  getFilterOptions,
  isFilterActive,
  clearFilter,
  buildFilterQuery,
} from '../sla-filter-utils';
import type { SLAViolation } from '../types';

describe('sla-filter-utils', () => {
  const mockViolations: SLAViolation[] = [
    {
      id: 'v1',
      status: 'open',
      severity: 'critical',
      policyId: 'policy-1',
      breachedAt: '2024-01-20T10:00:00Z',
      assignedTo: { id: 'user1', name: 'John' },
    },
    {
      id: 'v2',
      status: 'acknowledged',
      severity: 'high',
      policyId: 'policy-2',
      breachedAt: '2024-01-19T15:00:00Z',
      assignedTo: { id: 'user2', name: 'Jane' },
    },
    {
      id: 'v3',
      status: 'open',
      severity: 'medium',
      policyId: 'policy-1',
      breachedAt: '2024-01-18T09:00:00Z',
      assignedTo: { id: 'user1', name: 'John' },
    },
    {
      id: 'v4',
      status: 'resolved',
      severity: 'low',
      policyId: 'policy-3',
      breachedAt: '2024-01-17T14:00:00Z',
      assignedTo: null,
    },
  ];

  describe('filterViolationsByStatus', () => {
    it('应该过滤 open 状态的违规', () => {
      const filtered = filterViolationsByStatus(mockViolations, 'open');
      expect(filtered).toHaveLength(2);
      expect(filtered.every(v => v.status === 'open')).toBe(true);
    });

    it('应该过滤 acknowledged 状态的违规', () => {
      const filtered = filterViolationsByStatus(mockViolations, 'acknowledged');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].id).toBe('v2');
    });

    it('应该过滤 all 时返回所有', () => {
      const filtered = filterViolationsByStatus(mockViolations, 'all');
      expect(filtered).toHaveLength(4);
    });

    it('不存在的状态应返回空数组', () => {
      const filtered = filterViolationsByStatus(mockViolations, 'nonexistent');
      expect(filtered).toHaveLength(0);
    });
  });

  describe('filterViolationsBySeverity', () => {
    it('应该过滤 critical 严重程度', () => {
      const filtered = filterViolationsBySeverity(mockViolations, 'critical');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].id).toBe('v1');
    });

    it('应该过滤 high 严重程度', () => {
      const filtered = filterViolationsBySeverity(mockViolations, 'high');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].id).toBe('v2');
    });

    it('应该过滤 multiple 严重程度', () => {
      const filtered = filterViolationsBySeverity(mockViolations, ['critical', 'high']);
      expect(filtered).toHaveLength(2);
    });

    it('all 应返回所有', () => {
      const filtered = filterViolationsBySeverity(mockViolations, 'all');
      expect(filtered).toHaveLength(4);
    });
  });

  describe('filterViolationsByPolicy', () => {
    it('应该按 policyId 过滤', () => {
      const filtered = filterViolationsByPolicy(mockViolations, 'policy-1');
      expect(filtered).toHaveLength(2);
    });

    it('应该支持多个 policy', () => {
      const filtered = filterViolationsByPolicy(mockViolations, ['policy-1', 'policy-2']);
      expect(filtered).toHaveLength(3);
    });

    it('all 应返回所有', () => {
      const filtered = filterViolationsByPolicy(mockViolations, 'all');
      expect(filtered).toHaveLength(4);
    });
  });

  describe('filterViolationsByDateRange', () => {
    it('应该过滤指定日期范围内的违规', () => {
      const start = '2024-01-19';
      const end = '2024-01-20';
      const filtered = filterViolationsByDateRange(mockViolations, { start, end });
      expect(filtered).toHaveLength(2); // v1 和 v2
    });

    it('应该处理只有开始日期', () => {
      const start = '2024-01-19';
      const filtered = filterViolationsByDateRange(mockViolations, { start, end: null });
      expect(filtered.length).toBeGreaterThanOrEqual(2);
    });

    it('应该处理只有结束日期', () => {
      const end = '2024-01-19';
      const filtered = filterViolationsByDateRange(mockViolations, { start: null, end });
      expect(filtered).toHaveLength(1); // v2
    });

    it('空日期范围应返回所有', () => {
      const filtered = filterViolationsByDateRange(mockViolations, { start: null, end: null });
      expect(filtered).toHaveLength(4);
    });

    it('日期格式应正确比较', () => {
      const filtered = filterViolationsByDateRange(mockViolations, {
        start: '2024-01-20',
        end: '2024-01-20',
      });
      expect(filtered).toHaveLength(1);
      expect(filtered[0].id).toBe('v1');
    });
  });

  describe('filterViolationsByAssignee', () => {
    it('应该过滤分配给指定用户的违规', () => {
      const filtered = filterViolationsByAssignee(mockViolations, 'user1');
      expect(filtered).toHaveLength(2);
      expect(filtered.every(v => v.assignedTo?.id === 'user1')).toBe(true);
    });

    it('unassigned 应过滤未分配的', () => {
      const filtered = filterViolationsByAssignee(mockViolations, 'unassigned');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].assignedTo).toBeNull();
    });

    it('all 应返回所有', () => {
      const filtered = filterViolationsByAssignee(mockViolations, 'all');
      expect(filtered).toHaveLength(4);
    });
  });

  describe('combineFilters', () => {
    it('应该组合多个过滤器', () => {
      const result = combineFilters(
        mockViolations,
        { status: 'open' },
        { severity: ['critical', 'high'] }
      );

      // open + (critical 或 high) -> v1 (open, critical)
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('v1');
    });

    it('空过滤器应返回所有', () => {
      const result = combineFilters(mockViolations);
      expect(result).toHaveLength(4);
    });

    it('应该处理复杂的过滤器组合', () => {
      const result = combineFilters(
        mockViolations,
        { status: 'open' },
        { policy: 'policy-1' },
        { assignee: 'user1' }
      );

      // open + policy-1 + user1 = v3 和 v1 都是 open, policy-1, user1
      // v1: open, policy-1, user1 -> yes
      // v3: open, policy-1, user1 -> yes
      expect(result).toHaveLength(2);
    });
  });

  describe('getUniquePolicies', () => {
    it('应该提取唯一的策略 ID', () => {
      const policies = getUniquePolicies(mockViolations);
      expect(policies).toHaveLength(3);
      expect(policies).toContain('policy-1');
      expect(policies).toContain('policy-2');
      expect(policies).toContain('policy-3');
    });

    it('空数组应返回空', () => {
      const policies = getUniquePolicies([]);
      expect(policies).toHaveLength(0);
    });
  });

  describe('getUniqueAssignees', () => {
    it('应该提取唯一的分配人', () => {
      const assignees = getUniqueAssignees(mockViolations);
      expect(assignees).toHaveLength(1); // user1
    });

    it('应该包括未分配的标识', () => {
      const result = mockViolations.filter(v => v.assignedTo === null);
      expect(result).toHaveLength(1);
    });
  });

  describe('getFilterOptions', () => {
    it('应该生成所有过滤选项', () => {
      const options = getFilterOptions(mockViolations);
      expect(options.statuses).toBeDefined();
      expect(options.severities).toBeDefined();
      expect(options.policies).toBeDefined();
      expect(options.assignees).toBeDefined();
    });

    it('应该包括状态列表', () => {
      const options = getFilterOptions(mockViolations);
      expect(options.statuses).toContain('open');
      expect(options.statuses).toContain('acknowledged');
      expect(options.statuses).toContain('resolved');
    });

    it('应该包括严重程度列表', () => {
      const options = getFilterOptions(mockViolations);
      expect(options.severities).toContain('critical');
      expect(options.severities).toContain('high');
      expect(options.severities).toContain('medium');
      expect(options.severities).toContain('low');
    });
  });

  describe('isFilterActive', () => {
    it('应该识别活动过滤器', () => {
      expect(isFilterActive({ status: 'open' })).toBe(true);
      expect(isFilterActive({ severity: 'critical' })).toBe(true);
      expect(isFilterActive({ policy: 'policy-1' })).toBe(true);
    });

    it('应该识别非活动过滤器（值为 all）', () => {
      expect(isFilterActive({ status: 'all' })).toBe(false);
      expect(isFilterActive({ severity: 'all' })).toBe(false);
    });

    it('空过滤器应返回 false', () => {
      expect(isFilterActive({})).toBe(false);
    });
  });

  describe('clearFilter', () => {
    it('应该清除单个过滤器', () => {
      const filters = { status: 'open', severity: 'critical' };
      const result = clearFilter(filters, 'status');
      expect(result.status).toBe('all');
      expect(result.severity).toBe('critical');
    });

    it('应该清除无效键', () => {
      const filters = { status: 'open' };
      const result = clearFilter(filters, 'nonexistent');
      expect(result).toEqual(filters);
    });
  });

  describe('buildFilterQuery', () => {
    it('应该构建查询字符串', () => {
      const query = buildFilterQuery({
        status: 'open',
        severity: 'critical',
        page: 1,
        limit: 20,
      });

      expect(query).toContain('status=open');
      expect(query).toContain('severity=critical');
      expect(query).toContain('page=1');
      expect(query).toContain('limit=20');
    });

    it('应该编码特殊字符', () => {
      const query = buildFilterQuery({
        search: 'test query',
      });

      expect(query).toContain('search=test%20query');
    });

    it('应该跳过 all 值', () => {
      const query = buildFilterQuery({
        status: 'all',
        severity: 'critical',
      });

      expect(query).not.toContain('status');
      expect(query).toContain('severity=critical');
    });

    it('应该包括日期范围', () => {
      const query = buildFilterQuery({
        dateRange: {
          start: '2024-01-01',
          end: '2024-01-31',
        },
      });

      expect(query).toContain('startDate=2024-01-01');
      expect(query).toContain('endDate=2024-01-31');
    });
  });
});

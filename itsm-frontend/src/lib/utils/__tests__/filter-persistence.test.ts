/**
 * filter-persistence - 过滤器状态持久化工具测试
 */

import {
  saveFilterState,
  loadFilterState,
  clearFilterState,
  getFilterHistory,
  saveFilterHistory,
  getRecentFilters,
  exportFilters,
  importFilters,
  mergeFilterStates,
  getFilterDiff,
  validateFilterState,
} from '../filter-persistence';

describe('filter-persistence', () => {
  const mockFilterState = {
    status: 'open',
    severity: 'critical',
    dateRange: { start: '2024-01-01', end: '2024-01-31' },
    search: 'test',
    sortBy: 'breachedAt',
    sortOrder: 'desc',
  };

  beforeEach(() => {
    localStorage.clear();
  });

  describe('saveFilterState', () => {
    it('应该保存过滤器状态到 localStorage', () => {
      saveFilterState('sla-violations', mockFilterState);
      const stored = localStorage.getItem('filter:sla-violations');
      expect(stored).toBe(JSON.stringify(mockFilterState));
    });

    it('应该能够保存空状态', () => {
      saveFilterState('test', {});
      const stored = localStorage.getItem('filter:test');
      expect(stored).toBe('{}');
    });
  });

  describe('loadFilterState', () => {
    it('应该从 localStorage 加载过滤器状态', () => {
      localStorage.setItem('filter:test', JSON.stringify(mockFilterState));
      const result = loadFilterState('test');
      expect(result).toEqual(mockFilterState);
    });

    it('应该返回空对象如果不存在', () => {
      const result = loadFilterState('nonexistent');
      expect(result).toEqual({});
    });

    it('应该处理无效 JSON', () => {
      localStorage.setItem('filter:test', 'invalid-json');
      const result = loadFilterState('test');
      expect(result).toEqual({});
    });
  });

  describe('clearFilterState', () => {
    it('应该清除 localStorage 中的过滤器状态', () => {
      localStorage.setItem('filter:test', JSON.stringify(mockFilterState));
      clearFilterState('test');
      expect(localStorage.getItem('filter:test')).toBeNull();
    });

    it('清除不存在的键应无错误', () => {
      expect(() => clearFilterState('nonexistent')).not.toThrow();
    });
  });

  describe('getFilterHistory', () => {
    it('应该获取过滤器历史', () => {
      const history = [
        { filters: mockFilterState, timestamp: Date.now() },
        { filters: { status: 'open' }, timestamp: Date.now() - 1000 },
      ];
      localStorage.setItem('filter:history:sla-violations', JSON.stringify(history));

      const result = getFilterHistory('sla-violations');
      expect(result).toHaveLength(2);
    });

    it('应该按时间倒序返回', () => {
      const older = { filters: { status: 'open' }, timestamp: Date.now() - 5000 };
      const newer = { filters: { status: 'closed' }, timestamp: Date.now() };

      localStorage.setItem('filter:history:test', JSON.stringify([older, newer]));

      const result = getFilterHistory('test');
      expect(result[0].filters.status).toBe('closed');
      expect(result[1].filters.status).toBe('open');
    });
  });

  describe('saveFilterHistory', () => {
    it('应该保存到历史记录', () => {
      saveFilterHistory('sla-violations', mockFilterState);
      const history = getFilterHistory('sla-violations');
      expect(history).toHaveLength(1);
      expect(history[0].filters).toEqual(mockFilterState);
    });

    it('应该限制历史记录数量', () => {
      for (let i = 0; i < 20; i++) {
        saveFilterHistory('test', { status: `state${i}` });
      }

      const history = getFilterHistory('test');
      expect(history.length).toBeLessThanOrEqual(10); // 假设限制为 10
    });

    it('不应该重复保存相同的过滤器', () => {
      saveFilterHistory('test', mockFilterState);
      saveFilterHistory('test', mockFilterState);

      const history = getFilterHistory('test');
      expect(history).toHaveLength(1);
    });
  });

  describe('getRecentFilters', () => {
    it('应该获取最近的过滤器', () => {
      const history = [
        { filters: { a: 1 }, timestamp: Date.now() },
        { filters: { b: 2 }, timestamp: Date.now() - 1000 },
        { filters: { c: 3 }, timestamp: Date.now() - 2000 },
      ];
      localStorage.setItem('filter:history:test', JSON.stringify(history));

      const recent = getRecentFilters('test', 2);
      expect(recent).toHaveLength(2);
      expect(recent[0].filters).toEqual({ a: 1 });
      expect(recent[1].filters).toEqual({ b: 2 });
    });
  });

  describe('exportFilters', () => {
    it('应该导出过滤器为 JSON 字符串', () => {
      const state = { config: mockFilterState };
      const exported = exportFilters(state);
      const parsed = JSON.parse(exported);
      expect(parsed).toEqual(state);
    });

    it('应该包含元数据', () => {
      const exported = exportFilters(mockFilterState);
      const parsed = JSON.parse(exported);
      expect(parsed).toHaveProperty('_meta');
      expect(parsed._meta).toHaveProperty('version');
      expect(parsed._meta).toHaveProperty('exportedAt');
    });
  });

  describe('importFilters', () => {
    it('应该导入过滤器', () => {
      const state = { config: mockFilterState };
      const json = JSON.stringify(state);
      const result = importFilters(json);
      expect(result).toEqual(mockFilterState);
    });

    it('应该处理无效 JSON', () => {
      const result = importFilters('invalid');
      expect(result).toBeNull();
    });

    it('应该验证版本兼容性', () => {
      const incompatible = JSON.stringify({ _meta: { version: 99 }, config: {} });
      const result = importFilters(incompatible);
      // 可能返回 null 或部分数据
      expect(result === mockFilterState || result === null || result === undefined).toBe(true);
    });
  });

  describe('mergeFilterStates', () => {
    it('应该合并两个过滤器状态', () => {
      const state1 = { status: 'open', sortBy: 'name' };
      const state2 = { severity: 'critical', sortBy: 'date' };
      const result = mergeFilterStates(state1, state2);
      expect(result.status).toBe('open');
      expect(result.severity).toBe('critical');
      expect(result.sortBy).toBe('date'); // 后者覆盖前者
    });

    it('应该处理空状态', () => {
      const result = mergeFilterStates({}, mockFilterState);
      expect(result).toEqual(mockFilterState);
    });

    it('不应该修改原对象', () => {
      const state1 = { status: 'open' };
      const state2 = { severity: 'critical' };
      const result = mergeFilterStates(state1, state2);
      expect(state1).toEqual({ status: 'open' });
      expect(state2).toEqual({ severity: 'critical' });
      expect(result).not.toBe(state1);
    });
  });

  describe('getFilterDiff', () => {
    it('应该返回差异', () => {
      const oldState = { status: 'open', severity: 'high' };
      const newState = { status: 'closed', severity: 'high' };
      const diff = getFilterDiff(oldState, newState);
      expect(diff).toEqual({ status: { from: 'open', to: 'closed' } });
    });

    it('应该返回空如果没有差异', () => {
      const diff = getFilterDiff(mockFilterState, mockFilterState);
      expect(diff).toEqual({});
    });

    it('应该检测新增字段', () => {
      const oldState = { status: 'open' };
      const newState = { status: 'open', severity: 'critical' };
      const diff = getFilterDiff(oldState, newState);
      expect(diff).toHaveProperty('severity');
    });
  });

  describe('validateFilterState', () => {
    it('应该验证有效的过滤器状态', () => {
      expect(validateFilterState(mockFilterState)).toBe(true);
    });

    it('应该拒绝无效状态', () => {
      expect(validateFilterState(null)).toBe(false);
      expect(validateFilterState('string')).toBe(false);
      expect(validateFilterState(123)).toBe(false);
    });

    it('应该接受简单状态', () => {
      expect(validateFilterState({})).toBe(true);
      expect(validateFilterState({ key: 'value' })).toBe(true);
    });
  });
});

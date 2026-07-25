/**
 * Tests for Report Engine
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { ReportEngine } from '../report-engine';

const mockData = [
  { name: 'Alice', age: 30, department: 'Engineering', salary: 100000, createdAt: '2024-01-15T10:00:00Z' },
  { name: 'Bob', age: 25, department: 'Sales', salary: 80000, createdAt: '2024-01-15T14:00:00Z' },
  { name: 'Charlie', age: 35, department: 'Engineering', salary: 120000, createdAt: '2024-02-10T10:00:00Z' },
  { name: 'Diana', age: 28, department: 'Sales', salary: 90000, createdAt: '2024-02-10T15:00:00Z' },
  { name: 'Eve', age: 32, department: 'Marketing', salary: 95000, createdAt: '2024-03-05T10:00:00Z' },
];

describe('ReportEngine', () => {
  describe('applyFilters', () => {
    it('should filter with equals operator', () => {
      const filters = [{ field: 'department', operator: 'equals' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { department: 'Engineering' });
      expect(result).toHaveLength(2);
    });

    it('should filter with not_equals operator', () => {
      const filters = [{ field: 'department', operator: 'not_equals' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { department: 'Sales' });
      expect(result).toHaveLength(3);
    });

    it('should filter with contains operator', () => {
      const filters = [{ field: 'name', operator: 'contains' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { name: 'li' });
      expect(result).toHaveLength(2); // Alice, Charlie
    });

    it('should filter with not_contains operator', () => {
      const filters = [{ field: 'name', operator: 'not_contains' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { name: 'li' });
      expect(result).toHaveLength(3);
    });

    it('should filter with starts_with operator', () => {
      const filters = [{ field: 'name', operator: 'starts_with' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { name: 'A' });
      expect(result).toHaveLength(1);
    });

    it('should filter with ends_with operator', () => {
      const filters = [{ field: 'name', operator: 'ends_with' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { name: 'e' });
      expect(result).toHaveLength(3); // Alice, Charlie, Eve
    });

    it('should filter with greater_than operator', () => {
      const filters = [{ field: 'age', operator: 'greater_than' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { age: 30 });
      expect(result).toHaveLength(2); // Charlie 35, Eve 32
    });

    it('should filter with less_than operator', () => {
      const filters = [{ field: 'age', operator: 'less_than' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { age: 30 });
      expect(result).toHaveLength(2); // Bob 25, Diana 28
    });

    it('should filter with between operator', () => {
      const filters = [{ field: 'age', operator: 'between' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { age: [28, 32] });
      expect(result).toHaveLength(3); // Alice 30, Diana 28, Eve 32
    });

    it('should return false for between with invalid value', () => {
      const filters = [{ field: 'age', operator: 'between' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { age: 'invalid' });
      expect(result).toHaveLength(0);
    });

    it('should filter with in operator', () => {
      const filters = [{ field: 'department', operator: 'in' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { department: ['Engineering', 'Marketing'] });
      expect(result).toHaveLength(3);
    });

    it('should filter with not_in operator', () => {
      const filters = [{ field: 'department', operator: 'not_in' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { department: ['Engineering'] });
      expect(result).toHaveLength(3);
    });

    it('should filter with is_null operator', () => {
      const dataWithNull = [...mockData, { name: null, age: 40, department: 'HR', salary: 70000, createdAt: '2024-04-01' }];
      const filters = [{ field: 'name', operator: 'is_null' as any }];
      const result = ReportEngine.applyFilters(dataWithNull, filters, { name: true });
      expect(result).toHaveLength(1);
    });

    it('should filter with is_not_null operator', () => {
      const filters = [{ field: 'name', operator: 'is_not_null' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { name: true });
      expect(result).toHaveLength(5);
    });

    it('should skip filters with null/undefined values', () => {
      const filters = [{ field: 'department', operator: 'equals' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, {});
      expect(result).toHaveLength(5);
    });

    it('should handle default operator case', () => {
      const filters = [{ field: 'name', operator: 'unknown_operator' as any }];
      const result = ReportEngine.applyFilters(mockData, filters, { name: 'test' });
      expect(result).toHaveLength(5);
    });
  });

  describe('applySorting', () => {
    it('should sort ascending', () => {
      const sorting = [{ field: 'age', order: 'asc' as any }];
      const result = ReportEngine.applySorting(mockData, sorting);
      expect((result[0] as any).name).toBe('Bob');
      expect((result[4] as any).name).toBe('Charlie');
    });

    it('should sort descending', () => {
      const sorting = [{ field: 'age', order: 'desc' as any }];
      const result = ReportEngine.applySorting(mockData, sorting);
      expect((result[0] as any).name).toBe('Charlie');
    });

    it('should handle multiple sort fields', () => {
      const sorting = [
        { field: 'department', order: 'asc' as any },
        { field: 'age', order: 'asc' as any },
      ];
      const result = ReportEngine.applySorting(mockData, sorting);
      expect((result[0] as any).department).toBe('Engineering');
    });

    it('should return data unchanged when no sorting', () => {
      const result = ReportEngine.applySorting(mockData, []);
      expect(result).toEqual(mockData);
    });

    it('should handle null/undefined values in sort', () => {
      const dataWithNull = [
        { name: 'A', value: null },
        { name: 'B', value: 5 },
        { name: 'C', value: undefined },
      ];
      const sorting = [{ field: 'value', order: 'asc' as any }];
      const result = ReportEngine.applySorting(dataWithNull, sorting);
      expect((result[0] as any).name).toBe('B');
    });
  });

  describe('applyGrouping', () => {
    it('should group by field', () => {
      const grouping = [{ field: 'department' }];
      const result = ReportEngine.applyGrouping(mockData, grouping as any);

      expect(result['Engineering']).toHaveLength(2);
      expect(result['Sales']).toHaveLength(2);
      expect(result['Marketing']).toHaveLength(1);
    });

    it('should return all data in single group when no grouping', () => {
      const result = ReportEngine.applyGrouping(mockData, []);
      expect(result['all']).toHaveLength(5);
    });

    it('should handle multiple group fields', () => {
      const data = [
        { dept: 'Eng', level: 'Senior' },
        { dept: 'Eng', level: 'Junior' },
        { dept: 'Sales', level: 'Senior' },
      ];
      const grouping = [{ field: 'dept' }, { field: 'level' }];
      const result = ReportEngine.applyGrouping(data, grouping as any);

      expect(result['Eng|Senior']).toHaveLength(1);
      expect(result['Eng|Junior']).toHaveLength(1);
    });
  });

  describe('applyAggregation', () => {
    it('should calculate count', () => {
      const metrics = [{ name: 'count', field: 'salary', function: 'count' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['count']).toBe(5);
    });

    it('should calculate sum', () => {
      const metrics = [{ name: 'total', field: 'salary', function: 'sum' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['total']).toBe(485000);
    });

    it('should calculate avg', () => {
      const metrics = [{ name: 'average', field: 'salary', function: 'avg' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['average']).toBe(97000);
    });

    it('should calculate min', () => {
      const metrics = [{ name: 'min', field: 'salary', function: 'min' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['min']).toBe(80000);
    });

    it('should calculate max', () => {
      const metrics = [{ name: 'max', field: 'salary', function: 'max' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['max']).toBe(120000);
    });

    it('should calculate distinct', () => {
      const metrics = [{ name: 'distinct', field: 'salary', function: 'distinct' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['distinct']).toBe(5);
    });

    it('should use alias when provided', () => {
      const metrics = [{ name: 'count', field: 'salary', function: 'count' as any, alias: 'totalCount' }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['totalCount']).toBe(5);
    });

    it('should return 0 for empty data with avg/min/max', () => {
      const metrics = [
        { name: 'avg', field: 'salary', function: 'avg' as any },
        { name: 'min', field: 'salary', function: 'min' as any },
        { name: 'max', field: 'salary', function: 'max' as any },
      ];
      const result = ReportEngine.applyAggregation([], metrics);
      expect(result['avg']).toBe(0);
      expect(result['min']).toBe(0);
      expect(result['max']).toBe(0);
    });

    it('should handle unknown function', () => {
      const metrics = [{ name: 'unknown', field: 'salary', function: 'unknown_func' as any }];
      const result = ReportEngine.applyAggregation(mockData, metrics);
      expect(result['unknown']).toBe(0);
    });
  });

  describe('aggregateByTime', () => {
    it('should aggregate by month', () => {
      const metrics = [{ name: 'count', field: 'name', function: 'count' as any }];
      const result = ReportEngine.aggregateByTime(mockData, 'createdAt', 'month', metrics);

      expect(result.length).toBeGreaterThan(0);
      expect(result[0]).toHaveProperty('time');
      expect(result[0]).toHaveProperty('count');
    });

    it('should aggregate by day', () => {
      const metrics = [{ name: 'count', field: 'name', function: 'count' as any }];
      const result = ReportEngine.aggregateByTime(mockData, 'createdAt', 'day', metrics);
      expect(result.length).toBeGreaterThan(0);
    });

    it('should aggregate by hour', () => {
      const metrics = [{ name: 'count', field: 'name', function: 'count' as any }];
      const result = ReportEngine.aggregateByTime(mockData, 'createdAt', 'hour', metrics);
      expect(result.length).toBeGreaterThan(0);
    });

    it('should aggregate by week', () => {
      const metrics = [{ name: 'count', field: 'name', function: 'count' as any }];
      const result = ReportEngine.aggregateByTime(mockData, 'createdAt', 'week', metrics);
      expect(result.length).toBeGreaterThan(0);
    });

    it('should skip rows without time value', () => {
      const dataWithNull = [...mockData, { name: 'NoTime', age: 40, department: 'HR', salary: 70000, createdAt: null }];
      const metrics = [{ name: 'count', field: 'name', function: 'count' as any }];
      const result = ReportEngine.aggregateByTime(dataWithNull, 'createdAt', 'month', metrics);
      const totalCount = result.reduce((sum, r) => sum + (r['count'] as number), 0);
      expect(totalCount).toBe(5);
    });
  });

  describe('createPivotTable', () => {
    it('should create pivot table', () => {
      const result = ReportEngine.createPivotTable(
        mockData,
        ['department'],
        ['age'],
        [{ field: 'salary', aggregation: 'sum' }]
      );

      expect(result.rowHeaders).toBeDefined();
      expect(result.columnHeaders).toBeDefined();
      expect(result.data).toBeDefined();
    });
  });

  describe('formatAsTable', () => {
    it('should format data as table', () => {
      const result = ReportEngine.formatAsTable(mockData, ['name', 'age', 'department']);

      expect(result.columns).toEqual(['name', 'age', 'department']);
      expect(result.rows).toHaveLength(5);
      expect(result.rows[0]).toEqual(['Alice', '30', 'Engineering']);
    });

    it('should handle null/undefined values', () => {
      const data = [{ name: null, age: undefined }];
      const result = ReportEngine.formatAsTable(data, ['name', 'age']);
      expect(result.rows[0]).toEqual(['', '']);
    });
  });
});

/**
 * 报表引擎
 * 负责报表数据处理、聚合和计算
 */

import type {
  ReportDataSource,
  ReportFilter,
  ReportSorting,
  ReportGrouping,
  FilterOperator,
  AggregationMetric,
} from '@/types/reports';

export class ReportEngine {
  // ==================== 数据过滤 ====================

  /**
   * 应用过滤器
   */
  static applyFilters(
    data: any[],
    filters: ReportFilter[],
    filterValues: Record<string, any>
  ): any[] {
    let filteredData = [...data];

    for (const filter of filters) {
      const value = filterValues[filter.field];
      if (value === undefined || value === null) continue;

      filteredData = filteredData.filter((row) => {
        const fieldValue = this.getNestedValue(row, filter.field);
        return this.matchesFilter(fieldValue, filter.operator, value);
      });
    }

    return filteredData;
  }

  /**
   * 匹配过滤条件
   */
  private static matchesFilter(
    fieldValue: any,
    operator: FilterOperator,
    filterValue: any
  ): boolean {
    switch (operator) {
      case 'equals':
        return fieldValue === filterValue;
      
      case 'not_equals':
        return fieldValue !== filterValue;
      
      case 'contains':
        return String(fieldValue).toLowerCase().includes(String(filterValue).toLowerCase());
      
      case 'not_contains':
        return !String(fieldValue).toLowerCase().includes(String(filterValue).toLowerCase());
      
      case 'starts_with':
        return String(fieldValue).toLowerCase().startsWith(String(filterValue).toLowerCase());
      
      case 'ends_with':
        return String(fieldValue).toLowerCase().endsWith(String(filterValue).toLowerCase());
      
      case 'greater_than':
        return Number(fieldValue) > Number(filterValue);
      
      case 'less_than':
        return Number(fieldValue) < Number(filterValue);
      
      case 'between':
        if (Array.isArray(filterValue) && filterValue.length === 2) {
          return Number(fieldValue) >= Number(filterValue[0]) && 
                 Number(fieldValue) <= Number(filterValue[1]);
        }
        return false;
      
      case 'in':
        return Array.isArray(filterValue) && filterValue.includes(fieldValue);
      
      case 'not_in':
        return Array.isArray(filterValue) && !filterValue.includes(fieldValue);
      
      case 'is_null':
        return fieldValue === null || fieldValue === undefined;
      
      case 'is_not_null':
        return fieldValue !== null && fieldValue !== undefined;
      
      default:
        return true;
    }
  }

  // ==================== 数据排序 ====================

  /**
   * 应用排序
   */
  static applySorting(data: any[], sorting: ReportSorting[]): any[] {
    if (!sorting || sorting.length === 0) return data;

    return [...data].sort((a, b) => {
      for (const sort of sorting) {
        const aValue = this.getNestedValue(a, sort.field);
        const bValue = this.getNestedValue(b, sort.field);
        
        const comparison = this.compareValues(aValue, bValue);
        
        if (comparison !== 0) {
          return sort.order === 'asc' ? comparison : -comparison;
        }
      }
      return 0;
    });
  }

  /**
   * 比较值
   */
  private static compareValues(a: any, b: any): number {
    if (a === b) return 0;
    if (a === null || a === undefined) return 1;
    if (b === null || b === undefined) return -1;
    
    if (typeof a === 'string' && typeof b === 'string') {
      return a.localeCompare(b);
    }
    
    return a < b ? -1 : 1;
  }

  // ==================== 数据分组 ====================

  /**
   * 应用分组
   */
  static applyGrouping(
    data: any[],
    grouping: ReportGrouping[]
  ): Record<string, any[]> {
    if (!grouping || grouping.length === 0) {
      return { all: data };
    }

    const groups: Record<string, any[]> = {};
    
    for (const row of data) {
      const groupKey = this.generateGroupKey(row, grouping);
      
      if (!groups[groupKey]) {
        groups[groupKey] = [];
      }
      
      groups[groupKey].push(row);
    }

    return groups;
  }

  /**
   * 生成分组键
   */
  private static generateGroupKey(row: any, grouping: ReportGrouping[]): string {
    return grouping
      .map((g) => {
        const value = this.getNestedValue(row, g.field);
        return value !== null && value !== undefined ? String(value) : 'null';
      })
      .join('|');
  }

  // ==================== 数据聚合 ====================

  /**
   * 应用聚合
   */
  static applyAggregation(
    data: any[],
    metrics: AggregationMetric[]
  ): Record<string, any> {
    const result: Record<string, any> = {};

    for (const metric of metrics) {
      const key = metric.alias || metric.name;
      result[key] = this.calculateMetric(data, metric);
    }

    return result;
  }

  /**
   * 计算指标
   */
  private static calculateMetric(
    data: any[],
    metric: AggregationMetric
  ): number {
    const values = data
      .map((row) => this.getNestedValue(row, metric.field))
      .filter((v) => v !== null && v !== undefined);

    switch (metric.function) {
      case 'count':
        return values.length;
      
      case 'sum':
        return values.reduce((sum, v) => sum + Number(v), 0);
      
      case 'avg':
        if (values.length === 0) return 0;
        return values.reduce((sum, v) => sum + Number(v), 0) / values.length;
      
      case 'min':
        if (values.length === 0) return 0;
        return Math.min(...values.map(Number));
      
      case 'max':
        if (values.length === 0) return 0;
        return Math.max(...values.map(Number));
      
      case 'distinct':
        return new Set(values).size;
      
      default:
        return 0;
    }
  }

  // ==================== 时间序列聚合 ====================

  /**
   * 按时间聚合
   */
  static aggregateByTime(
    data: any[],
    timeField: string,
    granularity: 'hour' | 'day' | 'week' | 'month',
    metrics: AggregationMetric[]
  ): Array<{ time: string; [key: string]: any }> {
    // 按时间分组
    const timeGroups: Record<string, any[]> = {};
    
    for (const row of data) {
      const timeValue = this.getNestedValue(row, timeField);
      if (!timeValue) continue;
      
      const timeKey = this.formatTimeKey(new Date(timeValue), granularity);
      
      if (!timeGroups[timeKey]) {
        timeGroups[timeKey] = [];
      }
      
      timeGroups[timeKey].push(row);
    }

    // 计算每个时间段的指标
    const result: Array<{ time: string; [key: string]: any }> = [];
    
    for (const [timeKey, groupData] of Object.entries(timeGroups)) {
      const row: any = { time: timeKey };
      
      for (const metric of metrics) {
        const key = metric.alias || metric.name;
        row[key] = this.calculateMetric(groupData, metric);
      }
      
      result.push(row);
    }

    // 按时间排序
    return result.sort((a, b) => a.time.localeCompare(b.time));
  }

  /**
   * 格式化时间键
   */
  private static formatTimeKey(
    date: Date,
    granularity: 'hour' | 'day' | 'week' | 'month'
  ): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hour = String(date.getHours()).padStart(2, '0');

    switch (granularity) {
      case 'hour':
        return `${year}-${month}-${day} ${hour}:00`;
      
      case 'day':
        return `${year}-${month}-${day}`;
      
      case 'week':
        const weekStart = new Date(date);
        weekStart.setDate(date.getDate() - date.getDay());
        const weekYear = weekStart.getFullYear();
        const weekMonth = String(weekStart.getMonth() + 1).padStart(2, '0');
        const weekDay = String(weekStart.getDate()).padStart(2, '0');
        return `${weekYear}-${weekMonth}-${weekDay}`;
      
      case 'month':
        return `${year}-${month}`;
      
      default:
        return date.toISOString();
    }
  }

  // ==================== 透视表 ====================

  /**
   * 创建透视表
   */
  static createPivotTable(
    data: any[],
    rows: string[],
    columns: string[],
    values: Array<{ field: string; aggregation: string }>
  ): {
    rowHeaders: string[][];
    columnHeaders: string[][];
    data: any[][];
  } {
    // 获取所有唯一的行和列值
    const rowValues = this.getUniqueValueCombinations(data, rows);
    const columnValues = this.getUniqueValueCombinations(data, columns);

    // 创建数据矩阵
    const matrix: any[][] = [];
    
    for (const rowValue of rowValues) {
      const row: any[] = [];
      
      for (const columnValue of columnValues) {
        // 过滤匹配当前行列的数据
        const filteredData = data.filter((item) => {
          const rowMatch = rows.every((field, i) => 
            this.getNestedValue(item, field) === rowValue[i]
          );
          const colMatch = columns.every((field, i) => 
            this.getNestedValue(item, field) === columnValue[i]
          );
          return rowMatch && colMatch;
        });

        // 计算聚合值
        const cell: any = {};
        for (const value of values) {
          const metric: AggregationMetric = {
            name: value.field,
            field: value.field,
            function: value.aggregation as any,
          };
          cell[value.field] = this.calculateMetric(filteredData, metric);
        }
        
        row.push(cell);
      }
      
      matrix.push(row);
    }

    return {
      rowHeaders: rowValues,
      columnHeaders: columnValues,
      data: matrix,
    };
  }

  /**
   * 获取唯一值组合
   */
  private static getUniqueValueCombinations(
    data: any[],
    fields: string[]
  ): any[][] {
    const combinations = new Set<string>();
    
    for (const row of data) {
      const values = fields.map((field) => this.getNestedValue(row, field));
      combinations.add(JSON.stringify(values));
    }
    
    return Array.from(combinations)
      .map((str) => JSON.parse(str))
      .sort((a, b) => {
        for (let i = 0; i < a.length; i++) {
          const comparison = this.compareValues(a[i], b[i]);
          if (comparison !== 0) return comparison;
        }
        return 0;
      });
  }

  // ==================== 工具方法 ====================

  /**
   * 获取嵌套值
   */
  private static getNestedValue(obj: any, path: string): any {
    const keys = path.split('.');
    let value = obj;
    
    for (const key of keys) {
      if (value === null || value === undefined) return undefined;
      value = value[key];
    }
    
    return value;
  }

  /**
   * 格式化数据为表格
   */
  static formatAsTable(
    data: any[],
    columns: string[]
  ): {
    columns: string[];
    rows: any[][];
  } {
    const rows = data.map((row) =>
      columns.map((col) => this.getNestedValue(row, col))
    );

    return { columns, rows };
  }
}

export default ReportEngine;


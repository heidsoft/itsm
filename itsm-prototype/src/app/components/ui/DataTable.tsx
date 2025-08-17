import { Search, Download, ChevronDown, ChevronUp } from 'lucide-react';
import { Pagination } from 'antd';

import React, { useMemo, useState, useCallback } from "react";
import LoadingEmptyError from "./LoadingEmptyError";
interface Column<T> {
  key: keyof T;
  title: string;
  sortable?: boolean;
  filterable?: boolean;
  render?: (value: unknown, record: T) => React.ReactNode;
  width?: string;
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  pagination?: {
    current: number;
    pageSize: number;
    total: number;
    onChange: (page: number, pageSize: number) => void;
  };
  rowSelection?: {
    selectedRowKeys: string[];
    onChange: (selectedRowKeys: string[]) => void;
  };
  searchable?: boolean;
  exportable?: boolean;
  onExport?: () => void;
}

export const DataTable = React.memo(<T extends Record<string, unknown>>({
  data,
  columns,
  loading = false,
  pagination,
  rowSelection,
  searchable = true,
  exportable = false,
  onExport,
}: DataTableProps<T>) => {
  const [sortConfig, setSortConfig] = useState<{
    key: keyof T | null;
    direction: "asc" | "desc";
  }>({ key: null, direction: "asc" });
  const [searchTerm, setSearchTerm] = useState("");
  const [filters, setFilters] = useState<Record<string, string>>({});

  // 排序逻辑
  const sortedData = useMemo(() => {
    if (!sortConfig.key) return data;

    return [...data].sort((a, b) => {
      const aValue = a[sortConfig.key!];
      const bValue = b[sortConfig.key!];

      if (aValue < bValue) {
        return sortConfig.direction === "asc" ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortConfig.direction === "asc" ? 1 : -1;
      }
      return 0;
    });
  }, [data, sortConfig]);

  // 搜索和过滤逻辑
  const filteredData = useMemo(() => {
    let result = sortedData;

    // 全局搜索
    if (searchTerm) {
      result = result.filter((item) =>
        Object.values(item).some((value) =>
          String(value).toLowerCase().includes(searchTerm.toLowerCase())
        )
      );
    }

    // 列过滤
    Object.entries(filters).forEach(([key, value]) => {
      if (value) {
        result = result.filter((item) =>
          String(item[key]).toLowerCase().includes(value.toLowerCase())
        );
      }
    });

    return result;
  }, [sortedData, searchTerm, filters]);

  const handleSort = useCallback((key: keyof T) => {
    setSortConfig((prev) => ({
      key,
      direction: prev.key === key && prev.direction === "asc" ? "desc" : "asc",
    }));
  }, []);

  const handleFilterChange = useCallback((key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  }, []);

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200">
      {/* 工具栏 */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            {searchable && (
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
                <input
                  type="text"
                  placeholder="搜索..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            )}
            {rowSelection && (
              <span className="text-sm text-gray-600">
                已选择 {rowSelection.selectedRowKeys.length} 项
              </span>
            )}
          </div>
          <div className="flex items-center space-x-2">
            {exportable && (
              <button
                onClick={onExport}
                className="flex items-center px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                <Download className="w-4 h-4 mr-2" />
                导出
              </button>
            )}
          </div>
        </div>
      </div>

      {/* 表格 */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              {rowSelection && (
                <th className="px-4 py-3 text-left">
                  <input
                    type="checkbox"
                    checked={
                      rowSelection.selectedRowKeys.length === data.length
                    }
                    onChange={(e) => {
                      if (e.target.checked) {
                        rowSelection.onChange(
                          data.map((_, index) => String(index))
                        );
                      } else {
                        rowSelection.onChange([]);
                      }
                    }}
                    className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                </th>
              )}
              {columns.map((column) => (
                <th
                  key={String(column.key)}
                  className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                  style={{ width: column.width }}
                >
                  <div className="flex items-center space-x-1">
                    <span>{column.title}</span>
                    {column.sortable && (
                      <button
                        onClick={() => handleSort(column.key)}
                        className="text-gray-400 hover:text-gray-600"
                      >
                        {sortConfig.key === column.key ? (
                          sortConfig.direction === "asc" ? (
                            <ChevronUp className="w-4 h-4" />
                          ) : (
                            <ChevronDown className="w-4 h-4" />
                          )
                        ) : (
                          <ChevronDown className="w-4 h-4" />
                        )}
                      </button>
                    )}
                  </div>
                  {column.filterable && (
                    <input
                      type="text"
                      placeholder={`过滤${column.title}`}
                      value={filters[String(column.key)] || ""}
                      onChange={(e) =>
                        handleFilterChange(String(column.key), e.target.value)
                      }
                      className="mt-1 w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-1 focus:ring-blue-500"
                    />
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            <LoadingEmptyError
              state={loading ? 'loading' : filteredData.length === 0 ? 'empty' : 'success'}
              loadingText="加载中..."
              empty={{
                title: "暂无数据",
                description: "当前没有找到匹配的数据"
              }}
              minHeight={200}
            >
              {filteredData.map((record, index) => (
                <tr key={index} className="hover:bg-gray-50">
                  {rowSelection && (
                    <td className="px-4 py-4">
                      <input
                        type="checkbox"
                        checked={rowSelection.selectedRowKeys.includes(
                          String(index)
                        )}
                        onChange={(e) => {
                          if (e.target.checked) {
                            rowSelection.onChange([
                              ...rowSelection.selectedRowKeys,
                              String(index),
                            ]);
                          } else {
                            rowSelection.onChange(
                              rowSelection.selectedRowKeys.filter(
                                (key) => key !== String(index)
                              )
                            );
                          }
                        }}
                        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                    </td>
                  )}
                  {columns.map((column) => (
                    <td
                      key={String(column.key)}
                      className="px-4 py-4 whitespace-nowrap text-sm text-gray-900"
                    >
                      {column.render
                        ? column.render(record[column.key], record)
                        : String(record[column.key])}
                    </td>
                  ))}
                </tr>
              ))}
            </LoadingEmptyError>
          </tbody>
        </table>
      </div>

      {/* 分页 */}
      {pagination && (
        <div className="px-4 py-3 border-t border-gray-200 flex items-center justify-between">
          <div className="text-sm text-gray-700">
            显示 {(pagination.current - 1) * pagination.pageSize + 1} 到{" "}
            {Math.min(
              pagination.current * pagination.pageSize,
              pagination.total
            )}{" "}
            条， 共 {pagination.total} 条记录
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={() =>
                pagination.onChange(pagination.current - 1, pagination.pageSize)
              }
              disabled={pagination.current === 1}
              className="px-3 py-1 text-sm border border-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
            >
              上一页
            </button>
            <span className="text-sm text-gray-700">
              第 {pagination.current} 页，共{" "}
              {Math.ceil(pagination.total / pagination.pageSize)} 页
            </span>
            <button
              onClick={() =>
                pagination.onChange(pagination.current + 1, pagination.pageSize)
              }
              disabled={
                pagination.current >=
                Math.ceil(pagination.total / pagination.pageSize)
              }
              className="px-3 py-1 text-sm border border-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
            >
              下一页
            </button>
          </div>
        </div>
      )}
    </div>
  );
});

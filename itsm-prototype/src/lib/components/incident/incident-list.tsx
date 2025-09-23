'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { IncidentAPI, Incident, ListIncidentsRequest, INCIDENT_STATUS, INCIDENT_PRIORITY } from '../../incident-api';

// 事件列表属性
export interface IncidentListProps {
  className?: string;
  onIncidentSelect?: (incident: Incident) => void;
  onIncidentCreate?: () => void;
  onIncidentEdit?: (incident: Incident) => void;
  showActions?: boolean;
  showFilters?: boolean;
  pageSize?: number;
}

// 筛选器组件
const IncidentFilters: React.FC<{
  filters: ListIncidentsRequest;
  onFiltersChange: (filters: ListIncidentsRequest) => void;
  onReset: () => void;
}> = ({ filters, onFiltersChange, onReset }) => {
  const handleFilterChange = (key: keyof ListIncidentsRequest, value: string | boolean | undefined) => {
    onFiltersChange({
      ...filters,
      [key]: value === '' ? undefined : value,
      page: 1, // 重置到第一页
    });
  };

  return (
    <div className="bg-white p-4 rounded-lg shadow-sm border border-gray-200 mb-6">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* 状态筛选 */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">状态</label>
          <select
            value={filters.status || ''}
            onChange={(e) => handleFilterChange('status', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="">全部状态</option>
            <option value={INCIDENT_STATUS.NEW}>新建</option>
            <option value={INCIDENT_STATUS.IN_PROGRESS}>处理中</option>
            <option value={INCIDENT_STATUS.RESOLVED}>已解决</option>
            <option value={INCIDENT_STATUS.CLOSED}>已关闭</option>
            <option value={INCIDENT_STATUS.CANCELLED}>已取消</option>
          </select>
        </div>

        {/* 优先级筛选 */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">优先级</label>
          <select
            value={filters.priority || ''}
            onChange={(e) => handleFilterChange('priority', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="">全部优先级</option>
            <option value={INCIDENT_PRIORITY.LOW}>低</option>
            <option value={INCIDENT_PRIORITY.MEDIUM}>中</option>
            <option value={INCIDENT_PRIORITY.HIGH}>高</option>
            <option value={INCIDENT_PRIORITY.CRITICAL}>紧急</option>
          </select>
        </div>

        {/* 重大事件筛选 */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">重大事件</label>
          <select
            value={filters.is_major_incident === undefined ? '' : filters.is_major_incident.toString()}
            onChange={(e) => handleFilterChange('is_major_incident', e.target.value === '' ? undefined : e.target.value === 'true')}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="">全部</option>
            <option value="true">是</option>
            <option value="false">否</option>
          </select>
        </div>

        {/* 搜索关键词 */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">搜索</label>
          <input
            type="text"
            value={filters.keyword || ''}
            onChange={(e) => handleFilterChange('keyword', e.target.value)}
            placeholder="搜索标题、描述..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      <div className="mt-4 flex justify-end">
        <button
          onClick={onReset}
          className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 focus:outline-none"
        >
          重置筛选
        </button>
      </div>
    </div>
  );
};

// 事件状态标签
const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case INCIDENT_STATUS.NEW:
        return { label: '新建', className: 'bg-blue-100 text-blue-800' };
      case INCIDENT_STATUS.IN_PROGRESS:
        return { label: '处理中', className: 'bg-yellow-100 text-yellow-800' };
      case INCIDENT_STATUS.RESOLVED:
        return { label: '已解决', className: 'bg-green-100 text-green-800' };
      case INCIDENT_STATUS.CLOSED:
        return { label: '已关闭', className: 'bg-gray-100 text-gray-800' };
      case INCIDENT_STATUS.CANCELLED:
        return { label: '已取消', className: 'bg-red-100 text-red-800' };
      default:
        return { label: status, className: 'bg-gray-100 text-gray-800' };
    }
  };

  const config = getStatusConfig(status);
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.className}`}>
      {config.label}
    </span>
  );
};

// 优先级标签
const PriorityBadge: React.FC<{ priority: string }> = ({ priority }) => {
  const getPriorityConfig = (priority: string) => {
    switch (priority) {
      case INCIDENT_PRIORITY.LOW:
        return { label: '低', className: 'bg-gray-100 text-gray-800' };
      case INCIDENT_PRIORITY.MEDIUM:
        return { label: '中', className: 'bg-blue-100 text-blue-800' };
      case INCIDENT_PRIORITY.HIGH:
        return { label: '高', className: 'bg-orange-100 text-orange-800' };
      case INCIDENT_PRIORITY.CRITICAL:
        return { label: '紧急', className: 'bg-red-100 text-red-800' };
      default:
        return { label: priority, className: 'bg-gray-100 text-gray-800' };
    }
  };

  const config = getPriorityConfig(priority);
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.className}`}>
      {config.label}
    </span>
  );
};

// 事件行组件
const IncidentRow: React.FC<{
  incident: Incident;
  onSelect?: (incident: Incident) => void;
  onEdit?: (incident: Incident) => void;
  showActions?: boolean;
}> = ({ incident, onSelect, onEdit, showActions = true }) => {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN');
  };

  return (
    <tr className="hover:bg-gray-50 cursor-pointer" onClick={() => onSelect?.(incident)}>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center">
          <div>
            <div className="text-sm font-medium text-gray-900">
              {incident.incident_number}
            </div>
            <div className="text-sm text-gray-500">
              {incident.title}
            </div>
          </div>
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <StatusBadge status={incident.status} />
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <PriorityBadge priority={incident.priority} />
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
        {incident.requester_name || '未知'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
        {incident.assignee_name || '未分配'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        {formatDate(incident.created_at)}
      </td>
      {showActions && (
        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onEdit?.(incident);
            }}
            className="text-blue-600 hover:text-blue-900 mr-3"
          >
            编辑
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onSelect?.(incident);
            }}
            className="text-gray-600 hover:text-gray-900"
          >
            查看
          </button>
        </td>
      )}
    </tr>
  );
};

// 分页组件
const Pagination: React.FC<{
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}> = ({ currentPage, totalPages, onPageChange }) => {
  const getPageNumbers = () => {
    const pages = [];
    const maxVisible = 5;
    
    let start = Math.max(1, currentPage - Math.floor(maxVisible / 2));
    const end = Math.min(totalPages, start + maxVisible - 1);
    
    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1);
    }
    
    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    
    return pages;
  };

  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-between px-4 py-3 bg-white border-t border-gray-200 sm:px-6">
      <div className="flex justify-between flex-1 sm:hidden">
        <button
          onClick={() => onPageChange(currentPage - 1)}
          disabled={currentPage <= 1}
          className="relative inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          上一页
        </button>
        <button
          onClick={() => onPageChange(currentPage + 1)}
          disabled={currentPage >= totalPages}
          className="relative ml-3 inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          下一页
        </button>
      </div>
      <div className="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
        <div>
          <p className="text-sm text-gray-700">
            第 <span className="font-medium">{currentPage}</span> 页，共{' '}
            <span className="font-medium">{totalPages}</span> 页
          </p>
        </div>
        <div>
          <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px">
            <button
              onClick={() => onPageChange(currentPage - 1)}
              disabled={currentPage <= 1}
              className="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              上一页
            </button>
            {getPageNumbers().map((page) => (
              <button
                key={page}
                onClick={() => onPageChange(page)}
                className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${
                  page === currentPage
                    ? 'z-10 bg-blue-50 border-blue-500 text-blue-600'
                    : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'
                }`}
              >
                {page}
              </button>
            ))}
            <button
              onClick={() => onPageChange(currentPage + 1)}
              disabled={currentPage >= totalPages}
              className="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              下一页
            </button>
          </nav>
        </div>
      </div>
    </div>
  );
};

// 主要的事件列表组件
export const IncidentList: React.FC<IncidentListProps> = ({
  className = '',
  onIncidentSelect,
  onIncidentCreate,
  onIncidentEdit,
  showActions = true,
  showFilters = true,
  pageSize = 20,
}) => {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState<ListIncidentsRequest>({
    page: 1,
    page_size: pageSize,
  });

  // 获取事件列表
  const fetchIncidents = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await IncidentAPI.listIncidents(filters);
      
      if (!response || !response.incidents) {
        throw new Error('API响应数据格式错误');
      }
      
      setIncidents(response.incidents);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch incidents:', error);
      setError(error instanceof Error ? error.message : '获取事件列表失败');
    } finally {
      setLoading(false);
    }
  }, [filters]);

  // 初始加载和筛选变化时重新获取数据
  useEffect(() => {
    fetchIncidents();
  }, [fetchIncidents]);

  // 处理筛选器变化
  const handleFiltersChange = (newFilters: ListIncidentsRequest) => {
    setFilters(newFilters);
  };

  // 重置筛选器
  const handleResetFilters = () => {
    setFilters({
      page: 1,
      page_size: pageSize,
    });
  };

  // 处理分页变化
  const handlePageChange = (page: number) => {
    setFilters(prev => ({ ...prev, page }));
  };

  // 计算总页数
  const totalPages = Math.ceil(total / (filters.page_size || pageSize));

  if (loading && incidents.length === 0) {
    return (
      <div className={`${className}`}>
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-2 text-gray-600">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className={`${className}`}>
      {/* 头部操作栏 */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">事件管理</h1>
          <p className="text-gray-600">共 {total} 个事件</p>
        </div>
        {onIncidentCreate && (
          <button
            onClick={onIncidentCreate}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            创建事件
          </button>
        )}
      </div>

      {/* 筛选器 */}
      {showFilters && (
        <IncidentFilters
          filters={filters}
          onFiltersChange={handleFiltersChange}
          onReset={handleResetFilters}
        />
      )}

      {/* 错误提示 */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <p className="text-sm text-red-800">{error}</p>
              <button
                onClick={fetchIncidents}
                className="mt-2 text-sm text-red-600 hover:text-red-500 underline"
              >
                重试
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 事件表格 */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                事件信息
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                状态
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                优先级
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                请求人
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                处理人
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                创建时间
              </th>
              {showActions && (
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              )}
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {incidents.length === 0 ? (
              <tr>
                <td colSpan={showActions ? 7 : 6} className="px-6 py-12 text-center text-gray-500">
                  {loading ? '加载中...' : '暂无事件数据'}
                </td>
              </tr>
            ) : (
              incidents.map((incident) => (
                <IncidentRow
                  key={incident.id}
                  incident={incident}
                  onSelect={onIncidentSelect}
                  onEdit={onIncidentEdit}
                  showActions={showActions}
                />
              ))
            )}
          </tbody>
        </table>

        {/* 分页 */}
        <Pagination
          currentPage={filters.page || 1}
          totalPages={totalPages}
          onPageChange={handlePageChange}
        />
      </div>
    </div>
  );
};

export default IncidentList;
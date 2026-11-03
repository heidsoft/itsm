'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { Card, Input, Button, Tag, Pagination, Spin, Empty, Select, Alert, App } from 'antd';
import {
  FileText,
  RefreshCw,
  ChevronRight,
  Clock,
  Hourglass,
  CheckCircle,
  XCircle,
  Calendar,
  Search,
  Filter,
} from 'lucide-react';

// API 接口类型定义
interface ServiceRequest {
  id: number;
  catalogId: number;
  requesterId: number;
  status:
    | 'submitted'
    | 'manager_approved'
    | 'it_approved'
    | 'security_approved'
    | 'provisioning'
    | 'delivered'
    | 'failed'
    | 'rejected'
    | 'cancelled'
    | string;
  reason: string;
  createdAt: string;
  catalog?: {
    id: number;
    name: string;
    category: string;
    description: string;
  };
  requester?: {
    id: number;
    name: string;
    email: string;
  };
}

import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type { ServiceRequestStatus } from '@/types/service-catalog';
import { formatDateTime } from '@/lib/formatters';

const RequestStatusBadge = ({ status }: { status: string }) => {
  const statusConfig = {
    submitted: { label: '已提交', color: 'gold', icon: Clock, pulse: true },
    manager_approved: { label: '主管已批', color: 'blue', icon: Hourglass, pulse: true },
    it_approved: { label: 'IT已批', color: 'blue', icon: Hourglass, pulse: true },
    security_approved: { label: '安全已批', color: 'green', icon: CheckCircle, pulse: false },
    provisioning: { label: '交付中', color: 'processing', icon: Hourglass, pulse: true },
    delivered: { label: '已交付', color: 'success', icon: CheckCircle, pulse: false },
    failed: { label: '交付失败', color: 'error', icon: XCircle, pulse: false },
    rejected: { label: '已拒绝', color: 'error', icon: XCircle, pulse: false },
    cancelled: { label: '已取消', color: 'default', icon: XCircle, pulse: false },
  };

  const config = statusConfig[status as keyof typeof statusConfig] || { label: status || '-', color: 'default', icon: XCircle, pulse: false };
  const Icon = config.icon;

  return (
    <Tag color={config.color} className="flex items-center gap-1 px-2 py-1">
      <Icon className="w-3 h-3" />
      {config.label}
    </Tag>
  );
};

const RequestCard = ({ request }: { request: ServiceRequest }) => {
  const formatDate = (dateString?: string) => formatDateTime(dateString) || '-';

  return (
    <Card
      className="mb-4 rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow"
     
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <div className="flex items-center gap-3 mb-2">
            <span className="text-sm font-mono text-gray-500 bg-gray-50 px-2 py-1 rounded">
              REQ-{String(request.id).padStart(5, '0')}
            </span>
            <RequestStatusBadge status={request.status} />
          </div>
          <h3 className="text-lg font-semibold text-gray-900 mb-2">
            {request.catalog?.name || '未知服务'}
          </h3>
          <p className="text-sm text-gray-600 mb-3">
            {request.catalog?.description || request.reason}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between text-sm text-gray-500">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1">
            <Calendar className="w-4 h-4" />
            <span>{formatDate(request.createdAt)}</span>
          </div>
          <div className="flex items-center gap-1">
            <FileText className="w-4 h-4" />
            <span>{request.catalog?.category || '其他'}</span>
          </div>
        </div>
        <Link href={`/my-requests/${request.id}`}>
          <Button type="link" className="flex items-center gap-1 p-0 h-auto">
            查看详情
            <ChevronRight className="w-4 h-4" />
          </Button>
        </Link>
      </div>
    </Card>
  );
};

const MyRequestsPage = () => {
  const { message } = App.useApp();
  const [requests, setRequests] = useState<ServiceRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  // 各状态的真实总数由服务端返回；不能再用「当前页条数」冒充全量
  const [statusCounts, setStatusCounts] = useState<Record<string, number>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 10;

  // 竞态守卫：快速翻页/切筛选/改关键字时，旧响应不得覆盖新结果
  const listRequestIdRef = useRef(0);
  const countsRequestIdRef = useRef(0);

  // 获取服务请求数据（分页、状态、关键字全部由后端过滤）
  const fetchRequests = async (page: number, status: string, keyword: string) => {
    const requestId = ++listRequestIdRef.current;
    setLoading(true);
    setError(null);
    try {
      const data = await ServiceCatalogApi.getServiceRequests({
        page,
        pageSize,
        status: status === 'all' ? undefined : (status as ServiceRequestStatus),
        keyword: keyword || undefined,
      });
      if (requestId !== listRequestIdRef.current) return;

      // 本页的 ServiceRequest 是早期按页面字段攒的局部视图，与 API 层线上形状
      // （ServiceRequestDetail）已经漂移（本页要求 reason 必填，API 层不保证）。
      // 显式 as unknown as 是为了让这处漂移**可见**：此前 API 返回 any，编译器
      // 根本无从发现两套形状不一致。收敛两套类型属独立工作项。
      setRequests(data.items as unknown as ServiceRequest[]);
      setTotal(data.total);
      setTotalPages(Math.max(1, data.totalPages));
    } catch (error) {
      if (requestId !== listRequestIdRef.current) return;
      console.error('API调用失败:', error);
      setError('加载失败，请重试');
      message.error('加载服务请求失败');
      setRequests([]);
      setTotal(0);
      setTotalPages(1);
    } finally {
      if (requestId === listRequestIdRef.current) setLoading(false);
    }
  };

  // 获取各状态真实计数：与列表同一套关键字条件，各自取服务端 total
  const fetchStatusCounts = async (keyword: string) => {
    const requestId = ++countsRequestIdRef.current;
    const statuses = ['submitted', 'provisioning', 'delivered', 'rejected'];
    try {
      const [allResult, ...statusResults] = await Promise.all([
        ServiceCatalogApi.getServiceRequests({ page: 1, pageSize: 1, keyword: keyword || undefined }),
        ...statuses.map(status =>
          ServiceCatalogApi.getServiceRequests({
            page: 1,
            pageSize: 1,
            status: status as ServiceRequestStatus,
            keyword: keyword || undefined,
          })
        ),
      ]);
      if (requestId !== countsRequestIdRef.current) return;

      const counts: Record<string, number> = { all: allResult.total };
      statuses.forEach((status, index) => {
        counts[status] = statusResults[index]?.total ?? 0;
      });
      setStatusCounts(counts);
    } catch (error) {
      if (requestId !== countsRequestIdRef.current) return;
      // 计数失败不能影响列表主流程，同时不展示过期数字
      console.error('加载状态计数失败:', error);
      setStatusCounts({});
    }
  };

  // 输入防抖：避免每次按键都打服务端
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchTerm.trim());
      setCurrentPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchTerm]);

  useEffect(() => {
    fetchRequests(currentPage, filter, debouncedSearch);
  }, [currentPage, filter, debouncedSearch]);

  useEffect(() => {
    fetchStatusCounts(debouncedSearch);
  }, [debouncedSearch]);

  const filterOptions = [
    { value: 'all', label: '全部', count: statusCounts.all ?? 0 },
    { value: 'submitted', label: '已提交', count: statusCounts.submitted ?? 0 },
    { value: 'provisioning', label: '交付中', count: statusCounts.provisioning ?? 0 },
    { value: 'delivered', label: '已交付', count: statusCounts.delivered ?? 0 },
    { value: 'rejected', label: '已拒绝', count: statusCounts.rejected ?? 0 },
  ];

  return (
    <div className="min-h-screen p-6 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        {/* 页面头部 */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900 mb-1">我的请求</h1>
              <p className="text-gray-500">查看和跟踪您提交的所有服务请求</p>
            </div>
            <Button
              onClick={() => fetchRequests(currentPage, filter, debouncedSearch)}
              icon={<RefreshCw className="w-4 h-4" />}
            >
              刷新
            </Button>
          </div>
        </div>

        {/* 错误提示 */}
        {error && (
          <Alert
            title={error}
            description="请检查网络连接或稍后重试"
            type="error"
            showIcon
            className="mb-6"
            action={
              <Button size="small" onClick={() => fetchRequests(currentPage, filter, debouncedSearch)}>
                重试
              </Button>
            }
          />
        )}

        {/* 搜索和筛选 */}
        <Card className="mb-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex flex-col lg:flex-row gap-4">
            {/* 搜索框 */}
            <div className="flex-1">
              <Input
                placeholder="搜索服务名称或描述..."
                prefix={<Search className="text-gray-400 w-4 h-4" />}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className="w-full"
              />
            </div>

            {/* 状态筛选 */}
            <div className="flex items-center gap-2">
              <Filter className="w-5 h-5 text-gray-500" />
              <div className="flex gap-2 flex-wrap">
                {filterOptions.map(option => (
                  <Button
                    key={option.value}
                    type={filter === option.value ? 'primary' : 'default'}
                    onClick={() => {
                      setFilter(option.value);
                      setCurrentPage(1);
                    }}
                    className={filter !== option.value ? 'bg-gray-50 border-gray-200' : ''}
                  >
                    {option.label}
                    {option.count > 0 && (
                      <span className="ml-1.5 text-xs opacity-75">({option.count})</span>
                    )}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </Card>

        {/* 请求列表 */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Spin size="large" />
          </div>
        ) : requests.length > 0 ? (
          <div className="space-y-4">
            {requests.map(request => (
              <RequestCard key={request.id} request={request} />
            ))}
          </div>
        ) : (
          <Card
            className="text-center py-12 rounded-lg shadow-sm border border-gray-200"
           
          >
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div className="mb-4">
                  <h3 className="text-lg font-medium text-gray-900 mb-2">
                    {debouncedSearch ? '没有匹配的请求' : '暂无请求'}
                  </h3>
                  <p className="text-gray-500">
                    {debouncedSearch
                      ? `未找到包含“${debouncedSearch}”的请求`
                      : filter === 'all'
                        ? '您还没有提交任何服务请求'
                        : '当前状态下没有请求'}
                  </p>
                </div>
              }
            >
              {debouncedSearch ? (
                <Button onClick={() => setSearchTerm('')}>清除搜索</Button>
              ) : (
                <Link href="/service-catalog">
                  <Button type="primary" icon={<FileText className="w-4 h-4" />}>
                    浏览服务目录
                  </Button>
                </Link>
              )}
            </Empty>
          </Card>
        )}

        {/* 分页 */}
        {totalPages > 1 && (
          <Card
            className="mt-8 rounded-lg shadow-sm border border-gray-200"
           
            styles={{ body: { padding: '16px 24px' } }}
          >
            <div className="flex items-center justify-between">
              <div className="text-sm text-gray-500">
                显示第 {(currentPage - 1) * pageSize + 1} -{' '}
                {Math.min(currentPage * pageSize, total)} 条，共 {total} 条记录
              </div>
              <Pagination
                current={currentPage}
                total={total}
                pageSize={pageSize}
                onChange={page => setCurrentPage(page)}
                showSizeChanger={false}
              />
            </div>
          </Card>
        )}
      </div>
    </div>
  );
};

export default MyRequestsPage;

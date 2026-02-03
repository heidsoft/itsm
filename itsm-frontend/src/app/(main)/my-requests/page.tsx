'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { Card, Input, Button, Tag, Pagination, Spin, Empty, Select } from 'antd';
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
  catalog_id: number;
  requester_id: number;
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
  created_at: string;
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

const RequestStatusBadge = ({ status }: { status: string }) => {
  const statusConfig = {
    submitted: {
      label: '已提交',
      color: 'gold',
      icon: Clock,
      pulse: true,
    },
    manager_approved: {
      label: '主管已批',
      color: 'blue',
      icon: Hourglass,
      pulse: true,
    },
    it_approved: {
      label: 'IT已批',
      color: 'blue',
      icon: Hourglass,
      pulse: true,
    },
    security_approved: {
      label: '安全已批',
      color: 'green',
      icon: CheckCircle,
      pulse: false,
    },
    provisioning: {
      label: '交付中',
      color: 'processing',
      icon: Hourglass,
      pulse: true,
    },
    delivered: {
      label: '已交付',
      color: 'success',
      icon: CheckCircle,
      pulse: false,
    },
    failed: {
      label: '交付失败',
      color: 'error',
      icon: XCircle,
      pulse: false,
    },
    rejected: {
      label: '已拒绝',
      color: 'error',
      icon: XCircle,
      pulse: false,
    },
    cancelled: {
      label: '已取消',
      color: 'default',
      icon: XCircle,
      pulse: false,
    },
  };

  const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.submitted;
  const Icon = config.icon;

  return (
    <Tag color={config.color} className='flex items-center gap-1 px-2 py-1'>
      <Icon className='w-3 h-3' />
      {config.label}
    </Tag>
  );
};

const RequestCard = ({ request }: { request: ServiceRequest }) => {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <Card
      className='mb-4 rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow'
      variant="borderless"
    >
      <div className='flex items-start justify-between mb-4'>
        <div className='flex-1'>
          <div className='flex items-center gap-3 mb-2'>
            <span className='text-sm font-mono text-gray-500 bg-gray-50 px-2 py-1 rounded'>
              REQ-{String(request.id).padStart(5, '0')}
            </span>
            <RequestStatusBadge status={request.status} />
          </div>
          <h3 className='text-lg font-semibold text-gray-900 mb-2'>
            {request.catalog?.name || '未知服务'}
          </h3>
          <p className='text-sm text-gray-600 mb-3'>
            {request.catalog?.description || request.reason}
          </p>
        </div>
      </div>

      <div className='flex items-center justify-between text-sm text-gray-500'>
        <div className='flex items-center gap-4'>
          <div className='flex items-center gap-1'>
            <Calendar className='w-4 h-4' />
            <span>{formatDate(request.created_at)}</span>
          </div>
          <div className='flex items-center gap-1'>
            <FileText className='w-4 h-4' />
            <span>{request.catalog?.category || '其他'}</span>
          </div>
        </div>
        <Link href={`/my-requests/${request.id}`}>
          <Button type='link' className='flex items-center gap-1 p-0 h-auto'>
            查看详情
            <ChevronRight className='w-4 h-4' />
          </Button>
        </Link>
      </div>
    </Card>
  );
};

const MyRequestsPage = () => {
  const [requests, setRequests] = useState<ServiceRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 10;

  // 获取服务请求数据
  const fetchRequests = async (page = 1, status = filter) => {
    setLoading(true);
    try {
      const data = await ServiceCatalogApi.getServiceRequests({
        page,
        pageSize,
        status: status === 'all' ? undefined : status,
      } as any);

      setRequests((data.requests || []) as ServiceRequest[]);
      setTotal(data.total || 0);
      setTotalPages(Math.max(1, Math.ceil((data.total || 0) / pageSize)));
    } catch (error) {
      console.error('API调用失败:', error);
      setRequests([]);
      setTotal(0);
      setTotalPages(1);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRequests(currentPage, filter);
  }, [currentPage, filter]);

  // 筛选数据
  const filteredRequests = requests.filter(request => {
    const matchesSearch =
      !searchTerm ||
      request.catalog?.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      request.reason.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesSearch;
  });

  const filterOptions = [
    { value: 'all', label: '全部', count: total },
    { value: 'submitted', label: '已提交', count: 0 },
    { value: 'provisioning', label: '交付中', count: 0 },
    { value: 'delivered', label: '已交付', count: 0 },
    { value: 'rejected', label: '已拒绝', count: 0 },
  ];

  return (
    <div className='min-h-screen p-6 bg-gray-50'>
      <div className='max-w-7xl mx-auto'>
        {/* 页面头部 */}
        <div className='mb-8'>
          <div className='flex items-center justify-between'>
            <div>
              <h1 className='text-2xl font-bold text-gray-900 mb-1'>我的请求</h1>
              <p className='text-gray-500'>查看和跟踪您提交的所有服务请求</p>
            </div>
            <Button
              onClick={() => fetchRequests(currentPage, filter)}
              icon={<RefreshCw className='w-4 h-4' />}
            >
              刷新
            </Button>
          </div>
        </div>

        {/* 搜索和筛选 */}
        <Card className='mb-6 rounded-lg shadow-sm border border-gray-200' variant="borderless">
          <div className='flex flex-col lg:flex-row gap-4'>
            {/* 搜索框 */}
            <div className='flex-1'>
              <Input
                placeholder='搜索服务名称或描述...'
                prefix={<Search className='text-gray-400 w-4 h-4' />}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className='w-full'
              />
            </div>

            {/* 状态筛选 */}
            <div className='flex items-center gap-2'>
              <Filter className='w-5 h-5 text-gray-500' />
              <div className='flex gap-2 flex-wrap'>
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
                      <span className='ml-1.5 text-xs opacity-75'>({option.count})</span>
                    )}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </Card>

        {/* 请求列表 */}
        {loading ? (
          <div className='flex items-center justify-center py-12'>
            <Spin size='large' />
          </div>
        ) : filteredRequests.length > 0 ? (
          <div className='space-y-4'>
            {filteredRequests.map(request => (
              <RequestCard key={request.id} request={request} />
            ))}
          </div>
        ) : (
          <Card
            className='text-center py-12 rounded-lg shadow-sm border border-gray-200'
            variant="borderless"
          >
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div className='mb-4'>
                  <h3 className='text-lg font-medium text-gray-900 mb-2'>暂无请求</h3>
                  <p className='text-gray-500'>您还没有提交任何服务请求</p>
                </div>
              }
            >
              <Link href='/service-catalog'>
                <Button type='primary' icon={<FileText className='w-4 h-4' />}>
                  浏览服务目录
                </Button>
              </Link>
            </Empty>
          </Card>
        )}

        {/* 分页 */}
        {totalPages > 1 && (
          <Card
            className='mt-8 rounded-lg shadow-sm border border-gray-200'
            variant="borderless"
            bodyStyle={{ padding: '16px 24px' }}
          >
            <div className='flex items-center justify-between'>
              <div className='text-sm text-gray-500'>
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

'use client';

/**
 * 服务请求列表组件
 * 包含"我的请求"和"待办审批"两个视图
 */

import React, { useState, useEffect } from 'react';
import { App, Table, Tag, Button, Tabs, Card, Space, Tooltip } from 'antd';
import { Eye, RefreshCw, CheckCircle } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import LoadingEmptyError from '@/components/ui/LoadingEmptyError';

import { ServiceRequestApi } from '@/lib/api/';
import { ServiceRequestStatus, serviceRequestStatusLabel } from '@/constants/service-request';
import type { ServiceRequest } from '@/lib/api/service-request-api';

// 列表查询条件：这个组件只提供分页，状态过滤由后端 status 参数负责但当前视图没有入口，
// 因此不再维护一个从不发送的 scope 字段（旧 ServiceRequestQuery 与后端契约无关）。
interface ListQuery {
  page: number;
  pageSize: number;
}

// 状态标签颜色映射
const statusColors: Record<string, string> = {
  [ServiceRequestStatus.SUBMITTED]: 'blue',
  [ServiceRequestStatus.MANAGER_APPROVED]: 'cyan',
  [ServiceRequestStatus.IT_APPROVED]: 'geekblue',
  [ServiceRequestStatus.SECURITY_APPROVED]: 'purple',
  [ServiceRequestStatus.PROVISIONING]: 'processing',
  [ServiceRequestStatus.DELIVERED]: 'green',
  [ServiceRequestStatus.FAILED]: 'red',
  [ServiceRequestStatus.REJECTED]: 'red',
  [ServiceRequestStatus.CANCELLED]: 'default',
};

const ServiceRequestList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState('my-requests');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ServiceRequest[]>([]);
  const [total, setTotal] = useState(0);

  // 查询状态
  const [query, setQuery] = useState<ListQuery>({
    page: 1,
    pageSize: 10,
  });

  // 加载数据
  const loadData = async () => {
    setLoading(true);
    try {
      // 「我的请求」与「待办审批」是两个收件箱，由不同后端路径表达；
      // scope 查询参数从未被发送，也不再是前端状态。
      const resp =
        activeTab === 'approvals'
          ? await ServiceRequestApi.getPendingApprovals({ page: query.page, pageSize: query.pageSize })
          : await ServiceRequestApi.getUserServiceRequests({
              page: query.page,
              pageSize: query.pageSize,
            });
      setData(resp.items);
      setTotal(resp.total);
    } catch {
      message.error('加载服务请求失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
     
  }, [query, activeTab]);

  // 处理Tab切换
  const handleTabChange = (key: string) => {
    setActiveTab(key);
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  // 表格列定义
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '标题',
      dataIndex: 'title',
      render: (text: string, record: ServiceRequest) => (
        <div className="flex flex-col">
          <span className="font-medium text-gray-900">{text || `请求 #${record.id}`}</span>
          <span className="text-xs text-gray-500">{record.catalog?.name || '未知服务'}</span>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 150,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>{serviceRequestStatusLabel(status)}</Tag>
      ),
    },
    {
      title: '当前步骤',
      dataIndex: 'currentLevel',
      width: 120,
      render: (level: number, record: ServiceRequest) => (
        <span>
          Level {level} / {record.totalLevels}
        </span>
      ),
      responsive: ['md'],
    },
    {
      title: '提交时间',
      dataIndex: 'createdAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
      responsive: ['sm'],
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: ServiceRequest) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              aria-label={`查看服务请求 ${record.id} 详情`}
              icon={<Eye />}
              className="text-blue-600 hover:text-blue-800 hover:bg-blue-50"
              onClick={() => router.push(`/service-requests/${record.id}`)}
            />
          </Tooltip>
          {activeTab === 'approvals' && (
            <Tooltip title="审批">
              <Button
                type="text"
                aria-label={`审批服务请求 ${record.id}`}
                icon={<CheckCircle />}
                className="text-green-600 hover:text-green-800 hover:bg-green-50"
                onClick={() => router.push(`/service-requests/${record.id}`)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card className="rounded-lg shadow-sm border border-gray-200">
      <div className="flex justify-between items-center mb-4">
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          size="large"
          className="flex-1"
          items={[
            { label: '我的请求', key: 'my-requests' },
            { label: '待办审批', key: 'approvals' },
          ]}
        />
        <Button icon={<RefreshCw />} onClick={loadData}>
          刷新
        </Button>
      </div>

      <Table
        rowKey="id"
        columns={columns as any}
        dataSource={data}
        loading={loading}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: (
            <LoadingEmptyError
              state="empty"
              empty={{
                title: '暂无服务请求数据',
                actionText: '新建服务请求',
                onAction: () => router.push('/service-requests/new'),
                showAction: true,
                icon: <CheckCircle size={48} />,
              }}
            />
          ),
        }}
        pagination={{
          current: query.page,
          pageSize: query.pageSize,
          total: total,
          onChange: (page, pageSize) => setQuery(prev => ({ ...prev, page, pageSize })),
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: total => `共 ${total} 条记录`,
          pageSizeOptions: ['10', '20', '50', '100'],
        }}
      />
    </Card>
  );
};

export default ServiceRequestList;

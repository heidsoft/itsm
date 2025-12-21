'use client';

import React from 'react';
import { Table, Tag, Button, Space, Badge } from 'antd';
import { Eye, Edit, MoreHorizontal, AlertTriangle } from 'lucide-react';
import { Problem, ProblemStatus, ProblemPriority } from '@/lib/services/problem-service';
import { useRouter } from 'next/navigation';

interface ProblemListProps {
  problems: Problem[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
  pagination: any;
  onTableChange: (page: number, pageSize?: number) => void;
}

export const ProblemList: React.FC<ProblemListProps> = ({
  problems,
  loading,
  selectedRowKeys,
  onSelectedRowKeysChange,
  pagination,
  onTableChange,
}) => {
  const router = useRouter();

  const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    open: {
      color: '#fa8c16',
      text: '待处理',
      backgroundColor: '#fff7e6',
    },
    'in-progress': {
      color: '#1890ff',
      text: '处理中',
      backgroundColor: '#e6f7ff',
    },
    resolved: {
      color: '#52c41a',
      text: '已解决',
      backgroundColor: '#f6ffed',
    },
    closed: {
      color: '#00000073',
      text: '已关闭',
      backgroundColor: '#fafafa',
    },
  };

  const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> =
    {
      low: {
        color: '#52c41a',
        text: '低',
        backgroundColor: '#f6ffed',
      },
      medium: {
        color: '#1890ff',
        text: '中',
        backgroundColor: '#e6f7ff',
      },
      high: {
        color: '#fa8c16',
        text: '高',
        backgroundColor: '#fff7e6',
      },
      critical: {
        color: '#ff4d4f',
        text: '紧急',
        backgroundColor: '#fff2f0',
      },
    };

  const columns = [
    {
      title: '问题信息',
      key: 'problem_info',
      width: 300,
      render: (_: unknown, record: Problem) => (
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 40,
              height: 40,
              backgroundColor: '#e6f7ff',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
            }}
          >
            <AlertTriangle size={20} style={{ color: '#1890ff' }} />
          </div>
          <div>
            <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>
              {record.title}
            </div>
            <div style={{ fontSize: 'small', color: '#666' }}>
              #{record.id} • {record.category}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: ProblemStatus) => {
        const config = statusConfig[status];
        return (
          <span
            style={{
              padding: '4px 12px',
              borderRadius: 16,
              fontSize: 'small',
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: ProblemPriority) => {
        const config = priorityConfig[priority];
        return (
          <span
            style={{
              padding: '4px 12px',
              borderRadius: 16,
              fontSize: 'small',
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: '影响范围',
      dataIndex: 'impact',
      key: 'impact',
      width: 120,
      render: (impact: string) => {
        const impactConfig: Record<string, { color: string; text: string }> = {
          low: { color: 'green', text: '低' },
          medium: { color: 'orange', text: '中' },
          high: { color: 'red', text: '高' },
        };
        const config = impactConfig[impact] || { color: 'default', text: impact };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '负责人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 150,
      render: (assignee: { name: string }) => (
        <div style={{ fontSize: 'small' }}>{assignee?.name || '未分配'}</div>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (created_at: string) => (
        <div style={{ fontSize: 'small', color: '#666' }}>
          {new Date(created_at).toLocaleDateString('zh-CN')}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Problem) => (
        <Space size='small'>
          <Button
            type='text'
            size='small'
            icon={<Eye size={16} />}
            onClick={() => router.push(`/problems/${record.id}`)}
            className='text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors duration-200'
            title='查看详情'
          />
          <Button
            type='text'
            size='small'
            icon={<Edit size={16} />}
            onClick={() => router.push(`/problems/${record.id}/edit`)}
            className='text-green-600 hover:text-green-800 hover:bg-green-50 rounded-md transition-colors duration-200'
            title='编辑问题'
          />
          <Button
            type='text'
            size='small'
            icon={<MoreHorizontal size={16} />}
            className='text-gray-600 hover:text-gray-800 hover:bg-gray-50 rounded-md transition-colors duration-200'
            title='更多操作'
          />
        </Space>
      ),
    },
  ];

  return (
    <Table
      rowSelection={{
        selectedRowKeys,
        onChange: onSelectedRowKeysChange,
      }}
      columns={columns}
      dataSource={problems}
      rowKey='id'
      loading={loading}
      pagination={{
        current: pagination.current,
        pageSize: pagination.pageSize,
        total: pagination.total,
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        onChange: onTableChange,
      }}
      scroll={{ x: 1200 }}
      className='[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:border-b-2 [&_.ant-table-thead>tr>th]:border-gray-200 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-thead>tr>th]:text-gray-900 [&_.ant-table-tbody>tr:hover>td]:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-b [&_.ant-table-tbody>tr>td]:border-gray-100'
    />
  );
};

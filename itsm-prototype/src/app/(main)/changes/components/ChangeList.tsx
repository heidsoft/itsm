'use client';

import React from 'react';
import { Table, Button, Space, Badge } from 'antd';
import { Eye, Edit, MoreHorizontal } from 'lucide-react';
import { Change } from '@/lib/services/change-service';

const getChangeStatusColor = (status: string) => {
  switch (status) {
    case 'pending':
      return {
        color: '#fa8c16',
        text: '待审批',
        backgroundColor: '#fff7e6',
      };
    case 'approved':
      return {
        color: '#1890ff',
        text: '审批通过',
        backgroundColor: '#e6f7ff',
      };
    case 'implementing':
      return {
        color: '#722ed1',
        text: '实施中',
        backgroundColor: '#f9f0ff',
      };
    case 'completed':
      return {
        color: '#52c41a',
        text: '已完成',
        backgroundColor: '#f6ffed',
      };
    case 'cancelled':
      return {
        color: '#ff4d4f',
        text: '已取消',
        backgroundColor: '#fff2f0',
      };
    case 'draft':
      return {
        color: '#8c8c8c',
        text: '草稿',
        backgroundColor: '#f5f5f5',
      };
    case 'rejected':
      return {
        color: '#ff4d4f',
        text: '已拒绝',
        backgroundColor: '#fff2f0',
      };
    default:
      return {
        color: '#8c8c8c',
        text: status,
        backgroundColor: '#f5f5f5',
      };
  }
};

const getPriorityColor = (priority: string) => {
  switch (priority) {
    case 'urgent':
      return {
        color: '#ff4d4f',
        text: '紧急',
        backgroundColor: '#fff2f0',
      };
    case 'high':
      return {
        color: '#fa8c16',
        text: '高',
        backgroundColor: '#fff7e6',
      };
    case 'medium':
      return {
        color: '#1890ff',
        text: '中',
        backgroundColor: '#e6f7ff',
      };
    case 'low':
      return {
        color: '#52c41a',
        text: '低',
        backgroundColor: '#f6ffed',
      };
    default:
      return {
        color: '#8c8c8c',
        text: priority,
        backgroundColor: '#f5f5f5',
      };
  }
};

interface ChangeListProps {
  changes: Change[];
  loading: boolean;
  pagination: any;
  selectedRowKeys: React.Key[];
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
  onTableChange: (pagination: any) => void;
}

export const ChangeList: React.FC<ChangeListProps> = ({
  changes,
  loading,
  pagination,
  selectedRowKeys,
  onSelectedRowKeysChange,
  onTableChange,
}) => {
  const columns = [
    {
      title: '变更信息',
      dataIndex: 'title',
      key: 'title',
      width: 300,
      render: (_: unknown, record: Change) => (
        <div>
          <div className='font-medium text-gray-900 mb-1'>{record.title}</div>
          <div className='text-sm text-gray-500 mb-2'>{record.description}</div>
          <div className='flex items-center space-x-2'>
            <Badge
              color={
                record.type === 'emergency'
                  ? 'red'
                  : record.type === 'standard'
                  ? 'green'
                  : 'blue'
              }
              text={
                record.type === 'emergency'
                  ? '紧急变更'
                  : record.type === 'standard'
                  ? '标准变更'
                  : '普通变更'
              }
            />
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const statusConfig = getChangeStatusColor(status);
        return <Badge color={statusConfig.color} text={statusConfig.text} />;
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => {
        const priorityConfig = getPriorityColor(priority);
        return <Badge color={priorityConfig.color} text={priorityConfig.text} />;
      },
    },
    {
      title: '申请人',
      dataIndex: 'createdByName',
      key: 'createdByName',
      width: 120,
      render: (createdByName: string) => <div style={{ fontSize: 'small' }}>{createdByName}</div>,
    },
    {
      title: '审批人',
      dataIndex: 'assigneeName',
      key: 'assigneeName',
      width: 120,
      render: (assigneeName: string) => (
        <div style={{ fontSize: 'small' }}>{assigneeName || '未分配'}</div>
      ),
    },
    {
      title: '计划时间',
      key: 'plannedTime',
      width: 180,
      render: (_: unknown, record: Change) => (
        <div style={{ fontSize: 'small' }}>
          {record.plannedStartDate && (
            <>
              <div>{new Date(record.plannedStartDate).toLocaleDateString('zh-CN')}</div>
              <div style={{ color: '#666' }}>
                {new Date(record.plannedStartDate).toLocaleTimeString('zh-CN', {
                  hour: '2-digit',
                  minute: '2-digit',
                })}{' '}
                -{' '}
                {record.plannedEndDate &&
                  new Date(record.plannedEndDate).toLocaleTimeString('zh-CN', {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
              </div>
            </>
          )}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: Change) => (
        <Space size='small'>
          <Button
            type='text'
            icon={<Eye size={16} />}
            className='text-blue-600 hover:text-blue-800 hover:bg-blue-50'
            title='查看详情'
            onClick={() => console.log('查看变更', record.id)}
          />
          <Button
            type='text'
            icon={<Edit size={16} />}
            className='text-green-600 hover:text-green-800 hover:bg-green-50'
            title='编辑变更'
            onClick={() => console.log('编辑变更', record.id)}
          />
          <Button
            type='text'
            icon={<MoreHorizontal size={16} />}
            className='text-gray-600 hover:text-gray-800 hover:bg-gray-50'
            title='更多操作'
            onClick={() => console.log('更多操作', record.id)}
          />
        </Space>
      ),
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={changes}
      rowKey='id'
      loading={loading}
      pagination={{
        ...pagination,
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
      }}
      onChange={onTableChange}
      rowSelection={{
        selectedRowKeys,
        onChange: onSelectedRowKeysChange,
      }}
      className='[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-tbody>tr:hover>td]:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-gray-100'
    />
  );
};

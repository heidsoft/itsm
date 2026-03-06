'use client';

/**
 * 审批链表格组件
 */

import React, { useMemo, useCallback } from 'react';
import { Table, Tag, Button, Space, Tooltip, Dropdown, Switch } from 'antd';
import {
  Edit,
  Trash2,
  Eye,
  MoreHorizontal,
  Play,
  Pause,
  Copy,
} from 'lucide-react';
import { ApprovalChain, ApprovalChainFilters } from '@/types/approval-chain';
import { TableColumn, ActionButton } from '@/types/common';

interface ApprovalChainTableProps {
  dataSource: ApprovalChain[];
  loading?: boolean;
  pagination?: {
    current: number;
    pageSize: number;
    total: number;
  };
  selectedRowKeys: React.Key[];
  onPaginationChange: (page: number, pageSize: number) => void;
  onRowSelectionChange: (selectedRowKeys: React.Key[], selectedRows: ApprovalChain[]) => void;
  onEdit: (chain: ApprovalChain) => void;
  onView: (chain: ApprovalChain) => void;
  onDelete: (chain: ApprovalChain) => void;
  onToggleStatus: (chain: ApprovalChain) => void;
  onCopy: (chain: ApprovalChain) => void;
  onBatchDelete: (ids: React.Key[]) => void;
}

export function ApprovalChainTable({
  dataSource,
  loading = false,
  pagination,
  selectedRowKeys,
  onPaginationChange,
  onRowSelectionChange,
  onEdit,
  onView,
  onDelete,
  onToggleStatus,
  onCopy,
  onBatchDelete,
}: ApprovalChainTableProps) {
  // 表格列配置
  const columns: TableColumn<ApprovalChain>[] = useMemo(
    () => [
      {
        key: 'name',
        title: '审批链名称',
        dataIndex: 'name',
        width: 200,
        render: (value: unknown, record: ApprovalChain) => (
          <div>
            <div className="font-medium text-gray-900">{value as string}</div>
            {record.description && (
              <div className="text-sm text-gray-500 mt-1">{record.description}</div>
            )}
          </div>
        ),
      },
      {
        key: 'status',
        title: '状态',
        dataIndex: 'isActive',
        width: 100,
        render: (value: unknown) => (
          <Tag color={(value as boolean) ? 'green' : 'red'}>
            {(value as boolean) ? '活跃' : '非活跃'}
          </Tag>
        ),
      },
      {
        key: 'steps',
        title: '步骤数',
        dataIndex: 'steps',
        width: 100,
        render: (value: unknown) => (
          <span className="font-medium">{(value as any[])?.length || 0}</span>
        ),
      },
      {
        key: 'createdAt',
        title: '创建时间',
        dataIndex: 'createdAt',
        width: 150,
        render: (value: unknown) => new Date(value as string).toLocaleString(),
      },
      {
        key: 'updatedAt',
        title: '更新时间',
        dataIndex: 'updatedAt',
        width: 150,
        render: (value: unknown) => new Date(value as string).toLocaleString(),
      },
      {
        key: 'actions',
        title: '操作',
        width: 120,
        render: (_: unknown, record: ApprovalChain) => {
          const menuProps = {
            items: [
              {
                key: 'view',
                icon: <Eye className="w-4 h-4" />,
                label: '查看详情',
                onClick: () => onView(record),
              },
              {
                key: 'edit',
                icon: <Edit className="w-4 h-4" />,
                label: '编辑',
                onClick: () => onEdit(record),
              },
              {
                key: 'copy',
                icon: <Copy className="w-4 h-4" />,
                label: '复制',
                onClick: () => onCopy(record),
              },
              { type: 'divider' as const },
              {
                key: 'toggle',
                icon: record.isActive ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />,
                label: record.isActive ? '停用' : '启用',
                onClick: () => onToggleStatus(record),
              },
              { type: 'divider' as const },
              {
                key: 'delete',
                icon: <Trash2 className="w-4 h-4" />,
                label: '删除',
                danger: true,
                onClick: () => onDelete(record),
              },
            ],
          };

          return (
            <Space>
              <Tooltip title="查看详情">
                <Button
                  type="text"
                  icon={<Eye className="w-4 h-4" />}
                  onClick={() => onView(record)}
                  size="small"
                />
              </Tooltip>
              <Tooltip title="编辑">
                <Button
                  type="text"
                  icon={<Edit className="w-4 h-4" />}
                  onClick={() => onEdit(record)}
                  size="small"
                />
              </Tooltip>
              <Dropdown menu={menuProps} trigger={['click']}>
                <Button type="text" icon={<MoreHorizontal className="w-4 h-4" />} size="small" />
              </Dropdown>
            </Space>
          );
        },
      },
    ],
    [onView, onEdit, onCopy, onToggleStatus, onDelete]
  );

  // 行选择配置
  const rowSelection = useMemo(
    () => ({
      selectedRowKeys,
      onChange: onRowSelectionChange,
      getCheckboxProps: (record: ApprovalChain) => ({
        disabled: false,
      }),
    }),
    [selectedRowKeys, onRowSelectionChange]
  );

  // 批量操作按钮
  const batchActions: ActionButton[] = useMemo(
    () => [
      {
        key: 'batch-delete',
        label: '批量删除',
        icon: <Trash2 className="w-4 h-4" />,
        type: 'default',
        danger: true,
        disabled: selectedRowKeys.length === 0,
        onClick: () => onBatchDelete(selectedRowKeys),
      },
    ],
    [selectedRowKeys, onBatchDelete]
  );

  return (
    <div>
      {/* 批量操作 */}
      {selectedRowKeys.length > 0 && (
        <div className="mb-4 p-3 bg-blue-50 rounded-lg">
          <div className="flex items-center justify-between">
            <span className="text-sm text-blue-600">已选择 {selectedRowKeys.length} 项</span>
            <Space>
              {batchActions.map(action => (
                <Button
                  key={action.key}
                  type={action.type}
                  danger={action.danger}
                  disabled={action.disabled}
                  icon={action.icon}
                  onClick={() => action.onClick(selectedRowKeys)}
                  size="small"
                >
                  {action.label}
                </Button>
              ))}
            </Space>
          </div>
        </div>
      )}

      {/* 表格 */}
      <Table
        columns={columns}
        dataSource={dataSource}
        loading={loading}
        pagination={
          pagination
            ? {
                ...pagination,
                onChange: onPaginationChange,
                onShowSizeChange: onPaginationChange,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              }
            : false
        }
        rowSelection={rowSelection}
        rowKey="id"
        scroll={{ x: 800 }}
        size="middle"
      />
    </div>
  );
}

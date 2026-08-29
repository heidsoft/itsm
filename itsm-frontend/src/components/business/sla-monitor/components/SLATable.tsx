/**
 * SLA 违规列表表格组件（简化版，匹配真实 API 字段）
 */

import React from 'react';
import { Table, Tag, Space, Button } from 'antd';
import { Eye, CheckCircle, XCircle } from 'lucide-react';
import type { SLAViolation } from '../types';

interface SLATableProps {
  violations: SLAViolation[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onRowSelect: (keys: React.Key[]) => void;
  onView: (violation: SLAViolation) => void;
  onResolve: (violation: SLAViolation) => void;
  onAcknowledge: (violation: SLAViolation) => void;
}

const severityColors: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
};

const statusColors: Record<string, string> = {
  open: 'red',
  acknowledged: 'orange',
  resolved: 'green',
};

export const SLATable: React.FC<SLATableProps> = ({
  violations,
  loading,
  selectedRowKeys,
  onRowSelect,
  onView,
  onResolve,
  onAcknowledge,
}) => {
  // @ts-ignore - columns type complex, simplified for refactoring
  const columns: any = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '工单编号',
      dataIndex:'ticketNumber',
      key:'ticketNumber',
      width: 140,
      render: (number: string | undefined, record: SLAViolation) => (
        <span>{number || `#${record.ticketId}`}</span>
      ),
    },
    {
      title: '工单标题',
      dataIndex:'ticketTitle',
      key:'ticketTitle',
      width: 200,
      ellipsis: true,
      render: (title: string | undefined, record: SLAViolation) => (
        <span>{title || `Ticket #${record.ticketId}`}</span>
      ),
    },
    {
      title: '违规类型',
      dataIndex:'violationType',
      key:'violationType',
      width: 120,
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severityColors[severity] || 'default'}>{severity}</Tag>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, record: SLAViolation) => {
        const status = record.isResolved ? 'resolved' : 'open';
        return (
          <Tag color={statusColors[status] || 'default'}>
            {status === 'resolved' ? '已解决' : '待处理'}
          </Tag>
        );
      },
    },
    {
      title: '违规时间',
      dataIndex: 'violationTime',
      key: 'violationTime',
      width: 180,
      render: (time: string) => (time ? new Date(time).toLocaleString() : '-'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      fixed: 'right',
      render: (_: any, record: SLAViolation) => (
        <Space size="small">
          <Button
            size="small"
            icon={<Eye />}
            onClick={() => onView(record)}
          />
          {!record.isResolved && (
            <>
              <Button
                size="small"
                type="primary"
                icon={<CheckCircle />}
                onClick={() => onAcknowledge(record)}
              >
                确认
              </Button>
              <Button
                size="small"
                danger
                icon={<XCircle />}
                onClick={() => onResolve(record)}
              >
                解决
              </Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Table
      rowKey="id"
      columns={columns}
      dataSource={violations}
      loading={loading}
      rowSelection={{
        selectedRowKeys,
        onChange: onRowSelect,
      }}
      scroll={{ x: 1200 }}
      size="small"
    />
  );
};

SLATable.displayName = 'SLATable';

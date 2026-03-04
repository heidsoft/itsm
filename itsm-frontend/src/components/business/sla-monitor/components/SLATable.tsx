/**
 * SLA 违规列表表格组件（简化版，匹配真实 API 字段）
 */

import React from 'react';
import { Table, Tag, Space, Button } from 'antd';
import { EyeOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
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
      title: '工单ID',
      dataIndex: 'ticket_id',
      key: 'ticket_id',
      width: 100,
    },
    {
      title: '违规类型',
      dataIndex: 'violation_type',
      key: 'violation_type',
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
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>{status}</Tag>
      ),
    },
    {
      title: '延迟分钟数',
      dataIndex: 'delay_minutes',
      key: 'delay_minutes',
      width: 120,
      render: (minutes?: number) => <span style={{ color: '#ff4d4f' }}>{minutes || 0} 分钟</span>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => new Date(time).toLocaleString(),
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
            icon={<EyeOutlined />}
            onClick={() => onView(record)}
          />
          {record.status === 'open' && (
            <>
              <Button
                size="small"
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={() => onAcknowledge(record)}
              >
                确认
              </Button>
              <Button
                size="small"
                danger
                icon={<CloseCircleOutlined />}
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

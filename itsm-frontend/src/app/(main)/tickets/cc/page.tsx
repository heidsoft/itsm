'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { App, Button, Card, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Inbox, RefreshCw } from 'lucide-react';
import { TicketApi, type TicketCCRecord } from '@/lib/api/ticket-api';
import type { WorkflowUserInfo } from '@/types/ticket-workflow-state';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

/**
 * 抄送人/添加人列的显示值。
 *
 * 契约字段是 fullName（后端 WorkflowUserInfo 从不发 name）。抄送记录可以比用户活得久，
 * 后端此时回填全零的 WorkflowUserInfo，所以按 id 判定而不是按文案猜测。
 */
const ccActorLabel = (info: WorkflowUserInfo): string =>
  info.id > 0 ? info.fullName : 'User deleted';

const priorityColor: Record<string, string> = {
  critical: 'red',
  urgent: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'green',
};

const statusColor: Record<string, string> = {
  new: 'default',
  open: 'processing',
  in_progress: 'processing',
  inProgress: 'processing',
  pending: 'warning',
  resolved: 'success',
  closed: 'default',
  rejected: 'error',
  cancelled: 'error',
};

export default function MyTicketCCPage() {
  const { message } = App.useApp();
  const [records, setRecords] = useState<TicketCCRecord[]>([]);
  const [loading, setLoading] = useState(false);

  const loadRecords = async () => {
    try {
      setLoading(true);
      const resp = await TicketApi.getMyCCRecords();
      setRecords(resp.items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to load CC records');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRecords();
  }, []);

  const columns: ColumnsType<TicketCCRecord> = [
    {
      title: 'Ticket',
      dataIndex: 'title',
      key: 'title',
      render: (_, record) => (
        <Space orientation="vertical" size={0}>
          <Link href={`/tickets/${record.ticketId}`}>{record.title || record.ticketNumber}</Link>
          <Text type="secondary" className="text-xs">
            {record.ticketNumber}
          </Text>
        </Space>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: value => <Tag color={statusColor[value] || 'default'}>{value}</Tag>,
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 110,
      render: value => <Tag color={priorityColor[value] || 'default'}>{value}</Tag>,
    },
    {
      title: 'Cc person',
      dataIndex: ['user', 'fullName'],
      key: 'user',
      width: 140,
      render: (_, record) => ccActorLabel(record.user),
    },
    {
      title: 'Add person',
      dataIndex: ['addedBy', 'fullName'],
      key: 'addedBy',
      width: 140,
      render: (_, record) => ccActorLabel(record.addedBy),
    },
    {
      title: 'Cc time',
      dataIndex: 'addedAt',
      key: 'addedAt',
      width: 180,
      render: value => formatDateTime(value),
    },
  ];

  return (
    <div className="p-6">
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Space>
            <Inbox className="w-6 h-6 text-blue-600" />
            <div>
              <Title level={3} className="!mb-0">

                My CC
              </Title>
              <Text type="secondary">Centrally view all tickets and CC history sent to me</Text>
            </div>
          </Space>
          <Button icon={<RefreshCw size={16} />} onClick={loadRecords} loading={loading}>

            Refresh
          </Button>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={records}
          loading={loading}
          pagination={{ pageSize: 20, showSizeChanger: true }}
        />
      </Card>
    </div>
  );
}

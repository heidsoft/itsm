'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { App, Button, Card, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Inbox, RefreshCw } from 'lucide-react';
import { TicketApi } from '@/lib/api/ticket-api';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

interface TicketCCRecord {
  id: number;
  ticketId: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  user: { id: number; name: string; username: string; email: string };
  addedBy: { id: number; name: string; username: string; email: string };
  addedAt: string;
  isActive: boolean;
}

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
      setRecords(resp.records || []);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载抄送记录失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRecords();
  }, []);

  const columns: ColumnsType<TicketCCRecord> = [
    {
      title: '工单',
      dataIndex: 'title',
      key: 'title',
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <Link href={`/tickets/${record.ticketId}`}>{record.title || record.ticketNumber}</Link>
          <Text type="secondary" className="text-xs">
            {record.ticketNumber}
          </Text>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: value => <Tag color={statusColor[value] || 'default'}>{value}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 110,
      render: value => <Tag color={priorityColor[value] || 'default'}>{value}</Tag>,
    },
    {
      title: '抄送人',
      dataIndex: ['user', 'name'],
      key: 'user',
      width: 140,
      render: (_, record) => record.user?.name || record.user?.username || `User#${record.user?.id}`,
    },
    {
      title: '添加人',
      dataIndex: ['addedBy', 'name'],
      key: 'addedBy',
      width: 140,
      render: (_, record) =>
        record.addedBy?.name || record.addedBy?.username || `User#${record.addedBy?.id}`,
    },
    {
      title: '抄送时间',
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
                我的抄送
              </Title>
              <Text type="secondary">集中查看所有抄送给我的工单和抄送历史</Text>
            </div>
          </Space>
          <Button icon={<RefreshCw size={16} />} onClick={loadRecords} loading={loading}>
            刷新
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

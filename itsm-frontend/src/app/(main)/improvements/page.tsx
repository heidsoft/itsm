'use client';

import React, { useState } from 'react';
import { Table, Button, Tag, Space, Card, App, Empty } from 'antd';
import { Plus, RotateCcw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import type { ColumnsType } from 'antd/es/table';
import { useI18n } from '@/lib/i18n';
import { displayUserName, formatDateTime } from '@/lib/formatters';
import type { Ticket, TicketListResponse } from '@/lib/api/api-config';

type ImprovementStatus = 'In Progress' | 'Pending Review' | 'Completed';

interface Improvement {
  id: string;
  ticketId: number;
  title: string;
  status: ImprovementStatus;
  owner: string;
  target: string;
  createdAt: string;
}

const statusColors: Record<ImprovementStatus, string> = {
  'In Progress': 'blue',
  'Pending Review': 'gold',
  Completed: 'green',
};

const ImprovementListPage = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [improvements, setImprovements] = useState<Improvement[]>([]);
  const [filter, setFilter] = useState<string>("All");
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空列表。
  const [listLoading, setListLoading] = useState(true);

  // 加载改进计划
  React.useEffect(() => {
    loadImprovements();
  }, []);

  const loadImprovements = async () => {
    setListLoading(true);
    try {
      const { TicketApi } = await import('@/lib/api/ticket-api');
      const response: TicketListResponse = await TicketApi.getTickets({ type: 'improvement', page: 1, pageSize: 100 });

      const mappedImprovements: Improvement[] = response.items.map((ticket: Ticket) => ({
        id: ticket.ticketNumber || `IMP-${ticket.id}`,
        ticketId: ticket.id,
        title: ticket.title,
        status: mapStatus(ticket.status),
        owner: displayUserName(ticket.assigneeName, ticket.assigneeId, 'Unassigned'),
        target: ticket.description || 'No target description',
        createdAt: formatDateTime(ticket.createdAt) || '-',
      }));

      setImprovements(mappedImprovements);
    } catch (error) {
      console.error(t('common.getFailed') + ':', error);
      message.error(t('common.getFailed'));
    } finally {
      setListLoading(false);
    }
  };

  const mapStatus = (ticketStatus: string): ImprovementStatus => {
    switch (ticketStatus) {
      case 'open':
        return 'Pending Review';
      case 'in_progress':
        return 'In Progress';
      case 'resolved':
      case 'closed':
        return 'Completed';
      default:
        return 'Pending Review';
    }
  };

  const filteredImprovements = improvements.filter(imp => {
    if (filter === 'All') return true;
    return imp.status === filter;
  });

  const columns: ColumnsType<Improvement> = [
    {
      title: 'Plan ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: string, record: Improvement) => (
        <Button type="link" onClick={() => router.push(`/tickets/${record.ticketId}`)}>
          {id}
        </Button>
      ),
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: ImprovementStatus) => <Tag color={statusColors[status]}>{status}</Tag>,
    },
    {
      title: 'Assignee',
      dataIndex: 'owner',
      key: 'owner',
      width: 120,
    },
    {
      title: 'Target',
      dataIndex: 'target',
      key: 'target',
      ellipsis: true,
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 120,
    },
  ];

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <Card>
        <div className="mb-6 flex justify-between items-center">
          <div>
            <h2 className="text-2xl font-bold text-gray-800">Continuous improvement</h2>
            <p className="text-gray-500 mt-1">Identify, plan and implement improvements to IT services and processes</p>
          </div>
          <Space>
            <Button icon={<RotateCcw />} onClick={loadImprovements} loading={listLoading}>
              Refresh
            </Button>
            <Button
              type="primary"
              icon={<Plus />}
              onClick={() => router.push('/tickets/create?type=improvement')}
            >

              Create a new improvement plan
            </Button>
          </Space>
        </div>

        {/* 筛选器 */}
        <div className="mb-4 flex items-center gap-2">
          <span className="text-sm font-semibold">Filter:</span>
          <Space>
            {['All', 'In Progress', 'Pending Review', 'Completed'].map(f => (
              <Button
                key={f}
                type={filter === f ? 'primary' : 'default'}
                onClick={() => setFilter(f)}
                size="small"
              >
                {f}
              </Button>
            ))}
          </Space>
        </div>

        {/* 改进计划列表 */}
        {filteredImprovements.length === 0 && !listLoading ? (
          <Empty description="No improvement plan yet">
            <Button type="primary" onClick={() => router.push('/tickets/create?type=improvement')}>

              Create an improvement plan
            </Button>
          </Empty>
        ) : (
          <Table
            columns={columns}
            dataSource={filteredImprovements}
            rowKey="id"
            loading={listLoading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showTotal: total => `Total ${total} records`,
            }}
          />
        )}
      </Card>
    </div>
  );
};

export default ImprovementListPage;

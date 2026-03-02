'use client';

import React, { useState } from 'react';
import { Table, Button, Tag, Space, Card, App, Empty } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import type { ColumnsType } from 'antd/es/table';
import { useI18n } from '@/lib/i18n';

type ImprovementStatus = '进行中' | '待评估' | '已完成';

interface Improvement {
  id: string;
  title: string;
  status: ImprovementStatus;
  owner: string;
  target: string;
  createdAt: string;
}

const statusColors: Record<ImprovementStatus, string> = {
  '进行中': 'blue',
  '待评估': 'gold',
  '已完成': 'green',
};

const ImprovementListPage = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [improvements, setImprovements] = useState<Improvement[]>([]);
  const [filter, setFilter] = useState<string>('全部');
  const [loading, setLoading] = useState(false);

  // 加载改进计划
  React.useEffect(() => {
    loadImprovements();
  }, []);

  const loadImprovements = async () => {
    setLoading(true);
    try {
      const { TicketApi } = await import('@/lib/api/ticket-api');
      const response = await TicketApi.getTickets({ type: 'improvement', page: 1, size: 100 });

      const mappedImprovements: Improvement[] = response.tickets.map((ticket: any) => ({
        id: ticket.ticketNumber || `IMP-${ticket.id}`,
        title: ticket.title,
        status: mapStatus(ticket.status),
        owner: ticket.assignee?.name || '未分配',
        target: ticket.description || '无目标描述',
        createdAt: ticket.createdAt ? new Date(ticket.createdAt).toLocaleDateString() : '-'
      }));

      setImprovements(mappedImprovements);
    } catch (error) {
      console.error(t('common.getFailed') + ':', error);
      message.error(t('common.getFailed'));
    } finally {
      setLoading(false);
    }
  };

  const mapStatus = (ticketStatus: string): ImprovementStatus => {
    switch (ticketStatus) {
      case 'open': return '待评估';
      case 'in_progress': return '进行中';
      case 'resolved':
      case 'closed': return '已完成';
      default: return '待评估';
    }
  };

  const filteredImprovements = improvements.filter(imp => {
    if (filter === '全部') return true;
    return imp.status === filter;
  });

  const columns: ColumnsType<Improvement> = [
    {
      title: '计划ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: string) => (
        <Button type='link' onClick={() => router.push(`/improvements/${id}`)}>
          {id}
        </Button>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: ImprovementStatus) => (
        <Tag color={statusColors[status]}>{status}</Tag>
      ),
    },
    {
      title: '负责人',
      dataIndex: 'owner',
      key: 'owner',
      width: 120,
    },
    {
      title: '目标',
      dataIndex: 'target',
      key: 'target',
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 120,
    },
  ];

  return (
    <div className='p-6 bg-gray-50 min-h-full'>
      <Card>
        <div className='mb-6 flex justify-between items-center'>
          <div>
            <h2 className='text-2xl font-bold text-gray-800'>持续改进</h2>
            <p className='text-gray-500 mt-1'>识别、规划和实施IT服务和流程的改进</p>
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={loadImprovements} loading={loading}>
              刷新
            </Button>
            <Button type='primary' icon={<PlusOutlined />} onClick={() => router.push('/improvements/new')}>
              新建改进计划
            </Button>
          </Space>
        </div>

        {/* 筛选器 */}
        <div className='mb-4 flex items-center gap-2'>
          <span className='text-sm font-semibold'>筛选:</span>
          <Space>
            {['全部', '进行中', '待评估', '已完成'].map(f => (
              <Button
                key={f}
                type={filter === f ? 'primary' : 'default'}
                onClick={() => setFilter(f)}
                size='small'
              >
                {f}
              </Button>
            ))}
          </Space>
        </div>

        {/* 改进计划列表 */}
        {filteredImprovements.length === 0 && !loading ? (
          <Empty description='暂无改进计划'>
            <Button type='primary' onClick={() => router.push('/improvements/new')}>
              创建第一个改进计划
            </Button>
          </Empty>
        ) : (
          <Table
            columns={columns}
            dataSource={filteredImprovements}
            rowKey='id'
            loading={loading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showTotal: (total) => `共 ${total} 条记录`,
            }}
          />
        )}
      </Card>
    </div>
  );
};

export default ImprovementListPage;

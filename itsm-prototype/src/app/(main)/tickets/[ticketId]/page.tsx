'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';
import type { Ticket } from '@/app/lib/api-config';
import { ArrowLeft, AlertCircle, XCircle } from 'lucide-react';
import Link from 'next/link';
import { Button, Card, Typography, App, Badge, Tag, Descriptions, Space } from 'antd';
import { useAuthStore } from '@/lib/store/auth-store';

const { Title, Text } = Typography;

const TicketDetailPage: React.FC = () => {
  const params = useParams();
  const { message: antMessage } = App.useApp();
  const { user } = useAuthStore();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const ticketId = parseInt(params.ticketId as string);

  // Get ticket details
  const fetchTicket = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await TicketApi.getTicket(ticketId);
      setTicket(data as Ticket);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Network error');
    } finally {
      setLoading(false);
    }
  }, [ticketId]);

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId, fetchTicket]);

  // Handle approval (轻量版：仅改状态)
  const handleApprove = async () => {
    try {
      await TicketApi.updateTicketStatus(ticketId, 'approved');
      antMessage.success('批准成功');
      fetchTicket();
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle rejection (轻量版：仅改状态)
  const handleReject = async () => {
    try {
      await TicketApi.updateTicketStatus(ticketId, 'rejected');
      antMessage.success('已拒绝');
      fetchTicket();
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle assignment (轻量版：不做交互，只保留入口)
  const handleAssign = async () => {
    try {
      antMessage.info('分配能力待恢复（V0 后续）');
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle update (轻量版：不做复杂表单)
  const handleUpdate = async () => {
    try {
      antMessage.info('编辑能力待恢复（V0 后续）');
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  if (loading) {
    return (
      <div className='p-6'>
        <Card>
          <div className='text-center py-8'>
            <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto'></div>
            <Text className='mt-4 block'>加载中...</Text>
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className='p-6'>
        <Card>
          <div className='text-center py-8'>
            <AlertCircle className='w-12 h-12 text-red-500 mx-auto mb-4' />
            <Title level={4} className='text-red-600 mb-2'>
              加载失败
            </Title>
            <Text type='secondary'>{error}</Text>
            <div className='mt-4'>
              <Button type='primary' onClick={fetchTicket}>
                重试
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className='p-6'>
        <Card>
          <div className='text-center py-8'>
            <XCircle className='w-12 h-12 text-gray-400 mx-auto mb-4' />
            <Title level={4} className='text-gray-600 mb-2'>
              未找到工单
            </Title>
            <Text type='secondary'>未找到指定的工单</Text>
            <div className='mt-4'>
              <Link href='/tickets'>
                <Button type='primary'>返回工单列表</Button>
              </Link>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className='p-6'>
      {/* Page header */}
      <div className='mb-6'>
        <div className='flex items-center justify-between mb-4'>
          <div className='flex items-center space-x-4'>
            <Link href='/tickets'>
              <Button icon={<ArrowLeft />} type='text'>
                返回列表
              </Button>
            </Link>
            <div>
              <Title level={2} className='mb-1'>
                工单详情 #{ticket.id}
              </Title>
              <Text type='secondary'>{ticket.title}</Text>
            </div>
          </div>
          <div className='flex items-center space-x-2'>
            <Badge
              status={
                ticket.status === 'open'
                  ? 'processing'
                  : ticket.status === 'closed'
                  ? 'success'
                  : 'warning'
              }
              text={
                ticket.status === 'open'
                  ? '处理中'
                  : ticket.status === 'closed'
                  ? '已关闭'
                  : '待处理'
              }
            />
            <Tag
              color={
                ticket.priority === 'high'
                  ? 'red'
                  : ticket.priority === 'medium'
                  ? 'orange'
                  : 'green'
              }
            >
              {ticket.priority === 'high'
                ? '高优先级'
                : ticket.priority === 'medium'
                ? '中优先级'
                : '低优先级'}
            </Tag>
          </div>
        </div>
      </div>

      <Card>
        <Space direction='vertical' size={16} style={{ width: '100%' }}>
          <Descriptions column={2} bordered size='middle'>
            <Descriptions.Item label='标题'>{ticket.title}</Descriptions.Item>
            <Descriptions.Item label='编号'>{ticket.ticket_number}</Descriptions.Item>
            <Descriptions.Item label='状态'>{ticket.status}</Descriptions.Item>
            <Descriptions.Item label='优先级'>{ticket.priority}</Descriptions.Item>
            <Descriptions.Item label='创建时间'>{ticket.created_at}</Descriptions.Item>
            <Descriptions.Item label='更新时间'>{ticket.updated_at}</Descriptions.Item>
            <Descriptions.Item label='描述' span={2}>
              {ticket.description}
            </Descriptions.Item>
          </Descriptions>

          <Space>
            <Button type='primary' onClick={handleApprove}>
              批准
            </Button>
            <Button danger onClick={handleReject}>
              拒绝
            </Button>
            <Button onClick={handleAssign}>分配（待恢复）</Button>
            <Button onClick={handleUpdate}>编辑（待恢复）</Button>
          </Space>

          <Text type='secondary'>
            当前为 P0 轻量详情页，避免引入 legacy business 组件导致 TS/构建阻塞；后续会恢复完整详情面板与工作流操作。
          </Text>
        </Space>
      </Card>
    </div>
  );
};

export default TicketDetailPage;

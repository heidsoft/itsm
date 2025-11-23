'use client';

import React, { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';
import { Ticket, ApiResponse } from '@/lib/api-config';
import { TicketDetail } from '@/components/business/TicketDetail';
import { ArrowLeft, AlertCircle } from 'lucide-react';
import Link from 'next/link';
import { Button, Card, Typography, App, Badge, Tag } from 'antd';

const { Title, Text } = Typography;

const TicketDetailPage: React.FC = () => {
  const params = useParams();
  const { message: antMessage } = App.useApp();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const ticketId = parseInt(params.ticketId as string);

  // Get ticket details
  const fetchTicket = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = (await TicketApi.getTicket(ticketId)) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        setTicket(response.data);
      } else {
        setError(response.message || 'Failed to load ticket');
      }
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Network error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId]);

  // Handle approval
  const handleApprove = async () => {
    try {
      const response = (await TicketApi.updateTicket(ticketId, {
        status: 'approved',
      })) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success('批准成功');
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || '批准失败');
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle rejection
  const handleReject = async () => {
    try {
      const response = (await TicketApi.updateTicket(ticketId, {
        status: 'rejected',
      })) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success('已拒绝');
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || '操作失败');
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle assignment
  const handleAssign = async (assignee: string) => {
    try {
      const response = (await TicketApi.updateTicket(ticketId, {
        assignee_id: parseInt(assignee), // This needs to be handled according to actual situation
      })) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success('分配成功');
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || '分配失败');
      }
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle update
  const handleUpdate = async (updates: unknown) => {
    try {
      const response = (await TicketApi.updateTicket(
        ticketId,
        updates as Partial<Ticket>
      )) as unknown as ApiResponse<Ticket>;

      if (response.code === 0) {
        antMessage.success('更新成功');
        fetchTicket(); // Refresh data
      } else {
        antMessage.error(response.message || '更新失败');
      }
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

      {/* Ticket detail component */}
      <Card>
        <TicketDetail
          ticket={ticket}
          onApprove={handleApprove}
          onReject={handleReject}
          onAssign={handleAssign}
          onUpdate={handleUpdate}
        />
      </Card>
    </div>
  );
};

export default TicketDetailPage;

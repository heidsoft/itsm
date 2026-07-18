'use client';

import React, { useState, useEffect } from 'react';
import { Card, Descriptions, Tag, Button, Skeleton, Result, Space, App } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';

const ImprovementDetailPage = () => {
  const params = useParams() as { id?: string };
  const id = params?.id;
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [improvement, setImprovement] = useState<{
    id: string | number;
    title: string;
    description?: string;
    status?: string;
    priority?: string;
    assignee?: { name?: string };
    createdAt?: string;
    updatedAt?: string;
  } | null>(null);

  useEffect(() => {
    if (id) loadDetail();
  }, [id]);

  const loadDetail = async () => {
    if (!id) return;
    // 接受 ticketNumber（IMP-xxx 或数字）或纯数字 id
    const numericId = parseInt(id, 10);
    if (!isNaN(numericId) && numericId > 0 && /^\d+$/.test(id)) {
      setLoading(true);
      try {
        const data = await TicketApi.getTicket(numericId);
        setImprovement(data as unknown as typeof improvement);
      } catch (err) {
        console.error('Load improvement failed:', err);
        message.error('加载改进计划失败');
        setError('未找到该改进计划');
      } finally {
        setLoading(false);
      }
      return;
    }
    setError('无效的改进计划ID');
    setLoading(false);
  };

  if (loading) {
    return (
      <div className="p-6">
        <Card>
          <Skeleton active />
        </Card>
      </div>
    );
  }

  if (!improvement) {
    return (
      <div className="p-6">
        <Card>
          <Result
            status="404"
            title="404"
            subTitle={error || '抱歉，您访问的改进计划不存在'}
            extra={
              <Button type="primary" onClick={() => router.push('/improvements')}>
                返回列表
              </Button>
            }
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
        <Button
          icon={<ArrowLeft />}
          onClick={() => router.push('/improvements')}
          type="text"
        >
          返回列表
        </Button>

        <Card>
          <Space orientation="vertical" size="small" style={{ width: '100%' }}>
            <h2 className="text-2xl font-bold text-gray-800">{improvement.title}</h2>
            <Space>
              <Tag color="blue">{improvement.status || '待评估'}</Tag>
              <Tag>{improvement.priority || '中'}</Tag>
            </Space>
          </Space>

          <Descriptions bordered column={2} style={{ marginTop: 24 }}>
            <Descriptions.Item label="计划ID">{String(improvement.id)}</Descriptions.Item>
            <Descriptions.Item label="标题">{improvement.title}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color="blue">{improvement.status || '待评估'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="优先级">{improvement.priority || '中'}</Descriptions.Item>
            <Descriptions.Item label="负责人">
              {improvement.assignee?.name || '未分配'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{improvement.createdAt || '-'}</Descriptions.Item>
            <Descriptions.Item label="更新时间" span={2}>
              {improvement.updatedAt || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="目标描述" span={2}>
              {improvement.description || '-'}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      </Space>
    </div>
  );
};

export default ImprovementDetailPage;
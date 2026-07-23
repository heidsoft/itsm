'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Result,
  Skeleton,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ArrowLeft, FileText } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { TicketApi } from '@/lib/api/ticket-api';
import type { Ticket } from '@/lib/api/api-config';

const { Paragraph, Text, Title } = Typography;
type TemplateDetail = Awaited<ReturnType<typeof TicketApi.getTemplate>>;

const renderValue = (value: unknown): React.ReactNode => {
  if (value === null || value === undefined || value === '') return '-';
  if (typeof value === 'boolean') return value ? '是' : '否';
  if (Array.isArray(value)) return value.join('、') || '-';
  if (typeof value === 'object') return <Text code>{JSON.stringify(value)}</Text>;
  return String(value);
};

export default function TicketTemplateDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const templateId = Number(params.id);
  const [loading, setLoading] = useState(true);
  const [template, setTemplate] = useState<TemplateDetail | null>(null);
  const [tickets, setTickets] = useState<Ticket[]>([]);

  const loadDetail = useCallback(async () => {
    if (!Number.isInteger(templateId) || templateId <= 0) {
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [templateData, ticketData] = await Promise.all([
        TicketApi.getTemplate(templateId),
        TicketApi.getTickets({ page: 1, pageSize: 50, templateId }),
      ]);
      setTemplate(templateData);
      setTickets(ticketData.tickets || []);
    } catch {
      message.error('加载模板详情失败');
      setTemplate(null);
    } finally {
      setLoading(false);
    }
  }, [templateId]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const formEntries = useMemo(
    () => Object.entries(template?.formFields || {}),
    [template?.formFields]
  );

  const columns: ColumnsType<Ticket> = [
    {
      title: '工单编号',
      dataIndex: 'ticketNumber',
      render: (value: string, record) => (
        <Button type="link" onClick={() => router.push(`/tickets/${record.id}`)}>
          {value || `#${record.id}`}
        </Button>
      ),
    },
    { title: '标题', dataIndex: 'title', ellipsis: true },
    { title: '状态', dataIndex: 'status', render: value => <Tag>{value}</Tag> },
    {
      title: '优先级',
      dataIndex: 'priority',
      render: value => (
        <Tag color={value === 'high' || value === 'critical' ? 'red' : 'blue'}>{value}</Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      render: value => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-'),
    },
  ];

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }
  if (!template) {
    return (
      <Result
        status="404"
        title="模板不存在"
        subTitle="该模板可能已被删除，或您没有访问权限。"
        extra={<Button onClick={() => router.push('/tickets/templates')}>返回模板列表</Button>}
      />
    );
  }

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Card>
        <Button icon={<ArrowLeft size={16} />} onClick={() => router.push('/tickets/templates')}>
          返回模板列表
        </Button>
        <div className="mt-4 flex items-start justify-between">
          <div>
            <Title level={3} style={{ marginBottom: 4 }}>{template.name}</Title>
            <Paragraph type="secondary" style={{ marginBottom: 0 }}>
              {template.description || '暂无描述'}
            </Paragraph>
          </div>
          <Tag color={template.isActive ? 'green' : 'default'}>
            {template.isActive ? '已启用' : '已停用'}
          </Tag>
        </div>
      </Card>

      <Card title="基本信息">
        <Descriptions bordered column={{ xs: 1, sm: 2 }}>
          <Descriptions.Item label="模板 ID">{template.id}</Descriptions.Item>
          <Descriptions.Item label="分类">{template.category || '-'}</Descriptions.Item>
          <Descriptions.Item label="默认优先级">{template.priority || '-'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {dayjs(template.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {dayjs(template.updatedAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="关联表单配置" extra={<FileText size={18} />}>
        {formEntries.length > 0 ? (
          <Descriptions bordered column={1}>
            {formEntries.map(([key, value]) => (
              <Descriptions.Item key={key} label={key}>{renderValue(value)}</Descriptions.Item>
            ))}
          </Descriptions>
        ) : template.fields?.length ? (
          <Table
            rowKey={(_, index) => String(index)}
            pagination={false}
            dataSource={template.fields}
            columns={[
              { title: '字段', dataIndex: 'name' },
              { title: '标签', dataIndex: 'label' },
              { title: '类型', dataIndex: 'type' },
              { title: '必填', dataIndex: 'required', render: value => (value ? '是' : '否') },
            ]}
          />
        ) : (
          <Empty description="暂无关联表单配置" />
        )}
      </Card>

      <Card title={`关联工单（${tickets.length}）`}>
        <Table rowKey="id" columns={columns} dataSource={tickets} pagination={{ pageSize: 10 }} />
      </Card>
    </Space>
  );
}

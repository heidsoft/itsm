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
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
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
      setTickets(ticketData.items ?? []);
    } catch {
      message.error('Failed to load template details');
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
      title: 'Ticket Number',
      dataIndex: 'ticketNumber',
      render: (value: string, record) => (
        <Button
          type='link'
          onClick={() => {
            try {
              router.push(`/tickets/${record.id}`);
            } catch (navErr) {
              console.error('[nav]', navErr);
              message.error('Navigation failed, please try again later');
            }
          }}
        >
          {value || `#${record.id}`}
        </Button>
      ),
    },
    { title: 'Title', dataIndex: 'title', ellipsis: true },
    { title: 'Status', dataIndex: 'status', render: value => <Tag>{value}</Tag> },
    {
      title: 'Priority',
      dataIndex: 'priority',
      render: value => (
        <Tag color={value === 'high' || value === 'critical' ? 'red' : 'blue'}>{value}</Tag>
      ),
    },
    {
      title: 'Created',
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
        status='404'
        title="Template does not exist"
        subTitle="The template may have been deleted or you do not have access rights."
        extra={
          <Button
            onClick={() => {
              try {
                router.push('/tickets/templates');
              } catch (navErr) {
                console.error('[nav]', navErr);
                message.error('Navigation failed, please try again later');
              }
            }}
          >

            Return template list
          </Button>
        }
      />
    );
  }

  return (
    <Space orientation='vertical' size='large' style={{ width: '100%' }}>
      <Card>
        <Button
          icon={<ArrowLeft size={16} />}
          onClick={() => {
            try {
              router.push('/tickets/templates');
            } catch (navErr) {
              console.error('[nav]', navErr);
              message.error('Navigation failed, please try again later');
            }
          }}
        >

          Return template list
        </Button>
        <div className='mt-4 flex items-start justify-between'>
          <div>
            <Title level={3} style={{ marginBottom: 4 }}>
              {template.name}
            </Title>
            <Paragraph type='secondary' style={{ marginBottom: 0 }}>
              {template.description || 'No description'}
            </Paragraph>
          </div>
          <Tag color={template.isActive ? 'green' : 'default'}>
            {template.isActive ? 'Enabled' : 'Disabled'}
          </Tag>
        </div>
      </Card>

      <Card title="Basic Info">
        <Descriptions bordered column={{ xs: 1, sm: 2 }}>
          <Descriptions.Item label="Template ID">{template.id}</Descriptions.Item>
          <Descriptions.Item label="Category">{template.category || '-'}</Descriptions.Item>
          <Descriptions.Item label="Default Priority">{template.priority || '-'}</Descriptions.Item>
          <Descriptions.Item label="Created">
            {dayjs(template.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
          <Descriptions.Item label="Updated At">
            {dayjs(template.updatedAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="Association form configuration" extra={<FileText size={18} />}>
        {formEntries.length > 0 ? (
          <Descriptions bordered column={1}>
            {formEntries.map(([key, value]) => (
              <Descriptions.Item key={key} label={key}>
                {renderValue(value)}
              </Descriptions.Item>
            ))}
          </Descriptions>
        ) : template.fields?.length ? (
          <Table
            rowKey={(_, index) => String(index)}
            pagination={false}
            dataSource={template.fields}
            columns={[
              { title: 'Field', dataIndex: 'name' },
              { title: 'Tags', dataIndex: 'label' },
              { title: 'Type', dataIndex: 'type' },
              { title: 'Required', dataIndex: 'required', render: value => (value ? 'Yes' : 'No') },
            ]}
          />
        ) : (
          <Empty description="No associated form configuration yet" />
        )}
      </Card>

      <Card title={`Associated work order (${tickets.length}）`}>
        <Table rowKey='id' columns={columns} dataSource={tickets} pagination={{ pageSize: 10 }} />
      </Card>
    </Space>
  );
}

'use client';

'use client';

/**
 * 工单类型管理页面
 * Bug 9 修复：原本是 dev 占位文案，现在调 /api/v1/ticket-categories 渲染真实工单类型
 */

import React, { useEffect, useState } from 'react';
import {
  Card,
  Table,
  Tag,
  Typography,
  Spin,
  Alert,
  Space,
  Empty,
} from 'antd';
import { httpClient } from '@/lib/api/http-client';

const { Title, Text } = Typography;

interface TicketCategory {
  id: number;
  name: string;
  code?: string;
  description?: string;
  parent_id?: number | null;
  sla_hours?: number;
  priority?: string;
  required_fields?: string[];
  sort_order?: number;
}

export default function TicketTypesPage() {
  const [data, setData] = useState<TicketCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    httpClient
      .get<any>('/api/v1/ticket-categories', { page: 1, page_size: 200 })
      .then((res: any) => {
        const items = res?.data?.items || res?.items || res?.data || [];
        setData(items);
      })
      .catch(err => setError(err?.message || '加载工单类型失败'))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Spin size="large" tip="加载工单类型..." />
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto p-6">
      <Card>
        <Space orientation="vertical" size={8} className="mb-4">
          <Title level={2} style={{ marginBottom: 0 }}>
            工单类型
          </Title>
          <Text type="secondary">
            系统内置的工单分类及其 SLA / 必填字段 / 审批模板
          </Text>
        </Space>

        {error && (
          <Alert
            type="error"
            showIcon
            className="mb-4"
            message="加载失败"
            description={error}
          />
        )}

        {data.length === 0 && !error ? (
          <Empty description="暂无工单类型数据" />
        ) : (
          <Table<TicketCategory>
            rowKey="id"
            dataSource={data}
            pagination={{ pageSize: 20 }}
            columns={[
              {
                title: 'ID',
                dataIndex: 'id',
                width: 60,
              },
              {
                title: '名称',
                dataIndex: 'name',
                width: 160,
                render: (text: string) => <Text strong>{text}</Text>,
              },
              {
                title: '编码',
                dataIndex: 'code',
                width: 140,
                render: (text?: string) =>
                  text ? <Tag color="blue">{text}</Tag> : '-',
              },
              {
                title: '默认优先级',
                dataIndex: 'priority',
                width: 100,
                render: (text?: string) => {
                  if (!text) return '-';
                  const colorMap: Record<string, string> = {
                    critical: 'red',
                    high: 'orange',
                    medium: 'blue',
                    low: 'green',
                  };
                  return <Tag color={colorMap[text] || 'default'}>{text}</Tag>;
                },
              },
              {
                title: 'SLA（小时）',
                dataIndex: 'sla_hours',
                width: 100,
                render: (v?: number) => (v != null ? `${v} h` : '-'),
              },
              {
                title: '描述',
                dataIndex: 'description',
                ellipsis: true,
              },
              {
                title: '必填字段',
                dataIndex: 'required_fields',
                width: 200,
                render: (fields?: string[]) =>
                  fields && fields.length > 0 ? (
                    <Space size={4} wrap>
                      {fields.map(f => (
                        <Tag key={f}>{f}</Tag>
                      ))}
                    </Space>
                  ) : (
                    '-'
                  ),
              },
            ]}
          />
        )}
      </Card>
    </div>
  );
}

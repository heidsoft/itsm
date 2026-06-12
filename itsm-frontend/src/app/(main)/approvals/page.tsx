'use client';

/**
 * 审批中心首页
 * 汇总展示所有待我审批的工单、变更、服务请求、事件
 */

import React, { useEffect, useState } from 'react';
import {
  Card,
  Tabs,
  Table,
  Tag,
  Button,
  Space,
  Empty,
  Spin,
  Statistic,
  Row,
  Col,
  message,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  FileTextOutlined,
  ToolOutlined,
  AlertOutlined,
  CustomerServiceOutlined,
} from '@ant-design/icons';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { httpClient } from '@/lib/api/http-client';
import { useAuthStore } from '@/lib/store/auth-store';

const { Title, Text } = Typography;

interface PendingItem {
  id: number | string;
  type: 'ticket' | 'change' | 'service_request' | 'incident';
  title: string;
  status: string;
  priority?: string;
  createdAt: string;
  url: string;
  requester?: string;
}

export default function ApprovalsCenterPage() {
  const router = useRouter();
  const { user } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [tickets, setTickets] = useState<PendingItem[]>([]);
  const [changes, setChanges] = useState<PendingItem[]>([]);
  const [serviceRequests, setServiceRequests] = useState<PendingItem[]>([]);
  const [activeTab, setActiveTab] = useState('tickets');

  const load = async () => {
    setLoading(true);
    try {
      // 加载工单待审批
      const ticketsResp = await httpClient
        .get<{ items: any[]; total: number }>('/api/v1/tickets?status=pending&page=1&page_size=20')
        .catch(() => ({ items: [], total: 0 }));
      setTickets(
        (ticketsResp.items || []).map((t: any) => ({
          id: t.id,
          type: 'ticket' as const,
          title: t.title || `工单 #${t.id}`,
          status: t.status,
          priority: t.priority,
          createdAt: t.createdAt || t.created_at,
          url: `/tickets/${t.id}`,
          requester: t.requesterName || t.requester_name,
        }))
      );

      // 加载变更待审批
      const changesResp = await httpClient
        .get<{ items: any[]; total: number }>('/api/v1/changes?status=pending&page=1&page_size=20')
        .catch(() => ({ items: [], total: 0 }));
      setChanges(
        (changesResp.items || []).map((c: any) => ({
          id: c.id,
          type: 'change' as const,
          title: c.title || `变更 #${c.id}`,
          status: c.status,
          priority: c.priority,
          createdAt: c.scheduledStart || c.createdAt || c.created_at,
          url: `/changes/${c.id}`,
          requester: c.requesterName || c.requester_name,
        }))
      );

      // 加载服务请求待审批
      const srResp = await httpClient
        .get<{ items: any[]; total: number }>('/api/v1/service-requests?status=pending&page=1&page_size=20')
        .catch(() => ({ items: [], total: 0 }));
      setServiceRequests(
        (srResp.items || []).map((s: any) => ({
          id: s.id,
          type: 'service_request' as const,
          title: s.title || `服务请求 #${s.id}`,
          status: s.status,
          priority: s.priority,
          createdAt: s.createdAt || s.created_at,
          url: `/service-requests/${s.id}`,
          requester: s.requesterName || s.requester_name,
        }))
      );
    } catch (e) {
      message.error('加载待审批列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: PendingItem) => (
        <Link href={record.url}>{text}</Link>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (p: string) => {
        if (!p) return '-';
        const colorMap: Record<string, string> = {
          critical: 'red',
          high: 'orange',
          medium: 'blue',
          low: 'green',
        };
        return <Tag color={colorMap[p] || 'default'}>{p}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <Tag>{s}</Tag>,
    },
    {
      title: '申请人',
      dataIndex: 'requester',
      key: 'requester',
      width: 120,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (t: string) => t ? new Date(t).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: PendingItem) => (
        <Button type="link" size="small" onClick={() => router.push(record.url)}>
          查看
        </Button>
      ),
    },
  ];

  const totalPending = tickets.length + changes.length + serviceRequests.length;

  return (
    <div className="p-6">
      <div className="mb-4 flex justify-between items-center">
        <Title level={3} style={{ margin: 0 }}>
          <CheckCircleOutlined style={{ marginRight: 8 }} />
          审批中心
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Link href="/approvals/pending">
            <Button type="primary" icon={<ClockCircleOutlined />}>
              待我审批
            </Button>
          </Link>
        </Space>
      </div>

      <Row gutter={16} className="mb-4">
        <Col span={6}>
          <Card>
            <Statistic
              title="待审批总数"
              value={totalPending}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="工单待审"
              value={tickets.length}
              prefix={<FileTextOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="变更待审"
              value={changes.length}
              prefix={<ToolOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="服务请求待审"
              value={serviceRequests.length}
              prefix={<CustomerServiceOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'tickets',
              label: (
                <span>
                  <FileTextOutlined /> 工单 ({tickets.length})
                </span>
              ),
              children: tickets.length > 0 ? (
                <Table
                  rowKey="id"
                  dataSource={tickets}
                  columns={columns}
                  loading={loading}
                  pagination={{ pageSize: 10 }}
                />
              ) : (
                <Empty description="暂无待审批工单" />
              ),
            },
            {
              key: 'changes',
              label: (
                <span>
                  <ToolOutlined /> 变更 ({changes.length})
                </span>
              ),
              children: changes.length > 0 ? (
                <Table
                  rowKey="id"
                  dataSource={changes}
                  columns={columns}
                  loading={loading}
                  pagination={{ pageSize: 10 }}
                />
              ) : (
                <Empty description="暂无待审批变更" />
              ),
            },
            {
              key: 'service-requests',
              label: (
                <span>
                  <CustomerServiceOutlined /> 服务请求 ({serviceRequests.length})
                </span>
              ),
              children: serviceRequests.length > 0 ? (
                <Table
                  rowKey="id"
                  dataSource={serviceRequests}
                  columns={columns}
                  loading={loading}
                  pagination={{ pageSize: 10 }}
                />
              ) : (
                <Empty description="暂无待审批服务请求" />
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}

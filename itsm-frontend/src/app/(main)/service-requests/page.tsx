'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Tabs, Table, Tag, Button } from 'antd';
import ServiceRequestList from '@/components/service-request/ServiceRequestList';
import { FileText, Clock, CheckCircle, XCircle, Plus, Inbox } from 'lucide-react';
import { useRouter } from 'next/navigation';

const { Title, Text } = Typography;

// Mock pending approvals data
const pendingApprovals = [
  { id: 1, request_no: 'SR-001', title: '申请云服务器', applicant: '张三', date: '2024-01-15', priority: '高' },
  { id: 2, request_no: 'SR-002', title: '开通VPN权限', applicant: '李四', date: '2024-01-14', priority: '中' },
  { id: 3, request_no: 'SR-003', title: '数据库扩容', applicant: '王五', date: '2024-01-13', priority: '高' },
];

const priorityColors: Record<string, string> = {
  '高': 'red',
  '中': 'orange',
  '低': 'default',
};

export default function ServiceRequestsPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('requests');
  const [stats, setStats] = useState({
    totalRequests: 0,
    pending: 0,
    processing: 0,
    completed: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Mock data - in real app would call API
      setStats({
        totalRequests: 156,
        pending: 12,
        processing: 28,
        completed: 116,
      });
    } catch (error) {
      console.error('Failed to fetch service request stats:', error);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const approvalColumns = [
    {
      title: '请求号',
      dataIndex: 'request_no',
      key: 'request_no',
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: '申请人',
      dataIndex: 'applicant',
      key: 'applicant',
    },
    {
      title: '申请时间',
      dataIndex: 'date',
      key: 'date',
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (priority: string) => <Tag color={priorityColors[priority]}>{priority}</Tag>,
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Button size="small" type="link">审批</Button>
      ),
    },
  ];

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            服务请求
          </Title>
          <Text type="secondary">
            查看和管理服务请求及审批流程
          </Text>
        </div>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="请求总数"
              value={stats.totalRequests}
              prefix={<FileText className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="待审批"
              value={stats.pending}
              prefix={<Clock className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="处理中"
              value={stats.processing}
              prefix={<Inbox className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="已完成"
              value={stats.completed}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 主要内容 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'requests',
              label: (
                <span className="flex items-center gap-2">
                  <FileText />
                  我的请求
                </span>
              ),
              children: <ServiceRequestList />,
            },
            {
              key: 'approvals',
              label: (
                <span className="flex items-center gap-2">
                  <Clock />
                  待审批 ({pendingApprovals.length})
                </span>
              ),
              children: (
                <Table
                  columns={approvalColumns}
                  dataSource={pendingApprovals}
                  rowKey="id"
                  pagination={false}
                />
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}

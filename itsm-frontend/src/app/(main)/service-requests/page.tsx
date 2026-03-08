'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Tabs, Table, Tag, Button, message } from 'antd';
import ServiceRequestList from '@/components/service-request/ServiceRequestList';
import { FileText, Clock, CheckCircle, XCircle, Plus, Inbox } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { serviceRequestAPI } from '@/lib/api/service-request-api';

const { Title, Text } = Typography;

const priorityColors: Record<string, string> = {
  '高': 'red',
  '中': 'orange',
  '低': 'default',
};

export default function ServiceRequestsPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('requests');
  const [pendingApprovals, setPendingApprovals] = useState<any[]>([]);
  const [stats, setStats] = useState({
    totalRequests: 0,
    pending: 0,
    processing: 0,
    completed: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Fetch pending approvals
      const pendingData = await serviceRequestAPI.getPendingApprovals({ page: 1, size: 20 });
      setPendingApprovals(pendingData.requests.map((r: any) => ({
        id: r.id,
        request_no: r.id,
        title: r.title || r.catalog?.name || '服务请求',
        applicant: r.requester?.name || r.requester?.username || '-',
        date: r.created_at ? new Date(r.created_at).toLocaleDateString() : '-',
        priority: '中',
      })));

      // Fetch all requests for stats (with large page size)
      const allRequests = await serviceRequestAPI.getUserServiceRequests({ page: 1, size: 1000 });
      const requests = allRequests.requests || [];

      const pending = requests.filter((r: any) => r.status === 'submitted' || r.status === 'manager_approved' || r.status === 'it_approved' || r.status === 'security_approved').length;
      const processing = requests.filter((r: any) => r.status === 'provisioning').length;
      const completed = requests.filter((r: any) => r.status === 'delivered').length;

      setStats({
        totalRequests: allRequests.total || 0,
        pending,
        processing,
        completed,
      });
    } catch (error) {
      console.error('Failed to fetch service request stats:', error);
      message.error('获取服务请求统计数据失败，请稍后重试');
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

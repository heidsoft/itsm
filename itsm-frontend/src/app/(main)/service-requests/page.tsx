'use client';

import React, { useState, useEffect } from 'react';
import { App, Card, Row, Col, Statistic, Typography, Tabs, Table, Button } from 'antd';
import ServiceRequestList from '@/components/service-request/ServiceRequestList';
import { FileText, Clock, CheckCircle, Inbox, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { serviceRequestAPI } from '@/lib/api/service-request-api';
import { ServiceRequestStatus } from '@/constants/service-request';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

// 审批待办收件箱行：只承载后端 ServiceRequestResponse 真实存在的字段。
// 原来的「优先级」列取自 r.priority，而服务请求 DTO 从未定义该字段，
// 每一行都只能渲染成占位符「-」，属于把未实现伪装成已有能力，故整列删除。
interface PendingApprovalRow {
  id: number;
  requestNumber: string;
  title: string;
  applicant: string;
  createdAt: string;
}

export default function ServiceRequestsPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  const canCreateServiceRequest =
    hasPermission('service_request', 'write') || hasPermission('service_request', 'create');
  const [activeTab, setActiveTab] = useState('requests');
  const [pendingApprovals, setPendingApprovals] = useState<PendingApprovalRow[]>([]);
  const [stats, setStats] = useState({
    totalRequests: 0,
    pending: 0,
    processing: 0,
    completed: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // 两个收件箱并行取数；单个请求失败只让对应指标为 0，不阻塞整个页面。
      const [pendingResp, mineResp] = await Promise.all([
        serviceRequestAPI.getPendingApprovals({ page: 1, pageSize: 20 }).catch(() => null),
        serviceRequestAPI.getUserServiceRequests({ page: 1, pageSize: 100 }).catch(() => null),
      ]);

      setPendingApprovals(
        (pendingResp?.items ?? []).map(r => ({
          id: r.id,
          requestNumber: r.requestNumber,
          title: r.title || r.catalog?.name || 'Service Request',
          applicant: r.requester?.name || r.requester?.username || '-',
          createdAt: r.createdAt,
        }))
      );

      // 分状态计数来自 /me 的第一页（最多 100 条），只覆盖当前页而非全租户权威统计；
      // 需要精确口径时应由后端统计接口负责，已登记为待办。
      const mine = mineResp?.items ?? [];
      setStats({
        totalRequests: mineResp?.total ?? 0,
        pending: mine.filter(
          r =>
            r.status === ServiceRequestStatus.SUBMITTED ||
            r.status === ServiceRequestStatus.MANAGER_APPROVED ||
            r.status === ServiceRequestStatus.IT_APPROVED ||
            r.status === ServiceRequestStatus.SECURITY_APPROVED
        ).length,
        processing: mine.filter(r => r.status === ServiceRequestStatus.PROVISIONING).length,
        completed: mine.filter(r => r.status === ServiceRequestStatus.DELIVERED).length,
      });
    } catch (error) {
      console.error('Failed to fetch service request stats:', error);
      message.error('Failed to load service request statistics');
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const approvalColumns = [
    {
      title: 'Request number',
      dataIndex: 'requestNumber',
      key: 'requestNumber',
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: 'Requester',
      dataIndex: 'applicant',
      key: 'applicant',
    },
    {
      title: 'Application time',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (value: string) => formatDateTime(value),
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: PendingApprovalRow) => (
        <Button
          size="small"
          type="link"
          onClick={() => router.push(`/service-requests/${record.id}`)}
        >

          Approval
        </Button>
      ),
    },
  ];

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>

            Service Request
          </Title>
          <Text type="secondary">

            View and manage service requests and approval processes
          </Text>
        </div>
        {canCreateServiceRequest && (
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={() => router.push('/service-requests/new')}
          >

            New service request
          </Button>
        )}
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="My Requests"
              value={stats.totalRequests}
              prefix={<FileText className="text-blue-500 mr-2" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="Pending Approval"
              value={stats.pending}
              prefix={<Clock className="text-orange-500 mr-2" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="In Progress"
              value={stats.processing}
              prefix={<Inbox className="text-blue-500 mr-2" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="Completed"
              value={stats.completed}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              styles={{ content: { color: '#52c41a' } }}
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

                  My Requests
                </span>
              ),
              children: <ServiceRequestList />,
            },
            {
              key: 'approvals',
              label: (
                <span className="flex items-center gap-2">
                  <Clock />

                  Pending approval ({pendingApprovals.length})
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

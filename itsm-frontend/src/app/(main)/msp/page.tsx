'use client';

import { useEffect, useState } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Table,
  Tag,
  Button,
  Alert,
  Tabs,
  List,
  Avatar,
  Spin,
} from 'antd';
import {
  UserOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import MSPService from '@/services/msp-service';
import type { MSPAllocation, MSPCustomerReport, MSPContext } from '@/types/msp';

export default function MSPDashboardPage() {
  const [loading, setLoading] = useState(false);
  const [isMSP, setIsMSP] = useState(false);
  const [isAdmin, setIsAdmin] = useState(false);
  const [allocations, setAllocations] = useState<MSPAllocation[]>([]);
  const [customers, setCustomers] = useState<{ id: number; code: string; name: string }[]>([]);
  const [reports, setReports] = useState<MSPCustomerReport[]>([]);
  const [mspContext, setMSPContext] = useState<MSPContext | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    checkMSPAccess();
  }, []);

  useEffect(() => {
    if (isMSP || isAdmin) {
      loadDashboardData();
    }
  }, [isMSP, isAdmin]);

  const checkMSPAccess = async () => {
    try {
      const { isMSP: mspFlag, isAdmin: adminFlag } = await MSPService.isMSPUser();
      setIsMSP(mspFlag);
      setIsAdmin(adminFlag);
      if (!mspFlag && !adminFlag) {
        setError('您不是 MSP 员工，无法访问此页面');
      } else if (adminFlag) {
        setError(null); // 管理员可以访问
      }
    } catch (err: any) {
      setError(err.message || '检查 MSP 状态失败');
    }
  };

  const loadDashboardData = async () => {
    setLoading(true);
    try {
      // 并行加载所有数据
      const [allocRes, custRes, ctxRes] = await Promise.all([
        MSPService.getAllocations(),
        MSPService.getCustomers(),
        MSPService.getMSPContext(),
      ]);

      setAllocations(allocRes.allocations);
      setCustomers(custRes.customers);
      setMSPContext(ctxRes);

      // 获取最近30天报表
      const endDate = new Date().toISOString().split('T')[0];
      const startDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];
      const reportsData = await MSPService.getCustomerReports({
        start_date: startDate,
        end_date: endDate,
      });
      setReports(reportsData);
    } catch (err: any) {
      setError(err.message || '加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const allocationColumns = [
    {
      title: 'MSP 员工',
      dataIndex: 'msp_username',
      key: 'msp_username',
    },
    {
      title: '客户租户',
      dataIndex: 'customer_name',
      key: 'customer_name',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const color = role === 'primary' ? 'green' : role === 'backup' ? 'orange' : 'blue';
        return <Tag color={color}>{role.toUpperCase()}</Tag>;
      },
    },
    {
      title: '分配时间',
      dataIndex: 'assignedAt',
      key: 'assignedAt',
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
  ];

  const reportColumns = [
    {
      title: '客户名称',
      dataIndex: 'customer_name',
      key: 'customer_name',
    },
    {
      title: '工单总数',
      dataIndex: 'total_tickets',
      key: 'total_tickets',
      sorter: (a: MSPCustomerReport, b: MSPCustomerReport) => a.total_tickets - b.total_tickets,
    },
    {
      title: '已解决',
      dataIndex: 'resolved_tickets',
      key: 'resolved_tickets',
    },
    {
      title: '解决率',
      key: 'resolution_rate',
      render: (record: MSPCustomerReport) => {
        const rate =
          record.total_tickets > 0
            ? Number(((record.resolved_tickets / record.total_tickets) * 100).toFixed(1))
            : 0;
        return <Tag color={rate >= 90 ? 'green' : rate >= 70 ? 'orange' : 'red'}>{rate}%</Tag>;
      },
    },
    {
      title: '平均处理时长(小时)',
      dataIndex: 'msp_handling_time_avg',
      key: 'msp_handling_time_avg',
      render: (val: number) => val.toFixed(2),
    },
    {
      title: 'SLA 合规率',
      key: 'sla_compliance_rate',
      render: (val: number) => {
        const rate = (val * 100).toFixed(1);
        return <Tag color={val >= 0.95 ? 'green' : val >= 0.8 ? 'orange' : 'red'}>{rate}%</Tag>;
      },
    },
  ];

  // 对于非 MSP 用户，显示欢迎页面
  if (!isMSP) {
    return (
      <div style={{ padding: 24 }}>
        <Card>
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <div style={{ marginBottom: 24 }}>
              <Avatar size={80} icon={<UserOutlined />} style={{ backgroundColor: '#1890ff' }} />
            </div>
            <Alert
              message="MSP 管理控制台"
              description={
                <div>
                  <p>您当前不是 MSP 员工，无法访问 MSP 管理功能。</p>
                  <p style={{ marginTop: 8, color: '#888' }}>
                    MSP（Managed Service Provider）管理控制台用于管理多租户服务。
                  </p>
                  <p style={{ marginTop: 16, fontSize: 12, color: '#999' }}>
                    如需访问，请联系系统管理员为您分配 MSP 员工角色。
                  </p>
                </div>
              }
              type="info"
              showIcon
              style={{ maxWidth: 500, margin: '0 auto' }}
            />
          </div>
        </Card>
      </div>
    );
  }

  if (loading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" tip="加载 MSP 数据..." />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 24 }}>
        <Alert
          message="加载错误"
          description={error}
          type="error"
          showIcon
          action={
            <Button size="small" onClick={() => window.location.reload()}>
              重试
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="负责客户数" value={customers.length} prefix={<UserOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="分配数量" value={allocations.length} prefix={<FileTextOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="本月工单"
              value={reports.reduce((sum, r) => sum + r.total_tickets, 0)}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={reports.reduce((sum, r) => sum + r.resolved_tickets, 0)}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Tabs
        defaultActiveKey="allocations"
        items={[
          {
            key: 'allocations',
            label: '分配列表',
            children: (
              <Table
                columns={allocationColumns}
                dataSource={allocations}
                rowKey="id"
                loading={loading}
                pagination={{ pageSize: 10 }}
              />
            ),
          },
          {
            key: 'reports',
            label: '服务报表',
            children: (
              <Table
                columns={reportColumns}
                dataSource={reports}
                rowKey="customer_tenant_id"
                loading={loading}
                pagination={{ pageSize: 10 }}
              />
            ),
          },
          {
            key: 'context',
            label: 'MSP 上下文',
            children: (
              <Card title="当前 MSP 上下文">
                <List
                  dataSource={[
                    { label: '是否 MSP', value: mspContext?.is_msp },
                    { label: 'MSP 用户 ID', value: mspContext?.msp_user_id },
                    { label: '当前客户租户', value: mspContext?.customer_tenant_id || '未指定' },
                    { label: '分配角色', value: mspContext?.role || 'N/A' },
                    {
                      label: '允许访问的客户',
                      value: mspContext?.allowed_customers?.length
                        ? mspContext.allowed_customers.join(', ')
                        : '无',
                    },
                  ]}
                  renderItem={item => (
                    <List.Item>
                      <List.Item.Meta title={item.label} description={String(item.value)} />
                    </List.Item>
                  )}
                />
              </Card>
            ),
          },
        ]}
      />
    </div>
  );
}

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
  Select,
  Space,
  DatePicker,
  message,
} from 'antd';
import { User, FileText, Clock, AlertCircle, CheckCircle } from 'lucide-react';
import MSPService from '@/lib/services/msp-service';
import type { MSPAllocation, MSPCustomerReport, MSPContext, MSPAllocationHistory } from '@/types/msp';

const { RangePicker } = DatePicker;

export default function MSPDashboardPage() {
  const [loading, setLoading] = useState(false);
  const [isMSP, setIsMSP] = useState(false);
  const [isAdmin, setIsAdmin] = useState(false);
  const [allocations, setAllocations] = useState<MSPAllocation[]>([]);
  const [customers, setCustomers] = useState<{ id: number; code: string; name: string }[]>([]);
  const [reports, setReports] = useState<MSPCustomerReport[]>([]);
  const [mspContext, setMSPContext] = useState<MSPContext | null>(null);
  const [error, setError] = useState<string | null>(null);

  // 客户工单状态
  const [selectedCustomerId, setSelectedCustomerId] = useState<number | null>(null);
  const [customerTickets, setCustomerTickets] = useState<any[]>([]);
  const [ticketsLoading, setTicketsLoading] = useState(false);

  // 绩效报表状态
  const [performanceReports, setPerformanceReports] = useState<MSPCustomerReport[]>([]);
  const [perfLoading, setPerfLoading] = useState(false);

  // 分配历史状态
  const [allocationHistory, setAllocationHistory] = useState<MSPAllocationHistory[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);

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
        setError(null);
      }
    } catch (err: any) {
      setError(err.message || '检查 MSP 状态失败');
    }
  };

  const loadDashboardData = async () => {
    setLoading(true);
    try {
      const [allocRes, custRes, ctxRes] = await Promise.all([
        MSPService.getAllocations(),
        MSPService.getCustomers(),
        MSPService.getMSPContext(),
      ]);

      setAllocations(allocRes.allocations);
      setCustomers(custRes.customers);
      setMSPContext(ctxRes);

      const endDate = new Date().toISOString().split('T')[0];
      const startDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];
      const reportsData = await MSPService.getCustomerReports({
        startDate: startDate,
        endDate: endDate,
      });
      setReports(reportsData);
    } catch (err: any) {
      setError(err.message || '加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载客户工单
  const loadCustomerTickets = async (customerId: number) => {
    setSelectedCustomerId(customerId);
    setTicketsLoading(true);
    try {
      const result = await MSPService.getCustomerTickets(customerId);
      setCustomerTickets(result.tickets);
    } catch (err: any) {
      message.error(err.message || '加载客户工单失败');
      setCustomerTickets([]);
    } finally {
      setTicketsLoading(false);
    }
  };

  // 分配 MSP 技术员
  const handleAssignTechnician = async (ticketId: number) => {
    if (!selectedCustomerId) return;
    try {
      await MSPService.assignTechnician(ticketId, selectedCustomerId);
      message.success('技术员分配成功');
      loadCustomerTickets(selectedCustomerId);
    } catch (err: any) {
      message.error(err.message || '分配技术员失败');
    }
  };

  // 加载绩效报表
  const loadPerformanceReports = async (startDate?: string, endDate?: string) => {
    setPerfLoading(true);
    try {
      const end = endDate || new Date().toISOString().split('T')[0];
      const start = startDate || new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];
      const data = await MSPService.getCustomerReports({ startDate: start, endDate: end });
      setPerformanceReports(data);
    } catch (err: any) {
      message.error(err.message || '加载绩效报表失败');
    } finally {
      setPerfLoading(false);
    }
  };

  // 加载分配历史
  const loadAllocationHistory = async () => {
    setHistoryLoading(true);
    try {
      const data = await MSPService.getAllocationHistory({});
      setAllocationHistory(data);
    } catch (err: any) {
      message.error(err.message || '加载分配历史失败');
    } finally {
      setHistoryLoading(false);
    }
  };

  const allocationColumns = [
    { title: 'MSP 员工', dataIndex: 'mspUsername', key:'mspUsername' },
    { title: '客户租户', dataIndex: 'customerName', key:'customerName' },
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
      render: (date: string) => date ? new Date(date).toLocaleString('zh-CN') : '-',
    },
  ];

  const reportColumns = [
    { title: '客户名称', dataIndex: 'customerName', key:'customerName' },
    {
      title: '工单总数',
      dataIndex: 'totalTickets',
      key: 'totalTickets',
      sorter: (a: MSPCustomerReport, b: MSPCustomerReport) => a.totalTickets - b.totalTickets,
    },
    { title: '已解决', dataIndex: 'resolvedTickets', key: 'resolvedTickets' },
    {
      title: '解决率',
      key:'resolutionRate',
      render: (record: MSPCustomerReport) => {
        const rate = record.totalTickets > 0 ? Number(((record.resolvedTickets / record.totalTickets) * 100).toFixed(1)) : 0;
        return <Tag color={rate >= 90 ? 'green' : rate >= 70 ? 'orange' : 'red'}>{rate}%</Tag>;
      },
    },
    {
      title: '平均处理时长(小时)',
      dataIndex: 'mspHandlingTimeAvg',
      key: 'mspHandlingTimeAvg',
      render: (val: number) => val?.toFixed(2) || '-',
    },
    {
      title: 'SLA 合规率',
      key: 'slaComplianceRate',
      render: (_: any, record: MSPCustomerReport) => {
        const val = record.slaComplianceRate;
        const rate = (val * 100).toFixed(1);
        return <Tag color={val >= 0.95 ? 'green' : val >= 0.8 ? 'orange' : 'red'}>{rate}%</Tag>;
      },
    },
  ];

  const ticketColumns = [
    { title: '工单标题', dataIndex: 'title', key: 'title' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag>{status}</Tag>,
    },
    { title: '负责人', dataIndex: 'assigneeName', key: 'assigneeName', render: (v: string) => v || '未分配' },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Button
          size="small"
          type="link"
          onClick={() => handleAssignTechnician(record.id)}
        >
          分配技术员
        </Button>
      ),
    },
  ];

  const historyColumns = [
    { title: 'MSP 员工', dataIndex: 'mspUsername', key:'mspUsername' },
    { title: '客户', dataIndex: 'customerName', key:'customerName' },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const color = role === 'primary' ? 'green' : role === 'backup' ? 'orange' : 'blue';
        return <Tag color={color}>{role.toUpperCase()}</Tag>;
      },
    },
    { title: '分配时间', dataIndex: 'assignedAt', key:'assignedAt', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
    { title: '解除时间', dataIndex: 'deassignedAt', key:'deassignedAt', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
    { title: '解除原因', dataIndex:'deallocationReason', key:'deallocationReason', render: (v: string) => v || '-' },
  ];

  if (!isMSP && !isAdmin) {
    return (
      <div style={{ padding: 24 }}>
        <Card>
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <div style={{ marginBottom: 24 }}>
              <Avatar size={80} icon={<User />} style={{ backgroundColor: '#1890ff' }} />
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
            <Statistic title="负责客户数" value={customers.length} prefix={<User />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="分配数量" value={allocations.length} prefix={<FileText />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="本月工单"
              value={reports.reduce((sum, r) => sum + r.totalTickets, 0)}
              prefix={<Clock />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={reports.reduce((sum, r) => sum + r.resolvedTickets, 0)}
              prefix={<CheckCircle />}
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
            key: 'tickets',
            label: '客户工单',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Space>
                    <span>选择客户：</span>
                    <Select
                      placeholder="请选择客户"
                      style={{ width: 300 }}
                      value={selectedCustomerId || undefined}
                      onChange={loadCustomerTickets}
                      showSearch
                      optionFilterProp="children"
                    >
                      {customers.map(c => (
                        <Select.Option key={c.id} value={c.id}>
                          {c.code} - {c.name}
                        </Select.Option>
                      ))}
                    </Select>
                  </Space>
                </div>
                {selectedCustomerId ? (
                  <Table
                    columns={ticketColumns}
                    dataSource={customerTickets}
                    rowKey="id"
                    loading={ticketsLoading}
                    pagination={{ pageSize: 10 }}
                    locale={{ emptyText: '暂无工单数据' }}
                  />
                ) : (
                  <Alert message="请先选择一个客户以查看其工单" type="info" showIcon />
                )}
              </div>
            ),
          },
          {
            key: 'reports',
            label: '服务报表',
            children: (
              <Table
                columns={reportColumns}
                dataSource={reports}
                rowKey="customerTenantId"
                loading={loading}
                pagination={{ pageSize: 10 }}
              />
            ),
          },
          {
            key: 'performance',
            label: '绩效报表',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Button
                    type="primary"
                    loading={perfLoading}
                    onClick={() => loadPerformanceReports()}
                  >
                    加载绩效数据
                  </Button>
                </div>
                <Table
                  columns={reportColumns}
                  dataSource={performanceReports}
                  rowKey="customerTenantId"
                  loading={perfLoading}
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: '点击"加载绩效数据"按钮查看报表' }}
                />
              </div>
            ),
          },
          {
            key: 'history',
            label: '分配历史',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Button
                    type="primary"
                    loading={historyLoading}
                    onClick={loadAllocationHistory}
                  >
                    加载分配历史
                  </Button>
                </div>
                <Table
                  columns={historyColumns}
                  dataSource={allocationHistory}
                  rowKey="id"
                  loading={historyLoading}
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: '点击"加载分配历史"按钮查看历史记录' }}
                />
              </div>
            ),
          },
          {
            key: 'context',
            label: 'MSP 上下文',
            children: (
              <Card title="当前 MSP 上下文">
                <List
                  dataSource={[
                    { label: '是否 MSP', value: mspContext?.isMsp },
                    { label: 'MSP 用户 ID', value: mspContext?.mspUserId },
                    { label: '当前客户租户', value: mspContext?.customerTenantId || '未指定' },
                    { label: '分配角色', value: mspContext?.role || 'N/A' },
                    {
                      label: '允许访问的客户',
                      value: mspContext?.allowedCustomers?.length
                        ? mspContext.allowedCustomers.join(', ')
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

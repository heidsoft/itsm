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
  App,
} from 'antd';
import { User, FileText, Clock, AlertCircle, CheckCircle } from 'lucide-react';
import MSPService from '@/lib/services/msp-service';
import { hasProductCapability } from '@/config/product-capabilities';
import type { MSPAllocation, MSPCustomerReport, MSPPerformanceReport, MSPContext } from '@/types/msp';
import type { Ticket as ApiTicket } from '@/lib/api/ticket-api';

const { RangePicker } = DatePicker;

export default function MSPDashboardPage() {
  const { message } = App.useApp();
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
  const [customerTickets, setCustomerTickets] = useState<ApiTicket[]>([]);
  const [ticketsLoading, setTicketsLoading] = useState(false);

  // 绩效报表状态
  const [performanceReports, setPerformanceReports] = useState<MSPPerformanceReport[]>([]);
  const [perfLoading, setPerfLoading] = useState(false);

  // 分配历史状态
  const [allocationHistory, setAllocationHistory] = useState<MSPAllocation[]>([]);
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
      setCustomerTickets(result.items);
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
      const data = await MSPService.getPerformanceReports({ startDate: start, endDate: end });
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
      const res = await MSPService.getAllocationHistory({});
      setAllocationHistory(res.items);
    } catch (err: any) {
      message.error(err.message || '加载分配历史失败');
    } finally {
      setHistoryLoading(false);
    }
  };

  const allocationColumns = [
    { title: 'MSP 员工', dataIndex: 'mspUsername', key:'mspUsername' },
    { title: '客户租户', dataIndex: 'customerTenantName', key:'customerTenantName' },
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

  // 后端 /msp/reports/customers 是按 MSP 租户的区间聚合，只有 mspTenantId/区间/总数/状态分布；
  // 客户名称、解决率与 SLA 合规率需要按客户维度拆分，后端尚未产出，因此这里不展示这些列。
  const reportPeriodRender = (_: unknown, record: MSPCustomerReport | MSPPerformanceReport) => {
    const from = record.dateFrom ? record.dateFrom.slice(0, 10) : '不限';
    const to = record.dateTo ? record.dateTo.slice(0, 10) : '不限';
    return `${from} ~ ${to}`;
  };

  const reportColumns = [
    { title: 'MSP 租户', dataIndex: 'mspTenantId', key: 'mspTenantId' },
    { title: '统计区间', key: 'period', render: reportPeriodRender },
    {
      title: '工单总数',
      dataIndex: 'totalTickets',
      key: 'totalTickets',
      sorter: (a: MSPCustomerReport, b: MSPCustomerReport) => a.totalTickets - b.totalTickets,
    },
    {
      title: '状态分布',
      key: 'statusSummary',
      render: (_: unknown, record: MSPCustomerReport) => {
        const entries = Object.entries(record.statusSummary ?? {});
        if (entries.length === 0) return '-';
        return (
          <Space size={[4, 4]} wrap>
            {entries.map(([status, count]) => (
              <Tag key={status}>{`${status}: ${count}`}</Tag>
            ))}
          </Space>
        );
      },
    },
  ];

  const performanceReportColumns = [
    { title: 'MSP 租户', dataIndex: 'mspTenantId', key: 'mspTenantId' },
    { title: '统计区间', key: 'period', render: reportPeriodRender },
    {
      title: '工单总数',
      dataIndex: 'totalTickets',
      key: 'totalTickets',
      sorter: (a: MSPPerformanceReport, b: MSPPerformanceReport) => a.totalTickets - b.totalTickets,
    },
    { title: '已解决', dataIndex: 'resolvedTickets', key: 'resolvedTickets' },
    {
      title: '平均解决时长(小时)',
      dataIndex: 'avgResolutionHours',
      key: 'avgResolutionHours',
      render: (val: number) => val.toFixed(2),
    },
  ];

  const ticketColumns = [
    { title: '工单编号', dataIndex: 'ticketNumber', key: 'ticketNumber' },
    { title: '工单标题', dataIndex: 'title', key: 'title' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag>{status}</Tag>,
    },
    {
      title: '负责人',
      key: 'assignee',
      render: (_: unknown, record: ApiTicket) => record.assignee?.name || record.assignee?.username || '未分配',
    },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: ApiTicket) => (
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
    { title: '客户', dataIndex: 'customerTenantName', key:'customerTenantName' },
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
              message="您当前不是 MSP 员工，请联系系统管理员分配角色"
              type="warning"
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
              title="近 30 天工单"
              value={reports.reduce((sum, r) => sum + r.totalTickets, 0)}
              prefix={<Clock />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="近 30 天已解决"
              value={reports.reduce((sum, r) => sum + (r.statusSummary?.resolved ?? 0), 0)}
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
                rowKey="mspTenantId"
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
                  columns={performanceReportColumns}
                  dataSource={performanceReports}
                  rowKey="mspTenantId"
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
          // 分配历史依赖 GET /api/v1/msp/allocations/history，见 PRODUCT_CAPABILITIES.mspAllocationHistory。
        ].filter((item) => item.key !== 'history' || hasProductCapability('mspAllocationHistory'))}
      />
    </div>
  );
}

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
import { displayUserName, formatDateTime } from '@/lib/formatters';
import type { MSPAllocation, MSPCustomerReport, MSPPerformanceReport, MSPContext } from '@/types/msp';
import type { Ticket as ApiTicket } from '@/lib/api/ticket-api';

const { RangePicker } = DatePicker;

export default function MSPDashboardPage() {
  const { message } = App.useApp();
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧渲染出
  // 空态内容而不是整页 Spin 遮罩。各子区块另有独立标志（ticketsLoading / perfLoading / historyLoading）。
  const [listLoading, setListLoading] = useState(true);
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
        setError('You are not an MSP employee and cannot access this page');
      } else if (adminFlag) {
        setError(null);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to check MSP status');
    }
  };

  const loadDashboardData = async () => {
    setListLoading(true);
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
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setListLoading(false);
    }
  };

  // 加载客户工单
  const loadCustomerTickets = async (customerId: number) => {
    setSelectedCustomerId(customerId);
    setTicketsLoading(true);
    try {
      const result = await MSPService.getCustomerTickets(customerId);
      setCustomerTickets(result.items);
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'Failed to load customer ticket');
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
      message.success('Technician assigned successfully');
      loadCustomerTickets(selectedCustomerId);
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'Failed to assign technician');
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
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'Failed to load performance report');
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
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'Failed to load allocation history');
    } finally {
      setHistoryLoading(false);
    }
  };

  const allocationColumns = [
    { title: 'MSP employee', dataIndex: 'mspUsername', key:'mspUsername' },
    { title: 'Customer Tenant', dataIndex: 'customerTenantName', key:'customerTenantName' },
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const color = role === 'primary' ? 'green' : role === 'backup' ? 'orange' : 'blue';
        return <Tag color={color}>{role.toUpperCase()}</Tag>;
      },
    },
    {
      title: 'Allocate time',
      dataIndex: 'assignedAt',
      key: 'assignedAt',
      render: (date: string) => formatDateTime(date) || '-',
    },
  ];

  // 后端 /msp/reports/customers 是按 MSP 租户的区间聚合，只有 mspTenantId/区间/总数/状态分布；
  // 客户名称、解决率与 SLA 合规率需要按客户维度拆分，后端尚未产出，因此这里不展示这些列。
  const reportPeriodRender = (_: unknown, record: MSPCustomerReport | MSPPerformanceReport) => {
    const from = record.dateFrom ? record.dateFrom.slice(0, 10) : 'No limit';
    const to = record.dateTo ? record.dateTo.slice(0, 10) : 'No limit';
    return `${from} ~ ${to}`;
  };

  const reportColumns = [
    { title: 'MSP tenant', dataIndex: 'mspTenantId', key: 'mspTenantId' },
    { title: 'Statistical interval', key: 'period', render: reportPeriodRender },
    {
      title: 'Total number of work orders',
      dataIndex: 'totalTickets',
      key: 'totalTickets',
      sorter: (a: MSPCustomerReport, b: MSPCustomerReport) => a.totalTickets - b.totalTickets,
    },
    {
      title: 'Status distribution',
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
    { title: 'MSP tenant', dataIndex: 'mspTenantId', key: 'mspTenantId' },
    { title: 'Statistical interval', key: 'period', render: reportPeriodRender },
    {
      title: 'Total number of work orders',
      dataIndex: 'totalTickets',
      key: 'totalTickets',
      sorter: (a: MSPPerformanceReport, b: MSPPerformanceReport) => a.totalTickets - b.totalTickets,
    },
    { title: 'Resolved', dataIndex: 'resolvedTickets', key: 'resolvedTickets' },
    {
      title: 'Average resolution time (hours)',
      dataIndex: 'avgResolutionHours',
      key: 'avgResolutionHours',
      render: (val: number) => val.toFixed(2),
    },
  ];

  const ticketColumns = [
    { title: 'Ticket Number', dataIndex: 'ticketNumber', key: 'ticketNumber' },
    { title: 'Ticket Title', dataIndex: 'title', key: 'title' },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag>{status}</Tag>,
    },
    {
      title: 'Assignee',
      key: 'assignee',
      render: (_: unknown, record: ApiTicket) =>
        displayUserName(record.assigneeName, record.assigneeId, 'Unassigned'),
    },
    { title: 'Created', dataIndex: 'createdAt', key: 'createdAt', render: (v: string) => formatDateTime(v) || '-' },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: ApiTicket) => (
        <Button
          size="small"
          type="link"
          onClick={() => handleAssignTechnician(record.id)}
        >

          Assignment Technician
        </Button>
      ),
    },
  ];

  const historyColumns = [
    { title: 'MSP employee', dataIndex: 'mspUsername', key:'mspUsername' },
    { title: 'Customer', dataIndex: 'customerTenantName', key:'customerTenantName' },
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const color = role === 'primary' ? 'green' : role === 'backup' ? 'orange' : 'blue';
        return <Tag color={color}>{role.toUpperCase()}</Tag>;
      },
    },
    { title: 'Allocate time', dataIndex: 'assignedAt', key:'assignedAt', render: (v: string) => formatDateTime(v) || '-' },
    { title: 'Release time', dataIndex: 'deassignedAt', key:'deassignedAt', render: (v: string) => formatDateTime(v) || '-' },
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
              title="You are not currently an MSP employee, please contact your system administrator to assign a role"
              type="warning"
              showIcon
              style={{ maxWidth: 500, margin: '0 auto' }}
            />
          </div>
        </Card>
      </div>
    );
  }

  if (listLoading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" tip="Loading MSP data..." />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 24 }}>
        <Alert
          title="Loading error"
          description={error}
          type="error"
          showIcon
          action={
            <Button size="small" onClick={() => window.location.reload()}>

              Retry
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
            <Statistic title="Number of customers responsible" value={customers.length} prefix={<User />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="allocated quantity" value={allocations.length} prefix={<FileText />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Work orders in the past 30 days"
              value={reports.reduce((sum, r) => sum + r.totalTickets, 0)}
              prefix={<Clock />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Resolved in the past 30 days"
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
            label: 'Allocation list',
            children: (
              <Table
                columns={allocationColumns}
                dataSource={allocations}
                rowKey="id"
                loading={listLoading}
                pagination={{ pageSize: 10 }}
              />
            ),
          },
          {
            key: 'tickets',
            label: 'Customer ticket',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Space>
                    <span>Select customer:</span>
                    <Select
                      placeholder="Please select a customer"
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
                    locale={{ emptyText: 'No ticket data' }}
                  />
                ) : (
                  <Alert title="Please select a customer first to view their tickets" type="info" showIcon />
                )}
              </div>
            ),
          },
          {
            key: 'reports',
            label: 'Service Report',
            children: (
              <Table
                columns={reportColumns}
                dataSource={reports}
                rowKey="mspTenantId"
                loading={listLoading}
                pagination={{ pageSize: 10 }}
              />
            ),
          },
          {
            key: 'performance',
            label: 'Performance Report',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Button
                    type="primary"
                    loading={perfLoading}
                    onClick={() => loadPerformanceReports()}
                  >

                    Load performance data
                  </Button>
                </div>
                <Table
                  columns={performanceReportColumns}
                  dataSource={performanceReports}
                  rowKey="mspTenantId"
                  loading={perfLoading}
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: 'Click the "Load Performance Data" button to view the report' }}
                />
              </div>
            ),
          },
          {
            key: 'history',
            label: 'Allocation history',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Button
                    type="primary"
                    loading={historyLoading}
                    onClick={loadAllocationHistory}
                  >

                    Load allocation history
                  </Button>
                </div>
                <Table
                  columns={historyColumns}
                  dataSource={allocationHistory}
                  rowKey="id"
                  loading={historyLoading}
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: 'Click the "Load Allocation History" button to view the history' }}
                />
              </div>
            ),
          },
          {
            key: 'context',
            label: 'MSP context',
            children: (
              <Card title="Current MSP context">
                <List
                  dataSource={[
                    { label: 'Is it MSP?', value: mspContext?.isMsp },
                    { label: 'MSP user ID', value: mspContext?.mspUserId },
                    { label: 'Current customer tenant', value: mspContext?.customerTenantId || 'Not specified' },
                    { label: 'Assign roles', value: mspContext?.role || 'N/A' },
                    {
                      label: 'Clients allowed access',
                      value: mspContext?.allowedCustomers?.length
                        ? mspContext.allowedCustomers.join(', ')
                        : 'None',
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

'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Space,
  Tag,
  Typography,
  App,
} from 'antd';
import { Search, RefreshCw, ShieldCheck } from 'lucide-react';

import {
  listAuditLogs,
  type AuditLog,
  type ListAuditLogsParams,
} from '@/lib/api/auditlog-api';
import { useAuthStoreHydration } from '@/lib/store/auth-store';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

const METHOD_COLORS: Record<string, string> = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  PATCH: 'cyan',
  DELETE: 'red',
};

// 操作码中文映射：取值来自后端 middleware/audit.go 的 AuditableAction 常量、
// 登录审计的 LOGIN_SUCCESS/LOGIN_FAILED，以及 BPMN 审计动作（如 reassigned）。
const actionLabels: Record<string, string> = {
  login: 'Login',
  logout: 'Log out',
  LOGIN_SUCCESS: 'Login Successful',
  LOGIN_FAILED: 'Login Failed',
  create: 'Create',
  update: 'Edit',
  delete: 'Delete',
  view: 'View',
  read: 'Read',
  write: 'Write',
  search: 'Search',
  export: 'Export',
  import: 'Import',
  assign: 'Assign',
  escalate: 'Escalate',
  resolve: 'Resolve',
  close: 'Close',
  reopen: 'Reopen',
  comment: 'Comments',
  attachment: 'Attachments',
  permission: 'Permission change',
  configuration: 'Configuration change',
  submit: 'Submit',
  approve: 'ApprovalApprove',
  reject: 'Reject',
  schedule: 'Schedule',
  start: 'Start',
  complete: 'Complete',
  rollback: 'Rollback',
  cancel: 'Cancel',
  // BPMN 流程审计动作
  terminated: 'Process terminated',
  assigned: 'Task assignment',
  unassigned: 'Unassign',
  claimed: 'Receive mission',
  completed: 'Task completed',
  cancelled: 'Task canceled',
  escalated: 'Task upgrade',
  reassigned: 'reallocation',
  variable_changed: 'Variable change',
  activity_started: 'Activity starts',
  activity_completed: 'Activity completed',
  delegate: 'Delegate',
  add_approver: 'Add approver',
  system_decision: 'System Decision',
};

// 资源码中文映射：后端取 URL 的第三段（多为复数 kebab-case），
// 少数业务写入使用 `ticket_type:<id>` 形式，按冒号前缀回退。
const resourceLabels: Record<string, string> = {
  auth: 'Authentication',
  tickets: 'Ticket',
  ticket: 'Ticket',
  ticket_type: 'Ticket Types',
  incidents: 'Incident',
  incident: 'Incident',
  problems: 'Problem',
  problem: 'Problem',
  changes: 'Change',
  change: 'Change',
  releases: 'Release',
  release: 'Release',
  'service-requests': 'Service Request',
  service_request: 'Service Request',
  'service-catalog': 'Service Catalog',
  service_catalog: 'Service Catalog',
  knowledge: 'Knowledge Base',
  cmdb: 'CMDB',
  assets: 'Asset',
  asset: 'Asset',
  licenses: 'Licenses',
  license: 'Licenses',
  users: 'Users',
  user: 'Users',
  roles: 'Role',
  role: 'Role',
  menus: 'Menu',
  tenants: 'Tenant',
  tenant: 'Tenant',
  'audit-logs': 'Audit Logs',
  notifications: 'Notification',
  approvals: 'Approval',
  bpmn: 'Workflows',
  workflows: 'Workflow Instances',
  marketplace: 'ApplyMarketplace',
  connectors: 'Connectors',
  msp: 'MSP',
  reports: 'Report',
  dashboard: 'Dashboard',
  projects: 'Project',
  applications: 'Application',
  'email-intake': 'Email acceptance',
  ai: 'AI Capabilities',
  sla: 'SLA',
  approvals_workflows: 'Workflow',
};

const labelOr = (labels: Record<string, string>, code: string): string => {
  if (!code) return '-';
  if (labels[code]) return labels[code];
  // `ticket_type:1` 等前缀形式：优先映射前缀，再兜底原值
  const prefix = code.split(':')[0];
  return labels[prefix] ?? code;
};

const statusColor = (code: number): string => {
  if (code >= 500) return 'red';
  if (code >= 400) return 'orange';
  if (code >= 300) return 'gold';
  return 'green';
};

const AuditLogsPage: React.FC = () => {
  const { message } = App.useApp();
  useAuthStoreHydration();

  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  // 筛选条件（与后端 ListAuditLogsRequest 对齐）
  const [filters, setFilters] = useState({
    action: '',
    resource: '',
    method: undefined as string | undefined,
    statusCode: undefined as number | undefined,
    path: '',
  });
  // 已生效的查询条件（点击“查询”后提交）
  const [appliedFilters, setAppliedFilters] = useState(filters);

  const fetchLogs = useCallback(
    async (page: number, pageSize: number, query: typeof appliedFilters) => {
      setLoading(true);
      try {
        const params: ListAuditLogsParams = {
          page,
          pageSize,
          resource: query.resource || undefined,
          action: query.action || undefined,
          method: query.method,
          statusCode: query.statusCode,
          path: query.path || undefined,
        };
        const res = await listAuditLogs(params);
        setLogs(res.logs ?? []);
        setPagination({
          current: res.page || page,
          pageSize: res.pageSize || pageSize,
          total: res.total ?? 0,
        });
      } catch (err) {
        message.error(
          `Failed to load audit log:${err instanceof Error ? err.message : 'Unknown error'}`
        );
      } finally {
        setLoading(false);
      }
    },
    [message]
  );

  useEffect(() => {
    void fetchLogs(1, pagination.pageSize, appliedFilters);

  }, [appliedFilters]);

  const handleSearch = () => {
    setAppliedFilters({ ...filters });
  };

  const handleReset = () => {
    const empty = {
      action: '',
      resource: '',
      method: undefined,
      statusCode: undefined,
      path: '',
    };
    setFilters(empty);
    setAppliedFilters(empty);
  };

  const columns = [
    {
      title: 'Timestamp',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      // 后端返回 ISO 字符串（含时区后缀 25+ 字符），nowrap 直接撑爆 170px 列宽；
      // 改为本地化短格式，列宽内完整展示。
      render: (v: string) => (
        <Text style={{ whiteSpace: 'nowrap' }}>
          {formatDateTime(v) || '-'}
        </Text>
      ),
    },
    {
      title: 'Users',
      dataIndex: 'userId',
      key: 'userId',
      width: 120,
      // 优先显示后端 join user 表填充的 userName；缺失时回退 #userId
      render: (v: number, record: AuditLog) =>
        record.userName?.trim() ? record.userName : v ? `#${v}` : '-',
    },
    {
      title: 'Actions',
      dataIndex: 'action',
      key: 'action',
      width: 160,
      ellipsis: true,
      // 显示中文动作名；未收录的动作码保留原值，便于排查
      render: (v: string) => labelOr(actionLabels, v),
    },
    {
      title: 'Resources',
      dataIndex: 'resource',
      key: 'resource',
      width: 160,
      ellipsis: true,
      render: (v: string) => labelOr(resourceLabels, v),
    },
    {
      title: 'path',
      dataIndex: 'path',
      key: 'path',
      ellipsis: true,
      render: (v: string) => <Text code>{v || '-'}</Text>,
    },
    {
      title: 'Method',
      dataIndex: 'method',
      key: 'method',
      width: 90,
      render: (v: string) => (
        <Tag color={METHOD_COLORS[v] ?? 'default'}>{v}</Tag>
      ),
    },
    {
      title: 'status code',
      dataIndex: 'statusCode',
      key: 'statusCode',
      width: 90,
      render: (v: number) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
      ellipsis: true,
    },
  ];

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center gap-2">
        <ShieldCheck className="w-5 h-5 text-blue-500" />
        <Title level={4} style={{ margin: 0 }}>

          Audit Logs
        </Title>
      </div>

      <Card size="small">
        <Space wrap>
          <Input
            placeholder="ActionsType"
            allowClear
            style={{ width: 160 }}
            value={filters.action}
            onChange={(e) => setFilters({ ...filters, action: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Input
            placeholder="Resource name"
            allowClear
            style={{ width: 160 }}
            value={filters.resource}
            onChange={(e) => setFilters({ ...filters, resource: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Input
            placeholder="Operation path"
            allowClear
            style={{ width: 220 }}
            value={filters.path}
            onChange={(e) => setFilters({ ...filters, path: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder="Request method"
            allowClear
            style={{ width: 120 }}
            value={filters.method}
            onChange={(v) => setFilters({ ...filters, method: v })}
            options={['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((m) => ({
              value: m,
              label: m,
            }))}
          />
          <Select
            placeholder="Result Status"
            allowClear
            style={{ width: 110 }}
            value={filters.statusCode}
            onChange={(v) => setFilters({ ...filters, statusCode: v })}
            options={[200, 400, 401, 403, 404, 500].map((c) => ({
              value: c,
              label: String(c),
            }))}
          />
          <Button type="primary" icon={<Search className="w-4 h-4" />} onClick={handleSearch}>

            Query
          </Button>
          <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>

            Reset
          </Button>
        </Space>
      </Card>

      <Card size="small">
        <Table<AuditLog>
          rowKey="id"
          columns={columns}
          dataSource={logs}
          loading={loading}
          scroll={{ x: 1100 }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `Total ${total} items`,
            onChange: (page, pageSize) => {
              void fetchLogs(page, pageSize, appliedFilters);
            },
          }}
          locale={{ emptyText: 'No audit log yet' }}
        />
      </Card>
    </div>
  );
};

export default AuditLogsPage;

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
  login: '登录',
  logout: '登出',
  LOGIN_SUCCESS: '登录成功',
  LOGIN_FAILED: '登录失败',
  create: '创建',
  update: '编辑',
  delete: '删除',
  view: '查看',
  read: '读取',
  write: '写入',
  search: '搜索',
  export: '导出',
  import: '导入',
  assign: '分配',
  escalate: '升级',
  resolve: '解决',
  close: '关闭',
  reopen: '重新打开',
  comment: '评论',
  attachment: '附件',
  permission: '权限变更',
  configuration: '配置变更',
  submit: '提交',
  approve: '审批通过',
  reject: '审批拒绝',
  schedule: '排期',
  start: '开始',
  complete: '完成',
  rollback: '回滚',
  cancel: '取消',
  // BPMN 流程审计动作
  terminated: '流程终止',
  assigned: '任务分配',
  unassigned: '取消分配',
  claimed: '领取任务',
  completed: '任务完成',
  cancelled: '任务取消',
  escalated: '任务升级',
  reassigned: '重新分配',
  variable_changed: '变量变更',
  activity_started: '活动开始',
  activity_completed: '活动完成',
  delegate: '委派',
  add_approver: '添加审批人',
  system_decision: '系统决策',
};

// 资源码中文映射：后端取 URL 的第三段（多为复数 kebab-case），
// 少数业务写入使用 `ticket_type:<id>` 形式，按冒号前缀回退。
const resourceLabels: Record<string, string> = {
  auth: '认证',
  tickets: '工单',
  ticket: '工单',
  ticket_type: '工单类型',
  incidents: '事件',
  incident: '事件',
  problems: '问题',
  problem: '问题',
  changes: '变更',
  change: '变更',
  releases: '发布',
  release: '发布',
  'service-requests': '服务请求',
  service_request: '服务请求',
  'service-catalog': '服务目录',
  service_catalog: '服务目录',
  knowledge: '知识库',
  cmdb: 'CMDB',
  assets: '资产',
  asset: '资产',
  licenses: '许可证',
  license: '许可证',
  users: '用户',
  user: '用户',
  roles: '角色',
  role: '角色',
  menus: '菜单',
  tenants: '租户',
  tenant: '租户',
  'audit-logs': '审计日志',
  notifications: '通知',
  approvals: '审批',
  bpmn: '工作流',
  workflows: '工作流实例',
  marketplace: '应用市场',
  connectors: '连接器',
  msp: 'MSP',
  reports: '报表',
  dashboard: '仪表盘',
  projects: '项目',
  applications: '应用',
  'email-intake': '邮件受理',
  ai: 'AI 能力',
  sla: 'SLA',
  approvals_workflows: '审批流程',
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
          `加载审计日志失败：${err instanceof Error ? err.message : '未知错误'}`
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
      title: '时间',
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
      title: '用户',
      dataIndex: 'userId',
      key: 'userId',
      width: 120,
      // 优先显示后端 join user 表填充的 userName；缺失时回退 #userId
      render: (v: number, record: AuditLog) =>
        record.userName?.trim() ? record.userName : v ? `#${v}` : '-',
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 160,
      ellipsis: true,
      // 显示中文动作名；未收录的动作码保留原值，便于排查
      render: (v: string) => labelOr(actionLabels, v),
    },
    {
      title: '资源',
      dataIndex: 'resource',
      key: 'resource',
      width: 160,
      ellipsis: true,
      render: (v: string) => labelOr(resourceLabels, v),
    },
    {
      title: '路径',
      dataIndex: 'path',
      key: 'path',
      ellipsis: true,
      render: (v: string) => <Text code>{v || '-'}</Text>,
    },
    {
      title: '方法',
      dataIndex: 'method',
      key: 'method',
      width: 90,
      render: (v: string) => (
        <Tag color={METHOD_COLORS[v] ?? 'default'}>{v}</Tag>
      ),
    },
    {
      title: '状态码',
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
          审计日志
        </Title>
      </div>

      <Card size="small">
        <Space wrap>
          <Input
            placeholder="操作类型"
            allowClear
            style={{ width: 160 }}
            value={filters.action}
            onChange={(e) => setFilters({ ...filters, action: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Input
            placeholder="资源名称"
            allowClear
            style={{ width: 160 }}
            value={filters.resource}
            onChange={(e) => setFilters({ ...filters, resource: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Input
            placeholder="操作路径"
            allowClear
            style={{ width: 220 }}
            value={filters.path}
            onChange={(e) => setFilters({ ...filters, path: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder="请求方式"
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
            placeholder="结果状态"
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
            查询
          </Button>
          <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>
            重置
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
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              void fetchLogs(page, pageSize, appliedFilters);
            },
          }}
          locale={{ emptyText: '暂无审计日志' }}
        />
      </Card>
    </div>
  );
};

export default AuditLogsPage;

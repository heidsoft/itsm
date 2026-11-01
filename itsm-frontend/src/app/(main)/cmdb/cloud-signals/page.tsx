'use client';

import React, { Suspense, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  Alert,
  App,
  Breadcrumb,
  Button,
  Card,
  Form,
  Input,
  Popconfirm,
  Select,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { BellRing, RefreshCw, RotateCcw, Search, ShieldAlert } from 'lucide-react';

import type {
  AlertListQuery,
  FindingListQuery,
} from '@/lib/api/cloudsignal-api';
import { serverErrorMessage } from '@/lib/api/http-client';
import {
  useAcknowledgeAlertMutation,
  useCloudAlertsQuery,
  useCloudSignalSyncStatusQuery,
  useFindingSummaryQuery,
  useIgnoreFindingMutation,
  useReopenFindingMutation,
  useResolveAlertMutation,
  useSecurityFindingsQuery,
} from '@/lib/hooks/useCloudSignals';
import { useCloudAccountsQuery } from '@/lib/hooks/useCMDB';
import { usePermissions } from '@/lib/hooks/use-permissions';
import type {
  CloudAlert,
  CloudSignalSyncState,
  SecurityFinding,
} from '@/types/biz/cloudsignal';

type SignalTab = 'alerts' | 'vulnerabilities';

interface AlertFilterValues {
  status?: string[];
  severity?: string[];
  cloudAccountId?: number;
  search?: string;
}

interface FindingFilterValues extends AlertFilterValues {
  vulnType?: string;
  instanceId?: string;
}

const alertStatusMeta: Record<string, { color: string; text: string }> = {
  firing: { color: 'red', text: '触发中' },
  acknowledged: { color: 'gold', text: '已认领' },
  resolved: { color: 'green', text: '已关闭' },
};

const findingStatusMeta: Record<string, { color: string; text: string }> = {
  open: { color: 'red', text: '待修复' },
  fixing: { color: 'gold', text: '修复中' },
  fixed: { color: 'green', text: '已修复' },
  ignored: { color: 'default', text: '已忽略' },
};

const severityColor: Record<string, string> = {
  P1: 'red',
  P2: 'orange',
  P3: 'blue',
  P4: 'default',
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
  unknown: 'default',
};

const severityText: Record<string, string> = {
  critical: '严重',
  high: '高危',
  medium: '中危',
  low: '低危',
  unknown: '未知',
};

const vulnTypeText: Record<string, string> = {
  cve: 'Linux 软件漏洞',
  sys: 'Windows 系统漏洞',
  cms: 'Web-CMS 漏洞',
  app: '应用漏洞',
  emg: '紧急漏洞',
};

const syncStateMeta: Record<string, { color: string; text: string }> = {
  unconfigured: { color: 'default', text: '未配置' },
  unavailable: { color: 'red', text: '不可用' },
  ready: { color: 'green', text: '就绪' },
};

const signalKindText: Record<string, string> = {
  alarm: '云监控告警',
  vulnerability: '安全漏洞',
};

const alertStatusOptions = [
  { value: 'firing', label: '触发中' },
  { value: 'acknowledged', label: '已认领' },
  { value: 'resolved', label: '已关闭' },
];

const alertSeverityOptions = [
  { value: 'P1', label: 'P1' },
  { value: 'P2', label: 'P2' },
  { value: 'P3', label: 'P3' },
  { value: 'P4', label: 'P4' },
];

const findingStatusOptions = [
  { value: 'open', label: '待修复' },
  { value: 'fixing', label: '修复中' },
  { value: 'fixed', label: '已修复' },
  { value: 'ignored', label: '已忽略' },
];

const findingSeverityOptions = [
  { value: 'critical', label: '严重' },
  { value: 'high', label: '高危' },
  { value: 'medium', label: '中危' },
  { value: 'low', label: '低危' },
  { value: 'unknown', label: '未知' },
];

const vulnTypeOptions = Object.entries(vulnTypeText).map(([value, label]) => ({ value, label }));

function formatTime(value?: string) {
  return value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-';
}

function bridgeTag(record: CloudAlert) {
  const status = record.annotations?.bridgeStatus;
  if (status === 'bridged') {
    const incidentId = record.annotations?.incidentId;
    return <Tag color="green">已建单{incidentId ? ` #${incidentId}` : ''}</Tag>;
  }
  if (status === 'failed') return <Tag color="red">建单失败</Tag>;
  if (status === 'claimed') return <Tag color="gold">建单中</Tag>;
  return <span className="text-gray-400">-</span>;
}

function CloudSignalsPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  const canWrite = hasPermission('cmdb', 'write');

  const activeTab: SignalTab =
    searchParams.get('tab') === 'vulnerabilities' ? 'vulnerabilities' : 'alerts';
  const instanceIdParam = searchParams.get('instanceId') ?? '';

  const [alertForm] = Form.useForm<AlertFilterValues>();
  const [findingForm] = Form.useForm<FindingFilterValues>();
  const [alertFilters, setAlertFilters] = useState<AlertFilterValues>({});
  const [findingFilters, setFindingFilters] = useState<FindingFilterValues>({
    status: ['open', 'fixing'],
    instanceId: instanceIdParam || undefined,
  });
  const [alertPage, setAlertPage] = useState({ current: 1, pageSize: 20 });
  const [findingPage, setFindingPage] = useState({ current: 1, pageSize: 20 });

  const accountsQuery = useCloudAccountsQuery();
  const accounts = accountsQuery.data?.items ?? [];
  const accountOptions = useMemo(
    () =>
      accounts.map(account => ({
        value: account.id,
        label: `${account.accountName} (${account.provider})`,
      })),
    [accounts],
  );

  const alertQueryParams = useMemo<AlertListQuery>(
    () => ({
      status: alertFilters.status,
      severity: alertFilters.severity,
      cloudAccountId: alertFilters.cloudAccountId,
      search: alertFilters.search,
      page: alertPage.current,
      pageSize: alertPage.pageSize,
    }),
    [alertFilters, alertPage],
  );

  const findingQueryParams = useMemo<FindingListQuery>(
    () => ({
      status: findingFilters.status,
      severity: findingFilters.severity,
      vulnType: findingFilters.vulnType,
      instanceId: findingFilters.instanceId,
      cloudAccountId: findingFilters.cloudAccountId,
      search: findingFilters.search,
      page: findingPage.current,
      pageSize: findingPage.pageSize,
    }),
    [findingFilters, findingPage],
  );

  const alertsQuery = useCloudAlertsQuery(alertQueryParams);
  const findingsQuery = useSecurityFindingsQuery(findingQueryParams);
  const summaryQuery = useFindingSummaryQuery();
  const syncQuery = useCloudSignalSyncStatusQuery();

  const alerts = alertsQuery.data?.items ?? [];
  const alertTotal = alertsQuery.data?.total ?? 0;
  const findings = findingsQuery.data?.items ?? [];
  const findingTotal = findingsQuery.data?.total ?? 0;
  const syncStates = syncQuery.data?.items ?? [];
  const summary = summaryQuery.data;

  const acknowledgeMutation = useAcknowledgeAlertMutation();
  const resolveMutation = useResolveAlertMutation();
  const ignoreMutation = useIgnoreFindingMutation();
  const reopenMutation = useReopenFindingMutation();

  const applyAlertFilters = () => {
    const values = alertForm.getFieldsValue();
    setAlertFilters({
      status: values.status,
      severity: values.severity,
      cloudAccountId: values.cloudAccountId,
      search: values.search?.trim() || undefined,
    });
    setAlertPage(previous => ({ ...previous, current: 1 }));
  };

  const resetAlertFilters = () => {
    alertForm.resetFields();
    setAlertFilters({});
    setAlertPage(previous => ({ ...previous, current: 1 }));
  };

  const applyFindingFilters = () => {
    const values = findingForm.getFieldsValue();
    setFindingFilters({
      status: values.status,
      severity: values.severity,
      vulnType: values.vulnType,
      instanceId: values.instanceId?.trim() || undefined,
      cloudAccountId: values.cloudAccountId,
      search: values.search?.trim() || undefined,
    });
    setFindingPage(previous => ({ ...previous, current: 1 }));
  };

  const resetFindingFilters = () => {
    findingForm.resetFields();
    setFindingFilters({
      status: ['open', 'fixing'],
      instanceId: instanceIdParam || undefined,
    });
    setFindingPage(previous => ({ ...previous, current: 1 }));
  };

  const refreshAll = () => {
    void alertsQuery.refetch();
    void findingsQuery.refetch();
    void summaryQuery.refetch();
    void syncQuery.refetch();
  };

  const refreshing =
    alertsQuery.isFetching ||
    findingsQuery.isFetching ||
    summaryQuery.isFetching ||
    syncQuery.isFetching;

  const changeTab = (key: string) => {
    const nextTab: SignalTab = key === 'vulnerabilities' ? 'vulnerabilities' : 'alerts';
    const query = new URLSearchParams({ tab: nextTab });
    if (instanceIdParam) query.set('instanceId', instanceIdParam);
    router.replace(`/cmdb/cloud-signals?${query.toString()}`);
  };

  const handleAcknowledge = (record: CloudAlert) => {
    acknowledgeMutation.mutate(record.id, {
      onSuccess: () => message.success(`已认领告警「${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, '认领告警失败')),
    });
  };

  const handleResolve = (record: CloudAlert) => {
    resolveMutation.mutate(record.id, {
      onSuccess: () => message.success(`已关闭告警「${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, '关闭告警失败')),
    });
  };

  const handleIgnore = (record: SecurityFinding) => {
    ignoreMutation.mutate(record.id, {
      onSuccess: () => message.success(`已忽略漏洞「${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, '忽略漏洞失败')),
    });
  };

  const handleReopen = (record: SecurityFinding) => {
    reopenMutation.mutate(record.id, {
      onSuccess: () => message.success(`已重新打开漏洞「${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, '重新打开漏洞失败')),
    });
  };

  const alertColumns: ColumnsType<CloudAlert> = [
    {
      title: '告警',
      width: 260,
      render: (_: unknown, record: CloudAlert) => (
        <div>
          <div className="font-medium">{record.name || '-'}</div>
          <div className="text-xs text-gray-500">{record.alertId}</div>
          {record.description && (
            <Tooltip title={record.description}>
              <div className="text-xs text-gray-400 truncate max-w-[240px]">{record.description}</div>
            </Tooltip>
          )}
        </div>
      ),
    },
    {
      title: '级别',
      dataIndex: 'severity',
      width: 80,
      render: (value: string) => <Tag color={severityColor[value] || 'default'}>{value || '-'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value: string) => {
        const meta = alertStatusMeta[value];
        return <Tag color={meta?.color || 'default'}>{meta?.text || value || '-'}</Tag>;
      },
    },
    {
      title: '服务',
      dataIndex: 'service',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: '来源 IP',
      dataIndex: 'sourceIp',
      width: 140,
      render: (value?: string) => value || '-',
    },
    {
      title: '触发时间',
      dataIndex: 'firedAt',
      width: 170,
      render: (value?: string) => formatTime(value),
    },
    {
      title: '建单',
      width: 120,
      render: (_: unknown, record: CloudAlert) => bridgeTag(record),
    },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_: unknown, record: CloudAlert) => {
        if (!canWrite || record.status === 'resolved') return <span className="text-gray-400">-</span>;
        return (
          <Space size="small">
            {record.status === 'firing' && (
              <Button
                type="link"
                size="small"
                loading={acknowledgeMutation.isPending}
                onClick={() => handleAcknowledge(record)}
              >
                认领
              </Button>
            )}
            <Popconfirm
              title="关闭告警"
              description={`确认关闭告警「${record.name}」？本地关闭不会影响云监控侧状态。`}
              okText="关闭"
              cancelText="取消"
              onConfirm={() => handleResolve(record)}
            >
              <Button type="link" size="small" danger>
                关闭
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const findingColumns: ColumnsType<SecurityFinding> = [
    {
      title: '漏洞',
      width: 280,
      render: (_: unknown, record: SecurityFinding) => (
        <div>
          <div className="font-medium">{record.name || record.findingKey}</div>
          <div className="text-xs text-gray-500">
            {record.findingKey}
            {record.aliasName ? ` · ${record.aliasName}` : ''}
          </div>
        </div>
      ),
    },
    {
      title: '严重度',
      width: 120,
      render: (_: unknown, record: SecurityFinding) => (
        <Space orientation="vertical" size={0}>
          <Tag color={severityColor[record.severity] || 'default'}>
            {severityText[record.severity] || record.severity || '-'}
          </Tag>
          {typeof record.cvssScore === 'number' && (
            <span className="text-xs text-gray-500">CVSS {record.cvssScore.toFixed(1)}</span>
          )}
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'vulnType',
      width: 130,
      render: (value?: string) => (value ? vulnTypeText[value] || value : '-'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value: string) => {
        const meta = findingStatusMeta[value];
        return <Tag color={meta?.color || 'default'}>{meta?.text || value || '-'}</Tag>;
      },
    },
    {
      title: '实例',
      width: 200,
      render: (_: unknown, record: SecurityFinding) => (
        <div>
          <div>{record.instanceName || '-'}</div>
          <div className="text-xs text-gray-500">{record.instanceId || '-'}</div>
        </div>
      ),
    },
    {
      title: '公网 IP',
      dataIndex: 'internetIp',
      width: 140,
      render: (value?: string) => value || '-',
    },
    {
      title: '关联 CI',
      width: 160,
      render: (_: unknown, record: SecurityFinding) =>
        record.ciId ? (
          <Button type="link" size="small" onClick={() => router.push(`/cmdb/cis/${record.ciId}`)}>
            {record.ciName || record.ciNumber || `#${record.ciId}`}
          </Button>
        ) : (
          <span className="text-gray-400">未关联</span>
        ),
    },
    {
      title: '最近发现',
      width: 160,
      render: (_: unknown, record: SecurityFinding) => formatTime(record.lastSeenAt || record.lastSyncedAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: SecurityFinding) => {
        if (!canWrite) return <span className="text-gray-400">-</span>;
        const active = record.status === 'open' || record.status === 'fixing';
        if (active) {
          return (
            <Popconfirm
              title="忽略漏洞"
              description={`确认忽略漏洞「${record.name}」？忽略后不再计入未修复统计。`}
              okText="忽略"
              cancelText="取消"
              onConfirm={() => handleIgnore(record)}
            >
              <Button type="link" size="small" danger loading={ignoreMutation.isPending}>
                忽略
              </Button>
            </Popconfirm>
          );
        }
        return (
          <Button
            type="link"
            size="small"
            loading={reopenMutation.isPending}
            onClick={() => handleReopen(record)}
          >
            重新打开
          </Button>
        );
      },
    },
  ];

  const syncColumns: ColumnsType<CloudSignalSyncState> = [
    {
      title: '云账号',
      width: 200,
      render: (_: unknown, record: CloudSignalSyncState) => (
        <div>
          <div>{record.accountName || `#${record.cloudAccountId}`}</div>
          <div className="text-xs text-gray-500">{record.provider}</div>
        </div>
      ),
    },
    {
      title: '信号',
      width: 130,
      render: (_: unknown, record: CloudSignalSyncState) =>
        signalKindText[record.signalKind] || record.signalKind,
    },
    {
      title: '状态',
      width: 100,
      render: (_: unknown, record: CloudSignalSyncState) => {
        const meta = syncStateMeta[record.state];
        return <Tag color={meta?.color || 'default'}>{meta?.text || record.state}</Tag>;
      },
    },
    {
      title: '最近成功',
      width: 170,
      render: (_: unknown, record: CloudSignalSyncState) => formatTime(record.lastSuccessAt),
    },
    {
      title: '本批条数',
      dataIndex: 'itemsSynced',
      width: 100,
    },
    {
      title: '连续失败',
      dataIndex: 'consecutiveFailures',
      width: 100,
      render: (value: number) => (value > 0 ? <Tag color="red">{value}</Tag> : 0),
    },
    {
      title: '最近错误',
      render: (_: unknown, record: CloudSignalSyncState) =>
        record.lastErrorMessage ? (
          <span className="text-xs text-red-600">
            {record.lastErrorCode ? `[${record.lastErrorCode}] ` : ''}
            {record.lastErrorMessage}
          </span>
        ) : (
          '-'
        ),
    },
  ];

  const hasUnavailable = syncStates.some(state => state.state === 'unavailable');
  const hasUnconfigured = syncStates.some(state => state.state === 'unconfigured');
  const syncProblemStates = syncStates.filter(state => state.state !== 'ready');

  const summaryItems = [
    { key: 'critical', label: '严重' },
    { key: 'high', label: '高危' },
    { key: 'medium', label: '中危' },
    { key: 'low', label: '低危' },
    { key: 'unknown', label: '未知' },
  ];

  const alertErrorText = serverErrorMessage(alertsQuery.error, '加载云监控告警失败，请重试');
  const findingErrorText = serverErrorMessage(findingsQuery.error, '加载安全漏洞失败，请重试');
  const syncErrorText = serverErrorMessage(syncQuery.error, '加载同步状态失败，请重试');

  return (
    <Card>
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">云安全与告警</h1>
          <p className="text-gray-500 mt-1">
            云监控告警与云安全中心漏洞，按云账号定时同步；P1/P2 告警由后端自动建事件单。
          </p>
        </div>
        <Space>
          <Button icon={<RefreshCw />} loading={refreshing} onClick={refreshAll}>
            刷新
          </Button>
          <Button onClick={() => router.push('/cmdb/cloud-resources')}>云资源</Button>
        </Space>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: '云安全与告警' },
        ]}
      />

      {syncQuery.isError ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          title={syncErrorText}
          description="同步状态取不到时无法判断告警和漏洞是否是最新数据。"
          action={
            <Button size="small" onClick={() => syncQuery.refetch()}>
              重试
            </Button>
          }
        />
      ) : syncStates.length === 0 && !syncQuery.isLoading ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="还没有云信号同步记录"
          description="同步按云账号执行：先在云账号中配置 secret:// 或 env:// 凭据引用，worker 下一轮同步后会在这里写入状态。"
        />
      ) : hasUnavailable ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="部分云账号同步失败"
          description={syncProblemStates
            .filter(state => state.state === 'unavailable')
            .map(
              state =>
                `${state.accountName || `#${state.cloudAccountId}`}（${signalKindText[state.signalKind] || state.signalKind}）：${state.lastErrorMessage || '未知错误'}`,
            )
            .join('；')}
        />
      ) : hasUnconfigured ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="部分云账号尚未配置同步凭据"
          description="未配置的账号不会拉取告警和漏洞；其余账号已按计划同步。"
        />
      ) : (
        <Alert
          type="success"
          showIcon
          style={{ marginBottom: 16 }}
          title="云信号同步正常"
          description={`最近一轮：${formatTime(
            syncStates.reduce<string | undefined>(
              (latest, state) =>
                state.lastSuccessAt && (!latest || dayjs(state.lastSuccessAt).isAfter(dayjs(latest)))
                  ? state.lastSuccessAt
                  : latest,
              undefined,
            ),
          )}`}
        />
      )}

      {syncStates.length > 0 && (
        <Card size="small" title="同步状态" style={{ marginBottom: 16 }}>
          <Table
            rowKey={record => `${record.cloudAccountId}-${record.signalKind}`}
            size="small"
            columns={syncColumns}
            dataSource={syncStates}
            loading={syncQuery.isLoading}
            pagination={false}
            scroll={{ x: 900 }}
          />
        </Card>
      )}

      <Tabs
        activeKey={activeTab}
        onChange={changeTab}
        items={[
          {
            key: 'alerts',
            label: (
              <Space size={6}>
                <BellRing className="h-4 w-4" />
                云监控告警
              </Space>
            ),
            children: (
              <div>
                <Form form={alertForm} layout="inline" className="mb-4 flex-wrap gap-2">
                  <Form.Item name="status" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="状态"
                      style={{ minWidth: 160 }}
                      options={alertStatusOptions}
                    />
                  </Form.Item>
                  <Form.Item name="severity" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="级别"
                      style={{ minWidth: 140 }}
                      options={alertSeverityOptions}
                    />
                  </Form.Item>
                  <Form.Item name="cloudAccountId" className="!mb-0">
                    <Select
                      allowClear
                      showSearch
                      optionFilterProp="label"
                      placeholder="云账号"
                      style={{ minWidth: 180 }}
                      options={accountOptions}
                    />
                  </Form.Item>
                  <Form.Item name="search" className="!mb-0">
                    <Input
                      allowClear
                      placeholder="名称 / 告警 ID / 描述"
                      style={{ width: 220 }}
                      onPressEnter={applyAlertFilters}
                    />
                  </Form.Item>
                  <Form.Item className="!mb-0">
                    <Space>
                      <Button type="primary" icon={<Search />} onClick={applyAlertFilters}>
                        查询
                      </Button>
                      <Button icon={<RotateCcw />} onClick={resetAlertFilters}>
                        重置
                      </Button>
                    </Space>
                  </Form.Item>
                </Form>

                {alertsQuery.isError && (
                  <Alert
                    type="error"
                    showIcon
                    style={{ marginBottom: 16 }}
                    title={alertErrorText}
                    action={
                      <Button size="small" onClick={() => alertsQuery.refetch()}>
                        重试
                      </Button>
                    }
                  />
                )}

                <Table
                  rowKey="id"
                  columns={alertColumns}
                  dataSource={alerts}
                  loading={alertsQuery.isLoading}
                  scroll={{ x: 1200 }}
                  pagination={{
                    current: alertPage.current,
                    pageSize: alertPage.pageSize,
                    total: alertTotal,
                    showSizeChanger: true,
                    showTotal: total => `共 ${total} 条告警`,
                    onChange: (page, pageSize) => setAlertPage({ current: page, pageSize }),
                  }}
                />
              </div>
            ),
          },
          {
            key: 'vulnerabilities',
            label: (
              <Space size={6}>
                <ShieldAlert className="h-4 w-4" />
                安全漏洞
              </Space>
            ),
            children: (
              <div>
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Space size="large" wrap>
                    <Statistic
                      title="未修复合计"
                      value={summaryQuery.isError ? '—' : (summary?.total ?? 0)}
                      prefix={<ShieldAlert className="mr-2 text-red-500" />}
                    />
                    {summaryItems.map(item => (
                      <Statistic
                        key={item.key}
                        title={item.label}
                        value={
                          summaryQuery.isError
                            ? '—'
                            : (summary?.bySeverity?.[item.key] ?? 0)
                        }
                      />
                    ))}
                  </Space>
                  <div className="text-xs text-gray-500 mt-2">
                    只统计待修复与修复中的漏洞；已修复和已忽略不计入。
                  </div>
                </Card>

                <Form
                  form={findingForm}
                  layout="inline"
                  className="mb-4 flex-wrap gap-2"
                  initialValues={{ status: ['open', 'fixing'], instanceId: instanceIdParam || undefined }}
                >
                  <Form.Item name="status" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="状态"
                      style={{ minWidth: 160 }}
                      options={findingStatusOptions}
                    />
                  </Form.Item>
                  <Form.Item name="severity" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="严重度"
                      style={{ minWidth: 140 }}
                      options={findingSeverityOptions}
                    />
                  </Form.Item>
                  <Form.Item name="vulnType" className="!mb-0">
                    <Select
                      allowClear
                      placeholder="漏洞类型"
                      style={{ minWidth: 160 }}
                      options={vulnTypeOptions}
                    />
                  </Form.Item>
                  <Form.Item name="instanceId" className="!mb-0">
                    <Input allowClear placeholder="ECS 实例 ID" style={{ width: 180 }} />
                  </Form.Item>
                  <Form.Item name="cloudAccountId" className="!mb-0">
                    <Select
                      allowClear
                      showSearch
                      optionFilterProp="label"
                      placeholder="云账号"
                      style={{ minWidth: 180 }}
                      options={accountOptions}
                    />
                  </Form.Item>
                  <Form.Item name="search" className="!mb-0">
                    <Input
                      allowClear
                      placeholder="漏洞名称 / CVE"
                      style={{ width: 180 }}
                      onPressEnter={applyFindingFilters}
                    />
                  </Form.Item>
                  <Form.Item className="!mb-0">
                    <Space>
                      <Button type="primary" icon={<Search />} onClick={applyFindingFilters}>
                        查询
                      </Button>
                      <Button icon={<RotateCcw />} onClick={resetFindingFilters}>
                        重置
                      </Button>
                    </Space>
                  </Form.Item>
                </Form>

                {findingsQuery.isError && (
                  <Alert
                    type="error"
                    showIcon
                    style={{ marginBottom: 16 }}
                    title={findingErrorText}
                    action={
                      <Button size="small" onClick={() => findingsQuery.refetch()}>
                        重试
                      </Button>
                    }
                  />
                )}

                <Table
                  rowKey="id"
                  columns={findingColumns}
                  dataSource={findings}
                  loading={findingsQuery.isLoading}
                  scroll={{ x: 1500 }}
                  pagination={{
                    current: findingPage.current,
                    pageSize: findingPage.pageSize,
                    total: findingTotal,
                    showSizeChanger: true,
                    showTotal: total => `共 ${total} 条漏洞`,
                    onChange: (page, pageSize) => setFindingPage({ current: page, pageSize }),
                  }}
                />
              </div>
            ),
          },
        ]}
      />
    </Card>
  );
}

export default function CloudSignalsPage() {
  return (
    <Suspense fallback={<Card loading />}>
      <CloudSignalsPageContent />
    </Suspense>
  );
}

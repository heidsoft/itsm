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
  firing: { color: 'red', text: 'Triggering' },
  acknowledged: { color: 'gold', text: 'Claimed' },
  resolved: { color: 'green', text: 'Closed' },
};

const findingStatusMeta: Record<string, { color: string; text: string }> = {
  open: { color: 'red', text: 'To be fixed' },
  fixing: { color: 'gold', text: 'Under repair' },
  fixed: { color: 'green', text: 'Fixed' },
  ignored: { color: 'default', text: 'ignored' },
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
  critical: 'Critical',
  high: 'High risk',
  medium: 'Medium risk',
  low: 'Low risk',
  unknown: 'Unknown',
};

const vulnTypeText: Record<string, string> = {
  cve: 'Linux software vulnerability',
  sys: 'Windows system vulnerability',
  cms: 'Web-CMS Vulnerability',
  app: 'Application vulnerability',
  emg: 'Emergency vulnerability',
};

const syncStateMeta: Record<string, { color: string; text: string }> = {
  unconfigured: { color: 'default', text: 'Not configured' },
  unavailable: { color: 'red', text: 'Not available' },
  ready: { color: 'green', text: 'Ready' },
};

const signalKindText: Record<string, string> = {
  alarm: 'Cloud monitoring alarm',
  vulnerability: 'Security Vulnerability',
};

const alertStatusOptions = [
  { value: 'firing', label: 'Triggering' },
  { value: 'acknowledged', label: 'Claimed' },
  { value: 'resolved', label: 'Closed' },
];

const alertSeverityOptions = [
  { value: 'P1', label: 'P1' },
  { value: 'P2', label: 'P2' },
  { value: 'P3', label: 'P3' },
  { value: 'P4', label: 'P4' },
];

const findingStatusOptions = [
  { value: 'open', label: 'To be fixed' },
  { value: 'fixing', label: 'Under repair' },
  { value: 'fixed', label: 'Fixed' },
  { value: 'ignored', label: 'ignored' },
];

const findingSeverityOptions = [
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High risk' },
  { value: 'medium', label: 'Medium risk' },
  { value: 'low', label: 'Low risk' },
  { value: 'unknown', label: 'Unknown' },
];

const vulnTypeOptions = Object.entries(vulnTypeText).map(([value, label]) => ({ value, label }));

function formatTime(value?: string) {
  return value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-';
}

function bridgeTag(record: CloudAlert) {
  const status = record.annotations?.bridgeStatus;
  if (status === 'bridged') {
    const incidentId = record.annotations?.incidentId;
    return <Tag color="green">Order has been created{incidentId ? ` #${incidentId}` : ''}</Tag>;
  }
  if (status === 'failed') return <Tag color="red">Order creation failed</Tag>;
  if (status === 'claimed') return <Tag color="gold">Order is being created</Tag>;
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
      onSuccess: () => message.success(`Alarm claimed "${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, 'Failed to claim alarm')),
    });
  };

  const handleResolve = (record: CloudAlert) => {
    resolveMutation.mutate(record.id, {
      onSuccess: () => message.success(`Alarm closed "${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, 'Failed to close alarm')),
    });
  };

  const handleIgnore = (record: SecurityFinding) => {
    ignoreMutation.mutate(record.id, {
      onSuccess: () => message.success(`Vulnerability ignored"${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, 'Ignore vulnerability failed')),
    });
  };

  const handleReopen = (record: SecurityFinding) => {
    reopenMutation.mutate(record.id, {
      onSuccess: () => message.success(`Vulnerability has been reopened"${record.name}」`),
      onError: error => message.error(serverErrorMessage(error, 'Failed to reopen vulnerability')),
    });
  };

  const alertColumns: ColumnsType<CloudAlert> = [
    {
      title: 'Alarm',
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
      title: 'Level',
      dataIndex: 'severity',
      width: 80,
      render: (value: string) => <Tag color={severityColor[value] || 'default'}>{value || '-'}</Tag>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (value: string) => {
        const meta = alertStatusMeta[value];
        return <Tag color={meta?.color || 'default'}>{meta?.text || value || '-'}</Tag>;
      },
    },
    {
      title: 'Service',
      dataIndex: 'service',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Source IP',
      dataIndex: 'sourceIp',
      width: 140,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Trigger time',
      dataIndex: 'firedAt',
      width: 170,
      render: (value?: string) => formatTime(value),
    },
    {
      title: 'Create an order',
      width: 120,
      render: (_: unknown, record: CloudAlert) => bridgeTag(record),
    },
    {
      title: 'Actions',
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

                Claim
              </Button>
            )}
            <Popconfirm
              title="Close alarm"
              description={`Confirm to close the alarm "${record.name}"? Local shutdown will not affect the status of the cloud monitoring side.`}
              okText="Close"
              cancelText="Cancel"
              onConfirm={() => handleResolve(record)}
            >
              <Button type="link" size="small" danger>

                Close
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const findingColumns: ColumnsType<SecurityFinding> = [
    {
      title: 'Vulnerability',
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
      title: 'Severity',
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
      title: 'Type',
      dataIndex: 'vulnType',
      width: 130,
      render: (value?: string) => (value ? vulnTypeText[value] || value : '-'),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (value: string) => {
        const meta = findingStatusMeta[value];
        return <Tag color={meta?.color || 'default'}>{meta?.text || value || '-'}</Tag>;
      },
    },
    {
      title: 'Instances',
      width: 200,
      render: (_: unknown, record: SecurityFinding) => (
        <div>
          <div>{record.instanceName || '-'}</div>
          <div className="text-xs text-gray-500">{record.instanceId || '-'}</div>
        </div>
      ),
    },
    {
      title: 'Public IP',
      dataIndex: 'internetIp',
      width: 140,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Associated CI',
      width: 160,
      render: (_: unknown, record: SecurityFinding) =>
        record.ciId ? (
          <Button type="link" size="small" onClick={() => router.push(`/cmdb/cis/${record.ciId}`)}>
            {record.ciName || record.ciNumber || `#${record.ciId}`}
          </Button>
        ) : (
          <span className="text-gray-400">Not associated</span>
        ),
    },
    {
      title: 'Recently discovered',
      width: 160,
      render: (_: unknown, record: SecurityFinding) => formatTime(record.lastSeenAt || record.lastSyncedAt),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      render: (_: unknown, record: SecurityFinding) => {
        if (!canWrite) return <span className="text-gray-400">-</span>;
        const active = record.status === 'open' || record.status === 'fixing';
        if (active) {
          return (
            <Popconfirm
              title="Ignore vulnerability"
              description={`Confirm ignore vulnerability "${record.name}"? After being ignored, it will no longer be counted in the unrepaired statistics.`}
              okText="ignore"
              cancelText="Cancel"
              onConfirm={() => handleIgnore(record)}
            >
              <Button type="link" size="small" danger loading={ignoreMutation.isPending}>

                ignore
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

            Reopen
          </Button>
        );
      },
    },
  ];

  const syncColumns: ColumnsType<CloudSignalSyncState> = [
    {
      title: 'Cloud account',
      width: 200,
      render: (_: unknown, record: CloudSignalSyncState) => (
        <div>
          <div>{record.accountName || `#${record.cloudAccountId}`}</div>
          <div className="text-xs text-gray-500">{record.provider}</div>
        </div>
      ),
    },
    {
      title: 'signal',
      width: 130,
      render: (_: unknown, record: CloudSignalSyncState) =>
        signalKindText[record.signalKind] || record.signalKind,
    },
    {
      title: 'Status',
      width: 100,
      render: (_: unknown, record: CloudSignalSyncState) => {
        const meta = syncStateMeta[record.state];
        return <Tag color={meta?.color || 'default'}>{meta?.text || record.state}</Tag>;
      },
    },
    {
      title: 'Recent success',
      width: 170,
      render: (_: unknown, record: CloudSignalSyncState) => formatTime(record.lastSuccessAt),
    },
    {
      title: 'Number of items in this batch',
      dataIndex: 'itemsSynced',
      width: 100,
    },
    {
      title: 'Continuous failures',
      dataIndex: 'consecutiveFailures',
      width: 100,
      render: (value: number) => (value > 0 ? <Tag color="red">{value}</Tag> : 0),
    },
    {
      title: 'Recent error',
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
    { key: 'critical', label: 'Critical' },
    { key: 'high', label: 'High risk' },
    { key: 'medium', label: 'Medium risk' },
    { key: 'low', label: 'Low risk' },
    { key: 'unknown', label: 'Unknown' },
  ];

  const alertErrorText = serverErrorMessage(alertsQuery.error, 'Failed to load cloud monitoring alarm, please try again');
  const findingErrorText = serverErrorMessage(findingsQuery.error, 'Failed to load security vulnerability, please try again');
  const syncErrorText = serverErrorMessage(syncQuery.error, 'Failed to load sync status, please try again');

  return (
    <Card>
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Cloud Security and Alerting</h1>
          <p className="text-gray-500 mt-1">

            Cloud monitoring alarms and cloud security center vulnerabilities are synchronized regularly according to the cloud account; P1/P2 alarms are automatically created by the backend.
          </p>
        </div>
        <Space>
          <Button icon={<RefreshCw />} loading={refreshing} onClick={refreshAll}>

            Refresh
          </Button>
          <Button onClick={() => router.push('/cmdb/cloud-resources')}>Cloud Resources</Button>
        </Space>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: 'Home page' },
          { title: 'Configuration Management' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: 'Cloud Security and Alerting' },
        ]}
      />

      {syncQuery.isError ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          title={syncErrorText}
          description="When the synchronization status cannot be obtained, it is impossible to determine whether the alarms and vulnerabilities are the latest data."
          action={
            <Button size="small" onClick={() => syncQuery.refetch()}>

              Retry
            </Button>
          }
        />
      ) : syncStates.length === 0 && !syncQuery.isPending ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="There is no cloud signal synchronization record yet"
          description="Synchronization is performed according to the cloud account: first configure the secret:// or env:// credential reference in the cloud account, and the worker will write the status here after the next round of synchronization."
        />
      ) : hasUnavailable ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="Some cloud accounts failed to synchronize"
          description={syncProblemStates
            .filter(state => state.state === 'unavailable')
            .map(
              state =>
                `${state.accountName || `#${state.cloudAccountId}`}（${signalKindText[state.signalKind] || state.signalKind}）：${state.lastErrorMessage || 'Unknown error'}`,
            )
            .join('；')}
        />
      ) : hasUnconfigured ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="Some cloud accounts have not been configured with synchronization credentials."
          description="Unconfigured accounts will not pull alerts and vulnerabilities; other accounts have been synchronized as scheduled."
        />
      ) : (
        <Alert
          type="success"
          showIcon
          style={{ marginBottom: 16 }}
          title="Cloud signal synchronization is normal"
          description={`Latest round:${formatTime(
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
        <Card size="small" title="Sync Status" style={{ marginBottom: 16 }}>
          <Table
            rowKey={record => `${record.cloudAccountId}-${record.signalKind}`}
            size="small"
            columns={syncColumns}
            dataSource={syncStates}
            loading={syncQuery.isPending}
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

                Cloud monitoring alarm
              </Space>
            ),
            children: (
              <div>
                <Form form={alertForm} layout="inline" className="mb-4 flex-wrap gap-2">
                  <Form.Item name="status" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="Status"
                      style={{ minWidth: 160 }}
                      options={alertStatusOptions}
                    />
                  </Form.Item>
                  <Form.Item name="severity" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="Level"
                      style={{ minWidth: 140 }}
                      options={alertSeverityOptions}
                    />
                  </Form.Item>
                  <Form.Item name="cloudAccountId" className="!mb-0">
                    <Select
                      allowClear
                      showSearch
                      optionFilterProp="label"
                      placeholder="Cloud account"
                      style={{ minWidth: 180 }}
                      options={accountOptions}
                    />
                  </Form.Item>
                  <Form.Item name="search" className="!mb-0">
                    <Input
                      allowClear
                      placeholder="Name/Alarm ID/Description"
                      style={{ width: 220 }}
                      onPressEnter={applyAlertFilters}
                    />
                  </Form.Item>
                  <Form.Item className="!mb-0">
                    <Space>
                      <Button type="primary" icon={<Search />} onClick={applyAlertFilters}>

                        Query
                      </Button>
                      <Button icon={<RotateCcw />} onClick={resetAlertFilters}>

                        Reset
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

                        Retry
                      </Button>
                    }
                  />
                )}

                <Table
                  rowKey="id"
                  columns={alertColumns}
                  dataSource={alerts}
                  loading={alertsQuery.isPending}
                  scroll={{ x: 1200 }}
                  pagination={{
                    current: alertPage.current,
                    pageSize: alertPage.pageSize,
                    total: alertTotal,
                    showSizeChanger: true,
                    showTotal: total => `Total ${total} Alarm`,
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

                Security Vulnerability
              </Space>
            ),
            children: (
              <div>
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Space size="large" wrap>
                    <Statistic
                      title="Unrepaired total"
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

                    Only vulnerabilities that need to be fixed and those in the process of being fixed are counted; fixed and ignored vulnerabilities are not counted.
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
                      placeholder="Status"
                      style={{ minWidth: 160 }}
                      options={findingStatusOptions}
                    />
                  </Form.Item>
                  <Form.Item name="severity" className="!mb-0">
                    <Select
                      mode="multiple"
                      allowClear
                      placeholder="Severity"
                      style={{ minWidth: 140 }}
                      options={findingSeverityOptions}
                    />
                  </Form.Item>
                  <Form.Item name="vulnType" className="!mb-0">
                    <Select
                      allowClear
                      placeholder="Vulnerability type"
                      style={{ minWidth: 160 }}
                      options={vulnTypeOptions}
                    />
                  </Form.Item>
                  <Form.Item name="instanceId" className="!mb-0">
                    <Input allowClear placeholder="ECS instance ID" style={{ width: 180 }} />
                  </Form.Item>
                  <Form.Item name="cloudAccountId" className="!mb-0">
                    <Select
                      allowClear
                      showSearch
                      optionFilterProp="label"
                      placeholder="Cloud account"
                      style={{ minWidth: 180 }}
                      options={accountOptions}
                    />
                  </Form.Item>
                  <Form.Item name="search" className="!mb-0">
                    <Input
                      allowClear
                      placeholder="Vulnerability name / CVE"
                      style={{ width: 180 }}
                      onPressEnter={applyFindingFilters}
                    />
                  </Form.Item>
                  <Form.Item className="!mb-0">
                    <Space>
                      <Button type="primary" icon={<Search />} onClick={applyFindingFilters}>

                        Query
                      </Button>
                      <Button icon={<RotateCcw />} onClick={resetFindingFilters}>

                        Reset
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

                        Retry
                      </Button>
                    }
                  />
                )}

                <Table
                  rowKey="id"
                  columns={findingColumns}
                  dataSource={findings}
                  loading={findingsQuery.isPending}
                  scroll={{ x: 1500 }}
                  pagination={{
                    current: findingPage.current,
                    pageSize: findingPage.pageSize,
                    total: findingTotal,
                    showSizeChanger: true,
                    showTotal: total => `Total ${total} vulnerability`,
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

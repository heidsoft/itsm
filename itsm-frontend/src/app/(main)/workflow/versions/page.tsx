'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { App, Alert, Button, Card, Descriptions, Modal, Select, Space, Table, Tag } from 'antd';
import { Diff, Download, Eye, PlayCircle, RefreshCw, RotateCcw, Trash2 } from 'lucide-react';
import { useSearchParams } from 'next/navigation';

import BPMNDiagramPreview from '@/components/workflow/BPMNDiagramPreview';
import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import {
  WorkflowAPI,
  type BpmnProcessVersion,
  type BpmnVersionChange,
  type BpmnVersionComparison,
} from '@/lib/api/workflow-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { formatDateTime as formatStandardDateTime } from '@/lib/formatters';

// 全站时间统一为 YYYY-MM-DD HH:mm，避免 zh-CN locale 的斜杠格式与其它页并存。
const formatDateTime = (value?: string) => formatStandardDateTime(value) || '-';

const COMPATIBILITY_CONFIG: Record<
  string,
  { text: string; alertType: 'error' | 'warning' | 'success' }
> = {
  identical: { text: 'No difference', alertType: 'success' },
  compatible: { text: 'Backwards Compatibility', alertType: 'success' },
  risky: { text: 'High risk change', alertType: 'warning' },
  incompatible: { text: 'Incompatible', alertType: 'error' },
};

const CHANGE_TYPE_LABEL: Record<string, string> = {
  added: 'New',
  removed: 'Remove',
  modified: 'Modify',
};

const IMPACT_COLOR: Record<string, string> = {
  low: 'default',
  medium: 'blue',
  high: 'orange',
  critical: 'red',
};

// 差异行没有后端 ID，用变更内容本身组合作为 key，避免行错位。
const changeRowKey = (record: BpmnVersionChange) =>
  `${record.type}:${record.elementType}:${record.elementId}:${record.elementName}`;

export default function WorkflowVersionsPage() {
  const { message, modal } = App.useApp();
  const searchParams = useSearchParams();
  const [processKey, setProcessKey] = useState(searchParams.get('workflowId') || '');
  const [processKeyOptions, setProcessKeyOptions] = useState<{ label: string; value: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [versions, setVersions] = useState<BpmnProcessVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<BpmnProcessVersion | null>(null);
  const [comparison, setComparison] = useState<BpmnVersionComparison | null>(null);
  const [comparisonVisible, setComparisonVisible] = useState(false);
  const [comparingId, setComparingId] = useState<string | null>(null);

  // 流程 Key 必须来自真实定义列表；硬编码清单会让新建流程在版本页不可选。
  useEffect(() => {
    let cancelled = false;
    WorkflowAPI.getWorkflows({ pageSize: BACKEND_MAX_PAGE_SIZE })
      .then(result => {
        if (cancelled) return;
        setProcessKeyOptions(
          result.workflows
            .filter(item => Boolean(item.code))
            .map(item => ({ label: `${item.name || item.code}（${item.code}）`, value: item.code }))
        );
      })
      .catch(() => {
        if (!cancelled) setProcessKeyOptions([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const loadVersions = useCallback(async () => {
    if (!processKey) {
      setVersions([]);
      return;
    }

    try {
      setLoading(true);
      setVersions(await WorkflowAPI.listVersions(processKey));
    } catch (error) {
      console.error('Failed to load workflow versions:', error);
      message.error('Failed to load workflow versions');
      setVersions([]);
    } finally {
      setLoading(false);
    }
  }, [processKey, message]);

  useEffect(() => {
    void loadVersions();
  }, [loadVersions]);

  const handleCompare = useCallback(
    async (record: BpmnProcessVersion) => {
      // 后端按 version 倒序返回，首行即最新版本。
      const newest = versions[0];
      if (!newest || newest.version === record.version) {
        message.info('The current version is already the latest version, no need to compare.');
        return;
      }
      // base 必须是较低版本，否则后端给出的“新增/移除”方向会反过来。
      const [baseVersion, targetVersion] =
        newest.version > record.version
          ? [record.version, newest.version]
          : [newest.version, record.version];
      setComparingId(record.id);
      try {
        const result = await WorkflowAPI.compareVersions(
          record.processDefinitionKey,
          baseVersion,
          targetVersion
        );
        if (!result) {
          message.info('The versions are the same, no need to compare.');
          return;
        }
        setComparison(result);
        setComparisonVisible(true);
      } catch {
        message.error('Version comparison failed');
      } finally {
        setComparingId(null);
      }
    },
    [message, versions]
  );

  const columns = useMemo(
    () => [
      {
        title: 'Version',
        dataIndex: 'version',
        key: 'version',
        render: (value: number) => <span className="font-mono text-sm">v{value}</span>,
      },
      {
        title: 'Name',
        dataIndex: 'name',
        key: 'name',
      },
      {
        title: 'Status',
        dataIndex: 'isActive',
        key: 'isActive',
        render: (value: boolean) => (value ? <Tag color="green">Activated</Tag> : <Tag>Inactive</Tag>),
      },
      {
        title: 'Created',
        dataIndex: 'createdAt',
        key: 'createdAt',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: 'Updated At',
        dataIndex: 'updatedAt',
        key: 'updatedAt',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: 'Actions',
        key: 'actions',
        render: (_: unknown, record: BpmnProcessVersion) => (
          <Space>
            <Button
              type="text"
              icon={<Eye className="h-4 w-4" />}
              aria-label={`View version v ${record.version} Details`}
              onClick={() => setSelectedVersion(record)}
            />
            <Button
              type="text"
              icon={<PlayCircle className="h-4 w-4" />}
              aria-label={`Activate version v ${record.version}`}
              disabled={record.isActive}
              onClick={() => {
                modal.confirm({
                  title: 'Activated version',
                  content: `Confirm to activate version v ${record.version} ? Newly initiated process instances after activation will use this version.`,
                  okText: 'Active',
                  cancelText: 'Cancel',
                  onOk: async () => {
                    try {
                      await WorkflowAPI.activateVersion(record.processDefinitionKey, record.version);
                      message.success(`Version v activated ${record.version}`);
                      await loadVersions();
                    } catch {
                      message.error('Failed to activate version');
                    }
                  },
                });
              }}
            />
            <Button
              type="text"
              icon={<RotateCcw className="h-4 w-4" />}
              aria-label={`Rollback to version v ${record.version}`}
              onClick={() => {
                modal.confirm({
                  title: 'Rollback version',
                  content: `Are you sure you want to roll back to version v?${record.version} ? Rolling back replaces the currently active process definition.`,
                  okText: 'Rollback',
                  okType: 'danger',
                  cancelText: 'Cancel',
                  onOk: async () => {
                    try {
                      await WorkflowAPI.rollbackVersion(record.processDefinitionKey, record.version, 'Frontend rollback');
                      message.success(`Rolled back to version v ${record.version}`);
                      await loadVersions();
                    } catch {
                      message.error('Rollback version failed');
                    }
                  },
                });
              }}
            />
            <Button
              type="text"
              icon={<Diff className="h-4 w-4" />}
              aria-label={`Compare version v ${record.version} with the latest version`}
              loading={comparingId === record.id}
              onClick={() => void handleCompare(record)}
            />
            <Button
              type="text"
              danger
              icon={<Trash2 className="h-4 w-4" />}
              aria-label={`Delete version v ${record.version}`}
              disabled={record.isActive}
              onClick={() => {
                modal.confirm({
                  title: 'Delete version',
                  content: `Are you sure you want to delete version v?${record.version}（${record.name})? It cannot be recovered after deletion.`,
                  okText: 'Delete',
                  okType: 'danger',
                  cancelText: 'Cancel',
                  onOk: async () => {
                    try {
                      await WorkflowAPI.deleteVersion(record.processDefinitionKey, record.version);
                      message.success(`Version v deleted ${record.version}`);
                      await loadVersions();
                    } catch {
                      message.error('Failed to delete version');
                    }
                  },
                });
              }}
            />
          </Space>
        ),
      },
    ],
    [comparingId, handleCompare, loadVersions, message, modal]
  );

  const activeCount = versions.filter(item => item.isActive).length;

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="Workflow version"
        description="View the historical version and currently activated version of the process, supporting version activation, rollback and comparison."
        notice={
          <ManagementNotice
            message="This version"
            description="Supports version list, process graphical preview, activation, rollback and difference comparison. Import, export and version description editing will be added in subsequent versions."
          />
        }
      />

      <StatsOverview
        items={[
          { key: 'count', title: 'Total version number', value: versions.length, accentColor: '#1677ff' },
          { key: 'active', title: 'Activated version', value: activeCount, accentColor: '#52c41a' },
          {
            key: 'inactive',
            title: 'Inactive',
            value: versions.length - activeCount,
            accentColor: '#faad14',
          },
        ]}
        columns={{ xs: 24, sm: 12, lg: 8 }}
      />

      <FilterToolbarCard
        filters={
          <Select
            value={processKey || undefined}
            placeholder="Select Process"
            showSearch
            allowClear
            optionFilterProp="label"
            style={{ width: 320 }}
            onChange={value => setProcessKey(value || '')}
            options={processKeyOptions}
            notFoundContent={processKeyOptions.length ? undefined : 'No process definition yet'}
          />
        }
        actions={
          <>
            <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadVersions}>

              Refresh
            </Button>
            <Button icon={<Download className="h-4 w-4" />} disabled>

              Export version
            </Button>
          </>
        }
      />

      <Card className="rounded-xl shadow-sm">
        <LoadingEmptyError
          state={loading ? 'loading' : versions.length === 0 ? 'empty' : 'success'}
          loadingText="Loading workflow version..."
          empty={{
            title: processKey ? 'Version data not found' : 'Please select the process first',
            description: processKey
              ? 'There is no displayable version record for the current process.'
              : 'The version page needs to determine a process first.',
            actionText: processKey ? 'Reload' : undefined,
            onAction: processKey ? loadVersions : undefined,
            showAction: Boolean(processKey),
          }}
        >
          <Table columns={columns} dataSource={versions} rowKey="id" pagination={{ pageSize: 10 }} />
        </LoadingEmptyError>
      </Card>

      <Modal
        title={selectedVersion ? `Version details · v ${selectedVersion.version}` : 'Version details'}
        open={!!selectedVersion}
        onCancel={() => setSelectedVersion(null)}
        footer={null}
        width={860}
        destroyOnHidden
      >
        {selectedVersion && (
          <div className="space-y-4">
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Process Key" span={2}>
                {selectedVersion.processDefinitionKey}
              </Descriptions.Item>
              <Descriptions.Item label="Version">v{selectedVersion.version}</Descriptions.Item>
              <Descriptions.Item label="Status">
                <Tag color={selectedVersion.isActive ? 'green' : 'default'}>
                  {selectedVersion.isActive ? 'Activated' : 'Inactive'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Name" span={2}>
                {selectedVersion.name}
              </Descriptions.Item>
              <Descriptions.Item label="Description" span={2}>
                {selectedVersion.description || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="Created">
                {formatDateTime(selectedVersion.createdAt)}
              </Descriptions.Item>
              <Descriptions.Item label="Updated At">
                {formatDateTime(selectedVersion.updatedAt)}
              </Descriptions.Item>
            </Descriptions>

            <div>
              <div className="mb-2 text-sm font-medium">Process preview</div>
              <BPMNDiagramPreview xml={selectedVersion.bpmnXml} height={380} />
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={
          comparison
            ? `Version comparison · v ${comparison.baseVersion?.version ?? '-'} → v${comparison.targetVersion?.version ?? '-'}`
            : 'Version comparison results'
        }
        open={comparisonVisible}
        onCancel={() => setComparisonVisible(false)}
        footer={null}
        width={860}
        destroyOnHidden
      >
        {comparison && (
          <div className="space-y-4">
            <Alert
              type={COMPATIBILITY_CONFIG[comparison.compatibility]?.alertType || 'warning'}
              showIcon
              title={
                COMPATIBILITY_CONFIG[comparison.compatibility]?.text ||
                `Compatibility:${comparison.compatibility || 'Unknown'}`
              }
              description={
                comparison.breakingChanges.length > 0
                  ? `exists ${comparison.breakingChanges.length} A destructive change may affect running process instances.`
                  : 'No breaking changes found.'
              }
            />

            {comparison.breakingChanges.length > 0 && (
              <div>
                <div className="mb-2 text-sm font-medium">Destructive change</div>
                <ul className="list-disc space-y-1 pl-5 text-sm">
                  {comparison.breakingChanges.map((item, index) => (
                    <li key={`${item}-${index}`}>{item}</li>
                  ))}
                </ul>
              </div>
            )}

            <div>
              <div className="mb-2 text-sm font-medium">Change details</div>
              <Table<BpmnVersionChange>
                size="small"
                rowKey={changeRowKey}
                dataSource={comparison.changes}
                pagination={false}
                locale={{ emptyText: 'No structural changes detected between the two versions.' }}
                columns={[
                  {
                    title: 'Type',
                    dataIndex: 'type',
                    width: 80,
                    render: (value: string) => CHANGE_TYPE_LABEL[value] || value,
                  },
                  { title: 'Element type', dataIndex: 'elementType', width: 120 },
                  {
                    title: 'Element',
                    dataIndex: 'elementName',
                    render: (value: string, record) => (
                      <span>
                        {value || '-'}
                        <span className="ml-1 font-mono text-xs text-gray-400">
                          {record.elementId}
                        </span>
                      </span>
                    ),
                  },
                  { title: 'Description', dataIndex: 'description', ellipsis: true },
                  {
                    title: 'Impact',
                    dataIndex: 'impact',
                    width: 90,
                    render: (value: string) => (
                      <Tag color={IMPACT_COLOR[value] || 'default'}>{value || '-'}</Tag>
                    ),
                  },
                ]}
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

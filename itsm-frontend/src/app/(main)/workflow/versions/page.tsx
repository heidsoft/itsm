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
  identical: { text: '无差异', alertType: 'success' },
  compatible: { text: '向后兼容', alertType: 'success' },
  risky: { text: '高风险变更', alertType: 'warning' },
  incompatible: { text: '不兼容', alertType: 'error' },
};

const CHANGE_TYPE_LABEL: Record<string, string> = {
  added: '新增',
  removed: '移除',
  modified: '修改',
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
      message.error('加载工作流版本失败');
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
        message.info('当前版本已经是最新版本，无需比较。');
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
          message.info('版本相同，无需比较。');
          return;
        }
        setComparison(result);
        setComparisonVisible(true);
      } catch {
        message.error('版本比较失败');
      } finally {
        setComparingId(null);
      }
    },
    [message, versions]
  );

  const columns = useMemo(
    () => [
      {
        title: '版本',
        dataIndex: 'version',
        key: 'version',
        render: (value: number) => <span className="font-mono text-sm">v{value}</span>,
      },
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
      },
      {
        title: '状态',
        dataIndex: 'isActive',
        key: 'isActive',
        render: (value: boolean) => (value ? <Tag color="green">已激活</Tag> : <Tag>未激活</Tag>),
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        key: 'createdAt',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '更新时间',
        dataIndex: 'updatedAt',
        key: 'updatedAt',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        render: (_: unknown, record: BpmnProcessVersion) => (
          <Space>
            <Button
              type="text"
              icon={<Eye className="h-4 w-4" />}
              aria-label={`查看版本 v${record.version} 详情`}
              onClick={() => setSelectedVersion(record)}
            />
            <Button
              type="text"
              icon={<PlayCircle className="h-4 w-4" />}
              aria-label={`激活版本 v${record.version}`}
              disabled={record.isActive}
              onClick={() => {
                modal.confirm({
                  title: '激活版本',
                  content: `确定要激活版本 v${record.version} 吗？激活后新发起的流程实例将使用该版本。`,
                  okText: '激活',
                  cancelText: '取消',
                  onOk: async () => {
                    try {
                      await WorkflowAPI.activateVersion(record.processDefinitionKey, record.version);
                      message.success(`已激活版本 v${record.version}`);
                      await loadVersions();
                    } catch {
                      message.error('激活版本失败');
                    }
                  },
                });
              }}
            />
            <Button
              type="text"
              icon={<RotateCcw className="h-4 w-4" />}
              aria-label={`回滚到版本 v${record.version}`}
              onClick={() => {
                modal.confirm({
                  title: '回滚版本',
                  content: `确定要回滚到版本 v${record.version} 吗？回滚将替换当前生效的流程定义。`,
                  okText: '回滚',
                  okType: 'danger',
                  cancelText: '取消',
                  onOk: async () => {
                    try {
                      await WorkflowAPI.rollbackVersion(record.processDefinitionKey, record.version, '前端回滚');
                      message.success(`已回滚到版本 v${record.version}`);
                      await loadVersions();
                    } catch {
                      message.error('回滚版本失败');
                    }
                  },
                });
              }}
            />
            <Button
              type="text"
              icon={<Diff className="h-4 w-4" />}
              aria-label={`比较版本 v${record.version} 与最新版本`}
              loading={comparingId === record.id}
              onClick={() => void handleCompare(record)}
            />
            <Button
              type="text"
              danger
              icon={<Trash2 className="h-4 w-4" />}
              aria-label={`删除版本 v${record.version}`}
              disabled={record.isActive}
              onClick={() => {
                modal.confirm({
                  title: '删除版本',
                  content: `确定要删除版本 v${record.version}（${record.name}）吗？删除后不可恢复。`,
                  okText: '删除',
                  okType: 'danger',
                  cancelText: '取消',
                  onOk: async () => {
                    try {
                      await WorkflowAPI.deleteVersion(record.processDefinitionKey, record.version);
                      message.success(`已删除版本 v${record.version}`);
                      await loadVersions();
                    } catch {
                      message.error('删除版本失败');
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
        title="工作流版本"
        description="查看流程的历史版本和当前激活版本，支持版本激活、回滚和比较。"
        notice={
          <ManagementNotice
            message="本版本"
            description="支持版本列表、流程图形化预览、激活、回滚与差异比较。导入导出和版本说明编辑将在后续版本补充。"
          />
        }
      />

      <StatsOverview
        items={[
          { key: 'count', title: '总版本数', value: versions.length, accentColor: '#1677ff' },
          { key: 'active', title: '激活版本', value: activeCount, accentColor: '#52c41a' },
          {
            key: 'inactive',
            title: '未激活',
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
            placeholder="选择流程"
            showSearch
            allowClear
            optionFilterProp="label"
            style={{ width: 320 }}
            onChange={value => setProcessKey(value || '')}
            options={processKeyOptions}
            notFoundContent={processKeyOptions.length ? undefined : '暂无流程定义'}
          />
        }
        actions={
          <>
            <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadVersions}>
              刷新
            </Button>
            <Button icon={<Download className="h-4 w-4" />} disabled>
              导出版本
            </Button>
          </>
        }
      />

      <Card className="rounded-xl shadow-sm">
        <LoadingEmptyError
          state={loading ? 'loading' : versions.length === 0 ? 'empty' : 'success'}
          loadingText="正在加载工作流版本..."
          empty={{
            title: processKey ? '未找到版本数据' : '请先选择流程',
            description: processKey
              ? '当前流程还没有可显示的版本记录。'
              : '版本页需要先确定一个流程。',
            actionText: processKey ? '重新加载' : undefined,
            onAction: processKey ? loadVersions : undefined,
            showAction: Boolean(processKey),
          }}
        >
          <Table columns={columns} dataSource={versions} rowKey="id" pagination={{ pageSize: 10 }} />
        </LoadingEmptyError>
      </Card>

      <Modal
        title={selectedVersion ? `版本详情 · v${selectedVersion.version}` : '版本详情'}
        open={!!selectedVersion}
        onCancel={() => setSelectedVersion(null)}
        footer={null}
        width={860}
        destroyOnHidden
      >
        {selectedVersion && (
          <div className="space-y-4">
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="流程 Key" span={2}>
                {selectedVersion.processDefinitionKey}
              </Descriptions.Item>
              <Descriptions.Item label="版本">v{selectedVersion.version}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={selectedVersion.isActive ? 'green' : 'default'}>
                  {selectedVersion.isActive ? '已激活' : '未激活'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="名称" span={2}>
                {selectedVersion.name}
              </Descriptions.Item>
              <Descriptions.Item label="说明" span={2}>
                {selectedVersion.description || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {formatDateTime(selectedVersion.createdAt)}
              </Descriptions.Item>
              <Descriptions.Item label="更新时间">
                {formatDateTime(selectedVersion.updatedAt)}
              </Descriptions.Item>
            </Descriptions>

            <div>
              <div className="mb-2 text-sm font-medium">流程预览</div>
              <BPMNDiagramPreview xml={selectedVersion.bpmnXml} height={380} />
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={
          comparison
            ? `版本比较 · v${comparison.baseVersion?.version ?? '-'} → v${comparison.targetVersion?.version ?? '-'}`
            : '版本比较结果'
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
                `兼容性：${comparison.compatibility || '未知'}`
              }
              description={
                comparison.breakingChanges.length > 0
                  ? `存在 ${comparison.breakingChanges.length} 项破坏性变更，运行中的流程实例可能受影响。`
                  : '未发现破坏性变更。'
              }
            />

            {comparison.breakingChanges.length > 0 && (
              <div>
                <div className="mb-2 text-sm font-medium">破坏性变更</div>
                <ul className="list-disc space-y-1 pl-5 text-sm">
                  {comparison.breakingChanges.map((item, index) => (
                    <li key={`${item}-${index}`}>{item}</li>
                  ))}
                </ul>
              </div>
            )}

            <div>
              <div className="mb-2 text-sm font-medium">变更明细</div>
              <Table<BpmnVersionChange>
                size="small"
                rowKey={changeRowKey}
                dataSource={comparison.changes}
                pagination={false}
                locale={{ emptyText: '两个版本之间没有检测到结构变更。' }}
                columns={[
                  {
                    title: '类型',
                    dataIndex: 'type',
                    width: 80,
                    render: (value: string) => CHANGE_TYPE_LABEL[value] || value,
                  },
                  { title: '元素类型', dataIndex: 'elementType', width: 120 },
                  {
                    title: '元素',
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
                  { title: '说明', dataIndex: 'description', ellipsis: true },
                  {
                    title: '影响',
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

'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Descriptions, Modal, Select, Space, Table, Tag } from 'antd';
import { Diff, Download, Eye, PlayCircle, RefreshCw, RotateCcw, Trash2 } from 'lucide-react';
import { useSearchParams } from 'next/navigation';

import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import { WorkflowAPI } from '@/lib/api/workflow-api';

type VersionRow = {
  id: string;
  processKey: string;
  name: string;
  version: number;
  status: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
};

const formatDateTime = (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-');

// 工作流版本状态映射
const workflowVersionStatusMap: Record<string, { text: string; color: string }> = {
  active: { text: '已激活', color: 'green' },
  draft: { text: '草稿', color: 'default' },
  inactive: { text: '未激活', color: 'default' },
};

const getVersionStatusText = (status: string): string => {
  return workflowVersionStatusMap[status]?.text || status;
};

export default function WorkflowVersionsPage() {
  const { message } = App.useApp();
  const searchParams = useSearchParams();
  const initialProcessKey = searchParams.get('workflowId') || '';
  const [processKey, setProcessKey] = useState(initialProcessKey);
  const [loading, setLoading] = useState(false);
  const [versions, setVersions] = useState<VersionRow[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<VersionRow | null>(null);
  const [comparisonVisible, setComparisonVisible] = useState(false);
  const [comparisonText, setComparisonText] = useState('');

  const loadVersions = async () => {
    if (!processKey) {
      setVersions([]);
      return;
    }

    try {
      setLoading(true);
      const response = await WorkflowAPI.getWorkflowVersions(processKey);
      setVersions(
        response.map(item => ({
          id: item.id,
          processKey: item.code,
          name: item.name,
          version: Number(item.version || 1),
          status: String(item.status),
          description: item.description,
          createdAt:
            item.createdAt instanceof Date ? item.createdAt.toISOString() : new Date().toISOString(),
          updatedAt:
            item.updatedAt instanceof Date ? item.updatedAt.toISOString() : new Date().toISOString(),
        }))
      );
    } catch (error) {
      console.error('Failed to load workflow versions:', error);
      message.error('加载工作流版本失败');
      setVersions([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVersions();
  }, [processKey]);

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
        dataIndex: 'status',
        key: 'status',
        render: (value: string) => {
          const config = workflowVersionStatusMap[value] || { text: value, color: 'default' };
          return <Tag color={config.color}>{config.text}</Tag>;
        },
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        key: 'createdAt',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        render: (_: unknown, record: VersionRow) => (
          <Space>
            <Button type="text" icon={<Eye className="h-4 w-4" />} onClick={() => setSelectedVersion(record)} />
            <Button
              type="text"
              icon={<PlayCircle className="h-4 w-4" />}
              onClick={async () => {
                await WorkflowAPI.activateVersion(record.processKey, record.version);
                message.success(`已激活版本 v${record.version}`);
                loadVersions();
              }}
            />
            <Button
              type="text"
              icon={<RotateCcw className="h-4 w-4" />}
              onClick={async () => {
                await WorkflowAPI.rollbackVersion(record.processKey, record.version, '前端回滚');
                message.success(`已回滚到版本 v${record.version}`);
                loadVersions();
              }}
            />
            <Button
              type="text"
              icon={<Diff className="h-4 w-4" />}
              onClick={async () => {
                const latestVersion = versions[0];
                if (!latestVersion || latestVersion.version === record.version) {
                  setComparisonText('当前版本已经是最新版本，无需比较。');
                  setComparisonVisible(true);
                  return;
                }
                const result = await WorkflowAPI.compareVersions(
                  record.processKey,
                  latestVersion.version,
                  record.version
                );
                setComparisonText(JSON.stringify(result, null, 2));
                setComparisonVisible(true);
              }}
            />
            <Button
              type="text"
              danger
              icon={<Trash2 className="h-4 w-4" />}
              onClick={async () => {
                await WorkflowAPI.deleteVersion(record.processKey, record.version);
                message.success(`已删除版本 v${record.version}`);
                loadVersions();
              }}
            />
          </Space>
        ),
      },
    ],
    [message, versions]
  );

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="工作流版本"
        description="按流程 Key 查看历史版本、激活版本和回滚操作，先恢复版本管理入口可用性。"
        notice={
          <ManagementNotice
            message="版本页已恢复"
            description="当前版本页先聚焦列表、详情、激活、回滚和比较能力，后续继续补充导入导出和版本说明编辑。"
          />
        }
      />

      <StatsOverview
        items={[
          { key: 'count', title: '总版本数', value: versions.length, accentColor: '#1677ff' },
          {
            key: 'active',
            title: '激活版本',
            value: versions.filter(item => item.status === 'active').length,
            accentColor: '#52c41a',
          },
          {
            key: 'draft',
            title: '草稿/其他',
            value: versions.filter(item => item.status !== 'active').length,
            accentColor: '#faad14',
          },
        ]}
        columns={{ xs: 24, sm: 12, lg: 8 }}
      />

      <FilterToolbarCard
        filters={
          <Select
            value={processKey || undefined}
            placeholder="输入或选择流程 Key"
            showSearch
            allowClear
            style={{ width: 260 }}
            onChange={value => setProcessKey(value || '')}
            options={[
              { label: 'ticket_approval', value: 'ticket_approval' },
              { label: 'incident_process', value: 'incident_process' },
              { label: 'change_process', value: 'change_process' },
            ]}
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
            title: processKey ? '未找到版本数据' : '请先选择流程 Key',
            description: processKey ? '当前流程还没有可显示的版本记录。' : '版本页需要先确定一个流程定义 Key。',
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
        destroyOnHidden
      >
        {selectedVersion && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="流程 Key">{selectedVersion.processKey}</Descriptions.Item>
            <Descriptions.Item label="版本">{selectedVersion.version}</Descriptions.Item>
            <Descriptions.Item label="名称">{selectedVersion.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={workflowVersionStatusMap[selectedVersion.status]?.color || 'default'}>
                {getVersionStatusText(selectedVersion.status)}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="说明">{selectedVersion.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(selectedVersion.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(selectedVersion.updatedAt)}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      <Modal
        title="版本比较结果"
        open={comparisonVisible}
        onCancel={() => setComparisonVisible(false)}
        footer={null}
        width={760}
        destroyOnHidden
      >
        <pre className="max-h-[420px] overflow-auto rounded bg-slate-950 p-4 text-xs text-slate-100">
          {comparisonText}
        </pre>
      </Modal>
    </div>
  );
}

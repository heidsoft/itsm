'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Descriptions, Input, Modal, Select, Space, Table, Tag } from 'antd';
import { Eye, PauseCircle, PlayCircle, RefreshCw, StopCircle } from 'lucide-react';

import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import { WorkflowAPI } from '@/lib/api/workflow-api';

type InstanceRow = {
  id: string;
  businessKey: string;
  processDefinitionKey: string;
  status: string;
  startTime?: string;
  endTime?: string;
};

const formatDateTime = (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-');

const statusColorMap: Record<string, string> = {
  running: 'green',
  completed: 'blue',
  suspended: 'orange',
  terminated: 'red',
};

export default function WorkflowInstancesPage() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [instances, setInstances] = useState<InstanceRow[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    running: 0,
    completed: 0,
    suspended: 0,
    terminated: 0,
  });
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<string | undefined>();
  const [selectedInstance, setSelectedInstance] = useState<InstanceRow | null>(null);

  const loadData = async () => {
    try {
      setLoading(true);
      const [instanceResponse, statsResponse] = await Promise.all([
        WorkflowAPI.getInstances({
          workflowId: keyword || undefined,
          status,
          page: 1,
          pageSize: 50,
        }),
        WorkflowAPI.getInstanceStats({
          process_definition_key: keyword || undefined,
          status,
        }),
      ]);

      const rows = (instanceResponse.instances || []).map(instance => ({
        id: instance.id,
        businessKey:
          (instance as unknown as Record<string, string>).business_key ||
          instance.workflowId ||
          '-',
        processDefinitionKey: instance.workflowId || '-',
        status: String(instance.status),
        startTime:
          instance.startTime instanceof Date
            ? instance.startTime.toISOString()
            : String(instance.startTime || ''),
        endTime:
          instance.endTime instanceof Date ? instance.endTime.toISOString() : String(instance.endTime || ''),
      }));

      setInstances(rows);
      setStats(statsResponse);
    } catch (error) {
      console.error('Failed to load workflow instances:', error);
      message.error('加载工作流实例失败');
      setInstances([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [keyword, status]);

  const columns = useMemo(
    () => [
      {
        title: '实例 ID',
        dataIndex: 'id',
        key: 'id',
        render: (value: string) => <span className="font-mono text-sm">{value}</span>,
      },
      {
        title: '流程 Key',
        dataIndex: 'processDefinitionKey',
        key: 'processDefinitionKey',
      },
      {
        title: '业务键',
        dataIndex: 'businessKey',
        key: 'businessKey',
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{value}</Tag>,
      },
      {
        title: '启动时间',
        dataIndex: 'startTime',
        key: 'startTime',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '结束时间',
        dataIndex: 'endTime',
        key: 'endTime',
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        render: (_: unknown, record: InstanceRow) => (
          <Space>
            <Button type="text" icon={<Eye className="h-4 w-4" />} onClick={() => setSelectedInstance(record)} />
            {record.status === 'running' && (
              <>
                <Button
                  type="text"
                  icon={<PauseCircle className="h-4 w-4" />}
                  onClick={async () => {
                    await WorkflowAPI.suspendWorkflow(record.id);
                    message.success('实例已暂停');
                    loadData();
                  }}
                />
                <Button
                  type="text"
                  danger
                  icon={<StopCircle className="h-4 w-4" />}
                  onClick={async () => {
                    await WorkflowAPI.terminateWorkflow(record.id, '前端终止');
                    message.success('实例已终止');
                    loadData();
                  }}
                />
              </>
            )}
            {record.status === 'suspended' && (
              <Button
                type="text"
                icon={<PlayCircle className="h-4 w-4" />}
                onClick={async () => {
                  await WorkflowAPI.resumeWorkflow(record.id);
                  message.success('实例已恢复');
                  loadData();
                }}
              />
            )}
          </Space>
        ),
      },
    ],
    [message]
  );

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="工作流实例"
        description="统一查看流程实例状态、生命周期和关键时间点，先保证实例页可用和可筛选。"
        notice={
          <ManagementNotice
            message="实例页已恢复"
            description="这一版先收口列表、状态操作和详情查看，后续会继续补充任务列表与会签视图。"
          />
        }
      />

      <StatsOverview
        items={[
          { key: 'total', title: '总实例', value: stats.total, accentColor: '#1677ff' },
          { key: 'running', title: '运行中', value: stats.running, accentColor: '#52c41a' },
          { key: 'completed', title: '已完成', value: stats.completed, accentColor: '#1677ff' },
          { key: 'suspended', title: '已暂停', value: stats.suspended, accentColor: '#faad14' },
        ]}
      />

      <FilterToolbarCard
        filters={
          <>
            <Input
              placeholder="按流程 Key 搜索"
              value={keyword}
              allowClear
              onChange={event => setKeyword(event.target.value)}
            />
            <Select
              placeholder="状态筛选"
              allowClear
              value={status}
              onChange={setStatus}
              style={{ width: 180 }}
              options={[
                { label: '运行中', value: 'running' },
                { label: '已完成', value: 'completed' },
                { label: '已暂停', value: 'suspended' },
                { label: '已终止', value: 'terminated' },
              ]}
            />
          </>
        }
        actions={
          <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadData}>
            刷新
          </Button>
        }
      />

      <Card className="rounded-xl shadow-sm">
        <LoadingEmptyError
          state={loading ? 'loading' : instances.length === 0 ? 'empty' : 'success'}
          loadingText="正在加载工作流实例..."
          empty={{
            title: '暂无工作流实例',
            description: '当前筛选条件下没有实例数据。',
            actionText: '重新加载',
            onAction: loadData,
          }}
        >
          <Table columns={columns} dataSource={instances} rowKey="id" pagination={{ pageSize: 10 }} />
        </LoadingEmptyError>
      </Card>

      <Modal
        title={selectedInstance ? `实例详情 · ${selectedInstance.id}` : '实例详情'}
        open={!!selectedInstance}
        onCancel={() => setSelectedInstance(null)}
        footer={null}
        destroyOnHidden
      >
        {selectedInstance && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="实例 ID">{selectedInstance.id}</Descriptions.Item>
            <Descriptions.Item label="流程 Key">{selectedInstance.processDefinitionKey}</Descriptions.Item>
            <Descriptions.Item label="业务键">{selectedInstance.businessKey}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColorMap[selectedInstance.status] || 'default'}>
                {selectedInstance.status}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="启动时间">{formatDateTime(selectedInstance.startTime)}</Descriptions.Item>
            <Descriptions.Item label="结束时间">{formatDateTime(selectedInstance.endTime)}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}

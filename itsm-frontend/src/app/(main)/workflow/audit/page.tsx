'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  DatePicker,
  Space,
  Tag,
  Modal,
  Spin,
  Descriptions,
  Timeline,
  App,
} from 'antd';
import {
  Search,
  RefreshCw,
  Eye,
  Filter,
  Clock,
  User,
  Activity,
} from 'lucide-react';

import type {
  ProcessAuditLog,
  QueryAuditLogsRequest,
} from '@/lib/api/bpmn-dashboard-api';
import BPMNDashboardApi from '@/lib/api/bpmn-dashboard-api';
import { formatDateTime } from '@/lib/formatters';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

export default function AuditLogsPage() {
  const { t } = useI18n();
  const { message } = App.useApp();
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空列表。
  const [listLoading, setListLoading] = useState(true);
  const [logs, setLogs] = useState<ProcessAuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedLog, setSelectedLog] = useState<ProcessAuditLog | null>(null);
  const [timelineVisible, setTimelineVisible] = useState(false);
  const [timeline, setTimeline] = useState<ProcessAuditLog[]>([]);
  const [timelineLoading, setTimelineLoading] = useState(false);

  // 筛选条件
  const [filters, setFilters] = useState<QueryAuditLogsRequest>({
    page: 1,
    pageSize: 20,
  });

  const fetchLogs = async () => {
    setListLoading(true);
    try {
      // 租户由后端认证上下文收敛；前端不发送 tenantId 去猜测归属。
      const request: QueryAuditLogsRequest = {
        ...filters,
        page,
        pageSize: pageSize,
      };
      const result = await BPMNDashboardApi.queryAuditLogs(request);
      setLogs(result.items ?? []);
      setTotal(result.total);
    } catch (error) {
      console.error('Failed to fetch audit logs:', error);
      message.error('Failed to load audit log');
      setLogs([]);
      setTotal(0);
    } finally {
      setListLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [page, pageSize, filters]);

  const fetchTimeline = async (processInstanceKey: string) => {
    setTimelineLoading(true);
    try {
      const data = await BPMNDashboardApi.getProcessTimeline(processInstanceKey);
      setTimeline(data);
      setTimelineVisible(true);
    } catch (error) {
      console.error('Failed to fetch timeline:', error);
      message.error('Failed to load timeline');
    } finally {
      setTimelineLoading(false);
    }
  };

  const getActionColor = (action: string) => {
    switch (action) {
      case 'started':
      case 'activity_started':
        return 'blue';
      case 'completed':
      case 'activity_completed':
        return 'green';
      case 'assigned':
      case 'claimed':
        return 'cyan';
      case 'cancelled':
      case 'terminated':
        return 'red';
      case 'suspended':
      case 'resumed':
        return 'orange';
      case 'variable_changed':
        return 'purple';
      case 'escalated':
      case 'reassigned':
        return 'gold';
      default:
        return 'default';
    }
  };

  const getActivityTypeIcon = (type: string) => {
    return <Activity size={14} />;
  };

  const columns = [
    {
      title: t('bpmn.audit.timestamp') || 'Timestamp',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (val: string) => formatDateTime(val) || '-',
    },
    {
      title: t('bpmn.audit.action') || 'Actions',
      dataIndex: 'action',
      key: 'action',
      width: 120,
      render: (action: string) => (
        <Tag color={getActionColor(action)}>{action}</Tag>
      ),
    },
    {
      title: t('bpmn.audit.activityName') || 'Activity',
      dataIndex:'activityName',
      key:'activityName',
      width: 160,
      ellipsis: true,
      render: (name: string, record: ProcessAuditLog) => (
        <Space size={4} className="w-full min-w-0">
          {getActivityTypeIcon(record.activityType)}
          <span className="truncate">{name || record.activityId}</span>
        </Space>
      ),
    },
    {
      title: t('bpmn.audit.user') || 'Operator',
      dataIndex:'userName',
      key:'userName',
      width: 120,
    },
    {
      title: t('bpmn.audit.assignee') || 'Assignee',
      dataIndex:'assigneeName',
      key:'assigneeName',
      width: 120,
    },
    {
      title: t('bpmn.audit.processInstance') || 'Process Instances',
      dataIndex:'processInstanceKey',
      key:'processInstanceKey',
      width: 220,
      // 实例 key 形如 PI-change_normal_flow-1788572798064346374，无 ellipsis 时会撞宽列把后续字段顶出可视区
      ellipsis: true,
      render: (val: string) => (
        <span className="font-mono text-xs" title={val}>{val || '-'}</span>
      ),
    },
    {
      title: t('bpmn.audit.comment') || 'Notes',
      dataIndex: 'comment',
      key: 'comment',
      ellipsis: true,
    },
    {
      title: t('common.actions') || 'Actions',
      key: 'actions',
      width: 150,
      render: (_value: unknown, record: ProcessAuditLog) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<Eye size={14} />}
            onClick={() => setSelectedLog(record)}
          >
            {t('common.view') || 'Details'}
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => fetchTimeline(record.processInstanceKey)}
          >
            {t('bpmn.audit.timeline') || 'Timeline'}
          </Button>
        </Space>
      ),
    },
  ];

  const actionOptions = [
    { label: 'All', value: '' },
    { label: 'started', value: 'started' },
    { label: 'completed', value: 'completed' },
    { label: 'assigned', value: 'assigned' },
    { label: 'Signed (claimed)', value: 'claimed' },
    { label: 'suspended', value: 'suspended' },
    { label: 'resumed', value: 'resumed' },
    { label: 'terminated', value: 'terminated' },
    { label: 'Variable_changed', value: 'variable_changed' },
  ];

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">
          {t('bpmn.audit.title') || 'BPMN Audit Logs'}
        </h1>
        <Button icon={<RefreshCw size={16} />} onClick={fetchLogs}>
          {t('common.refresh') || 'Refresh'}
        </Button>
      </div>

      {/* Filters */}
      <Card size="small">
        <Space wrap>
          <Input
            placeholder={t('bpmn.audit.processDefinitionKey') || 'Process Definition Key'}
            style={{ width: 200 }}
            onChange={(e) => setFilters({ ...filters, processDefinitionKey: e.target.value || undefined })}
            allowClear
          />
          <Select
            placeholder={t('bpmn.audit.action') || 'ActionsType'}
            style={{ width: 150 }}
            options={actionOptions}
            onChange={(val) => setFilters({ ...filters, action: val || undefined })}
            allowClear
          />
          <Input
            placeholder={t('bpmn.audit.userId') || 'User ID'}
            type="number"
            style={{ width: 100 }}
            onChange={(e) => setFilters({ ...filters, userId: e.target.value ? parseInt(e.target.value) : undefined })}
            allowClear
          />
          <RangePicker
            onChange={(dates, dateStrings) => {
              setFilters({
                ...filters,
                startTime: dateStrings[0] || undefined,
                endTime: dateStrings[1] || undefined,
              });
            }}
          />
        </Space>
      </Card>

      {/* Table */}
      <Card>
        <Table
          dataSource={logs}
          columns={columns}
          rowKey="id"
          loading={listLoading}
          scroll={{ x: 1200 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => `Total ${total} items`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          size="small"
        />
      </Card>

      {/* Detail Modal */}
      <Modal
        title={t('bpmn.audit.detail') || 'Audit Log Details'}
        open={!!selectedLog}
        onCancel={() => setSelectedLog(null)}
        footer={null}
        width={700}
      >
        {selectedLog && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="Timestamp" span={2}>
              {formatDateTime(selectedLog.timestamp) || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Actions">
              <Tag color={getActionColor(selectedLog.action)}>{selectedLog.action}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Activity type">
              {selectedLog.activityType}
            </Descriptions.Item>
            <Descriptions.Item label="Activity" span={2}>
              {selectedLog.activityName}
            </Descriptions.Item>
            <Descriptions.Item label="Operator">
              {selectedLog.userName} (ID: {selectedLog.userId})
            </Descriptions.Item>
            <Descriptions.Item label="Assignee">
              {selectedLog.assigneeName} (ID: {selectedLog.assigneeId})
            </Descriptions.Item>
            <Descriptions.Item label="Process Instance Key" span={2}>
              {selectedLog.processInstanceKey}
            </Descriptions.Item>
            <Descriptions.Item label="Process Definition Key" span={2}>
              {selectedLog.processDefinitionKey}
            </Descriptions.Item>
            <Descriptions.Item label="Notes" span={2}>
              {selectedLog.comment || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="IP Address" span={2}>
              {selectedLog.ipAddress || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Before variable change" span={2}>
              <pre className="text-xs bg-gray-50 p-2 rounded">
                {JSON.stringify(selectedLog.variablesBefore, null, 2) || '-'}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="After variable change" span={2}>
              <pre className="text-xs bg-gray-50 p-2 rounded">
                {JSON.stringify(selectedLog.variablesAfter, null, 2) || '-'}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      {/* Timeline Modal */}
      <Modal
        title={t('bpmn.audit.processTimeline') || 'Process Timeline'}
        open={timelineVisible}
        onCancel={() => setTimelineVisible(false)}
        footer={null}
        width={800}
      >
        {timelineLoading ? (
          <div className="flex justify-center py-8">
            <Spin />
          </div>
        ) : (
          <Timeline
            items={timeline.map((log) => ({
              color: getActionColor(log.action),
              children: (
                <div>
                  <Space>
                    <Tag color={getActionColor(log.action)}>{log.action}</Tag>
                    <span>{log.activityName}</span>
                  </Space>
                  <div className="text-sm text-gray-500 mt-1">
                    <Space>
                      <User size={12} /> {log.userName}
                      <Clock size={12} /> {formatDateTime(log.timestamp) || '-'}
                    </Space>
                  </div>
                </div>
              ),
            }))}
          />
        )}
      </Modal>
    </div>
  );
}

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

import BPMNDashboardApi, {
  ProcessAuditLog,
  QueryAuditLogsRequest,
} from '@/lib/api/bpmn-dashboard-api';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

export default function AuditLogsPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [logs, setLogs] = useState<ProcessAuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedLog, setSelectedLog] = useState<ProcessAuditLog | null>(null);
  const [timelineVisible, setTimelineVisible] = useState(false);
  const [timeline, setTimeline] = useState<ProcessAuditLog[]>([]);

  // 筛选条件
  const [filters, setFilters] = useState<QueryAuditLogsRequest>({
    tenant_id: 1,
    page: 1,
    page_size: 20,
  });

  const tenantId = 1;

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const request: QueryAuditLogsRequest = {
        ...filters,
        tenant_id: tenantId,
        page,
        page_size: pageSize,
      };
      const result = await BPMNDashboardApi.queryAuditLogs(request);
      setLogs(result.list);
      setTotal(result.total);
    } catch (error) {
      console.error('Failed to fetch audit logs:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [page, pageSize, filters]);

  const fetchTimeline = async (processInstanceKey: string) => {
    try {
      const data = await BPMNDashboardApi.getProcessTimeline(processInstanceKey);
      setTimeline(data);
      setTimelineVisible(true);
    } catch (error) {
      console.error('Failed to fetch timeline:', error);
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
      title: t('bpmn.audit.timestamp') || '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (val: string) => new Date(val).toLocaleString(),
    },
    {
      title: t('bpmn.audit.action') || '操作',
      dataIndex: 'action',
      key: 'action',
      width: 120,
      render: (action: string) => (
        <Tag color={getActionColor(action)}>{action}</Tag>
      ),
    },
    {
      title: t('bpmn.audit.activity_name') || '活动',
      dataIndex: 'activity_name',
      key: 'activity_name',
      width: 150,
      render: (name: string, record: ProcessAuditLog) => (
        <Space>
          {getActivityTypeIcon(record.activity_type)}
          {name || record.activity_id}
        </Space>
      ),
    },
    {
      title: t('bpmn.audit.user') || '操作人',
      dataIndex: 'user_name',
      key: 'user_name',
      width: 120,
    },
    {
      title: t('bpmn.audit.assignee') || '受理人',
      dataIndex: 'assignee_name',
      key: 'assignee_name',
      width: 120,
    },
    {
      title: t('bpmn.audit.process_instance') || '流程实例',
      dataIndex: 'process_instance_key',
      key: 'process_instance_key',
      width: 180,
    },
    {
      title: t('bpmn.audit.comment') || '备注',
      dataIndex: 'comment',
      key: 'comment',
      ellipsis: true,
    },
    {
      title: t('common.actions') || '操作',
      key: 'actions',
      width: 150,
      render: (_: any, record: ProcessAuditLog) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<Eye size={14} />}
            onClick={() => setSelectedLog(record)}
          >
            {t('common.view') || '详情'}
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => fetchTimeline(record.process_instance_key)}
          >
            {t('bpmn.audit.timeline') || '时间线'}
          </Button>
        </Space>
      ),
    },
  ];

  const actionOptions = [
    { label: '全部', value: '' },
    { label: '启动 (started)', value: 'started' },
    { label: '完成 (completed)', value: 'completed' },
    { label: '分配 (assigned)', value: 'assigned' },
    { label: '签收 (claimed)', value: 'claimed' },
    { label: '暂停 (suspended)', value: 'suspended' },
    { label: '恢复 (resumed)', value: 'resumed' },
    { label: '终止 (terminated)', value: 'terminated' },
    { label: '变量变更 (variable_changed)', value: 'variable_changed' },
  ];

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">
          {t('bpmn.audit.title') || 'BPMN审计日志'}
        </h1>
        <Button icon={<RefreshCw size={16} />} onClick={fetchLogs}>
          {t('common.refresh') || '刷新'}
        </Button>
      </div>

      {/* Filters */}
      <Card size="small">
        <Space wrap>
          <Input
            placeholder={t('bpmn.audit.process_instance_key') || '流程实例Key'}
            style={{ width: 200 }}
            onChange={(e) => setFilters({ ...filters, process_definition_key: e.target.value || undefined })}
            allowClear
          />
          <Select
            placeholder={t('bpmn.audit.action') || '操作类型'}
            style={{ width: 150 }}
            options={actionOptions}
            onChange={(val) => setFilters({ ...filters, action: val || undefined })}
            allowClear
          />
          <Input
            placeholder={t('bpmn.audit.user_id') || '用户ID'}
            type="number"
            style={{ width: 100 }}
            onChange={(e) => setFilters({ ...filters, user_id: e.target.value ? parseInt(e.target.value) : undefined })}
            allowClear
          />
          <RangePicker
            onChange={(dates, dateStrings) => {
              setFilters({
                ...filters,
                start_time: dateStrings[0] || undefined,
                end_time: dateStrings[1] || undefined,
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
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => `总计 ${total} 条`,
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
        title={t('bpmn.audit.detail') || '审计日志详情'}
        open={!!selectedLog}
        onCancel={() => setSelectedLog(null)}
        footer={null}
        width={700}
      >
        {selectedLog && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="时间" span={2}>
              {new Date(selectedLog.timestamp).toLocaleString()}
            </Descriptions.Item>
            <Descriptions.Item label="操作">
              <Tag color={getActionColor(selectedLog.action)}>{selectedLog.action}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="活动类型">
              {selectedLog.activity_type}
            </Descriptions.Item>
            <Descriptions.Item label="活动名称" span={2}>
              {selectedLog.activity_name}
            </Descriptions.Item>
            <Descriptions.Item label="操作人">
              {selectedLog.user_name} (ID: {selectedLog.user_id})
            </Descriptions.Item>
            <Descriptions.Item label="受理人">
              {selectedLog.assignee_name} (ID: {selectedLog.assignee_id})
            </Descriptions.Item>
            <Descriptions.Item label="流程实例Key" span={2}>
              {selectedLog.process_instance_key}
            </Descriptions.Item>
            <Descriptions.Item label="流程定义Key" span={2}>
              {selectedLog.process_definition_key}
            </Descriptions.Item>
            <Descriptions.Item label="备注" span={2}>
              {selectedLog.comment || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="IP地址" span={2}>
              {selectedLog.ip_address || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="变量变更前" span={2}>
              <pre className="text-xs bg-gray-50 p-2 rounded">
                {JSON.stringify(selectedLog.variables_before, null, 2) || '-'}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="变量变更后" span={2}>
              <pre className="text-xs bg-gray-50 p-2 rounded">
                {JSON.stringify(selectedLog.variables_after, null, 2) || '-'}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      {/* Timeline Modal */}
      <Modal
        title={t('bpmn.audit.process_timeline') || '流程时间线'}
        open={timelineVisible}
        onCancel={() => setTimelineVisible(false)}
        footer={null}
        width={800}
      >
        <Timeline
          items={timeline.map((log) => ({
            color: getActionColor(log.action),
            children: (
              <div>
                <Space>
                  <Tag color={getActionColor(log.action)}>{log.action}</Tag>
                  <span>{log.activity_name}</span>
                </Space>
                <div className="text-sm text-gray-500 mt-1">
                  <Space>
                    <User size={12} /> {log.user_name}
                    <Clock size={12} /> {new Date(log.timestamp).toLocaleString()}
                  </Space>
                </div>
              </div>
            ),
          }))}
        />
      </Modal>
    </div>
  );
}

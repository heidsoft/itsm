'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Select,
  DatePicker,
  Space,
  Tag,
  Row,
  Col,
  Statistic,
  Alert,
} from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import { AlertTriangle, CheckCircle, Clock, RefreshCw } from 'lucide-react';

import BPMNDashboardApi, { SLAViolation, ProcessMetrics } from '@/lib/api/bpmn-dashboard-api';
import { useI18n } from '@/lib/i18n';
import { WorkflowApi } from '@/lib/api/workflow-api';

const { RangePicker } = DatePicker;

export default function SLAMonitoringPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [violations, setViolations] = useState<SLAViolation[]>([]);
  const [selectedProcess, setSelectedProcess] = useState<string>('');
  const [processes, setProcesses] = useState<{ key: string; name: string }[]>([]);
  const [processMetrics, setProcessMetrics] = useState<ProcessMetrics | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()]);

  const tenantId = 1;

  const fetchProcesses = async () => {
    try {
      const data = await WorkflowApi.getWorkflows({ page: 1, pageSize: 100 });
      setProcesses(
        data.workflows.map(workflow => ({
          key: workflow.code,
          name: workflow.name || workflow.code,
        }))
      );
    } catch (error) {
      console.error('Failed to fetch processes:', error);
    }
  };

  const fetchViolations = async () => {
    setLoading(true);
    try {
      const data = await BPMNDashboardApi.getSLAViolations(tenantId);
      setViolations(data);
    } catch (error) {
      console.error('Failed to fetch violations:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchProcessMetrics = async () => {
    if (!selectedProcess) return;
    setLoading(true);
    try {
      const data = await BPMNDashboardApi.getProcessMetrics(
        selectedProcess,
        tenantId,
        dateRange[0].format('YYYY-MM-DD'),
        dateRange[1].format('YYYY-MM-DD')
      );
      setProcessMetrics(data);
    } catch (error) {
      console.error('Failed to fetch process metrics:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProcesses();
    fetchViolations();
  }, []);

  useEffect(() => {
    if (selectedProcess) {
      fetchProcessMetrics();
    }
  }, [selectedProcess, dateRange]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'breached':
        return 'red';
      case 'warning':
        return 'orange';
      case 'ok':
        return 'green';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'breached':
        return <AlertTriangle size={14} />;
      case 'warning':
        return <Clock size={14} />;
      case 'ok':
        return <CheckCircle size={14} />;
      default:
        return null;
    }
  };

  const violationColumns = [
    {
      title: t('bpmn.sla.resource_type') || '资源类型',
      dataIndex: 'resource_type',
      key: 'resource_type',
      width: 120,
      render: (val: string) => <Tag>{val}</Tag>,
    },
    {
      title: t('bpmn.sla.resource_key') || '资源Key',
      dataIndex: 'resource_key',
      key: 'resource_key',
    },
    {
      title: t('bpmn.sla.status') || '状态',
      dataIndex: 'sla_status',
      key: 'sla_status',
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)} icon={getStatusIcon(status)}>
          {status}
        </Tag>
      ),
    },
    {
      title: t('bpmn.sla.start_time') || '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 160,
      render: (val: string) => new Date(val).toLocaleString(),
    },
    {
      title: t('bpmn.sla.deadline') || '截止时间',
      dataIndex: 'deadline',
      key: 'deadline',
      width: 160,
      render: (val: string) => new Date(val).toLocaleString(),
    },
    {
      title: t('bpmn.sla.elapsed_minutes') || '已耗时(分钟)',
      dataIndex: 'elapsed_minutes',
      key: 'elapsed_minutes',
      width: 120,
      render: (val: number) => <span className={val > 480 ? 'text-red-500' : ''}>{val}</span>,
    },
  ];

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">{t('bpmn.sla.title') || 'BPMN SLA监控'}</h1>
        <Button
          icon={<RefreshCw size={16} />}
          onClick={() => {
            fetchViolations();
            fetchProcessMetrics();
          }}
        >
          {t('common.refresh') || '刷新'}
        </Button>
      </div>

      {/* Process Metrics */}
      <Card title={t('bpmn.sla.process_metrics') || '流程SLA指标'}>
        <Space wrap className="mb-4">
          <Select
            placeholder={t('bpmn.sla.select_process') || '选择流程'}
            style={{ width: 250 }}
            value={selectedProcess || undefined}
            onChange={setSelectedProcess}
            options={processes.map(p => ({ label: p.name, value: p.key }))}
            allowClear
          />
          <RangePicker
            value={[dateRange[0], dateRange[1]]}
            onChange={dates => {
              if (dates && dates[0] && dates[1]) {
                setDateRange([dates[0], dates[1]]);
              }
            }}
          />
        </Space>

        {selectedProcess && processMetrics && (
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.total_instances') || '总实例数'}
                value={processMetrics.total_instances}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.running_instances') || '进行中'}
                value={processMetrics.running_instances}
                styles={{ content: { color: '#1890ff' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.completed_instances') || '已完成'}
                value={processMetrics.completed_instances}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.completion_rate') || '完成率'}
                value={processMetrics.completion_rate}
                suffix="%"
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.sla_compliance_rate') || 'SLA合规率'}
                value={processMetrics.sla_compliance_rate}
                suffix="%"
                styles={{
                  content: {
                    color:
                      processMetrics.sla_compliance_rate >= 90
                        ? '#52c41a'
                        : processMetrics.sla_compliance_rate >= 70
                          ? '#faad14'
                          : '#ff4d4f',
                  },
                }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.avg_completion_time') || '平均完成时间'}
                value={processMetrics.avg_completion_time_minutes?.toFixed(1) || 0}
                suffix="分钟"
              />
            </Col>
          </Row>
        )}
      </Card>

      {/* SLA Violations */}
      <Card
        title={t('bpmn.sla.violations') || 'SLA违规告警'}
        extra={
          <Tag color="red">
            {violations.length} {t('bpmn.sla.items') || '项'}
          </Tag>
        }
      >
        {violations.length === 0 ? (
          <Alert title={t('bpmn.sla.no_violations') || '暂无SLA违规'} type="success" showIcon />
        ) : (
          <Table
            dataSource={violations}
            columns={violationColumns}
            rowKey={record => `${record.resource_type}-${record.resource_id}`}
            loading={loading}
            pagination={false}
            size="small"
          />
        )}
      </Card>

      {/* SLA Status Summary */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('bpmn.sla.breached') || '已逾期'}
              value={violations.filter(v => v.sla_status === 'breached').length}
              prefix={<AlertTriangle size={20} />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('bpmn.sla.warning') || '预警中'}
              value={violations.filter(v => v.sla_status === 'warning').length}
              prefix={<Clock size={20} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('bpmn.sla.ok') || '正常'}
              value={violations.filter(v => v.sla_status === 'ok').length}
              prefix={<CheckCircle size={20} />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}

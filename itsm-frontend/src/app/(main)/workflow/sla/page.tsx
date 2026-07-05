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
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { AlertTriangle, CheckCircle, Clock, RefreshCw } from 'lucide-react';

import type { SLAViolation, ProcessMetrics } from '@/lib/api/bpmn-dashboard-api';
import BPMNDashboardApi from '@/lib/api/bpmn-dashboard-api';
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
      title: t('bpmn.sla.resourceType') || '资源类型',
      dataIndex: 'resourceType',
      key: 'resourceType',
      width: 120,
      render: (val: string) => <Tag>{val}</Tag>,
    },
    {
      title: t('bpmn.sla.resourceKey') || '资源Key',
      dataIndex:'resourceKey',
      key:'resourceKey',
    },
    {
      title: t('bpmn.sla.status') || '状态',
      dataIndex:'slaStatus',
      key:'slaStatus',
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)} icon={getStatusIcon(status)}>
          {status}
        </Tag>
      ),
    },
    {
      title: t('bpmn.sla.startTime') || '开始时间',
      dataIndex:'startTime',
      key:'startTime',
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
      title: t('bpmn.sla.elapsedMinutes') || '已耗时(分钟)',
      dataIndex:'elapsedMinutes',
      key:'elapsedMinutes',
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
      <Card title={t('bpmn.sla.processMetrics') || '流程SLA指标'}>
        <Space wrap className="mb-4">
          <Select
            placeholder={t('bpmn.sla.selectProcess') || '选择流程'}
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
                title={t('bpmn.sla.totalInstances') || '总实例数'}
                value={processMetrics.totalInstances}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.runningInstances') || '进行中'}
                value={processMetrics.runningInstances}
                styles={{ content: { color: '#1890ff' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.completedInstances') || '已完成'}
                value={processMetrics.completedInstances}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.completionRate') || '完成率'}
                value={processMetrics.completionRate}
                suffix="%"
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.slaComplianceRate') || 'SLA合规率'}
                value={processMetrics.slaComplianceRate}
                suffix="%"
                styles={{
                  content: {
                    color:
                      processMetrics.slaComplianceRate >= 90
                        ? '#52c41a'
                        : processMetrics.slaComplianceRate >= 70
                          ? '#faad14'
                          : '#ff4d4f',
                  },
                }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('bpmn.sla.avgCompletionTime') || '平均完成时间'}
                value={processMetrics.avgCompletionTimeMinutes?.toFixed(1) || 0}
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
          <Alert title={t('bpmn.sla.noViolations') || '暂无SLA违规'} type="success" showIcon />
        ) : (
          <Table
            dataSource={violations}
            columns={violationColumns}
            rowKey={record => `${record.resourceType}-${record.resourceId}`}
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
              value={violations.filter(v => v.slaStatus === 'breached').length}
              prefix={<AlertTriangle size={20} />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('bpmn.sla.warning') || '预警中'}
              value={violations.filter(v => v.slaStatus === 'warning').length}
              prefix={<Clock size={20} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title={t('bpmn.sla.ok') || '正常'}
              value={violations.filter(v => v.slaStatus === 'ok').length}
              prefix={<CheckCircle size={20} />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}

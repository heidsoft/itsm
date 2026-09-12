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
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n';
import { WorkflowApi } from '@/lib/api/workflow-api';

const { RangePicker } = DatePicker;

export default function SLAMonitoringPage() {
  const { t } = useI18n();
  const currentTenant = useAuthStore(s => s.currentTenant);
  const [loading, setLoading] = useState(false);
  const [violations, setViolations] = useState<SLAViolation[]>([]);
  const [selectedProcess, setSelectedProcess] = useState<string>('');
  const [processes, setProcesses] = useState<{ key: string; name: string }[]>([]);
  const [processMetrics, setProcessMetrics] = useState<ProcessMetrics | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()]);

  // 使用当前登录租户；未登录时不加载数据
  const tenantId = currentTenant?.id;

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
    if (!tenantId) {
      setLoading(false);
      return;
    }
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
    if (!selectedProcess || !tenantId) return;
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

  const VIOLATION_TYPE_LABEL: Record<string, string> = {
    response_time: '响应超时',
    resolution_time: '解决超时',
  };

  const severityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'red';
      case 'high':
        return 'orange';
      case 'medium':
        return 'gold';
      case 'low':
        return 'blue';
      default:
        return 'default';
    }
  };

  const violationColumns = [
    {
      title: '工单',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      width: 220,
      render: (val: string, record: SLAViolation) => (
        <div className="min-w-0">
          <div className="font-medium">{val || `#${record.ticketId}`}</div>
          {record.ticketTitle && (
            <div className="truncate text-xs text-gray-500">{record.ticketTitle}</div>
          )}
        </div>
      ),
    },
    {
      title: 'SLA',
      dataIndex: 'slaName',
      key: 'slaName',
      width: 140,
      render: (val: string) => val || '-',
    },
    {
      title: '违规类型',
      dataIndex: 'violationType',
      key: 'violationType',
      width: 110,
      render: (val: string) => <Tag>{VIOLATION_TYPE_LABEL[val] || val || '-'}</Tag>,
    },
    {
      title: '严重级别',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (val: string) => <Tag color={severityColor(val)}>{val || '-'}</Tag>,
    },
    {
      title: '违规时间',
      dataIndex: 'violationTime',
      key: 'violationTime',
      width: 160,
      render: (val: string) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '状态',
      dataIndex: 'isResolved',
      key: 'isResolved',
      width: 100,
      render: (isResolved: boolean) =>
        isResolved ? (
          <Tag color="green" icon={<CheckCircle size={14} />}>
            已处理
          </Tag>
        ) : (
          <Tag color="red" icon={<AlertTriangle size={14} />}>
            未处理
          </Tag>
        ),
    },
  ];

  return (
    <div className='p-6 space-y-6'>
      {/* Header */}
      <div className='flex justify-between items-center'>
        <h1 className='text-2xl font-bold'>{t('workflow.sla.title') || 'BPMN SLA监控'}</h1>
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
      <Card title={t('workflow.sla.processMetrics') || '流程SLA指标'}>
        <Space wrap className='mb-4'>
          <Select
            placeholder={t('workflow.sla.selectProcess') || '选择流程'}
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
                title={t('workflow.sla.totalInstances') || '总实例数'}
                value={processMetrics.totalInstances}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.runningInstances') || '进行中'}
                value={processMetrics.runningInstances}
                styles={{ content: { color: '#1890ff' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.completedInstances') || '已完成'}
                value={processMetrics.completedInstances}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.completionRate') || '完成率'}
                value={processMetrics.completionRate}
                suffix='%'
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.slaComplianceRate') || 'SLA合规率'}
                value={processMetrics.slaComplianceRate}
                suffix='%'
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
                title={t('workflow.sla.avgCompletionTime') || '平均完成时间'}
                value={processMetrics.avgCompletionTimeMinutes?.toFixed(1) || 0}
                suffix='分钟'
              />
            </Col>
          </Row>
        )}
      </Card>

      {/* SLA Violations */}
      <Card
        title={t('workflow.sla.violations') || 'SLA违规告警'}
        extra={
          <Tag color='red'>
            {violations.length} {t('workflow.sla.items') || '项'}
          </Tag>
        }
      >
        {violations.length === 0 ? (
          <Alert message={t('workflow.sla.noViolations') || '暂无SLA违规'} type='success' showIcon />
        ) : (
          <Table
            dataSource={violations}
            columns={violationColumns}
            rowKey={record => record.id}
            loading={loading}
            pagination={false}
            size='small'
          />
        )}
      </Card>

      {/* SLA Status Summary */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="未处理"
              value={violations.filter(v => !v.isResolved).length}
              prefix={<AlertTriangle size={20} />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="高严重级别"
              value={violations.filter(v => v.severity === 'critical' || v.severity === 'high').length}
              prefix={<Clock size={20} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="已处理"
              value={violations.filter(v => v.isResolved).length}
              prefix={<CheckCircle size={20} />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}

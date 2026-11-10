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
  Skeleton,
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
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「违规总数 0」的假统计卡与空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [violations, setViolations] = useState<SLAViolation[]>([]);
  // 违规列表Failed to load必须与「暂无违规」区分：2026-10-14 修复前失败被渲染成成功态空列表
  const [violationsError, setViolationsError] = useState(false);
  const [selectedProcess, setSelectedProcess] = useState<string>('');
  const [processes, setProcesses] = useState<{ key: string; name: string }[]>([]);
  const [processMetrics, setProcessMetrics] = useState<ProcessMetrics | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()]);

  // 页面用 currentTenant 是否已水合作为「等登录后再取数」的门槛。
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
      setListLoading(false);
      return;
    }
    setListLoading(true);
    try {
      const data = await BPMNDashboardApi.getSLAViolations(tenantId);
      setViolations(data);
      setViolationsError(false);
    } catch (error) {
      console.error('Failed to fetch violations:', error);
      setViolations([]);
      setViolationsError(true);
    } finally {
      setListLoading(false);
    }
  };

  const fetchProcessMetrics = async () => {
    if (!selectedProcess || !tenantId) return;
    setListLoading(true);
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
      setListLoading(false);
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
    response_time: 'Response timeout',
    resolution_time: 'Resolving timeout',
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
      title: 'Ticket',
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
      title: 'Violation Type',
      dataIndex: 'violationType',
      key: 'violationType',
      width: 110,
      render: (val: string) => <Tag>{VIOLATION_TYPE_LABEL[val] || val || '-'}</Tag>,
    },
    {
      title: 'Severity level',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (val: string) => <Tag color={severityColor(val)}>{val || '-'}</Tag>,
    },
    {
      title: 'Violation Time',
      dataIndex: 'violationTime',
      key: 'violationTime',
      width: 160,
      render: (val: string) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: 'Status',
      dataIndex: 'isResolved',
      key: 'isResolved',
      width: 100,
      render: (isResolved: boolean) =>
        isResolved ? (
          <Tag color="green" icon={<CheckCircle size={14} />}>

            Processed
          </Tag>
        ) : (
          <Tag color="red" icon={<AlertTriangle size={14} />}>

            Not processed
          </Tag>
        ),
    },
  ];

  return (
    <div className='p-6 space-y-6'>
      {/* Header */}
      <div className='flex justify-between items-center'>
        <h1 className='text-2xl font-bold'>{t('workflow.sla.title') || 'BPMN SLA Monitor'}</h1>
        <Button
          icon={<RefreshCw size={16} />}
          onClick={() => {
            fetchViolations();
            fetchProcessMetrics();
          }}
        >
          {t('common.refresh') || 'Refresh'}
        </Button>
      </div>

      {/* Process Metrics */}
      <Card title={t('workflow.sla.processMetrics') || 'Process SLA Metrics'}>
        <Space wrap className='mb-4'>
          <Select
            placeholder={t('workflow.sla.selectProcess') || 'Select Process'}
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
                title={t('workflow.sla.totalInstances') || 'Total Instances'}
                value={processMetrics.totalInstances}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.runningInstances') || 'In Progress'}
                value={processMetrics.runningInstances}
                styles={{ content: { color: '#1890ff' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.completedInstances') || 'Completed'}
                value={processMetrics.completedInstances}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.completionRate') || 'Completion Rate'}
                value={processMetrics.completionRate}
                suffix='%'
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('workflow.sla.slaComplianceRate') || 'SLA Compliance Rate'}
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
                title={t('workflow.sla.avgCompletionTime') || 'Avg Completion Time'}
                value={processMetrics.avgCompletionTimeMinutes?.toFixed(1) || 0}
                suffix="minutes"
              />
            </Col>
          </Row>
        )}
      </Card>

      {/* SLA Violations */}
      <Card
        title={t('workflow.sla.violations') || 'SLA Violations'}
        extra={
          <Tag color='red'>
            {violations.length} {t('workflow.sla.items') || 'items'}
          </Tag>
        }
      >
        {violationsError ? (
          <Alert
            title="SLA violation data loading failed"
            description="Please try again later; failure does not mean there is no violation, please do not judge the health of the system based on this."
            type='error'
            showIcon
          />
        ) : violations.length === 0 ? (
          // 空态必须等列表加载完成后再判定：首帧 violations 仍是空数组，
          // 直接渲染「No SLA violations」会把「还没拉到」误报成「系统没有违规」。
          listLoading ? (
            <Skeleton active paragraph={{ rows: 3 }} />
          ) : (
            <Alert title={t('workflow.sla.noViolations') || 'No SLA violations'} type='success' showIcon />
          )
        ) : (
          <Table
            dataSource={violations}
            columns={violationColumns}
            rowKey={record => record.id}
            loading={listLoading}
            pagination={false}
            size='small'
          />
        )}
      </Card>

      {/* SLA Status Summary */}
      {/* 统计卡必须门控：还没拉到数据时渲染成「未处理 0」会被误读成「没有违规」 */}
      {!listLoading && (
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="Not processed"
              value={violations.filter(v => !v.isResolved).length}
              prefix={<AlertTriangle size={20} />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="High severity"
              value={violations.filter(v => v.severity === 'critical' || v.severity === 'high').length}
              prefix={<Clock size={20} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="Processed"
              value={violations.filter(v => v.isResolved).length}
              prefix={<CheckCircle size={20} />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
      </Row>
      )}
    </div>
  );
}

'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Button, Card, Col, Empty, Row, Space, Spin, Statistic, Tag, Tooltip } from 'antd';
import { AlertTriangle, Download, RefreshCw } from 'lucide-react';
import dayjs from 'dayjs';
import { SLAApi, type SLAComplianceReport } from '@/lib/api/sla-api';

interface SLAStats {
  totalDefinitions: number;
  activeDefinitions: number;
  totalViolations: number;
  openViolations: number;
  overallComplianceRate: number;
}

const priorityConfig: Record<string, { label: string; color: string }> = {
  critical: { label: '紧急', color: 'magenta' },
  urgent: { label: '紧急', color: 'magenta' },
  high: { label: '高', color: 'red' },
  medium: { label: '普通', color: 'gold' },
  normal: { label: '普通', color: 'blue' },
  low: { label: '低', color: 'green' },
};

export default function SLAPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshedAt, setRefreshedAt] = useState<Date>();
  const [stats, setStats] = useState<SLAStats>({
    totalDefinitions: 0,
    activeDefinitions: 0,
    totalViolations: 0,
    openViolations: 0,
    overallComplianceRate: 0,
  });
  const [report, setReport] = useState<SLAComplianceReport>();
  const [alerts, setAlerts] = useState<Awaited<ReturnType<typeof SLAApi.getSLAAlerts>>>([]);

  const dateRange = useMemo(() => ({
    startDate: dayjs().subtract(29, 'day').startOf('day').toISOString(),
    endDate: dayjs().endOf('day').toISOString(),
  }), []);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [statsResult, reportResult, alertResult] = await Promise.all([
        SLAApi.getSLAStats(),
        SLAApi.getSLAComplianceReport(dateRange),
        SLAApi.getSLAAlerts(),
      ]);
      setStats(statsResult);
      setReport(reportResult);
      setAlerts(alertResult);
      setRefreshedAt(new Date());
    } catch (loadError) {
      console.error('Failed to load SLA dashboard:', loadError);
      setError('SLA 数据加载失败，请检查服务连接后重试。');
    } finally {
      setLoading(false);
    }
  }, [dateRange]);

  useEffect(() => { void loadData(); }, [loadData]);

  const exportReport = () => {
    const rows = [
      ['指标', '数值'],
      ['统计周期', `${dayjs(dateRange.startDate).format('YYYY-MM-DD')} 至 ${dayjs(dateRange.endDate).format('YYYY-MM-DD')}`],
      ['总工单', report?.totalTickets ?? 0],
      ['符合 SLA', report?.metSla ?? 0],
      ['违反 SLA', report?.violatedSla ?? stats.totalViolations],
      ['未解决违规', stats.openViolations],
      ['合规率', `${(report?.complianceRate ?? stats.overallComplianceRate).toFixed(1)}%`],
      ['平均响应时间（分钟）', report?.avgResponseTime ?? 0],
      ['平均解决时间 MTTR（分钟）', report?.avgResolutionTime ?? 0],
    ];
    const csv = `\uFEFF${rows.map(row => row.map(value => `"${String(value).replaceAll('"', '""')}"`).join(',')).join('\n')}`;
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `sla-breach-report-${dayjs().format('YYYY-MM-DD')}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  if (loading && !refreshedAt) {
    return <div className="flex min-h-[400px] items-center justify-center"><Spin size="large" tip="正在加载 SLA 指标…" /></div>;
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="mb-1 text-2xl font-bold">SLA 服务级别管理</h1>
          <p className="m-0 text-sm text-gray-500">近 30 天服务质量与实时违约风险</p>
        </div>
        <Space wrap>
          {refreshedAt && <span className="text-xs text-gray-500">更新于 {dayjs(refreshedAt).format('HH:mm:ss')}</span>}
          <Button icon={<RefreshCw size={16} />} loading={loading} onClick={() => void loadData()}>刷新</Button>
          <Button type="primary" danger icon={<Download size={16} />} onClick={exportReport}>导出 SLA 违约报告</Button>
        </Space>
      </div>

      {error && <Alert className="mb-4" type="error" showIcon message={error} action={<Button size="small" onClick={() => void loadData()}>重试</Button>} />}
      {stats.openViolations > 0 && (
        <Alert className="mb-4" type="error" showIcon icon={<AlertTriangle />} message={`${stats.openViolations} 个 SLA 违规尚未处理`} description="请立即检查紧急与高优先级工单，避免升级影响扩大。" action={<Button danger onClick={() => router.push('/sla-monitor')}>查看违规</Button>} />
      )}

      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} xl={6}><Card><Statistic title="SLA 合规率" value={report?.complianceRate ?? stats.overallComplianceRate} precision={1} suffix="%" valueStyle={{ color: (report?.complianceRate ?? stats.overallComplianceRate) >= 95 ? '#389e0d' : '#d46b08' }} /></Card></Col>
        <Col xs={24} sm={12} xl={6}><Card><Statistic title="平均响应时间" value={report?.avgResponseTime ?? 0} suffix="分钟" /></Card></Col>
        <Col xs={24} sm={12} xl={6}><Card><Statistic title="MTTR / 平均解决时间" value={report?.avgResolutionTime ?? 0} suffix="分钟" /></Card></Col>
        <Col xs={24} sm={12} xl={6}><Card><Tooltip title="当前 SLA API 尚未提供故障间隔数据"><Statistic title="MTBF / 平均故障间隔" value="—" /></Tooltip></Card></Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={15}>
          <Card title="实时风险告警" extra={<Button type="link" onClick={() => router.push('/sla-monitor')}>全部监控</Button>}>
            {alerts.length === 0 ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="当前没有 SLA 风险告警" /> : (
              <div className="space-y-3">
                {alerts.slice(0, 6).map(alert => {
                  const priority = priorityConfig[alert.priority] ?? priorityConfig.normal;
                  return <div key={`${alert.ticketId}-${alert.createdAt}`} className="flex flex-wrap items-center justify-between gap-3 rounded border border-gray-200 p-3">
                    <div><div className="font-medium">{alert.ticketTitle}</div><div className="text-xs text-gray-500">{alert.slaDefinition} · 工单 #{alert.ticketId}</div></div>
                    <Space><Tag color={priority.color}>{priority.label}优先级</Tag><Tag color={alert.timeRemaining <= 0 ? 'red' : 'orange'}>{alert.timeRemaining <= 0 ? `已超时 ${Math.abs(alert.timeRemaining)} 分钟` : `剩余 ${alert.timeRemaining} 分钟`}</Tag></Space>
                  </div>;
                })}
              </div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={9}>
          <Card title="优先级处置指引">
            <Space orientation="vertical" size="middle" className="w-full">
              <div><Tag color="magenta">紧急</Tag><span>立即升级，持续跟进直至恢复</span></div>
              <div><Tag color="red">高</Tag><span>进入当班队列，优先处理</span></div>
              <div><Tag color="blue">普通</Tag><span>按 SLA 目标有序处理</span></div>
            </Space>
            <div className="mt-6 grid grid-cols-2 gap-3">
              <Card size="small"><Statistic title="生效 SLA" value={stats.activeDefinitions} /></Card>
              <Card size="small"><Statistic title="累计违规" value={stats.totalViolations} /></Card>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}

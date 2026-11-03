'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Button, Card, Col, Empty, Row, Space, Spin, Statistic, Tag, Tooltip } from 'antd';
import { AlertTriangle, Download, RefreshCw } from 'lucide-react';
import dayjs from 'dayjs';
import { SLAApi, type SLAComplianceReport } from '@/lib/api/sla-api';
import SLATrendPredictionCard from '@/components/business/sla/SLATrendPredictionCard';
import { useI18n } from '@/lib/i18n/useI18n';

interface SLAStats {
  totalDefinitions: number;
  activeDefinitions: number;
  totalViolations: number;
  openViolations: number;
  overallComplianceRate: number;
}

type PriorityKey = 'critical' | 'urgent' | 'high' | 'medium' | 'normal' | 'low';

const priorityKeys: PriorityKey[] = ['critical', 'urgent', 'high', 'medium', 'normal', 'low'];

// 剩余时间必须按量级格式化：直接展示「3018 分钟」会让用户自己换算成天/小时。
// 单位随语言切换，避免英文界面出现中文单位。
function formatDuration(minutes: number, language: string): string {
  const abs = Math.abs(Math.round(minutes));
  const days = Math.floor(abs / 1440);
  const hours = Math.floor((abs % 1440) / 60);
  const mins = abs % 60;
  if (language === 'en-US') {
    if (days > 0) return hours > 0 ? `${days}d ${hours}h` : `${days}d`;
    if (hours > 0) return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
    return `${mins}m`;
  }
  if (days > 0) return hours > 0 ? `${days} 天 ${hours} 小时` : `${days} 天`;
  if (hours > 0) return mins > 0 ? `${hours} 小时 ${mins} 分钟` : `${hours} 小时`;
  return `${mins} 分钟`;
}

export default function SLAPage() {
  const router = useRouter();
  const { t, language } = useI18n();

  // 告警规则名沿用了 SLA 定义名前缀（SLA-P3-低-响应告警 / SLA-服务请求-响应告警），
  // 两个维度混排会让人误以为告警类型不同。页面已展示工单与优先级，
  // 规则名只保留告警维度（响应/解决），统一成同一套命名。
  const alertKindLabel = (name?: string | null) => {
    if (name?.endsWith('-响应告警')) return t('sla.alertKindResponse');
    if (name?.endsWith('-解决告警')) return t('sla.alertKindResolution');
    return name || t('sla.alertKindGeneric');
  };
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

  const priorityConfig = useMemo<Record<string, { label: string; color: string }>>(() => {
    const map: Record<string, { label: string; color: string }> = {
      critical: { label: t('sla.priorityCritical'), color: 'magenta' },
      urgent: { label: t('sla.priorityCritical'), color: 'magenta' },
      high: { label: t('sla.priorityHigh'), color: 'red' },
      medium: { label: t('sla.priorityNormal'), color: 'gold' },
      normal: { label: t('sla.priorityNormal'), color: 'blue' },
      low: { label: t('sla.priorityNormal'), color: 'green' },
    };
    priorityKeys.forEach(key => {
      if (!map[key]) {
        map[key] = { label: key, color: 'default' };
      }
    });
    return map;
  }, [t]);

  const dateRange = useMemo(() => ({
    startDate: dayjs().subtract(29, 'day').startOf('day').toISOString(),
    endDate: dayjs().endOf('day').toISOString(),
  }), []);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [statsRes, reportRes, alertsRes] = await Promise.allSettled([
        SLAApi.getSLAStats(),
        SLAApi.getSLAComplianceReport(dateRange),
        SLAApi.getSLAAlerts(),
      ]);
      if (statsRes.status === 'fulfilled') setStats(statsRes.value);
      if (reportRes.status === 'fulfilled') setReport(reportRes.value);
      if (alertsRes.status === 'fulfilled') setAlerts(alertsRes.value);
      if (statsRes.status === 'rejected' && reportRes.status === 'rejected' && alertsRes.status === 'rejected') {
        throw new Error('SLA data load failed');
      }
      setRefreshedAt(new Date());
    } catch (loadError) {
      console.error('Failed to load SLA dashboard:', loadError);
      setError(t('sla.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [dateRange, t]);

  useEffect(() => { void loadData(); }, [loadData]);

  const exportReport = () => {
    const rows = [
      [t('sla.exportMetric'), t('sla.exportValue')],
      [t('sla.exportPeriod'), `${dayjs(dateRange.startDate).format('YYYY-MM-DD')} ~ ${dayjs(dateRange.endDate).format('YYYY-MM-DD')}`],
      [t('sla.exportTotalTickets'), report?.totalTickets ?? 0],
      [t('sla.exportMetSla'), report?.metSla ?? 0],
      [t('sla.exportViolatedSla'), report?.violatedSla ?? stats.totalViolations],
      [t('sla.exportOpenViolations'), stats.openViolations],
      [t('sla.exportComplianceRate'), `${(report?.complianceRate ?? stats.overallComplianceRate).toFixed(1)}%`],
      [t('sla.exportAvgResponse'), report?.avgResponseTime ?? 0],
      [t('sla.exportAvgResolution'), report?.avgResolutionTime ?? 0],
    ];
    const csv = `\uFEFF${rows.map(row => row.map(value => `"${String(value).replaceAll('"', '""')}"`).join(',')).join('\n')}`;
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = t('sla.exportFileName', { date: dayjs().format('YYYY-MM-DD') });
    anchor.click();
    URL.revokeObjectURL(url);
  };

  if (loading && !refreshedAt) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <Spin size="large" tip={t('sla.loading')} />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="mb-1 text-2xl font-bold">{t('sla.dashboardTitle')}</h1>
          <p className="m-0 text-sm text-gray-500">{t('sla.dashboardSubtitle')}</p>
        </div>
        <Space wrap>
          {refreshedAt && (
            <span className="text-xs text-gray-500">
              {t('sla.refreshedAt', { time: dayjs(refreshedAt).format('HH:mm:ss') })}
            </span>
          )}
          <Button icon={<RefreshCw size={16} />} loading={loading} onClick={() => void loadData()}>
            {t('sla.refresh')}
          </Button>
          <Button type="primary" danger icon={<Download size={16} />} onClick={exportReport}>
            {t('sla.exportReport')}
          </Button>
        </Space>
      </div>

      {error && (
        <Alert
          className="mb-4"
          type="error"
          showIcon
          title={error}
          action={
            <Button size="small" onClick={() => void loadData()}>
              {t('sla.retry')}
            </Button>
          }
        />
      )}
      {stats.openViolations > 0 && (
        <Alert
          className="mb-4"
          type="error"
          showIcon
          icon={<AlertTriangle />}
          title={t('sla.openViolationsAlert', { count: stats.openViolations })}
          description={t('sla.openViolationsDesc')}
          action={
            <Button danger onClick={() => router.push('/sla-monitor')}>
              {t('sla.viewViolations')}
            </Button>
          }
        />
      )}

      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} xl={6}>
          <Card>
            <Statistic
              title={t('sla.complianceRate')}
              value={report?.complianceRate ?? stats.overallComplianceRate}
              precision={1}
              suffix="%"
              styles={{ content: {
                color:
                  (report?.complianceRate ?? stats.overallComplianceRate) >= 95 ? '#389e0d' : '#d46b08',
              } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card>
            <Statistic
              title={t('sla.avgResponseTime')}
              // 后端返回的是浮点分钟数；不设 precision 时 Statistic 会原样打印
              // 7.862243183333334 这种全精度值，直接把卡片撑爆（2026-10-25 实测）。
              value={Number.isFinite(report?.avgResponseTime) ? report!.avgResponseTime : 0}
              precision={1}
              suffix={t('sla.minutes')}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card>
            <Statistic
              title={t('sla.mttr')}
              value={Number.isFinite(report?.avgResolutionTime) ? report!.avgResolutionTime : 0}
              precision={1}
              suffix={t('sla.minutes')}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card>
            <Tooltip title={t('sla.mtbfTooltip')}>
              <Statistic title={t('sla.mtbf')} value="—" />
            </Tooltip>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24}>
          <SLATrendPredictionCard />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={15}>
          <Card
            title={t('sla.realTimeAlerts')}
            extra={
              <Button type="link" onClick={() => router.push('/sla-monitor')}>
                {t('sla.allMonitor')}
              </Button>
            }
          >
            {alerts.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('sla.noAlerts')} />
            ) : (
              <div className="space-y-3">
                {alerts.slice(0, 6).map(alert => {
                  const priority = priorityConfig[alert.priority] ?? priorityConfig.normal;
                  // timeRemaining 是 { hours, deadline } 结构：hours 可为负（已超时），
                  // 无截止时间的告警后端返回 null，此时不渲染剩余时间标签。
                  const remainingMinutes = alert.timeRemaining
                    ? Math.round(alert.timeRemaining.hours * 60)
                    : null;
                  return (
                    <div
                      key={alert.id}
                      className="flex flex-wrap items-center justify-between gap-3 rounded border border-gray-200 p-3"
                    >
                      <div>
                        <div className="font-medium">{alert.ticketTitle}</div>
                        <div className="text-xs text-gray-500">
                          {alertKindLabel(alert.alertRuleName)} · {t('sla.ticketRef', { id: alert.ticketId })}
                        </div>
                      </div>
                      <Space>
                        <Tag color={priority.color}>
                          {priority.label}
                          {t('sla.prioritySuffix')}
                        </Tag>
                        {remainingMinutes !== null && (
                          <Tag color={remainingMinutes <= 0 ? 'red' : 'orange'}>
                            {remainingMinutes <= 0
                              ? t('sla.overdueDuration', { duration: formatDuration(remainingMinutes, language) })
                              : t('sla.remainingDuration', { duration: formatDuration(remainingMinutes, language) })}
                          </Tag>
                        )}
                      </Space>
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={9}>
          <Card title={t('sla.priorityGuide')}>
            <Space orientation="vertical" size="middle" className="w-full">
              <div>
                <Tag color="magenta">{t('sla.priorityCritical')}</Tag>
                <span>{t('sla.criticalAction')}</span>
              </div>
              <div>
                <Tag color="red">{t('sla.priorityHigh')}</Tag>
                <span>{t('sla.highAction')}</span>
              </div>
              <div>
                <Tag color="blue">{t('sla.priorityNormal')}</Tag>
                <span>{t('sla.normalAction')}</span>
              </div>
            </Space>
            <div className="mt-6 grid grid-cols-2 gap-3">
              <Card size="small">
                <Statistic title={t('sla.activeSlas')} value={stats.activeDefinitions} />
              </Card>
              <Card size="small">
                <Statistic title={t('sla.totalViolations')} value={stats.totalViolations} />
              </Card>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}

'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Space, Typography, Tag, Avatar, theme, Button, Progress, Skeleton } from 'antd';
import { Activity, AlertCircle, CheckCircle, RefreshCw, XCircle, Database, Zap, Gauge } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import { formatMetricWindow, formatUptime } from '@/lib/utils/metric-format';
import type { SystemStats } from '@/types/dashboard';

const { Title, Text } = Typography;

type HealthLevel = 'excellent' | 'good' | 'warning' | 'critical';

type MetricStatus = 'success' | 'warning' | 'error' | 'default';

interface MetricCard {
  key: string;
  icon: React.ComponentType<{ className?: string; style?: React.CSSProperties }>;
  name: string;
  value: string;
  status?: MetricStatus;
  statusText?: string;
  percent?: number;
  sub: string;
}

interface ResourceRow {
  key: string;
  name: string;
  value: number | null;
}

// 根据指标阈值计算健康等级
const levelFromRate = (rate: number, good: number, warn: number): HealthLevel =>
  rate <= good ? 'excellent' : rate <= warn ? 'warning' : 'critical';

const LEVEL_ORDER: HealthLevel[] = ['excellent', 'good', 'warning', 'critical'];

export const SystemHealth: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<SystemStats | null>(null);

  const loadStats = useCallback(async () => {
    setLoading(true);
    try {
      const data = await DashboardAPI.getSystemStats();
      setStats(data ?? null);
    } catch (error) {
      console.error('Failed to load system stats:', error);
      setStats(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadStats();
  }, [loadStats]);

  const getHealthStatus = (level: HealthLevel | 'unknown') => {
    switch (level) {
      case 'excellent':
        return { text: t('admin.excellent'), color: token.colorSuccess };
      case 'good':
        return { text: t('admin.good'), color: token.colorInfo };
      case 'warning':
        return { text: t('admin.warning'), color: token.colorWarning };
      case 'critical':
        return { text: t('admin.critical'), color: token.colorError };
      default:
        return { text: t('admin.unknown'), color: token.colorTextSecondary };
    }
  };

  const getHealthIcon = (level: HealthLevel | 'unknown') => {
    switch (level) {
      case 'excellent':
      case 'good':
        return CheckCircle;
      case 'warning':
        return AlertCircle;
      case 'critical':
        return XCircle;
      default:
        return Activity;
    }
  };

  // 只采信后端标记为 measured 的指标：未接入的读数默认是 0，
  // 一起评级会把「没采集到」渲染成「命中率 0%，严重」。
  const deriveLevel = (s: SystemStats): HealthLevel | 'unknown' => {
    const levels: HealthLevel[] = [];
    if (s.availability.cpu === 'measured') levels.push(levelFromRate(s.cpuUsage, 70, 90));
    if (s.availability.memory === 'measured') levels.push(levelFromRate(s.memoryUsage, 70, 90));
    if (s.availability.disk === 'measured') levels.push(levelFromRate(s.diskUsage, 75, 90));
    if (s.availability.http === 'measured') levels.push(levelFromRate(s.errorRate, 1, 3));
    if (s.availability.cache === 'measured') {
      levels.push(levelFromRate(100 - s.cacheHitRate, 5, 20)); // 命中率越低越差
    }
    if (levels.length === 0) return 'unknown';
    return levels.sort((a, b) => LEVEL_ORDER.indexOf(b) - LEVEL_ORDER.indexOf(a))[0];
  };

  if (loading) {
    return (
      <Card title={<Space><Activity className="w-5 h-5" />{t('admin.systemHealth')}</Space>}>
        <Skeleton active paragraph={{ rows: 5 }} />
      </Card>
    );
  }

  const level = stats ? deriveLevel(stats) : 'unknown';
  const healthStatus = getHealthStatus(level);
  const HealthIcon = getHealthIcon(level);
  const notMeasuredText = t('admin.notMeasured');

  const metricCards: MetricCard[] = stats
    ? [
        {
          key: 'db',
          icon: Database,
          name: t('admin.database'),
          value: t('admin.dbConnections', { count: stats.dbConnections }),
          sub:
            stats.availability.dbSize === 'measured'
              ? `${(stats.dbSize / (1024 * 1024)).toFixed(0)} MB`
              : notMeasuredText,
        },
        {
          key: 'cache',
          icon: Zap,
          name: t('admin.cache'),
          value: stats.availability.cache === 'measured' ? `${stats.cacheHitRate.toFixed(1)}%` : notMeasuredText,
          status: stats.availability.cache === 'measured'
            ? (stats.cacheHitRate >= 90 ? 'success' : stats.cacheHitRate >= 70 ? 'warning' : 'error')
            : 'default',
          statusText:
            stats.availability.cache !== 'measured'
              ? notMeasuredText
              : stats.cacheHitRate >= 90
                ? t('admin.excellent')
                : stats.cacheHitRate >= 70
                  ? t('admin.warning')
                  : t('admin.critical'),
          percent: stats.availability.cache === 'measured' ? Math.round(stats.cacheHitRate) : undefined,
          sub: t('admin.cacheHitRate'),
        },
        {
          key: 'api',
          icon: Gauge,
          name: t('admin.apiService'),
          value: stats.availability.http === 'measured' ? `${stats.errorRate.toFixed(2)}%` : notMeasuredText,
          status: stats.availability.http === 'measured'
            ? (stats.errorRate <= 1 ? 'success' : stats.errorRate <= 3 ? 'warning' : 'error')
            : 'default',
          statusText:
            stats.availability.http !== 'measured'
              ? notMeasuredText
              : stats.errorRate <= 1
                ? t('admin.excellent')
                : stats.errorRate <= 3
                  ? t('admin.warning')
                  : t('admin.critical'),
          percent:
            stats.availability.http === 'measured'
              ? Math.min(100, Math.round(stats.errorRate * 10))
              : undefined,
          sub: `${t('admin.avgResponseTime')} ${stats.avgResponseTime.toFixed(0)}ms`,
        },
        {
          key: 'rps',
          icon: Activity,
          name: t('admin.requestsPerSecond'),
          value: stats.availability.http === 'measured' ? stats.requestsPerSecond.toFixed(1) : notMeasuredText,
          sub: `${t('admin.sampleWindow')} ${formatMetricWindow(stats.metricsWindowSeconds)}`,
        },
      ]
    : [];

  const resourceUsage: ResourceRow[] = stats
    ? [
        {
          key: 'cpu',
          name: t('admin.cpuUsage'),
          value: stats.availability.cpu === 'measured' ? stats.cpuUsage : null,
        },
        {
          key: 'memory',
          name: t('admin.memoryUsage'),
          value: stats.availability.memory === 'measured' ? stats.memoryUsage : null,
        },
        {
          key: 'disk',
          name: t('admin.diskUsage'),
          value: stats.availability.disk === 'measured' ? stats.diskUsage : null,
        },
      ]
    : [];

  const sampledAtText =
    stats && !Number.isNaN(Date.parse(stats.sampledAt))
      ? new Date(stats.sampledAt).toLocaleString('zh-CN')
      : '—';

  return (
    <Card
      title={
        <Space>
          <Activity className="w-5 h-5" />
          {t('admin.systemHealth')}
        </Space>
      }
      extra={
        <Button
          type="text"
          icon={<RefreshCw className="w-4 h-4" />}
          size="small"
          aria-label="刷新系统健康状态"
          onClick={() => void loadStats()}
        />
      }
    >
      <div style={{ marginBottom: token.marginLG }}>
        <Space align="center" size="large">
          <Avatar
            size={48}
            style={{ backgroundColor: healthStatus.color, border: 'none' }}
            icon={<HealthIcon className="w-6 h-6" />}
          />
          <div>
            <Title level={3} style={{ margin: 0, color: healthStatus.color }}>
              {healthStatus.text}
            </Title>
            <Text type="secondary">
              {t('admin.uptime')}: {stats ? formatUptime(stats.uptime) : '—'}
            </Text>
          </div>
        </Space>
      </div>

      {/* 核心指标状态 */}
      <div className="grid grid-cols-2 gap-3">
        {metricCards.map(item => {
          const ItemIcon = item.icon;
          return (
            <div
              key={item.key}
              style={{
                border: `1px solid ${token.colorBorder}`,
                borderRadius: token.borderRadius,
                padding: token.paddingSM,
              }}
            >
              <Space align="center" size="small">
                <ItemIcon className="w-4 h-4" style={{ color: token.colorTextSecondary }} />
                <Text style={{ fontSize: 13 }}>{item.name}</Text>
              </Space>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginTop: 6 }}>
                <Text strong style={{ fontSize: 16 }}>{item.value}</Text>
                {item.statusText ? (
                  <Tag color={item.status ?? 'default'} style={{ marginInlineEnd: 0 }}>
                    {item.statusText}
                  </Tag>
                ) : null}
              </div>
              {typeof item.percent === 'number' ? (
                <Progress
                  percent={item.percent}
                  showInfo={false}
                  size="small"
                  strokeColor={
                    item.status === 'success'
                      ? token.colorSuccess
                      : item.status === 'warning'
                        ? token.colorWarning
                        : token.colorError
                  }
                  style={{ marginBottom: 0, marginTop: 4 }}
                />
              ) : null}
              <Text type="secondary" style={{ fontSize: 11 }}>{item.sub}</Text>
            </div>
          );
        })}
      </div>

      {/* 资源使用率 */}
      <div
        style={{
          marginTop: token.marginLG,
          paddingTop: token.paddingSM,
          borderTop: `1px solid ${token.colorBorder}`,
        }}
      >
        {resourceUsage.map(res => (
          <div key={res.key} style={{ marginBottom: 10 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <Text type="secondary" style={{ fontSize: 13 }}>{res.name}</Text>
              <Text style={{ fontSize: 13 }}>
                {res.value === null ? notMeasuredText : `${res.value.toFixed(1)}%`}
              </Text>
            </div>
            {res.value !== null ? (
              <Progress
                percent={Math.round(res.value)}
                showInfo={false}
                strokeColor={res.value <= 70 ? token.colorSuccess : res.value <= 90 ? token.colorWarning : token.colorError}
              />
            ) : null}
          </div>
        ))}
        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
          {t('admin.lastUpdate')}: {sampledAtText}
        </Text>
      </div>
    </Card>
  );
};

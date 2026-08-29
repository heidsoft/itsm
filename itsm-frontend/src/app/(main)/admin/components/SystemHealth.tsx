'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Space, Typography, List, Tag, Avatar, theme, Button, Progress, Skeleton } from 'antd';
import { Activity, AlertCircle, CheckCircle, RefreshCw, XCircle, Database, Zap, Gauge } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { SystemStats } from '@/types/dashboard';

const { Title, Text } = Typography;

type HealthLevel = 'excellent' | 'good' | 'warning' | 'critical';

const formatUptime = (seconds: number): string => {
  if (!Number.isFinite(seconds) || seconds <= 0) return '—';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}天 ${h}小时`;
  if (h > 0) return `${h}小时 ${m}分钟`;
  return `${m}分钟`;
};

// 根据指标阈值计算健康等级
const levelFromRate = (rate: number, good: number, warn: number): HealthLevel =>
  rate <= good ? 'excellent' : rate <= warn ? 'warning' : 'critical';

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
        return { type: 'success' as const, text: t('admin.excellent'), color: token.colorSuccess };
      case 'good':
        return { type: 'info' as const, text: t('admin.good'), color: token.colorInfo };
      case 'warning':
        return { type: 'warning' as const, text: t('admin.warning'), color: token.colorWarning };
      case 'critical':
        return { type: 'error' as const, text: t('admin.critical'), color: token.colorError };
      default:
        return { type: 'default' as const, text: t('admin.unknown'), color: token.colorTextSecondary };
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

  // 基于真实指标推导整体健康等级
  const deriveLevel = (s: SystemStats): HealthLevel => {
    const errorLevel = levelFromRate(s.errorRate, 1, 3);
    const cacheLevel = levelFromRate(100 - s.cacheHitRate, 5, 20); // 命中率越低越差
    const cpuLevel = levelFromRate(s.cpuUsage, 70, 90);
    const memLevel = levelFromRate(s.memoryUsage, 70, 90);
    const diskLevel = levelFromRate(s.diskUsage, 75, 90);
    const order: HealthLevel[] = ['excellent', 'good', 'warning', 'critical'];
    const worst = [errorLevel, cacheLevel, cpuLevel, memLevel, diskLevel].sort(
      (a, b) => order.indexOf(b) - order.indexOf(a)
    )[0];
    return worst;
  };

  if (loading) {
    return (
      <Card title={<Space><Activity className="w-5 h-5" />{t('admin.systemHealth')}</Space>}>
        <Skeleton active paragraph={{ rows: 5 }} />
      </Card>
    );
  }

  const level: HealthLevel | 'unknown' = stats ? deriveLevel(stats) : 'unknown';
  const healthStatus = getHealthStatus(level);
  const HealthIcon = getHealthIcon(level);

  // 指标卡（真实数据）
  const metricCards = stats
    ? [
        {
          key: 'db',
          icon: Database,
          name: t('admin.database'),
          value: `${stats.dbConnections} 连接`,
          status: stats.dbConnections > 0 ? ('success' as const) : ('error' as const),
          statusText: stats.dbConnections > 0 ? t('admin.good') : t('admin.critical'),
          percent: Math.min(100, Math.round((stats.dbSize / (1024 * 1024 * 1024)) * 100) || 0),
          sub: `${(stats.dbSize / (1024 * 1024)).toFixed(0)} MB`,
        },
        {
          key: 'cache',
          icon: Zap,
          name: t('admin.cache'),
          value: `${stats.cacheHitRate.toFixed(1)}%`,
          status: (stats.cacheHitRate >= 90 ? 'success' : stats.cacheHitRate >= 70 ? 'warning' : 'error'),
          statusText: stats.cacheHitRate >= 90 ? t('admin.excellent') : stats.cacheHitRate >= 70 ? t('admin.warning') : t('admin.critical'),
          percent: Math.round(stats.cacheHitRate),
          sub: t('admin.cacheHitRate'),
        },
        {
          key: 'api',
          icon: Gauge,
          name: t('admin.apiService'),
          value: `${stats.errorRate.toFixed(2)}%`,
          status: (stats.errorRate <= 1 ? 'success' : stats.errorRate <= 3 ? 'warning' : 'error'),
          statusText: stats.errorRate <= 1 ? t('admin.excellent') : stats.errorRate <= 3 ? t('admin.warning') : t('admin.critical'),
          percent: Math.min(100, Math.round(stats.errorRate * 10)),
          sub: `${stats.avgResponseTime.toFixed(0)}ms`,
        },
        {
          key: 'rps',
          icon: Activity,
          name: t('admin.requestsPerSecond'),
          value: stats.requestsPerSecond.toFixed(1),
          status: 'success' as const,
          statusText: t('admin.good'),
          percent: Math.min(100, Math.round(stats.requestsPerSecond)),
          sub: t('admin.avgResponseTime'),
        },
      ]
    : [];

  const resourceUsage = stats
    ? [
        { name: t('admin.cpuUsage'), value: stats.cpuUsage, color: token.colorPrimary },
        { name: t('admin.memoryUsage'), value: stats.memoryUsage, color: token.colorInfo },
        { name: t('admin.diskUsage'), value: stats.diskUsage, color: token.colorWarning },
      ]
    : [];

  return (
    <Card
      title={
        <Space>
          <Activity className="w-5 h-5" />
          {t('admin.systemHealth')}
          <Tag color="blue">实时</Tag>
        </Space>
      }
      extra={
        <Button type="text" icon={<RefreshCw className="w-4 h-4" />} size="small" onClick={() => void loadStats()} />
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
      <List
        grid={{ column: 2, gutter: 12 }}
        dataSource={metricCards}
        renderItem={item => {
          const ItemIcon = item.icon;
          return (
            <List.Item>
              <div
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
                  <Tag color={item.status} style={{ marginInlineEnd: 0 }}>{item.statusText}</Tag>
                </div>
                <Progress
                  percent={item.percent}
                  showInfo={false}
                  size="small"
                  strokeColor={item.status === 'success' ? token.colorSuccess : item.status === 'warning' ? token.colorWarning : token.colorError}
                  style={{ marginBottom: 0, marginTop: 4 }}
                />
                <Text type="secondary" style={{ fontSize: 11 }}>{item.sub}</Text>
              </div>
            </List.Item>
          );
        }}
      />

      {/* 资源使用率 */}
      <div
        style={{
          marginTop: token.marginLG,
          paddingTop: token.paddingSM,
          borderTop: `1px solid ${token.colorBorder}`,
        }}
      >
        {resourceUsage.map(res => (
          <div key={res.name} style={{ marginBottom: 10 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <Text type="secondary" style={{ fontSize: 13 }}>{res.name}</Text>
              <Text style={{ fontSize: 13 }}>{res.value.toFixed(1)}%</Text>
            </div>
            <Progress
              percent={Math.round(res.value)}
              showInfo={false}
              strokeColor={res.value <= 70 ? token.colorSuccess : res.value <= 90 ? token.colorWarning : token.colorError}
            />
          </div>
        ))}
        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
          {t('admin.lastUpdate')}: {new Date().toLocaleString('zh-CN')}
        </Text>
      </div>
    </Card>
  );
};

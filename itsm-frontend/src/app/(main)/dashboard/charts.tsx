'use client';

import React, { useEffect, useState } from 'react';
import { App, Card, Empty, Skeleton } from 'antd';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CloudResource } from '@/types/biz/cmdb';
import { useChartTheme } from '@/lib/design-system/chart-theme';

interface ProviderCount {
  name: string;
  value: number;
  color?: string;
}

interface StatusCount {
  name: string;
  value: number;
  color: string;
}

const PROVIDER_LABEL_MAP: Record<string, string> = {
  aliyun: 'Aliyun',
  huawei: 'Huawei Cloud',
  tencent: 'Tencent Cloud',
  azure: 'Azure',
  aws: 'AWS',
  volcengine: 'Volcengine',
  k8s: 'Kubernetes',
  onprem: 'Private Cloud',
  private: 'Private Cloud',
  gcp: 'GCP',
};

const STATUS_LABEL_MAP: Record<string, { label: string; color: string }> = {
  running: { label: 'Running', color: '#22c55e' },
  active: { label: 'Running', color: '#22c55e' },
  stopped: { label: 'Stopped', color: '#64748b' },
  inactive: { label: 'Stopped', color: '#64748b' },
  available: { label: 'Available', color: '#22c55e' },
  unavailable: { label: 'Not available', color: '#ef4444' },
  pending: { label: 'In Progress', color: '#f59e0b' },
  warning: { label: 'Warning', color: '#f59e0b' },
  error: { label: 'Error', color: '#ef4444' },
  failed: { label: 'Failed', color: '#ef4444' },
};

const FALLBACK_COLORS = ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#0891b2', '#7c3aed'];

const aggregateByProvider = (resources: CloudResource[]): ProviderCount[] => {
  const counts = new Map<string, number>();
  for (const resource of resources) {
    const meta = (resource.metadata ?? {}) as Record<string, unknown>;
    const raw =
      (typeof meta.provider === 'string' && meta.provider) ||
      (typeof meta.cloudProvider === 'string' && meta.cloudProvider) ||
      (typeof meta.vendor === 'string' && meta.vendor) ||
      resource.region ||
      'unknown';
    const key = String(raw).toLowerCase();
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  const entries = Array.from(counts.entries());
  if (entries.length === 0) return [];
  return entries.map(([key, value], index) => ({
    name: PROVIDER_LABEL_MAP[key] ?? key,
    value,
    color: FALLBACK_COLORS[index % FALLBACK_COLORS.length],
  }));
};

const aggregateByStatus = (resources: CloudResource[]): StatusCount[] => {
  const counts = new Map<string, number>();
  for (const resource of resources) {
    const raw = (resource.status || 'unknown').toString();
    const key = raw.toLowerCase();
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return Array.from(counts.entries()).map(([key, value]) => {
    const meta = STATUS_LABEL_MAP[key] ?? { label: key, color: '#94a3b8' };
    return {
      name: meta.label,
      value,
      color: meta.color,
    };
  });
};

// 多云资源分布图 — 从 CMDB 云资源接口聚合真实数据
export const ResourceDistributionChart: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<ProviderCount[]>([]);
  const [loading, setLoading] = useState(false);
  const chart = useChartTheme();

  useEffect(() => {
    let isMounted = true;
    const load = async () => {
      setLoading(true);
      try {
        // 分布要按厂商计数，样本必须是全量而不是第一页；页长上限 100，靠翻页读满 200 条。
        const list = await CMDBApi.getAllCloudResources();
        if (!isMounted) return;
        setData(aggregateByProvider(list));
      } catch (error) {
        if (!isMounted) return;
        console.warn('Failed to load cloud resource distribution:', error);
        message.error('Failed to load cloud resource distribution');
        setData([]);
      } finally {
        if (isMounted) setLoading(false);
      }
    };
    load();
    return () => {
      isMounted = false;
    };
  }, [message]);

  if (loading) {
    return <Skeleton active paragraph={{ rows: 4 }} />;
  }

  if (data.length === 0) {
    return <Empty description="No cloud resource distribution data yet" />;
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
        <XAxis
          dataKey="name"
          tick={{ fill: chart.axisText, fontSize: 12 }}
          axisLine={{ stroke: chart.axisLine }}
          tickLine={{ stroke: chart.axisLine }}
        />
        <YAxis
          allowDecimals={false}
          tick={{ fill: chart.axisText, fontSize: 12 }}
          axisLine={{ stroke: chart.axisLine }}
          tickLine={{ stroke: chart.axisLine }}
        />
        <Tooltip
          contentStyle={{
            background: chart.tooltipBg,
            border: `1px solid ${chart.tooltipBorder}`,
            borderRadius: 8,
            color: chart.tooltipText,
          }}
          itemStyle={{ color: chart.tooltipText }}
          labelStyle={{ color: chart.tooltipText }}
        />
        <Legend wrapperStyle={{ color: chart.legendText }} />
        <Bar dataKey="value" name="Number of resources" fill={chart.palette[0]} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
};

// 资源健康状态饼图 — 从 CMDB 云资源接口聚合真实状态分布
export const ResourceHealthPieChart: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<StatusCount[]>([]);
  const [loading, setLoading] = useState(false);
  const chart = useChartTheme();

  useEffect(() => {
    let isMounted = true;
    const load = async () => {
      setLoading(true);
      try {
        const list = await CMDBApi.getAllCloudResources();
        if (!isMounted) return;
        setData(aggregateByStatus(list));
      } catch (error) {
        if (!isMounted) return;
        console.warn('Failed to load resource health status:', error);
        message.error('Failed to load resource health status');
        setData([]);
      } finally {
        if (isMounted) setLoading(false);
      }
    };
    load();
    return () => {
      isMounted = false;
    };
  }, [message]);

  if (loading) {
    return <Skeleton active paragraph={{ rows: 4 }} />;
  }

  if (data.length === 0) {
    return <Empty description="No resource health status data yet" />;
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <PieChart margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          labelLine={false}
          outerRadius={100}
          fill={chart.palette[0]}
          dataKey="value"
          label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${entry.name}-${index}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            background: chart.tooltipBg,
            border: `1px solid ${chart.tooltipBorder}`,
            borderRadius: 8,
            color: chart.tooltipText,
          }}
          itemStyle={{ color: chart.tooltipText }}
          labelStyle={{ color: chart.tooltipText }}
        />
        <Legend wrapperStyle={{ color: chart.legendText }} />
      </PieChart>
    </ResponsiveContainer>
  );
};

// 兼容旧 Card 包装器 — 占位用，后续可由真实组件替换
export const CloudResourceDistributionCard: React.FC<{ title?: string }> = ({ title }) => (
  <Card title={title ?? 'Multi-cloud resource distribution'}>
    <ResourceDistributionChart />
  </Card>
);

// 默认导出 — 保留向后兼容
const Charts = {
  ResourceDistributionChart,
  ResourceHealthPieChart,
};

export default Charts;
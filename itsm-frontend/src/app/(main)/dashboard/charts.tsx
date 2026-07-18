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
  aliyun: '阿里云',
  huawei: '华为云',
  tencent: '腾讯云',
  azure: 'Azure',
  aws: 'AWS',
  onprem: '私有云',
  private: '私有云',
  gcp: 'GCP',
};

const STATUS_LABEL_MAP: Record<string, { label: string; color: string }> = {
  running: { label: '运行中', color: '#22c55e' },
  active: { label: '运行中', color: '#22c55e' },
  stopped: { label: '已停止', color: '#64748b' },
  inactive: { label: '已停止', color: '#64748b' },
  available: { label: '可用', color: '#22c55e' },
  unavailable: { label: '不可用', color: '#ef4444' },
  pending: { label: '处理中', color: '#f59e0b' },
  warning: { label: '警告', color: '#f59e0b' },
  error: { label: '错误', color: '#ef4444' },
  failed: { label: '失败', color: '#ef4444' },
};

const FALLBACK_COLORS = ['#8884d8', '#82ca9d', '#ffc658', '#ff7f50', '#8dd1e1', '#a4de6c'];

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

  useEffect(() => {
    let isMounted = true;
    const load = async () => {
      setLoading(true);
      try {
        // 仅取少量样本即可用于分布展示
        const list = await CMDBApi.getCloudResources({ limit: 200 });
        if (!isMounted) return;
        setData(aggregateByProvider(list || []));
      } catch (error) {
        if (!isMounted) return;
        console.warn('加载云资源分布失败:', error);
        message.error('加载云资源分布失败');
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
    return <Empty description="暂无云资源分布数据" />;
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis dataKey="name" />
        <YAxis allowDecimals={false} />
        <Tooltip />
        <Legend />
        <Bar dataKey="value" name="资源数" fill="#8884d8" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
};

// 资源健康状态饼图 — 从 CMDB 云资源接口聚合真实状态分布
export const ResourceHealthPieChart: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<StatusCount[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let isMounted = true;
    const load = async () => {
      setLoading(true);
      try {
        const list = await CMDBApi.getCloudResources({ limit: 200 });
        if (!isMounted) return;
        setData(aggregateByStatus(list || []));
      } catch (error) {
        if (!isMounted) return;
        console.warn('加载资源健康状态失败:', error);
        message.error('加载资源健康状态失败');
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
    return <Empty description="暂无资源健康状态数据" />;
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
          fill="#8884d8"
          dataKey="value"
          label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${entry.name}-${index}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  );
};

// 兼容旧 Card 包装器 — 占位用，后续可由真实组件替换
export const CloudResourceDistributionCard: React.FC<{ title?: string }> = ({ title }) => (
  <Card title={title ?? '多云资源分布'}>
    <ResourceDistributionChart />
  </Card>
);

// 默认导出 — 保留向后兼容
const Charts = {
  ResourceDistributionChart,
  ResourceHealthPieChart,
};

export default Charts;
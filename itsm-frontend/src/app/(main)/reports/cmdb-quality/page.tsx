'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { App, Button, Card, Col, Empty, Row, Skeleton, Statistic, Typography } from 'antd';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { RotateCcw } from 'lucide-react';

import { CMDBApi } from '@/lib/api/cmdb-api';

const { Title, Text } = Typography;

const COLORS = {
  completeness: '#1890ff',
  accuracy: '#52c41a',
  consistency: '#faad14',
};

interface QualityData {
  name: string;
  count: number;
  completeness: number;
  accuracy: number;
  consistency: number;
}

interface ConfigurationItem {
  id: number;
  name?: string;
  type?: string;
  status?: string;
  environment?: string;
  criticality?: string;
  assetTag?: string;
  serialNumber?: string;
  assignedTo?: string;
  ownedBy?: string;
}

const VALID_STATUSES = new Set(['active', 'inactive', 'maintenance', 'retired']);
const VALID_CRITICALITIES = new Set(['critical', 'high', 'medium', 'low']);
const VALID_ENVIRONMENTS = new Set(['production', 'staging', 'development', 'test']);

// CMDB 数据质量维度定义：
// - completeness：必填核心字段（name/type/status/env/criticality/assetTag/serialNumber/assignedTo/ownedBy）填充率
// - accuracy：status/criticality 取值落在受控词表内的比例（值合法）
// - consistency：environment 与 status 的一致性（生产环境必须是 active，staging 可以是 active/maintenance，
//   其他环境不应为 retired）。不一致的比例即为扣分点。
const computeQualityForGroup = (items: ConfigurationItem[]): Omit<QualityData, 'name' | 'count'> => {
  if (items.length === 0) {
    return { completeness: 0, accuracy: 0, consistency: 0 };
  }

  const requiredFields: Array<keyof ConfigurationItem> = [
    'name',
    'type',
    'status',
    'environment',
    'criticality',
    'assetTag',
    'serialNumber',
    'assignedTo',
    'ownedBy',
  ];

  let completeCount = 0;
  let accurateCount = 0;
  let consistentCount = 0;

  for (const item of items) {
    // completeness
    const filled = requiredFields.filter((f) => {
      const v = item[f];
      return v !== undefined && v !== null && String(v).trim() !== '';
    }).length;
    if (filled === requiredFields.length) completeCount += 1;

    // accuracy
    const statusOk = !!item.status && VALID_STATUSES.has(String(item.status));
    const critOk = !!item.criticality && VALID_CRITICALITIES.has(String(item.criticality));
    if (statusOk && critOk) accurateCount += 1;

    // consistency
    const env = String(item.environment || '');
    const status = String(item.status || '');
    let envStatusOk = false;
    if (!env || !VALID_ENVIRONMENTS.has(env)) {
      // 未填环境按不扣分处理，避免对未打标的 CI 误报
      envStatusOk = true;
    } else if (env === 'production') {
      envStatusOk = status === 'active' || status === 'maintenance';
    } else if (env === 'staging') {
      envStatusOk = status === 'active' || status === 'maintenance' || status === 'inactive';
    } else {
      envStatusOk = status !== 'retired';
    }
    if (envStatusOk) consistentCount += 1;
  }

  return {
    completeness: Math.round((completeCount / items.length) * 100),
    accuracy: Math.round((accurateCount / items.length) * 100),
    consistency: Math.round((consistentCount / items.length) * 100),
  };
};

const CMDBQualityReport = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [qualityData, setQualityData] = useState<QualityData[]>([]);
  const [overall, setOverall] = useState<{
    total: number;
    completeness: number;
    accuracy: number;
    consistency: number;
  } | null>(null);
  const [hasData, setHasData] = useState<boolean>(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setHasData(false);
    try {
      // 拉取 CI 列表（带 stats 用于兜底），按 type 分组后计算真实质量指标。
      // 注意：分页尽量大，但若后端 size 上限有限，分页拉完所有 CIs。
      const collected: ConfigurationItem[] = [];
      const pageSize = 200;
      for (let page = 1; page <= 20; page += 1) {
        const resp: any = await CMDBApi.getCIs({ page, size: pageSize });
        const batch: ConfigurationItem[] = (resp?.items || []) as ConfigurationItem[];
        collected.push(...batch);
        const total = resp?.total ?? collected.length;
        if (batch.length < pageSize || collected.length >= total) break;
      }

      if (collected.length === 0) {
        setQualityData([]);
        setOverall(null);
        setHasData(false);
        return;
      }

      // 按 type 分组
      const groups = new Map<string, ConfigurationItem[]>();
      for (const ci of collected) {
        const key = ci.type || 'unknown';
        if (!groups.has(key)) groups.set(key, []);
        groups.get(key)!.push(ci);
      }

      const sortedTypes = Array.from(groups.entries()).sort((a, b) => b[1].length - a[1].length);
      const data: QualityData[] = sortedTypes.map(([type, items]) => {
        const metrics = computeQualityForGroup(items);
        return {
          name: type,
          count: items.length,
          ...metrics,
        };
      });

      // 整体指标（用全部 CI 重新算一次，避免被分组大小加权稀释）
      const overallMetrics = computeQualityForGroup(collected);
      setQualityData(data);
      setOverall({
        total: collected.length,
        completeness: overallMetrics.completeness,
        accuracy: overallMetrics.accuracy,
        consistency: overallMetrics.consistency,
      });
      setHasData(true);
    } catch (error) {
      console.warn('获取 CMDB 数据失败:', error);
      message.error('加载CMDB质量数据失败');
      setQualityData([]);
      setOverall(null);
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const renderContent = () => {
    if (loading) {
      return (
        <div className="space-y-4">
          <Row gutter={[16, 16]}>
            {[0, 1, 2].map((i) => (
              <Col xs={24} sm={8} key={`skeleton-${i}`}>
                <Card>
                  <Skeleton active />
                </Card>
              </Col>
            ))}
          </Row>
          <Skeleton active paragraph={{ rows: 6 }} />
        </div>
      );
    }

    if (!hasData || qualityData.length === 0) {
      return (
        <Card>
          <Empty description="暂无 CMDB 配置项，无法计算数据质量" />
        </Card>
      );
    }

    return (
      <>
        {/* 整体统计 */}
        <Row gutter={[16, 16]} className="mb-6">
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="配置项总数"
                value={overall?.total ?? 0}
                suffix="项"
                styles={{ content: { color: COLORS.completeness } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="整体完整度"
                value={overall?.completeness ?? 0}
                suffix="%"
                styles={{ content: { color: COLORS.completeness } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="整体准确度"
                value={overall?.accuracy ?? 0}
                suffix="%"
                styles={{ content: { color: COLORS.accuracy } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="整体一致度"
                value={overall?.consistency ?? 0}
                suffix="%"
                styles={{ content: { color: COLORS.consistency } }}
              />
            </Card>
          </Col>
        </Row>

        {/* 质量指标对比图 */}
        <Card title="各类型配置项数据质量对比" className="mb-6">
          <ResponsiveContainer width="100%" height={350}>
            <BarChart data={qualityData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis domain={[0, 100]} />
              <Tooltip content={<QualityTooltip />} />
              <Legend />
              <Bar dataKey="completeness" name="完整度" fill={COLORS.completeness} />
              <Bar dataKey="accuracy" name="准确度" fill={COLORS.accuracy} />
              <Bar dataKey="consistency" name="一致度" fill={COLORS.consistency} />
            </BarChart>
          </ResponsiveContainer>
        </Card>

        {/* 质量趋势 */}
        <Card title="数据质量趋势（按类型）" className="mb-6">
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={qualityData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis domain={[0, 100]} />
              <Tooltip content={<QualityTooltip />} />
              <Legend />
              <Line
                type="monotone"
                dataKey="completeness"
                name="完整度"
                stroke={COLORS.completeness}
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="accuracy"
                name="准确度"
                stroke={COLORS.accuracy}
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="consistency"
                name="一致度"
                stroke={COLORS.consistency}
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        {/* 数量分布参考 */}
        <Card title="各类型配置项数量">
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={qualityData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip content={<CountTooltip />} />
              <Legend />
              <Bar dataKey="count" name="配置项数量" fill={COLORS.completeness} />
            </BarChart>
          </ResponsiveContainer>
        </Card>
      </>
    );
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>CMDB数据质量报表</Title>
        <p className="text-gray-500 mt-1">
          基于配置项实际字段填充、字段取值合法性与环境/状态一致性计算。
          完整度 = 9 项核心字段均已填写；准确度 = status/criticality 在受控词表内；一致度 = 环境与状态组合符合规范。
        </p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">配置项数据质量监控</Text>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
              刷新数据
            </Button>
          </Col>
        </Row>
      </Card>

      {renderContent()}
    </div>
  );
};

interface TooltipPayloadEntry {
  color?: string;
  name?: string;
  value?: number | string;
}

const QualityTooltip = ({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: TooltipPayloadEntry[];
  label?: string | number;
}) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
        <p className="font-semibold text-gray-800 mb-2">{`类型: ${label ?? ''}`}</p>
        {payload.map((entry, index) => (
          <p
            key={`${entry.name ?? 'item'}-${index}`}
            className="text-sm"
            style={{ color: entry.color }}
          >
            {`${entry.name ?? ''}: ${entry.value ?? 0}%`}
          </p>
        ))}
      </div>
    );
  }
  return null;
};

const CountTooltip = ({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: TooltipPayloadEntry[];
  label?: string | number;
}) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
        <p className="font-semibold text-gray-800 mb-2">{`类型: ${label ?? ''}`}</p>
        <p className="text-sm" style={{ color: payload[0].color }}>
          {`配置项数量: ${payload[0].value ?? 0}`}
        </p>
      </div>
    );
  }
  return null;
};

export default CMDBQualityReport;

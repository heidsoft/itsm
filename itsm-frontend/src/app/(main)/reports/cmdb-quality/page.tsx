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
      // 拉取 CI 列表，按 type 分组后计算真实质量指标。
      // 平台页长上限 100，一次性索要 size:200 会被后端回落成默认 20，
      // 所以 getAllCIs 按页翻读到 4000 条上限。
      const collected = await CMDBApi.getAllCIs({}, 4000);

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
      console.warn('Failed to obtain CMDB data:', error);
      message.error('Failed to load CMDB quality data');
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
          <Empty description="There is no CMDB configuration item and data quality cannot be calculated." />
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
                title="Total CIs"
                value={overall?.total ?? 0}
                suffix="items"
                styles={{ content: { color: COLORS.completeness } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="Overall integrity"
                value={overall?.completeness ?? 0}
                suffix="%"
                styles={{ content: { color: COLORS.completeness } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="Overall accuracy"
                value={overall?.accuracy ?? 0}
                suffix="%"
                styles={{ content: { color: COLORS.accuracy } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={6}>
            <Card>
              <Statistic
                title="Overall consistency"
                value={overall?.consistency ?? 0}
                suffix="%"
                styles={{ content: { color: COLORS.consistency } }}
              />
            </Card>
          </Col>
        </Row>

        {/* 质量指标对比图 */}
        <Card title="Comparison of data quality of various types of configuration items" className="mb-6">
          <ResponsiveContainer width="100%" height={350}>
            <BarChart data={qualityData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis domain={[0, 100]} />
              <Tooltip content={<QualityTooltip />} />
              <Legend />
              <Bar dataKey="completeness" name="Completeness" fill={COLORS.completeness} />
              <Bar dataKey="accuracy" name="Accuracy" fill={COLORS.accuracy} />
              <Bar dataKey="consistency" name="Consistency" fill={COLORS.consistency} />
            </BarChart>
          </ResponsiveContainer>
        </Card>

        {/* 质量趋势 */}
        <Card title="Data quality trends by type" className="mb-6">
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
                name="Completeness"
                stroke={COLORS.completeness}
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="accuracy"
                name="Accuracy"
                stroke={COLORS.accuracy}
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="consistency"
                name="Consistency"
                stroke={COLORS.consistency}
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        {/* 数量分布参考 */}
        <Card title="Number of configuration items of each type">
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={qualityData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip content={<CountTooltip />} />
              <Legend />
              <Bar dataKey="count" name="Number of configuration items" fill={COLORS.completeness} />
            </BarChart>
          </ResponsiveContainer>
        </Card>
      </>
    );
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>CMDB data quality report</Title>
        <p className="text-gray-500 mt-1">

          Calculated based on actual field filling of configuration items, legality of field values, and environment/status consistency.
Completeness = 9 core fields are filled in; Accuracy = status/criticality is within the controlled vocabulary; Consistency = Environment and status combinations meet specifications.
        </p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">Configuration item data quality monitoring</Text>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>

              Refresh Data
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
        <p className="font-semibold text-gray-800 mb-2">{`Type: ${label ?? ''}`}</p>
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
        <p className="font-semibold text-gray-800 mb-2">{`Type: ${label ?? ''}`}</p>
        <p className="text-sm" style={{ color: payload[0].color }}>
          {`Number of configuration items: ${payload[0].value ?? 0}`}
        </p>
      </div>
    );
  }
  return null;
};

export default CMDBQualityReport;

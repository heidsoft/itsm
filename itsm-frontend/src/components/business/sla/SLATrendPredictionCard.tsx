'use client';

/**
 * SLA 趋势预测卡片
 *
 * 调 POST /api/v1/ai/predictions（P1#1 落地）：
 *  - 后端契约：itsm-backend/dto/ticket_prediction_dto.go
 *  - 4 种 predictionType: volume | type | priority | resource
 *  - 历史窗口：预测起始日期前 6 个月
 *  - 当前为简化统计版（线性增长），后端已装配 PredictionService
 *
 * 显示未来 6 周每周一个预测点（含置信区间上下界）。
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Empty,
  Radio,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  aiPredict,
  type PredictionDataPoint,
  type TrendPredictionResponse,
  type TrendPredictionType,
} from '@/lib/api/ai-api';
import { useI18n } from '@/lib/i18n/useI18n';
import { TrendingUp, RefreshCw, Sparkles } from 'lucide-react';
import dayjs from 'dayjs';

const { Text } = Typography;

interface Props {
  /** 历史数据起始日（含）；默认今天往前 6 个月 */
  startDate?: string;
  /** 预测窗口结束日；默认今天往后 6 周 */
  endDate?: string;
}

const TYPE_OPTIONS: Array<{ value: TrendPredictionType; label: string }> = [
  { value: 'volume', label: '工单量' },
  { value: 'priority', label: '按优先级' },
  { value: 'type', label: '按类型' },
  { value: 'resource', label: '资源需求' },
];

export const SLATrendPredictionCard: React.FC<Props> = ({
  startDate,
  endDate,
}) => {
  const { t } = useI18n();
  const [type, setType] = useState<TrendPredictionType>('volume');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<TrendPredictionResponse | null>(null);

  const range = useMemo(() => {
    const end = endDate ?? dayjs().add(6, 'week').format('YYYY-MM-DD');
    const start = startDate ?? dayjs().subtract(6, 'month').format('YYYY-MM-DD');
    return [start, end] as [string, string];
  }, [startDate, endDate]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await aiPredict({
        predictionType: type,
        timeRange: range,
      });
      setData(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : '预测失败');
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [type, range]);

  useEffect(() => {
    void load();
  }, [load]);

  const points = data?.predictions ?? [];
  const summary = useMemo(() => {
    if (!points.length) return null;
    const sum = points.reduce((acc, p) => acc + p.predictedValue, 0);
    const avg = sum / points.length;
    const peak = points.reduce(
      (m, p) => (p.predictedValue > m.predictedValue ? p : m),
      points[0],
    );
    return { sum, avg, peak };
  }, [points]);

  return (
    <Card
      title={
        <Space>
          <Sparkles size={16} />
          <span>AI 趋势预测</span>
        </Space>
      }
      extra={
        <Space>
          <Radio.Group
            size="small"
            value={type}
            onChange={e => setType(e.target.value)}
            optionType="button"
          >
            {TYPE_OPTIONS.map(o => (
              <Radio.Button key={o.value} value={o.value}>
                {o.label}
              </Radio.Button>
            ))}
          </Radio.Group>
          <Tooltip title="刷新预测">
            <Button
              size="small"
              icon={<RefreshCw size={14} />}
              onClick={() => void load()}
              loading={loading}
            />
          </Tooltip>
        </Space>
      }
    >
      {error && (
        <Alert
          className="mb-3"
          type="error"
          showIcon
          message="预测请求失败"
          description={error}
          action={
            <Button size="small" onClick={() => void load()}>
              重试
            </Button>
          }
        />
      )}

      <Spin spinning={loading}>
        {!data || points.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无预测数据" />
        ) : (() => {
          // 后端 predictedValue 全 0(历史样本不足)时,即使置信度返回高值也是误导。
          // 这里给出「样本不足」的明确提示,避免渲染一条全 0 的曲线给用户。
          const allZero = points.every(p => !p.predictedValue);
          if (allZero) {
            return (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <Space orientation="vertical" size={4}>
                    <span>当前时间窗口内历史工单样本不足,暂无可信预测。</span>
                    <Text type="secondary">
                      模型:<Tag color="blue">{data.model}</Tag>
                      整体置信度:
                      <Tag color="orange">
                        {(data.confidence * 100).toFixed(0)}%
                      </Tag>
                      (仅供参考,不建议作为决策依据)
                    </Text>
                  </Space>
                }
              />
            );
          }
          return (
            <>
              <Space size="large" className="mb-3" wrap>
                <Text type="secondary">
                  模型：<Tag color="blue">{data.model}</Tag>
                </Text>
                <Text type="secondary">
                  整体置信度：
                  <Tag color={data.confidence >= 0.8 ? 'green' : 'orange'}>
                    {(data.confidence * 100).toFixed(0)}%
                  </Tag>
                </Text>
              {summary && (
                <>
                  <Text type="secondary">
                    平均：<strong>{summary.avg.toFixed(1)}</strong>
                  </Text>
                  <Text type="secondary">
                    峰值：<strong>{summary.peak.predictedValue.toFixed(1)}</strong>
                    <Text type="secondary" className="ml-1 text-xs">
                      ({summary.peak.date})
                    </Text>
                  </Text>
                </>
              )}
            </Space>

            <PredictionList points={points} />
          </>
        );
        })()}
      </Spin>

      <Text type="secondary" className="mt-2 block text-xs">
        预测窗口：{range[0]} ~ {range[1]} · 由 itsm-backend PredictionService 提供
      </Text>
    </Card>
  );
};

/** 极简列表式渲染，避免引入额外图表依赖；后续可换成 G2/echarts */
const PredictionList: React.FC<{ points: PredictionDataPoint[] }> = ({ points }) => {
  const top = points.slice(0, 8);
  return (
    <div className="space-y-2">
      {top.map((p, idx) => {
        const widthPct = Math.min(100, Math.max(8, (p.predictedValue / (top[0].predictedValue || 1)) * 100));
        return (
          <div key={`${p.date}-${idx}`} className="flex items-center gap-3">
            <Text className="w-24 shrink-0 text-xs text-gray-500">{p.date}</Text>
            <div className="relative h-5 flex-1 overflow-hidden rounded bg-gray-100">
              <div
                className="absolute inset-y-0 left-0 bg-blue-400/70"
                style={{ width: `${widthPct}%` }}
                aria-label="预测值"
              />
              <div
                className="absolute inset-y-0 left-0 border-r border-red-400"
                style={{
                  width: `${Math.min(100, Math.max(0, (p.lowerBound / (p.predictedValue || 1)) * widthPct))}%`,
                }}
                aria-label="下界"
              />
              <div
                className="absolute inset-y-0 left-0 border-r-2 border-green-500"
                style={{
                  width: `${Math.min(100, (p.upperBound / (p.predictedValue || 1)) * widthPct)}%`,
                }}
                aria-label="上界"
              />
            </div>
            <Text className="w-20 shrink-0 text-right text-sm font-medium">
              {p.predictedValue.toFixed(1)}
            </Text>
            <Tooltip title={`置信度 ${(p.confidence * 100).toFixed(0)}%`}>
              <TrendingUp size={14} className="shrink-0 text-gray-400" />
            </Tooltip>
          </div>
        );
      })}
    </div>
  );
};

export default SLATrendPredictionCard;
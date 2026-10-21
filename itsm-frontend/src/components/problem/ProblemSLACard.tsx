'use client';

/**
 * 问题详情 SLA 卡片（2026-10-21 GA 落地）
 *
 * 数据源：GET /api/v1/problems/:id/sla（dto.ProblemSLAResponse 镜像）。
 *
 * 渲染红线：`bound=false`（租户没配 problem 类型 SLA 定义 / 优先级无匹配）
 * 必须如实渲染「未绑定 SLA」，绝不能渲染成「SLA 正常」——这正是
 * 旧 503 unready 桩的契约理由：调用方必须能区分「能力没接」与「没触发 SLA」。
 */

import React, { useCallback, useEffect, useState } from 'react';
import { Card, Descriptions, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { ProblemApi, type ProblemSLA } from '@/lib/api/problem-api';

const { Text } = Typography;

const statusMeta: Record<string, { label: string; color: string }> = {
  unbound: { label: '未绑定', color: 'default' },
  active: { label: '进行中', color: 'blue' },
  warning: { label: '临近期限', color: 'orange' },
  breached: { label: '已超时', color: 'red' },
  paused: { label: '已暂停', color: 'purple' },
  met: { label: '已达成', color: 'green' },
};

export interface ProblemSLACardProps {
  problemId: number;
}

export const ProblemSLACard: React.FC<ProblemSLACardProps> = ({ problemId }) => {
  const [sla, setSla] = useState<ProblemSLA | null>(null);
  const [loaded, setLoaded] = useState(false);

  const fetchSLA = useCallback(async () => {
    try {
      const data = await ProblemApi.getProblemSLA(problemId);
      setSla(data);
    } catch {
      // SLA 读取失败时静默隐藏卡片，不阻断详情页主体
      setSla(null);
    } finally {
      setLoaded(true);
    }
  }, [problemId]);

  useEffect(() => {
    if (Number.isFinite(problemId) && problemId > 0) {
      void fetchSLA();
    }
  }, [problemId, fetchSLA]);

  if (!loaded || !sla) return null;

  const fmt = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');
  const meta = statusMeta[sla.status] ?? { label: sla.status, color: 'default' };

  return (
    <Card size="small" title="SLA" className="mt-4" data-testid="problem-sla-card">
      {!sla.bound ? (
        <Text type="secondary">该问题未绑定 SLA（租户未配置 problem 类型的 SLA 定义）</Text>
      ) : (
        <Descriptions size="small" column={1}>
          <Descriptions.Item label="状态">
            <Tag color={meta.color}>{meta.label}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="响应截止">
            <Text type={sla.status === 'breached' ? 'danger' : undefined}>
              {fmt(sla.responseDeadline)}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="解决截止">
            <Text type={sla.status === 'breached' ? 'danger' : undefined}>
              {fmt(sla.resolutionDeadline)}
            </Text>
          </Descriptions.Item>
          {sla.firstResponseAt && (
            <Descriptions.Item label="首次响应">{fmt(sla.firstResponseAt)}</Descriptions.Item>
          )}
          {sla.resolvedAt && (
            <Descriptions.Item label="解决时间">{fmt(sla.resolvedAt)}</Descriptions.Item>
          )}
        </Descriptions>
      )}
    </Card>
  );
};

export default ProblemSLACard;

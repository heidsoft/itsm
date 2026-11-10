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
  unbound: { label: 'Not bound', color: 'default' },
  active: { label: 'In Progress', color: 'blue' },
  warning: { label: 'Deadline approaching', color: 'orange' },
  breached: { label: 'Timed out', color: 'red' },
  paused: { label: 'Paused', color: 'purple' },
  met: { label: 'Achieved', color: 'green' },
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
        <Text type="secondary">The problem is not bound to an SLA (the tenant does not configure an SLA definition of the problem type)</Text>
      ) : (
        <Descriptions size="small" column={1}>
          <Descriptions.Item label="Status">
            <Tag color={meta.color}>{meta.label}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Response Deadline">
            <Text type={sla.status === 'breached' ? 'danger' : undefined}>
              {fmt(sla.responseDeadline)}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="Resolution Deadline">
            <Text type={sla.status === 'breached' ? 'danger' : undefined}>
              {fmt(sla.resolutionDeadline)}
            </Text>
          </Descriptions.Item>
          {sla.firstResponseAt && (
            <Descriptions.Item label="First response">{fmt(sla.firstResponseAt)}</Descriptions.Item>
          )}
          {sla.resolvedAt && (
            <Descriptions.Item label="Resolved At">{fmt(sla.resolvedAt)}</Descriptions.Item>
          )}
        </Descriptions>
      )}
    </Card>
  );
};

export default ProblemSLACard;

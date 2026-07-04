'use client';

/**
 * 问题SLA信息卡片
 * 展示响应/解决倒计时，SLA状态标记
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Tag, Space, Statistic, Row, Col } from 'antd';
import { Clock, AlertTriangle, CheckCircle } from 'lucide-react';
import { ProblemApi, type Problem } from '@/lib/api/problem-api';

interface ProblemSLACardProps {
  problem: Problem;
}

interface SLAInfo {
  responseDeadline?: string;
  resolutionDeadline?: string;
  responseTimeUsed: number;
  resolutionTimeUsed: number;
  responseBreached: boolean;
  resolutionBreached: boolean;
  slaStatus: string;
}

const SLA_STATUS_CONFIG: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
  ok: { color: 'success', text: '正常', icon: <CheckCircle /> },
  warning: { color: 'warning', text: '即将到期', icon: <AlertTriangle /> },
  breached: { color: 'error', text: '已违规', icon: <AlertTriangle /> },
};

const ProblemSLACard: React.FC<ProblemSLACardProps> = ({ problem }) => {
  const [slaInfo, setSlaInfo] = useState<SLAInfo | null>(null);

  const loadSLA = useCallback(async () => {
    if (!problem.id) return;
    try {
      const info = await ProblemApi.getProblemSLA(problem.id);
      setSlaInfo(info as unknown as SLAInfo);
    } catch {
      // SLA信息获取失败时静默处理
    }
  }, [problem.id]);

  useEffect(() => {
    loadSLA();
  }, [loadSLA]);

  // 优先使用已嵌入问题中的SLA字段
  const status = slaInfo?.slaStatus || problem.slaStatus || 'ok';
  const config = SLA_STATUS_CONFIG[status] || SLA_STATUS_CONFIG.ok;

  const formatDeadline = (deadline?: string) => {
    if (!deadline) return '-';
    const d = new Date(deadline);
    return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
  };

  const getTimeRemaining = (deadline?: string) => {
    if (!deadline) return null;
    const remaining = new Date(deadline).getTime() - Date.now();
    if (remaining <= 0) return '已超时';
    const hours = Math.floor(remaining / (1000 * 60 * 60));
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
    if (hours > 24) return `${Math.floor(hours / 24)}天${hours % 24}小时`;
    return `${hours}小时${minutes}分钟`;
  };

  const responseDeadline = slaInfo?.responseDeadline || (problem as any).responseDeadline;
  const resolutionDeadline = slaInfo?.resolutionDeadline || (problem as any).resolutionDeadline;
  const responseBreached = slaInfo?.responseBreached || false;
  const resolutionBreached = slaInfo?.resolutionBreached || false;

  return (
    <Card
      size="small"
      title={
        <Space>
          <Clock />
          <span>SLA信息</span>
          <Tag color={config.color} icon={config.icon}>
            {config.text}
          </Tag>
        </Space>
      }
    >
      <Row gutter={16}>
        <Col span={12}>
          <Statistic
            title="响应截止"
            value={formatDeadline(responseDeadline)}
            valueStyle={{
              fontSize: 14,
              color: responseBreached ? '#ff4d4f' : undefined,
            }}
            suffix={
              !responseBreached && responseDeadline ? (
                <span style={{ fontSize: 12, color: '#999' }}>
                  (剩余 {getTimeRemaining(responseDeadline)})
                </span>
              ) : responseBreached ? (
                <Tag color="error" style={{ marginLeft: 8 }}>已超时</Tag>
              ) : null
            }
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="解决截止"
            value={formatDeadline(resolutionDeadline)}
            valueStyle={{
              fontSize: 14,
              color: resolutionBreached ? '#ff4d4f' : undefined,
            }}
            suffix={
              !resolutionBreached && resolutionDeadline ? (
                <span style={{ fontSize: 12, color: '#999' }}>
                  (剩余 {getTimeRemaining(resolutionDeadline)})
                </span>
              ) : resolutionBreached ? (
                <Tag color="error" style={{ marginLeft: 8 }}>已超时</Tag>
              ) : null
            }
          />
        </Col>
      </Row>
    </Card>
  );
};

export default ProblemSLACard;

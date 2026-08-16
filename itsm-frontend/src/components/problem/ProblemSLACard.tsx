'use client';

/**
 * 问题SLA信息卡片
 * 展示响应/解决倒计时，SLA状态标记
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Tag, Space, Statistic, Row, Col } from 'antd';
import { Clock, AlertTriangle, CheckCircle } from 'lucide-react';
import { ProblemApi, type Problem } from '@/lib/api/problem-api';
import { useI18n } from '@/lib/i18n';

interface ProblemSLACardProps {
  problem: ProblemWithSLA;
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

/**
 * 扩展的 Problem 类型，包含 SLA 相关字段（向后兼容）
 */
interface ProblemWithSLA extends Problem {
  responseDeadline?: string;
  resolutionDeadline?: string;
}

const ProblemSLACard: React.FC<ProblemSLACardProps> = ({ problem }) => {
  const { t } = useI18n();
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

  const SLA_STATUS_CONFIG: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
    ok: { color: 'success', text: t('problemSla.status.ok'), icon: <CheckCircle /> },
    warning: { color: 'warning', text: t('problemSla.status.warning'), icon: <AlertTriangle /> },
    breached: { color: 'error', text: t('problemSla.status.breached'), icon: <AlertTriangle /> },
  };

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
    if (remaining <= 0) return t('problemSla.expired');
    const hours = Math.floor(remaining / (1000 * 60 * 60));
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
    if (hours > 24) {
      return t('problemSla.timeRemaining.daysHours', {
        days: Math.floor(hours / 24),
        hours: hours % 24,
      });
    }
    return t('problemSla.timeRemaining.hoursMinutes', {
      hours,
      minutes,
    });
  };

  const renderRemaining = (breached: boolean, deadline?: string) => {
    if (breached) {
      return <Tag color="error" style={{ marginLeft: 8 }}>{t('problemSla.expired')}</Tag>;
    }
    if (!deadline) return null;
    const remaining = getTimeRemaining(deadline);
    if (!remaining) return null;
    return <span style={{ fontSize: 12, color: '#999' }}>{t('problemSla.remaining', { time: remaining })}</span>;
  };

  const responseDeadline = slaInfo?.responseDeadline || problem.responseDeadline;
  const resolutionDeadline = slaInfo?.resolutionDeadline || problem.resolutionDeadline;
  const responseBreached = slaInfo?.responseBreached || false;
  const resolutionBreached = slaInfo?.resolutionBreached || false;

  return (
    <Card
      size="small"
      title={
        <Space>
          <Clock />
          <span>{t('problemSla.title')}</span>
          <Tag color={config.color} icon={config.icon}>
            {config.text}
          </Tag>
        </Space>
      }
    >
      <Row gutter={16}>
        <Col span={12}>
          <Statistic
            title={t('problemSla.responseDeadline')}
            value={formatDeadline(responseDeadline)}
            valueStyle={{
              fontSize: 14,
              color: responseBreached ? '#ff4d4f' : undefined,
            }}
            suffix={renderRemaining(responseBreached, responseDeadline)}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title={t('problemSla.resolutionDeadline')}
            value={formatDeadline(resolutionDeadline)}
            valueStyle={{
              fontSize: 14,
              color: resolutionBreached ? '#ff4d4f' : undefined,
            }}
            suffix={renderRemaining(resolutionBreached, resolutionDeadline)}
          />
        </Col>
      </Row>
    </Card>
  );
};

export default ProblemSLACard;

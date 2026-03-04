/**
 * SLA 统计卡片组件
 */

import React from 'react';
import { Card, Row, Col, Statistic, Spin } from 'antd';
import {
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
} from 'lucide-react';
import type { SLAStats } from '../types';

interface SLAStatisticsCardsProps {
  stats: SLAStats;
  loading?: boolean;
}

export const SLAStatisticsCards: React.FC<SLAStatisticsCardsProps> = ({
  stats,
  loading = false,
}) => {
  if (loading) {
    return (
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Spin spinning={true} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Spin spinning={true} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Spin spinning={true} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Spin spinning={true} />
          </Card>
        </Col>
      </Row>
    );
  }

  return (
    <Row gutter={16}>
      <Col span={6}>
        <Card>
          <Statistic
            title="总违规数"
            value={stats.total}
            prefix={<AlertTriangle size={18} className="text-orange-500" />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="待处理"
            value={stats.open}
            prefix={<Clock size={18} className="text-red-500" />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="已解决"
            value={stats.resolved}
            prefix={<CheckCircle size={18} className="text-green-500" />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="严重违规"
            value={stats.critical}
            prefix={<XCircle size={18} className="text-red-600" />}
          />
        </Card>
      </Col>
    </Row>
  );
};

SLAStatisticsCards.displayName = 'SLAStatisticsCards';

/**
 * SLA 违规图表面板组件（简化版）
 * 可以集成 ECharts/Recharts 等图表库
 */

import React from 'react';
import { Card, Row, Col, Empty } from 'antd';
import type { SLAViolation } from '../types';

interface SLAChartPanelProps {
  violations: SLAViolation[];
}

export const SLAChartPanel: React.FC<SLAChartPanelProps> = ({ violations }) => {
  // 统计各严重程度的数量
  const severityStats = React.useMemo(() => {
    const stats: Record<string, number> = {};
    violations.forEach(v => {
      stats[v.severity] = (stats[v.severity] || 0) + 1;
    });
    return stats;
  }, [violations]);

  // 统计各状态的数量
  const statusStats = React.useMemo(() => {
    const stats: Record<string, number> = {};
    violations.forEach(v => {
      stats[v.status] = (stats[v.status] || 0) + 1;
    });
    return stats;
  }, [violations]);

  if (violations.length === 0) {
    return (
      <Card title="违规趋势分析">
        <Empty description="暂无数据" />
      </Card>
    );
  }

  return (
    <Row gutter={16}>
      <Col span={12}>
        <Card title="严重程度分布" size="small">
          <div className="space-y-2">
            {Object.entries(severityStats).map(([severity, count]) => (
              <div key={severity} className="flex items-center justify-between">
                <span className="capitalize">{severity}</span>
                <div className="flex items-center gap-2">
                  <div className="w-32 h-2 bg-gray-200 rounded overflow-hidden">
                    <div
                      className="h-full bg-blue-500"
                      style={{ width: `${(count / violations.length) * 100}%` }}
                    />
                  </div>
                  <span className="text-sm font-medium">{count}</span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      </Col>
      <Col span={12}>
        <Card title="状态分布" size="small">
          <div className="space-y-2">
            {Object.entries(statusStats).map(([status, count]) => (
              <div key={status} className="flex items-center justify-between">
                <span className="capitalize">{status}</span>
                <div className="flex items-center gap-2">
                  <div className="w-32 h-2 bg-gray-200 rounded overflow-hidden">
                    <div
                      className="h-full bg-green-500"
                      style={{ width: `${(count / violations.length) * 100}%` }}
                    />
                  </div>
                  <span className="text-sm font-medium">{count}</span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      </Col>
    </Row>
  );
};

SLAChartPanel.displayName = 'SLAChartPanel';

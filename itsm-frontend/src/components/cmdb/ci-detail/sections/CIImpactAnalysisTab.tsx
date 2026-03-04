/**
 * CI 影响分析标签页组件
 */

import React from 'react';
import { Card, Button, Alert, Space, Tag, Typography, List, Tabs, Empty } from 'antd';
import { ClusterOutlined } from '@ant-design/icons';
import type { ImpactAnalysisData, ImpactAnalysisItem, AffectedTicket, AffectedIncident } from '../types';
import { RISK_LEVEL_COLORS, RISK_LEVEL_LABELS } from '../constants';

const { Text } = Typography;

interface CIImpactAnalysisTabProps {
  impactAnalysis: ImpactAnalysisData | null;
  impactLoading: boolean;
  onAnalyze: () => void;
}

export const CIImpactAnalysisTab: React.FC<CIImpactAnalysisTabProps> = ({
  impactAnalysis,
  impactLoading,
  onAnalyze,
}) => {
  const renderImpactList = (
    data: ImpactAnalysisItem[] | undefined,
    emptyText: string
  ) => {
    if (!data || data.length === 0) {
      return <Empty description={emptyText} />;
    }

    return (
      <List
        size="small"
        dataSource={data}
        renderItem={(item: ImpactAnalysisItem) => (
          <List.Item>
            <Space>
              <Tag
                color={
                  item.impact_level === 'critical'
                    ? 'red'
                    : item.impact_level === 'high'
                      ? 'orange'
                      : 'blue'
                }
              >
                {item.impact_level}
              </Tag>
              <Text>{item.ci_name}</Text>
              <Text type="secondary">({item.ci_type})</Text>
              <Text type="secondary">- {item.relationship}</Text>
            </Space>
          </List.Item>
        )}
      />
    );
  };

  const renderAffectedTickets = (
    tickets: AffectedTicket[] | undefined,
    emptyText: string
  ) => {
    if (!tickets || tickets.length === 0) {
      return <Empty description={emptyText} />;
    }

    return (
      <List
        size="small"
        dataSource={tickets}
        renderItem={(item: AffectedTicket) => (
          <List.Item>
            <Space>
              <Tag>{item.status}</Tag>
              <Text>{item.title}</Text>
              <Text type="secondary">优先级: {item.priority}</Text>
            </Space>
          </List.Item>
        )}
      />
    );
  };

  const renderAffectedIncidents = (
    incidents: AffectedIncident[] | undefined,
    emptyText: string
  ) => {
    if (!incidents || incidents.length === 0) {
      return <Empty description={emptyText} />;
    }

    return (
      <List
        size="small"
        dataSource={incidents}
        renderItem={(item: AffectedIncident) => (
          <List.Item>
            <Space>
              <Tag
                color={
                  item.severity === 'critical'
                    ? 'red'
                    : item.severity === 'high'
                      ? 'orange'
                      : 'blue'
                }
              >
                {item.severity}
              </Tag>
              <Text>{item.title}</Text>
              <Tag>{item.status}</Tag>
            </Space>
          </List.Item>
        )}
      />
    );
  };

  return (
    <Card title="CI影响分析" size="small">
      <Button
        type="primary"
        onClick={onAnalyze}
        loading={impactLoading}
        style={{ marginBottom: 16 }}
      >
        执行影响分析
      </Button>

      {impactAnalysis && (
        <>
          <Alert
            message="风险等级"
            description={
              <Space>
                <Tag color={RISK_LEVEL_COLORS[impactAnalysis.risk_level]}>
                  {RISK_LEVEL_LABELS[impactAnalysis.risk_level]}
                </Tag>
                <Text>{impactAnalysis.summary}</Text>
              </Space>
            }
            type={
              impactAnalysis.risk_level === 'critical'
                ? 'error'
                : impactAnalysis.risk_level === 'high'
                  ? 'warning'
                  : 'info'
            }
            style={{ marginBottom: 16 }}
          />

          <Tabs
            items={[
              {
                key: 'upstream',
                label: `上游影响 (${impactAnalysis.upstream_impact?.length || 0})`,
                children: renderImpactList(impactAnalysis.upstream_impact, '无上游依赖'),
              },
              {
                key: 'downstream',
                label: `下游影响 (${impactAnalysis.downstream_impact?.length || 0})`,
                children: renderImpactList(impactAnalysis.downstream_impact, '无下游影响'),
              },
              {
                key: 'tickets',
                label: `关联工单 (${impactAnalysis.affected_tickets?.length || 0})`,
                children: renderAffectedTickets(impactAnalysis.affected_tickets, '无关联工单'),
              },
              {
                key: 'incidents',
                label: `关联事件 (${impactAnalysis.affected_incidents?.length || 0})`,
                children: renderAffectedIncidents(impactAnalysis.affected_incidents, '无关联事件'),
              },
            ]}
          />
        </>
      )}

      {!impactAnalysis && !impactLoading && (
        <Empty description="点击上方按钮执行影响分析" />
      )}
    </Card>
  );
};

CIImpactAnalysisTab.displayName = 'CIImpactAnalysisTab';

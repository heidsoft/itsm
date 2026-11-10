/**
 * CI 影响分析标签页组件
 */

import React from 'react';
import { Card, Button, Alert, Space, Tag, Typography, List, Tabs, Empty } from 'antd';
import { Network } from 'lucide-react';
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
                  item.impactLevel === 'critical'
                    ? 'red'
                    : item.impactLevel === 'high'
                      ? 'orange'
                      : 'blue'
                }
              >
                {item.impactLevel}
              </Tag>
              <Text>{item.ciName}</Text>
              <Text type="secondary">- {item.ciType}</Text>
              <Text type="secondary">- {item.relationship}</Text>
              <Text type="secondary">- Distance: {item.distance}</Text>
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
              <Text type="secondary">Priority: {item.priority}</Text>
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
    <Card title="CI Impact Analysis" size="small">
      <Button
        type="primary"
        onClick={onAnalyze}
        loading={impactLoading}
        style={{ marginBottom: 16 }}
      >

        Perform impact analysis
      </Button>

      {impactAnalysis && (
        <>
          <Alert
            title="Risk Level"
            description={
              <Space>
                <Tag color={RISK_LEVEL_COLORS[impactAnalysis.riskLevel]}>
                  {RISK_LEVEL_LABELS[impactAnalysis.riskLevel]}
                </Tag>
                <Text>{impactAnalysis.summary}</Text>
              </Space>
            }
            type={
              impactAnalysis.riskLevel === 'critical'
                ? 'error'
                : impactAnalysis.riskLevel === 'high'
                  ? 'warning'
                  : 'info'
            }
            style={{ marginBottom: 16 }}
          />

          <Tabs
            items={[
              {
                key: 'upstream',
                label: `Upstream Impact (${impactAnalysis.upstreamImpact?.length || 0})`,
                children: renderImpactList(impactAnalysis.upstreamImpact, 'No upstream dependencies'),
              },
              {
                key: 'downstream',
                label: `Downstream Impact (${impactAnalysis.downstreamImpact?.length || 0})`,
                children: renderImpactList(impactAnalysis.downstreamImpact, 'No downstream effects'),
              },
              {
                key: 'tickets',
                label: `Associated work order (${impactAnalysis.affectedTickets?.length || 0})`,
                children: renderAffectedTickets(impactAnalysis.affectedTickets, 'No related work order'),
              },
              {
                key: 'incidents',
                label: `Correlation event (${impactAnalysis.affectedIncidents?.length || 0})`,
                children: renderAffectedIncidents(impactAnalysis.affectedIncidents, 'No associated events'),
              },
            ]}
          />
        </>
      )}

      {!impactAnalysis && !impactLoading && (
        <Empty description="Click the button above to perform an impact analysis" />
      )}
    </Card>
  );
};

CIImpactAnalysisTab.displayName = 'CIImpactAnalysisTab';

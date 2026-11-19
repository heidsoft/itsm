/**
 * CI 影响分析标签页组件
 */

import React from 'react';
import { Card, Button, Alert, Space, Tag, Typography, List, Tabs, Empty } from 'antd';
import { Network } from 'lucide-react';
import type { ImpactAnalysisData, ImpactAnalysisItem, AffectedTicket, AffectedIncident } from '../types';
import { RISK_LEVEL_COLORS, RISK_LEVEL_LABELS } from '../constants';
import { buildCriticalityOptions, buildEnumLabelResolver } from '../../ci-editor-shared';
import { useI18n } from '@/lib/i18n';

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
  const { t } = useI18n();
  // 影响面里的「优先级/重要性/风险等级」与 create/edit 表单同一枚举，复用同一份词表；
  // 未知值原样回退，避免新枚举被吞成空白。
  const criticalityLabel = buildEnumLabelResolver(buildCriticalityOptions(t));
  const riskLevelLabel = buildEnumLabelResolver(
    Object.entries(RISK_LEVEL_LABELS).map(([value, label]) => ({ value, label }))
  );
  // 工单 status / incident status 目前没有全产品单一词表（散落多份 PRIORITY_LABELS），
  // 这里不做猜测式映射，避免把工单状态套用 CI 状态词表造成语义错配；
  // 统一词表落地后再接（见 CHANGELOG 待办）。
  const rawLabel = (value?: string | null) => value || '-';

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
                  RISK_LEVEL_COLORS[item.impactLevel] ||
                  (item.impactLevel === 'critical'
                    ? 'red'
                    : item.impactLevel === 'high'
                      ? 'orange'
                      : 'blue')
                }
              >
                {riskLevelLabel(item.impactLevel)}
              </Tag>
              <Text>{item.ciName}</Text>
              <Text type="secondary">- {item.ciType}</Text>
              <Text type="secondary">- {item.relationship}</Text>
              <Text type="secondary">- 距离: {item.distance}</Text>
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
              <Tag>{rawLabel(item.status)}</Tag>
              <Text>{item.title}</Text>
              <Text type="secondary">优先级: {criticalityLabel(item.priority)}</Text>
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
                  RISK_LEVEL_COLORS[item.severity] ||
                  (item.severity === 'critical'
                    ? 'red'
                    : item.severity === 'high'
                      ? 'orange'
                      : 'blue')
                }
              >
                {riskLevelLabel(item.severity)}
              </Tag>
              <Text>{item.title}</Text>
              <Tag>{rawLabel(item.status)}</Tag>
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
            title="风险等级"
            description={
              <Space>
                <Tag color={RISK_LEVEL_COLORS[impactAnalysis.riskLevel]}>
                  {riskLevelLabel(impactAnalysis.riskLevel)}
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
                label: `上游影响 (${impactAnalysis.upstreamImpact?.length || 0})`,
                children: renderImpactList(impactAnalysis.upstreamImpact, '无上游依赖'),
              },
              {
                key: 'downstream',
                label: `下游影响 (${impactAnalysis.downstreamImpact?.length || 0})`,
                children: renderImpactList(impactAnalysis.downstreamImpact, '无下游影响'),
              },
              {
                key: 'tickets',
                label: `关联工单 (${impactAnalysis.affectedTickets?.length || 0})`,
                children: renderAffectedTickets(impactAnalysis.affectedTickets, '无关联工单'),
              },
              {
                key: 'incidents',
                label: `关联事件 (${impactAnalysis.affectedIncidents?.length || 0})`,
                children: renderAffectedIncidents(impactAnalysis.affectedIncidents, '无关联事件'),
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

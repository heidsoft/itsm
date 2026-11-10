'use client';

import React, { useEffect, useState } from 'react';
import { Card, Descriptions, Empty, Spin, Tag, Typography, Alert, List, Space } from 'antd';
import { AlertTriangle, Zap, Lightbulb } from 'lucide-react';
import { ChangeApi, type ChangeCMDBImpactSummary } from '@/lib/api/change-api';

const { Text, Title } = Typography;

const riskLevelColors: Record<string, string> = {
  low: 'green',
  medium: 'orange',
  high: 'red',
};

const impactScopeColors: Record<string, string> = {
  low: 'green',
  medium: 'orange',
  high: 'red',
};

const itilPracticeLabels: Record<string, string> = {
  serviceConfigurationManagement: 'Service configuration management',
  changeEnablement: 'Change enabled',
  incidentManagement: 'Incidents',
  riskManagement: 'Risk Management',
  monitoringAndEventManagement: 'Monitoring and Event Management',
};

export interface ChangeCMDBImpactPanelProps {
  changeId: number;
}

export default function ChangeCMDBImpactPanel({ changeId }: ChangeCMDBImpactPanelProps) {
  const [data, setData] = useState<ChangeCMDBImpactSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError(null);
      try {
        const res = await ChangeApi.getCMDBImpactSummary(changeId);
        if (!cancelled) setData(res);
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'LoadingFailed');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    if (changeId) load();
    return () => {
      cancelled = true;
    };
  }, [changeId]);

  if (loading) {
    return (
      <div className="py-8 text-center">
        <Spin />
      </div>
    );
  }

  if (error) {
    return <Alert type="error" title={`CMDB impact summary load failed:${error}`} />;
  }

  if (!data) {
    return <Empty description="No CMDB impact data yet" />;
  }

  return (
    <Space orientation="vertical" size="middle" className="w-full">
      <Card size="small" title={<><Zap />  Overall assessment</>}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="Recommended risk level">
            <Tag color={riskLevelColors[data.recommendedRiskLevel] || 'default'}>
              {data.recommendedRiskLevel?.toUpperCase()}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Recommended scope of influence">
            <Tag color={impactScopeColors[data.recommendedImpactScope] || 'default'}>
              {data.recommendedImpactScope?.toUpperCase()}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Requires review team approval">
            {data.requiresReview ? <Tag color="red">Yes</Tag> : <Tag color="green">No</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="Rollback plan required">
            {data.requiresBackoutPlan ? <Tag color="orange">Yes</Tag> : <Tag color="default">No</Tag>}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card size="small" title="Affected CI">
        <Descriptions column={2} size="small">
          <Descriptions.Item label="Total number of CIs">{data.totalAffectedCIs}</Descriptions.Item>
          <Descriptions.Item label="Key CI number">{data.criticalCICount}</Descriptions.Item>
          <Descriptions.Item label="Number of high-risk dependencies">{data.highRiskDependencyCount}</Descriptions.Item>
          <Descriptions.Item label="Number of unclosed events">{data.openIncidentCount}</Descriptions.Item>
        </Descriptions>
      </Card>

      {data.workflowHints && data.workflowHints.length > 0 && (
        <Card
          size="small"
          title={<><AlertTriangle />  Workflow Tips</>}
        >
          <List
            size="small"
            dataSource={data.workflowHints}
            renderItem={hint => <List.Item>{hint}</List.Item>}
          />
        </Card>
      )}

      {data.itilPractices && data.itilPractices.length > 0 && (
        <Card size="small" title={<><Lightbulb />  ITIL 4 Practical Advice</>}>
          <Space wrap>
            {data.itilPractices.map(p => (
              <Tag key={p} color="blue">
                {itilPracticeLabels[p] || p}
              </Tag>
            ))}
          </Space>
        </Card>
      )}

      {data.affectedCIs && data.affectedCIs.length > 0 && (
        <Card size="small" title="List of affected CI IDs">
          <Space wrap>
            {data.affectedCIs.map(id => (
              <Tag key={id}>#{id}</Tag>
            ))}
          </Space>
        </Card>
      )}
    </Space>
  );
}
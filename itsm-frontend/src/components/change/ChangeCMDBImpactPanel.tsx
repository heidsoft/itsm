'use client';

import React, { useEffect, useState } from 'react';
import { Card, Descriptions, Empty, Spin, Tag, Typography, Alert, List, Space } from 'antd';
import { ThunderboltOutlined, AlertOutlined, BulbOutlined } from '@ant-design/icons';
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
  service_configuration_management: '服务配置管理',
  change_enablement: '变更启用',
  incident_management: '事件管理',
  risk_management: '风险管理',
  monitoring_and_event_management: '监控与事件管理',
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
          setError(e instanceof Error ? e.message : '加载失败');
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
    return <Alert type="error" message={`CMDB 影响摘要加载失败：${error}`} />;
  }

  if (!data) {
    return <Empty description="暂无 CMDB 影响数据" />;
  }

  return (
    <Space direction="vertical" size="middle" className="w-full">
      <Card size="small" title={<><ThunderboltOutlined /> 总体评估</>}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="推荐风险等级">
            <Tag color={riskLevelColors[data.recommendedRiskLevel] || 'default'}>
              {data.recommendedRiskLevel?.toUpperCase()}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="推荐影响范围">
            <Tag color={impactScopeColors[data.recommendedImpactScope] || 'default'}>
              {data.recommendedImpactScope?.toUpperCase()}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="需要 CAB 审批">
            {data.requiresCAB ? <Tag color="red">是</Tag> : <Tag color="green">否</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="需要回滚计划">
            {data.requiresBackoutPlan ? <Tag color="orange">是</Tag> : <Tag color="default">否</Tag>}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card size="small" title="受影响 CI">
        <Descriptions column={2} size="small">
          <Descriptions.Item label="CI 总数">{data.totalAffectedCIs}</Descriptions.Item>
          <Descriptions.Item label="关键 CI 数">{data.criticalCICount}</Descriptions.Item>
          <Descriptions.Item label="高风险依赖数">{data.highRiskDependencyCount}</Descriptions.Item>
          <Descriptions.Item label="未关闭事件数">{data.openIncidentCount}</Descriptions.Item>
        </Descriptions>
      </Card>

      {data.workflowHints && data.workflowHints.length > 0 && (
        <Card
          size="small"
          title={<><AlertOutlined /> 工作流提示</>}
        >
          <List
            size="small"
            dataSource={data.workflowHints}
            renderItem={hint => <List.Item>{hint}</List.Item>}
          />
        </Card>
      )}

      {data.itilPractices && data.itilPractices.length > 0 && (
        <Card size="small" title={<><BulbOutlined /> ITIL 4 实践建议</>}>
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
        <Card size="small" title="受影响 CI ID 列表">
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
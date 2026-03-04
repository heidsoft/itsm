/**
 * 重构后的 CIDetail 组件
 * 简化版：使用拆分的子组件和自定义 hooks
 */

import React from 'react';
import { Card, Tabs, Breadcrumb, Button, Space, Tag, Typography, Result, Skeleton } from 'antd';
import { ArrowLeftOutlined, NodeIndexOutlined, LinkOutlined, ClusterOutlined, HistoryOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';

import { useCIDetail } from './hooks/useCIDetail';
import { STATUS_COLORS } from './constants';
import { CIStatusLabels } from '@/constants/cmdb';
import { CIBasicInfo } from './sections/CIBasicInfo';
import { CITopologyGraph } from './sections/CITopologyGraph';
import { CIRelationshipsTab } from './sections/CIRelationshipsTab';
import { CIImpactAnalysisTab } from './sections/CIImpactAnalysisTab';
import { CIChangeHistoryTab } from './sections/CIChangeHistoryTab';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';

const { Title, Text } = Typography;

export const CIDetail: React.FC = () => {
  const router = useRouter();
  const {
    ci,
    loading,
    impactAnalysis,
    impactLoading,
    changeHistory,
    historyLoading,
    loadImpactAnalysis,
    loadChangeHistory,
    typeInfo,
  } = useCIDetail();

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!ci) {
    return (
      <Card>
        <Result
          status="404"
          title="404"
          subTitle="抱歉，您访问的配置项不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/cmdb')}>
              返回列表
            </Button>
          }
        />
      </Card>
    );
  }

  const tabItems = [
    {
      key: 'basic',
      label: '基本信息',
      children: <CIBasicInfo ci={ci} typeInfo={typeInfo} />,
    },
    {
      key: 'topology',
      label: (
        <span>
          <NodeIndexOutlined /> 服务拓扑
        </span>
      ),
      children: <CITopologyGraph ciId={ci.id} />,
    },
    {
      key: 'relationships',
      label: (
        <span>
          <LinkOutlined /> 关系管理
        </span>
      ),
      children: (
        <CIRelationshipsTab
          ciId={ci.id}
          ciName={ci.name}
          onRefresh={() => {
            // 可以触发拓扑图刷新等
          }}
        />
      ),
    },
    {
      key: 'impact',
      label: (
        <span>
          <ClusterOutlined /> 影响分析
        </span>
      ),
      children: (
        <CIImpactAnalysisTab
          impactAnalysis={impactAnalysis}
          impactLoading={impactLoading}
          onAnalyze={loadImpactAnalysis}
        />
      ),
    },
    {
      key: 'history',
      label: (
        <span>
          <HistoryOutlined /> 变更历史
        </span>
      ),
      children: (
        <CIChangeHistoryTab
          changeHistory={changeHistory}
          historyLoading={historyLoading}
          onLoad={loadChangeHistory}
        />
      ),
    },
  ];

  return (
    <div style={{ padding: '0 0 24px' }}>
      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
          { title: '资产详情' },
        ]}
      />

      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => router.push('/cmdb')}
            style={{ marginBottom: 16 }}
          >
            返回列表
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <Space orientation="vertical" size={0}>
              <Title level={3} style={{ margin: 0 }}>
                {ci.name}
              </Title>
              <Text type="secondary">配置项 ID: {ci.id}</Text>
            </Space>
            <Tag color={STATUS_COLORS[ci.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
              {CIStatusLabels[ci.status] || ci.status}
            </Tag>
          </div>
        </div>

        <Tabs defaultActiveKey="basic" items={tabItems} />
      </Card>
    </div>
  );
};

export default CIDetail;

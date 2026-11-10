'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  App, Button, Card, Col, Descriptions, Empty, Row, Space, Spin, Tag, Typography, Divider,
} from 'antd';
import {
  Activity, CheckCircle, Database, RefreshCw, Wifi, WifiOff, Zap,
} from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import { VectorStoreApi } from '@/lib/api/vector-store-api';
import { formatDateTime } from '@/lib/formatters';
import type { VectorStoreStatus, VectorStoreTestResult } from '@/lib/api/vector-store-api';

const { Text } = Typography;

const CAPABILITY_META: Record<string, { color: string; label: string }> = {
  ready: { color: 'green', label: 'Ready' },
  degraded: { color: 'orange', label: 'Downgrade' },
  unconfigured: { color: 'default', label: 'Not configured' },
  error: { color: 'red', label: 'Error' },
};

export default function VectorStoreAdminPage() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<VectorStoreStatus | null>(null);
  const [testResult, setTestResult] = useState<VectorStoreTestResult | null>(null);
  const [testing, setTesting] = useState(false);

  const loadStatus = useCallback(async () => {
    setLoading(true);
    try {
      const data = await VectorStoreApi.getStatus();
      setStatus(data);
    } catch {
      message.error('Failed to load vector storage state');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    void loadStatus();
  }, [loadStatus]);

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const result = await VectorStoreApi.testConnection();
      setTestResult(result);
      if (result.ok) {
        message.success(`Connectivity test successful (${result.backend}，${result.latencyMs}ms）`);
      } else {
        message.warning(`Connectivity test failed:${result.message}`);
      }
    } catch {
      message.error('Connectivity test request failed');
    } finally {
      setTesting(false);
    }
  };

  const cap = CAPABILITY_META[(status?.capability || '').toLowerCase()];
  const hasVectors = (status?.vectorCount ?? 0) > 0;

  return (
    <PageContainer
      header={{
        title: 'Vector storage configuration',
        breadcrumb: {
          items: [
            { title: 'System Admin' },
            { title: 'Vector storage configuration' },
          ],
        },
      }}
      extra={
        <Space>
          <Button icon={<RefreshCw size={14} />} onClick={loadStatus} loading={loading}>

            Refresh
          </Button>
          <Button
            type="primary"
            icon={<Wifi size={14} />}
            onClick={handleTest}
            loading={testing}
            disabled={!status?.configured}
          >

            Connectivity test
          </Button>
        </Space>
      }
    >
      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="How to configure and test"
        intro="This page is used to view the running status of vector storage (knowledge base RAG semantic retrieval) and test connectivity; the actual configuration is completed in the backend deployment environment variable VECTOR_STORE_CONFIG and is not edited on the page."
        steps={[
          'Keyword search (backend: keyword) is used by default. When the vector library is not configured, the system can still provide a certain search fallback, which is not considered a fault.',
          'To enable semantic retrieval: set VECTOR_STORE_CONFIG for the backend in deployment configuration (docker-compose/K8s), backend optional pgvector/qdrant/milvus, connection parameters written to config, support ${ENV_VAR} references to avoid plaintext keys.',
          'After restarting the backend, click "Refresh" and confirm that the "Configuration Status" is configured and the "Capability Status" is Ready.',
          'Click "Connectivity Test" in the upper right corner to verify the read and write latency; the button is grayed out to indicate that it has not been configured.',
          'After it is ready, go to the knowledge base to re-index the article, and AI Q&A can use semantic search; when the capability status is "downgraded", the search will fall back to keyword results.',
        ]}
      />
      <Spin spinning={loading}>
        {!status && !loading ? (
          <Empty description="Unable to obtain vector storage status" />
        ) : status && (
          <>
            {/* 顶部状态卡片 */}
            <Row gutter={[16, 16]} className="mb-4">
              <Col xs={24} sm={8}>
                <Card size="small" style={{ height: '100%' }}>
                  <div className="flex items-center gap-3">
                    <Database size={20} className="text-blue-500" />
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>Backend engine</Text>
                      <div><Text strong>{status.backend || 'Not configured'}</Text></div>
                    </div>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={8}>
                <Card size="small" style={{ height: '100%' }}>
                  <div className="flex items-center gap-3">
                    {status.configured
                      ? <CheckCircle size={20} className="text-green-500" />
                      : <WifiOff size={20} className="text-gray-400" />}
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>Configuration status</Text>
                      <div>
                        <Tag color={status.configured ? 'green' : 'default'}>
                          {status.configured ? 'Configured' : 'Not configured'}
                        </Tag>
                      </div>
                    </div>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={8}>
                <Card size="small" style={{ height: '100%' }}>
                  <div className="flex items-center gap-3">
                    <Zap size={20} className={cap?.color === 'green' ? 'text-green-500' : 'text-orange-500'} />
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>Capability status</Text>
                      <div>
                        <Tag color={cap?.color || 'default'}>
                          {cap?.label || status.capability || 'Unknown'}
                        </Tag>
                      </div>
                    </div>
                  </div>
                </Card>
              </Col>
            </Row>

            {/* 详情描述 */}
            <Card title="Storage details" className="mb-4">
              <Descriptions column={{ xs: 1, sm: 2 }} size="small" bordered>
                <Descriptions.Item label="Data source">{status.source || '-'}</Descriptions.Item>
                <Descriptions.Item label="Backend engine">{status.backend || '-'}</Descriptions.Item>
                <Descriptions.Item label="Collection">{status.collection || '-'}</Descriptions.Item>
                <Descriptions.Item label="Vector quantity">
                  {hasVectors ? status.vectorCount.toLocaleString() : <Text type="secondary">No data</Text>}
                </Descriptions.Item>
                <Descriptions.Item label="Downgrade rollback">
                  <Tag color={status.fallbackEnabled ? 'blue' : 'default'}>
                    {status.fallbackEnabled ? 'Enabled' : 'Not Enabled'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Check time">
                  {formatDateTime(status.checkedAt) || '-'}
                </Descriptions.Item>
                {status.message && (
                  <Descriptions.Item label="Prompt message" span={2}>
                    <Text type="secondary">{status.message}</Text>
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>

            {/* 配置参数 */}
            {status.settings && Object.keys(status.settings).length > 0 && (
              <Card title="Configuration parameters" className="mb-4">
                <Descriptions column={1} size="small" bordered>
                  {Object.entries(status.settings).map(([k, v]) => (
                    <Descriptions.Item key={k} label={k}>
                      {typeof v === 'string' ? v : JSON.stringify(v)}
                    </Descriptions.Item>
                  ))}
                </Descriptions>
              </Card>
            )}

            {/* 连通性测试结果 */}
            {testResult && (
              <>
                <Divider />
                <Card
                  title={
                    <Space>
                      <Activity size={16} />

                      Connectivity test results
                    </Space>
                  }
                >
                  <Descriptions column={{ xs: 1, sm: 2 }} size="small" bordered>
                    <Descriptions.Item label="Test results">
                      <Tag color={testResult.ok ? 'green' : 'red'} icon={testResult.ok ? <CheckCircle /> : <WifiOff />}>
                        {testResult.ok ? 'Success' : 'Failed'}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="Backend engine">{testResult.backend || '-'}</Descriptions.Item>
                    <Descriptions.Item label="Delay">{testResult.latencyMs}ms</Descriptions.Item>
                    <Descriptions.Item label="Downgrade rollback">
                      <Tag color={testResult.fallback ? 'orange' : 'default'}>
                        {testResult.fallback ? 'Triggered' : 'Not triggered'}
                      </Tag>
                    </Descriptions.Item>
                    {testResult.message && (
                      <Descriptions.Item label="Messages" span={2}>{testResult.message}</Descriptions.Item>
                    )}
                    <Descriptions.Item label="Test time">
                      {formatDateTime(testResult.checkedAt) || '-'}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              </>
            )}
          </>
        )}
      </Spin>
    </PageContainer>
  );
}

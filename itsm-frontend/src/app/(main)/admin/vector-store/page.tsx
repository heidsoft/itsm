'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  App, Button, Card, Col, Descriptions, Empty, Row, Space, Spin, Tag, Typography, Divider,
} from 'antd';
import {
  Activity, CheckCircle, Database, RefreshCw, Wifi, WifiOff, Zap,
} from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import { VectorStoreApi } from '@/lib/api/vector-store-api';
import type { VectorStoreStatus, VectorStoreTestResult } from '@/lib/api/vector-store-api';

const { Text } = Typography;

const CAPABILITY_META: Record<string, { color: string; label: string }> = {
  ready: { color: 'green', label: '就绪' },
  degraded: { color: 'orange', label: '降级' },
  unconfigured: { color: 'default', label: '未配置' },
  error: { color: 'red', label: '错误' },
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
      message.error('加载向量存储状态失败');
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
        message.success(`连通性测试成功（${result.backend}，${result.latencyMs}ms）`);
      } else {
        message.warning(`连通性测试失败：${result.message}`);
      }
    } catch {
      message.error('连通性测试请求失败');
    } finally {
      setTesting(false);
    }
  };

  const cap = CAPABILITY_META[(status?.capability || '').toLowerCase()];
  const hasVectors = (status?.vectorCount ?? 0) > 0;

  return (
    <PageContainer
      header={{
        title: '向量存储配置',
        breadcrumb: {
          items: [
            { title: '系统管理' },
            { title: '向量存储配置' },
          ],
        },
      }}
      extra={
        <Space>
          <Button icon={<RefreshCw size={14} />} onClick={loadStatus} loading={loading}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<Wifi size={14} />}
            onClick={handleTest}
            loading={testing}
            disabled={!status?.configured}
          >
            连通性测试
          </Button>
        </Space>
      }
    >
      <Spin spinning={loading}>
        {!status && !loading ? (
          <Empty description="无法获取向量存储状态" />
        ) : status && (
          <>
            {/* 顶部状态卡片 */}
            <Row gutter={[16, 16]} className="mb-4">
              <Col xs={24} sm={8}>
                <Card size="small" style={{ height: '100%' }}>
                  <div className="flex items-center gap-3">
                    <Database size={20} className="text-blue-500" />
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>后端引擎</Text>
                      <div><Text strong>{status.backend || '未配置'}</Text></div>
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
                      <Text type="secondary" style={{ fontSize: 12 }}>配置状态</Text>
                      <div>
                        <Tag color={status.configured ? 'green' : 'default'}>
                          {status.configured ? '已配置' : '未配置'}
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
                      <Text type="secondary" style={{ fontSize: 12 }}>能力状态</Text>
                      <div>
                        <Tag color={cap?.color || 'default'}>
                          {cap?.label || status.capability || '未知'}
                        </Tag>
                      </div>
                    </div>
                  </div>
                </Card>
              </Col>
            </Row>

            {/* 详情描述 */}
            <Card title="存储详情" className="mb-4">
              <Descriptions column={{ xs: 1, sm: 2 }} size="small" bordered>
                <Descriptions.Item label="数据源">{status.source || '-'}</Descriptions.Item>
                <Descriptions.Item label="后端引擎">{status.backend || '-'}</Descriptions.Item>
                <Descriptions.Item label="Collection">{status.collection || '-'}</Descriptions.Item>
                <Descriptions.Item label="向量数量">
                  {hasVectors ? status.vectorCount.toLocaleString() : <Text type="secondary">暂无数据</Text>}
                </Descriptions.Item>
                <Descriptions.Item label="降级回退">
                  <Tag color={status.fallbackEnabled ? 'blue' : 'default'}>
                    {status.fallbackEnabled ? '已启用' : '未启用'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="检查时间">
                  {status.checkedAt
                    ? new Date(status.checkedAt).toLocaleString('zh-CN', { hour12: false })
                    : '-'}
                </Descriptions.Item>
                {status.message && (
                  <Descriptions.Item label="提示信息" span={2}>
                    <Text type="secondary">{status.message}</Text>
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>

            {/* 配置参数 */}
            {status.settings && Object.keys(status.settings).length > 0 && (
              <Card title="配置参数" className="mb-4">
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
                      连通性测试结果
                    </Space>
                  }
                >
                  <Descriptions column={{ xs: 1, sm: 2 }} size="small" bordered>
                    <Descriptions.Item label="测试结果">
                      <Tag color={testResult.ok ? 'green' : 'red'} icon={testResult.ok ? <CheckCircle /> : <WifiOff />}>
                        {testResult.ok ? '成功' : '失败'}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="后端引擎">{testResult.backend || '-'}</Descriptions.Item>
                    <Descriptions.Item label="延迟">{testResult.latencyMs}ms</Descriptions.Item>
                    <Descriptions.Item label="降级回退">
                      <Tag color={testResult.fallback ? 'orange' : 'default'}>
                        {testResult.fallback ? '已触发' : '未触发'}
                      </Tag>
                    </Descriptions.Item>
                    {testResult.message && (
                      <Descriptions.Item label="消息" span={2}>{testResult.message}</Descriptions.Item>
                    )}
                    <Descriptions.Item label="测试时间">
                      {testResult.checkedAt
                        ? new Date(testResult.checkedAt).toLocaleString('zh-CN', { hour12: false })
                        : '-'}
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

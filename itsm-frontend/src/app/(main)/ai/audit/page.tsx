'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Select,
  Space,
  Tag,
  Typography,
  Progress,
  Drawer,
  Descriptions,
  Empty,
  App,
  Tooltip,
  Button,
} from 'antd';
import { BarChart3, RefreshCw, ShieldCheck, Target } from 'lucide-react';

import {
  aiGetAuditLogs,
  aiGetEvaluation,
  aiUpdateAuditAccepted,
  type AIAuditEntry,
  type AIEvaluationReport,
} from '@/lib/api/ai-api';
import { useAuthStoreHydration } from '@/lib/store/auth-store';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

const SCENARIO_LABELS: Record<string, string> = {
  triage: 'Intelligent triage',
  summarize: 'Work order summary',
  analyze: 'Root Cause Analysis',
  rag_search: 'Knowledge retrieval',
  chat: 'AI dialogue',
  sla_forecast: 'SLA prediction',
};

const scenarioLabel = (kind: string): string => SCENARIO_LABELS[kind] ?? kind;

const pct = (v: number): string => `${Math.round(v * 100)}%`;

const healthColor = (score: number): string => {
  if (score >= 80) return '#52c41a';
  if (score >= 60) return '#faad14';
  return '#ff4d4f';
};

const AIAuditConsole: React.FC = () => {
  const { message } = App.useApp();
  useAuthStoreHydration();

  const [report, setReport] = useState<AIEvaluationReport | null>(null);
  const [evalLoading, setEvalLoading] = useState(false);
  const [logs, setLogs] = useState<AIAuditEntry[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [kindFilter, setKindFilter] = useState<string>('');
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [detail, setDetail] = useState<AIAuditEntry | null>(null);
  const [flipPending, setFlipPending] = useState(false);

  const fetchEvaluation = useCallback(async () => {
    setEvalLoading(true);
    try {
      setReport(await aiGetEvaluation(30));
    } catch (e) {
      message.error(`Failed to load AI evaluation:${(e as Error).message}`);
    } finally {
      setEvalLoading(false);
    }
  }, [message]);

  const fetchLogs = useCallback(
    async (page: number, pageSize: number, kind: string) => {
      setLogsLoading(true);
      try {
        const res = await aiGetAuditLogs({ page, pageSize, kind: kind || undefined, days: 90 });
        // 信封五键由后端单一所有者给出，这里不再用 ?? / || 猜第二套缺省：
        // 静默回退会让页码与真实读数分家。
        setLogs(res.items);
        setPagination({ current: res.page, pageSize: res.pageSize, total: res.total });
      } catch (e) {
        message.error(`Failed to load AI audit log:${(e as Error).message}`);
      } finally {
        setLogsLoading(false);
      }
    },
    [message]
  );

  useEffect(() => {
    fetchEvaluation();
    fetchLogs(1, 20, '');
  }, [fetchEvaluation, fetchLogs]);

  const handleKindChange = (kind: string) => {
    setKindFilter(kind);
    fetchLogs(1, pagination.pageSize, kind);
  };

  const columns = [
    {
      title: 'Timestamp',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: 'Scene',
      dataIndex: 'scenario',
      key: 'scenario',
      width: 110,
      render: (v: string) => <Tag color="geekblue">{scenarioLabel(v)}</Tag>,
    },
    {
      title: 'Input reference',
      dataIndex: 'inputRef',
      key: 'inputRef',
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      title: 'Model',
      dataIndex: 'model',
      key: 'model',
      width: 130,
      render: (v: string) => v || '-',
    },
    {
      title: 'Policy version',
      dataIndex: 'promptVersion',
      key: 'promptVersion',
      width: 110,
      render: (v: string) => (v ? <Tag>{v}</Tag> : '-'),
    },
    {
      title: 'Credibility',
      dataIndex: 'confidence',
      key: 'confidence',
      width: 110,
      render: (v: number) => pct(v ?? 0),
    },
    {
      title: 'Adopted',
      dataIndex: 'accepted',
      key: 'accepted',
      width: 80,
      render: (v: boolean) =>
        v ? <Tag color="success">Adopted</Tag> : <Tag color="default">Not adopted</Tag>,
    },
    {
      title: 'Actions',
      key: 'action',
      width: 80,
      render: (_: unknown, record: AIAuditEntry) => (
        <a onClick={() => setDetail(record)}>Details</a>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
          <Space align="center">
            <ShieldCheck size={22} color="#1677ff" />
            <Title level={4} style={{ margin: 0 }}>

              AI Assessment and Audit Console
            </Title>
          </Space>
          <Space>
            <Select
              style={{ width: 160 }}
              placeholder="All scenes"
              allowClear
              value={kindFilter || undefined}
              onChange={handleKindChange}
              options={Object.entries(SCENARIO_LABELS).map(([k, label]) => ({
                value: k,
                label,
              }))}
            />
            <a onClick={() => { fetchEvaluation(); fetchLogs(1, pagination.pageSize, kindFilter); }}>
              <RefreshCw size={16} />  Refresh
            </a>
          </Space>
        </Space>

        {/* 评估总览 */}
        <Card title="Assessment overview (last 30 days)" loading={evalLoading} extra={<BarChart3 size={16} />}>
          {report?.hasData ? (
            <Space size={32} wrap>
              <div style={{ textAlign: 'center' }}>
                <Progress
                  type="dashboard"
                  percent={Math.round(report.healthScore)}
                  strokeColor={healthColor(report.healthScore)}
                  format={(p) => `${p}`}
                />
                <div>
                  <Text strong>Health points</Text>
                </div>
              </div>
              <div>
                <Descriptions column={2} size="small" colon={false}>
                  <Descriptions.Item label="Feedback sample">
                    <Text strong>{report.totalFeedback}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Effective rate">
                    <Text strong>{pct(report.usefulRate)}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Recommendation adoption rate">
                    <Text strong>{pct(report.acceptedRate)}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Average confidence">
                    <Text strong>{pct(report.avgConfidence)}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Number of model calls">
                    <Text strong>{report.platform.llmCallCount}  times</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Call success rate">
                    <Text strong>{pct(report.platform.successRate)}</Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Avg Response Time">
                    <Text strong>{Math.round(report.platform.avgLatencyMs)} ms</Text>
                  </Descriptions.Item>
                </Descriptions>
              </div>
            </Space>
          ) : (
            <Empty description="No AI evaluation data yet" />
          )}
        </Card>

        {/* 按场景 + 置信度校准 */}
        {report?.hasData && (
          <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
            <Card title="Evaluate by scenario" style={{ flex: 1 }}>
              <Table
                size="small"
                rowKey="kind"
                dataSource={report.byScenario}
                pagination={false}
                columns={[
                  { title: 'Scene', dataIndex: 'kind', render: (v: string) => scenarioLabel(v) },
                  { title: 'Number of samples', dataIndex: 'count', width: 80 },
                  {
                    title: 'Effective rate',
                    dataIndex: 'usefulRate',
                    width: 120,
                    render: (v: number) => (
                      <Progress percent={Math.round(v * 100)} size="small" strokeColor={v >= 0.6 ? '#52c41a' : '#faad14'} />
                    ),
                  },
                  {
                    title: 'Adoption rate',
                    dataIndex: 'acceptedRate',
                    width: 100,
                    render: (v: number) => (v > 0 ? pct(v) : '-'),
                  },
                  {
                    title: 'Average confidence',
                    dataIndex: 'avgConfidence',
                    width: 100,
                    render: (v: number) => pct(v),
                  },
                ]}
              />
            </Card>
            <Card
              title={
                <Space>
                  <Target size={16} />

                  Confidence calibration (high confidence ≠ high usefulness)
                </Space>
              }
              style={{ flex: 1 }}
            >
              <Table
                size="small"
                rowKey="bucket"
                dataSource={report.confidenceCalibration}
                pagination={false}
                columns={[
                  { title: 'Confidence interval', dataIndex: 'bucket', width: 100 },
                  { title: 'Number of samples', dataIndex: 'count', width: 70 },
                  {
                    title: 'Actual useful rate',
                    dataIndex: 'usefulRate',
                    width: 130,
                    render: (v: number, r: { calibrationError: number; count: number }) =>
                      r.count > 0 ? (
                        <Progress
                          percent={Math.round(v * 100)}
                          size="small"
                          strokeColor={r.calibrationError <= 0.15 ? '#52c41a' : '#ff4d4f'}
                        />
                      ) : (
                        '-'
                      ),
                  },
                  {
                    title: 'Calibration deviation',
                    dataIndex: 'calibrationError',
                    width: 90,
                    render: (v: number, r: { count: number }) =>
                      r.count > 0 ? (
                        <Tooltip title="|Actual useful rate - confidence midpoint|, the smaller the value, the more credible the confidence is">
                          <Tag color={v <= 0.15 ? 'green' : v <= 0.3 ? 'orange' : 'red'}>
                            {Math.round(v * 100)}%
                          </Tag>
                        </Tooltip>
                      ) : (
                        '-'
                      ),
                  },
                ]}
              />
            </Card>
          </div>
        )}

        {/* 审计日志 */}
        <Card title={`AI audit log (last 90 days, total ${pagination.total} Article)`}>
          <Table
            rowKey="id"
            size="middle"
            loading={logsLoading}
            dataSource={logs}
            columns={columns}
            pagination={{
              current: pagination.current,
              pageSize: pagination.pageSize,
              total: pagination.total,
              showSizeChanger: true,
              showTotal: (t) => `Total ${t} items`,
              onChange: (page, pageSize) => fetchLogs(page, pageSize, kindFilter),
            }}
            locale={{ emptyText: 'No AI audit record yet' }}
          />
        </Card>
      </Space>

      {/* 审计详情 */}
      <Drawer
        title={`AI Audit Details · ${detail ? scenarioLabel(detail.scenario) : ''}`}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={520}
      >
        {detail && (
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Request ID">{detail.requestId}</Descriptions.Item>
            <Descriptions.Item label="Timestamp">
              {formatDateTime(detail.createdAt)}
            </Descriptions.Item>
            <Descriptions.Item label="Scene">{scenarioLabel(detail.scenario)}</Descriptions.Item>
            <Descriptions.Item label="Input reference">{detail.inputRef || '-'}</Descriptions.Item>
            <Descriptions.Item label="Model">{detail.model || '-'}</Descriptions.Item>
            <Descriptions.Item label="Policy version">{detail.promptVersion || '-'}</Descriptions.Item>
            <Descriptions.Item label="Credibility">{pct(detail.confidence)}</Descriptions.Item>
            <Descriptions.Item label="Adoption results">
              {detail.accepted ? <Tag color="success">Adopted</Tag> : <Tag>Not adopted</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="Actions">
              <Space>
                <Button
                  size="small"
                  type={detail.accepted ? 'default' : 'primary'}
                  disabled={detail.accepted || flipPending}
                  onClick={async () => {
                    if (!detail) return;
                    setFlipPending(true);
                    try {
                      await aiUpdateAuditAccepted(detail.id, true);
                      setLogs(prev => prev.map(e => (
                        e.id === detail.id ? { ...e, accepted: true } : e
                      )));
                      setDetail({ ...detail, accepted: true });
                      message.success(`AI Audit ${detail.id} Marked for adoption`);
                    } catch (err) {
                      message.error((err as Error).message ?? 'Adoption failed');
                    } finally {
                      setFlipPending(false);
                    }
                  }}
                >

                  Adopted
                </Button>
                <Button
                  size="small"
                  danger
                  disabled={!detail.accepted || flipPending}
                  onClick={async () => {
                    if (!detail) return;
                    setFlipPending(true);
                    try {
                      await aiUpdateAuditAccepted(detail.id, false);
                      setLogs(prev => prev.map(e => (
                        e.id === detail.id ? { ...e, accepted: false } : e
                      )));
                      setDetail({ ...detail, accepted: false });
                      message.success(`AI Audit ${detail.id} Flagged for rejection`);
                    } catch (err) {
                      message.error((err as Error).message ?? 'Rejection failed');
                    } finally {
                      setFlipPending(false);
                    }
                  }}
                >

                  Rejected
                </Button>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="Suggested content">
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontSize: 12 }}>
                {detail.suggestion ? JSON.stringify(detail.suggestion, null, 2) : '-'}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="Original comment">
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontSize: 12 }}>
                {detail.notes || '-'}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
};

export default AIAuditConsole;

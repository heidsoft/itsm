'use client';

/**
 * BPMN 节点停留时间分析页面（任务 5）
 *
 * 对应后端 BPMNMonitoringController.GetProcessMetricsByKey
 * 端点：GET /api/v1/bpmn/monitoring/metrics/:processKey
 *
 * 展示：
 *   1. 流程指标（吞吐、成功率、平均耗时）
 *   2. 瓶颈任务列表（含新指标：WaitTimeSeconds / ProcessingTimeSeconds / P95）
 *   3. 最慢路径
 *   4. 资源约束
 *   5. 优化建议
 */

import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Empty,
  Input,
  Progress,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import { Search, Clock, AlertTriangle, BarChart3, Zap, Rocket } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { BPMNMonitoringApi, type BottleneckTask } from '@/lib/api/bpmn-monitoring-api';
import { BPMNDashboardApi, type ProcessStat } from '@/lib/api/bpmn-dashboard-api';

const { Title, Paragraph, Text } = Typography;

const severityColorMap: Record<string, string> = {
  low: 'green',
  medium: 'gold',
  high: 'orange',
  critical: 'red',
};

const bottleneckTypeLabel: Record<string, string> = {
  processing: '处理耗时',
  waiting: '等待耗时',
  resource: '资源受限',
};

function formatSeconds(s: number): string {
  if (!s || s <= 0) return '-';
  if (s < 60) return `${s} 秒`;
  if (s < 3600) return `${(s / 60).toFixed(1)} 分`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} 小时`;
  return `${(s / 86400).toFixed(1)} 天`;
}

export default function BottlenecksPage() {
  const { message } = App.useApp();
  const [processList, setProcessList] = useState<ProcessStat[]>([]);
  const [selectedProcess, setSelectedProcess] = useState<string | null>(null);
  const [keyword, setKeyword] = useState('');
  const [timeRange, setTimeRange] = useState<string>('24h');
  const [loading, setLoading] = useState(false);
  const [analysis, setAnalysis] = useState<{
    bottleneckTasks: BottleneckTask[];
    severity: string;
    recommendations: string[];
    slowestPaths: Array<{
      pathId: string;
      pathName: string;
      totalDuration: string;
      taskCount: number;
      bottleneckTasks: string[];
      optimization: string;
    }>;
    resourceConstraints: Array<{
      resourceType: string;
      resourceName: string;
      utilization: number;
      capacity: number;
      currentLoad: number;
      constraintType: string;
    }>;
  } | null>(null);

  // 加载流程列表
  useEffect(() => {
    let alive = true;
    BPMNDashboardApi.getDashboardMetrics(1)
      .then(metrics => {
        if (!alive) return;
        const list = metrics.topProcesses ?? [];
        setProcessList(list);
        if (list.length > 0 && !selectedProcess) {
          setSelectedProcess(list[0].processDefinitionKey);
        }
      })
      .catch(() => {
        // 静默失败，允许用户手动输入
      });
    return () => {
      alive = false;
    };
  }, [selectedProcess]);

  // 加载瓶颈分析
  useEffect(() => {
    if (!selectedProcess) return;
    let alive = true;
    setLoading(true);
    BPMNMonitoringApi.getBottleneckAnalysis(selectedProcess, timeRange)
      .then(data => {
        if (!alive) return;
        setAnalysis({
          bottleneckTasks: data.bottleneckTasks ?? [],
          slowestPaths: data.slowestPaths ?? [],
          resourceConstraints: data.resourceConstraints ?? [],
          recommendations: data.recommendations ?? [],
          severity: data.severity ?? 'low',
        });
      })
      .catch(err => {
         
        console.error(err);
        if (alive) {
          message.error('加载瓶颈分析失败');
          setAnalysis(null);
        }
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [selectedProcess, timeRange, message]);

  const filteredProcesses = useMemo(() => {
    if (!keyword) return processList;
    const k = keyword.toLowerCase();
    return processList.filter(p => p.processDefinitionKey.toLowerCase().includes(k));
  }, [processList, keyword]);

  const totalWait = useMemo(
    () => (analysis?.bottleneckTasks ?? []).reduce((acc, t) => acc + (t.waitTimeSeconds ?? 0), 0),
    [analysis]
  );
  const totalProcessing = useMemo(
    () => (analysis?.bottleneckTasks ?? []).reduce((acc, t) => acc + (t.processingTimeSeconds ?? 0), 0),
    [analysis]
  );

  const taskColumns = [
    {
      title: '任务',
      key: 'task',
      render: (_: unknown, t: BottleneckTask) => (
        <Space size="small" wrap>
          <Text strong>{t.taskName}</Text>
          <Tag color="default">{t.taskId}</Tag>
          <Tag color="geekblue">{bottleneckTypeLabel[t.bottleneckType] ?? t.bottleneckType}</Tag>
        </Space>
      ),
    },
    {
      title: '影响分',
      dataIndex: 'impactScore',
      key: 'impactScore',
      width: 130,
      render: (v: number) => (
        <Progress
          percent={Math.round(v)}
          size="small"
          status={v >= 80 ? 'exception' : v >= 50 ? 'active' : 'normal'}
        />
      ),
    },
    {
      title: '等待时间',
      key: 'wait',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => (
        <Space size={4}>
          <Clock />
          <Text strong>{formatSeconds(t.waitTimeSeconds)}</Text>
        </Space>
      ),
    },
    {
      title: '处理时间',
      key: 'processing',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => formatSeconds(t.processingTimeSeconds),
    },
    {
      title: 'P95 等待',
      key: 'p95wait',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => formatSeconds(t.p95WaitTimeSeconds),
    },
    {
      title: 'P95 处理',
      key: 'p95processing',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => formatSeconds(t.p95ProcessingTimeSeconds),
    },
    {
      title: '样本数',
      dataIndex: 'sampleCount',
      key: 'sampleCount',
      width: 90,
    },
    {
      title: '处理人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 110,
      render: (v: string) => v || '-',
    },
    {
      title: '队列长度',
      dataIndex: 'queueLength',
      key: 'queueLength',
      width: 90,
    },
    {
      title: '优化建议',
      dataIndex: 'recommendation',
      key: 'recommendation',
      render: (v: string) => (v ? <Text type="secondary">{v}</Text> : '-'),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          <BarChart3 style={{ marginRight: 8 }} />
          BPMN 节点停留时间分析
        </Title>
        <Paragraph type="secondary">
          分析每个 BPMN 节点的等待时间和处理时间，识别流程瓶颈。等待时间 = 节点创建到被领取；处理时间 = 领取到完成；总耗时 = 等待 + 处理。
        </Paragraph>
      </div>

      <Card>
        <Space wrap>
          <Select
            showSearch
            allowClear
            placeholder="选择流程 key"
            style={{ minWidth: 320 }}
            value={selectedProcess ?? undefined}
            onChange={v => setSelectedProcess(v ?? null)}
            options={filteredProcesses.map(p => ({
              label: `${p.processDefinitionKey} (${p.totalInstances} 实例)`,
              value: p.processDefinitionKey,
            }))}
            filterOption={(input, option) =>
              (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
            }
          />
          <Input
            placeholder="或输入流程 key 检索"
            allowClear
            prefix={<Search />}
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
            style={{ width: 240 }}
          />
          <Select
            value={timeRange}
            onChange={setTimeRange}
            style={{ width: 140 }}
            options={[
              { label: '最近 1 小时', value: '1h' },
              { label: '最近 24 小时', value: '24h' },
              { label: '最近 7 天', value: '7d' },
              { label: '最近 30 天', value: '30d' },
            ]}
          />
          <Button
            icon={<Rocket />}
            onClick={() => {
              if (keyword) setSelectedProcess(keyword);
            }}
          >
            分析
          </Button>
        </Space>
      </Card>

      {!selectedProcess ? (
        <Card>
          <Empty description="请选择要分析的流程" />
        </Card>
      ) : (
        <>
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic
                  title="严重程度"
                  value={analysis?.severity?.toUpperCase() ?? '-'}
                  valueStyle={{ color: severityColorMap[analysis?.severity ?? 'low'] ?? 'green' }}
                  prefix={<AlertTriangle />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic
                  title="瓶颈任务数"
                  value={analysis?.bottleneckTasks.length ?? 0}
                  prefix={<Zap />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic title="总等待时间" value={formatSeconds(totalWait)} valueStyle={{ color: '#fa8c16' }} />
              </Card>
            </Col>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic
                  title="总处理时间"
                  value={formatSeconds(totalProcessing)}
                  valueStyle={{ color: '#1677ff' }}
                />
              </Card>
            </Col>
          </Row>

          {analysis?.recommendations && analysis.recommendations.length > 0 && (
            <Alert
              type={analysis.severity === 'critical' ? 'error' : 'warning'}
              showIcon
              message="系统优化建议"
              description={
                <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
                  {analysis.recommendations.map((r, i) => (
                    <li key={i}>{r}</li>
                  ))}
                </ul>
              }
            />
          )}

          <Card title={`瓶颈任务明细 - ${selectedProcess}`} loading={loading}>
            {analysis?.bottleneckTasks.length === 0 ? (
              <Empty description="暂无瓶颈任务" />
            ) : (
              <Table
                rowKey="taskId"
                dataSource={analysis?.bottleneckTasks ?? []}
                columns={taskColumns}
                pagination={false}
                scroll={{ x: 'max-content' }}
                size="small"
              />
            )}
          </Card>

          {analysis && analysis.slowestPaths.length > 0 && (
            <Card title="最慢路径">
              <Table
                rowKey="pathId"
                dataSource={analysis.slowestPaths}
                columns={[
                  { title: '路径', dataIndex: 'pathName', key: 'pathName' },
                  {
                    title: '总耗时',
                    dataIndex: 'totalDuration',
                    key: 'totalDuration',
                    width: 120,
                  },
                  { title: '任务数', dataIndex: 'taskCount', key: 'taskCount', width: 100 },
                  {
                    title: '瓶颈任务',
                    dataIndex: 'bottleneckTasks',
                    key: 'bottleneckTasks',
                    render: (tasks: string[]) => (
                      <Space size={4} wrap>
                        {tasks.map(t => (
                          <Tag key={t} color="red">
                            {t}
                          </Tag>
                        ))}
                      </Space>
                    ),
                  },
                  { title: '优化建议', dataIndex: 'optimization', key: 'optimization' },
                ]}
                pagination={false}
                size="small"
              />
            </Card>
          )}

          {analysis && analysis.resourceConstraints.length > 0 && (
            <Card title="资源约束">
              <Table
                rowKey={r => `${r.resourceType}-${r.resourceName}`}
                dataSource={analysis.resourceConstraints}
                columns={[
                  { title: '类型', dataIndex: 'resourceType', key: 'resourceType' },
                  { title: '资源', dataIndex: 'resourceName', key: 'resourceName' },
                  {
                    title: '利用率',
                    dataIndex: 'utilization',
                    key: 'utilization',
                    render: (v: number) => (
                      <Progress
                        percent={Math.round(v)}
                        size="small"
                        status={v >= 90 ? 'exception' : v >= 70 ? 'active' : 'normal'}
                      />
                    ),
                  },
                  { title: '容量', dataIndex: 'capacity', key: 'capacity' },
                  { title: '当前负载', dataIndex: 'currentLoad', key: 'currentLoad' },
                  {
                    title: '约束',
                    dataIndex: 'constraintType',
                    key: 'constraintType',
                    render: (v: string) => <Tag color="orange">{v}</Tag>,
                  },
                ]}
                pagination={false}
                size="small"
              />
            </Card>
          )}
        </>
      )}
    </div>
  );
}
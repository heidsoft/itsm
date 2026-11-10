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
import type { ColumnsType } from 'antd/es/table';
import { Search, Clock, AlertTriangle, BarChart3, Zap, Rocket } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import {
  BPMNMonitoringApi,
  type BottleneckAnalysis,
  type BottleneckTask,
} from '@/lib/api/bpmn-monitoring-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';

const { Title, Paragraph, Text } = Typography;

const severityColorMap: Record<string, string> = {
  low: 'green',
  medium: 'gold',
  high: 'orange',
  critical: 'red',
};

const bottleneckTypeLabel: Record<string, string> = {
  processing: 'Processing time',
  waiting: 'Waiting time consuming',
  resource: 'Resource limited',
};

// Severity、资源类型、约束类型都是后端枚举，统一映射中文，不直接回显原始值。
const severityLabel: Record<string, string> = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  critical: 'Critical',
};

const resourceTypeLabel: Record<string, string> = {
  human: 'Manual',
  system: 'System',
  external: 'External',
};

const constraintTypeLabel: Record<string, string> = {
  capacity: 'Processing capacity',
  skill: 'Skill matching',
  permission: 'Permissions',
  dependency: 'External dependencies',
};

function formatSeconds(s: number): string {
  if (!s || s <= 0) return '-';
  if (s < 60) return `${s} seconds`;
  if (s < 3600) return `${(s / 60).toFixed(1)} points`;
  if (s < 86400) return `${(s / 3600).toFixed(1)} hours`;
  return `${(s / 86400).toFixed(1)} days`;
}

export default function BottlenecksPage() {
  const { message } = App.useApp();
  const [processList, setProcessList] = useState<{ key: string; name: string }[]>([]);
  const [selectedProcess, setSelectedProcess] = useState<string | null>(null);
  const [keyword, setKeyword] = useState('');
  const [timeRange, setTimeRange] = useState<string>('24h');
  // 列表初值必须为 true：mount 后会发起分析请求，初值 false 会让首帧先渲染
  // 「瓶颈任务数 0」的假统计卡，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [analysis, setAnalysis] = useState<BottleneckAnalysis | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  // 加载流程列表：以流程定义为准（未运行过实例的流程也能分析），
  // 不再用 dashboard topProcesses（仅统计有实例的流程，新环境会恒空）
  useEffect(() => {
    let alive = true;
    WorkflowApi.getWorkflows({ page: 1, pageSize: BACKEND_MAX_PAGE_SIZE })
      .then(result => {
        if (!alive) return;
        const list = (result.workflows || [])
          .map(w => ({ key: w.code, name: w.name }))
          .filter(w => w.key);
        setProcessList(list);
        if (list.length > 0 && !selectedProcess) {
          setSelectedProcess(list[0].key);
        }
      })
      .catch(err => {
        console.error(err);
        if (alive) {
          message.warning('Failed to load the process list, you can directly enter the process key for analysis');
        }
      });
    return () => {
      alive = false;
    };
  }, [selectedProcess, message]);

  // 加载瓶颈分析
  useEffect(() => {
    if (!selectedProcess) return;
    let alive = true;
    setListLoading(true);
    BPMNMonitoringApi.getBottleneckAnalysis(selectedProcess, timeRange)
      .then(data => {
        if (!alive) return;
        setAnalysis(data);
        setLoadError(null);
      })
      .catch(err => {
        console.error(err);
        if (alive) {
          message.error('Loading bottleneck analysis failed');
          setAnalysis(null);
          setLoadError('Bottleneck analysis failed. Try again later or adjust the time range.');
        }
      })
      .finally(() => {
        if (alive) setListLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [selectedProcess, timeRange, message]);

  const filteredProcesses = useMemo(() => {
    if (!keyword) return processList;
    const k = keyword.toLowerCase();
    return processList.filter(
      p => p.key.toLowerCase().includes(k) || p.name?.toLowerCase().includes(k)
    );
  }, [processList, keyword]);

  const totalWait = useMemo(
    () => (analysis?.bottleneckTasks ?? []).reduce((acc, t) => acc + (t.waitTimeSeconds ?? 0), 0),
    [analysis]
  );
  const totalProcessing = useMemo(
    () => (analysis?.bottleneckTasks ?? []).reduce((acc, t) => acc + (t.processingTimeSeconds ?? 0), 0),
    [analysis]
  );

  const taskColumns: ColumnsType<BottleneckTask> = [
    {
      title: 'Task',
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
      title: 'Impact points',
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
      title: 'Waiting time',
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
      title: 'Processing time',
      key: 'processing',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => formatSeconds(t.processingTimeSeconds),
    },
    {
      title: 'P95 waiting',
      key: 'p95wait',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => formatSeconds(t.p95WaitTimeSeconds),
    },
    {
      title: 'P95 processing',
      key: 'p95processing',
      width: 120,
      render: (_: unknown, t: BottleneckTask) => formatSeconds(t.p95ProcessingTimeSeconds),
    },
    {
      title: 'Number of samples',
      dataIndex: 'sampleCount',
      key: 'sampleCount',
      width: 90,
    },
    {
      title: 'Assignee',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 110,
      render: (v: string) => v || '-',
    },
    {
      title: 'Queue length',
      dataIndex: 'queueLength',
      key: 'queueLength',
      width: 90,
    },
    {
      title: 'Optimization Suggestions',
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

          BPMN node residence time analysis
        </Title>
        <Paragraph type="secondary">

          Analyze the waiting time and processing time of each BPMN node to identify process bottlenecks. Waiting time = node creation to collection; processing time = collection to completion; total time spent = waiting + processing.
        </Paragraph>
      </div>

      <Card>
        <Space wrap>
          <Select
            showSearch
            allowClear
            placeholder="Select process key"
            style={{ minWidth: 320 }}
            value={selectedProcess ?? undefined}
            onChange={v => setSelectedProcess(v ?? null)}
            options={filteredProcesses.map(p => ({
              label: p.name ? `${p.name} (${p.key})` : p.key,
              value: p.key,
            }))}
            filterOption={(input, option) =>
              (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
            }
          />
          <Input
            placeholder="Or enter the process key to retrieve"
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
              { label: 'Last 1 hour', value: '1h' },
              { label: 'Last 24 hours', value: '24h' },
              { label: 'Last 7 days', value: '7d' },
              { label: 'Last 30 days', value: '30d' },
            ]}
          />
          <Button
            icon={<Rocket />}
            onClick={() => {
              if (keyword) setSelectedProcess(keyword);
            }}
          >

            Analysis
          </Button>
        </Space>
      </Card>

      {!selectedProcess ? (
        <Card>
          <Empty description="Please select the process to analyze" />
        </Card>
      ) : (
        <>
          {/* 统计卡必须门控：还没拉到分析结果时渲染成「瓶颈任务数 0」会被误读成「没有瓶颈」 */}
          {!listLoading && (
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic
                  title="Severity"
                  value={
                    analysis
                      ? (severityLabel[analysis.severity] ?? analysis.severity)
                      : loadError
                        ? 'Failed to load'
                        : 'No data'
                  }
                  styles={{
                    content: {
                      color: analysis ? (severityColorMap[analysis.severity] ?? undefined) : undefined,
                    },
                  }}
                  prefix={<AlertTriangle />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic
                  title="Number of bottleneck tasks"
                  value={analysis?.bottleneckTasks.length ?? 0}
                  prefix={<Zap />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic title="Total wait time" value={formatSeconds(totalWait)} styles={{ content: { color: '#fa8c16' } }} />
              </Card>
            </Col>
            <Col xs={24} sm={6}>
              <Card>
                <Statistic
                  title="Total processing time"
                  value={formatSeconds(totalProcessing)}
                  styles={{ content: { color: '#1677ff' } }}
                />
              </Card>
            </Col>
          </Row>
          )}

          {analysis?.recommendations && analysis.recommendations.length > 0 && (
            <Alert
              type={analysis.severity === 'critical' ? 'error' : 'warning'}
              showIcon
              title="System optimization suggestions"
              description={
                <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
                  {analysis.recommendations.map((r, i) => (
                    <li key={i}>{r}</li>
                  ))}
                </ul>
              }
            />
          )}

          <Card title={`Bottleneck task details - ${selectedProcess}`} loading={listLoading}>
            {loadError ? (
              <Alert type="error" showIcon title="Failed to load bottleneck analysis" description={loadError} />
            ) : !analysis ? (
              <Empty description="The server returned no bottleneck analysis data" />
            ) : analysis.bottleneckTasks.length === 0 ? (
              <Empty description="No bottleneck tasks" />
            ) : (
              <Table<BottleneckTask>
                rowKey="taskId"
                dataSource={analysis.bottleneckTasks}
                columns={taskColumns}
                pagination={false}
                scroll={{ x: 'max-content' }}
                size="small"
              />
            )}
          </Card>

          {analysis && analysis.slowestPaths.length > 0 && (
            <Card title="Slowest path">
              <Table
                rowKey="pathId"
                dataSource={analysis.slowestPaths}
                columns={[
                  { title: 'path', dataIndex: 'pathName', key: 'pathName' },
                  {
                    title: 'Total time spent',
                    dataIndex: 'totalDurationSeconds',
                    key: 'totalDurationSeconds',
                    width: 120,
                    render: (value: number) => formatSeconds(value),
                  },
                  { title: 'Number of tasks', dataIndex: 'taskCount', key: 'taskCount', width: 100 },
                  {
                    title: 'Bottleneck task',
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
                  { title: 'Optimization Suggestions', dataIndex: 'optimization', key: 'optimization' },
                ]}
                pagination={false}
                size="small"
              />
            </Card>
          )}

          {analysis && analysis.resourceConstraints.length > 0 && (
            <Card title="Resource constraints">
              <Table
                rowKey={r => `${r.resourceType}-${r.resourceName}`}
                dataSource={analysis.resourceConstraints}
                columns={[
                  {
                    title: 'Type',
                    dataIndex: 'resourceType',
                    key: 'resourceType',
                    render: (v: string) => resourceTypeLabel[v] ?? v,
                  },
                  { title: 'Resources', dataIndex: 'resourceName', key: 'resourceName' },
                  {
                    title: 'Utilization',
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
                  { title: 'Capacity', dataIndex: 'capacity', key: 'capacity' },
                  { title: 'Current load', dataIndex: 'currentLoad', key: 'currentLoad' },
                  {
                    title: 'Constraints',
                    dataIndex: 'constraintType',
                    key: 'constraintType',
                    render: (v: string) => (
                      <Tag color="orange">{constraintTypeLabel[v] ?? v}</Tag>
                    ),
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
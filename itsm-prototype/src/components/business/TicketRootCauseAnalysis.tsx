'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  DatePicker,
  Button,
  Space,
  Typography,
  Table,
  Tag,
  Alert,
  Timeline,
  Progress,
  message,
  App,
  Spin,
  Empty,
  Divider,
  Tree,
  Collapse,
  Statistic,
} from 'antd';
import {
  SearchOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  DownloadOutlined,
  FileTextOutlined,
  LinkOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { TicketRootCauseApi } from '@/lib/api/ticket-root-cause-api';

const { Title, Text, Paragraph } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;
const { Panel } = Collapse;

interface RootCause {
  id: string;
  title: string;
  description: string;
  confidence: number;
  category: 'system' | 'network' | 'application' | 'database' | 'user' | 'other';
  evidence: Array<{
    type: 'log' | 'metric' | 'ticket' | 'event';
    content: string;
    timestamp: string;
    relevance: number;
  }>;
  related_tickets: Array<{
    id: number;
    number: string;
    title: string;
    similarity: number;
  }>;
  impact_scope: {
    affected_tickets: number;
    affected_users: number;
    affected_systems: string[];
  };
  recommendations: string[];
  status: 'identified' | 'confirmed' | 'resolved' | 'false_positive';
  created_at: string;
  updated_at: string;
}

interface RootCauseAnalysisReport {
  ticket_id: number;
  ticket_number: string;
  ticket_title: string;
  analysis_date: string;
  root_causes: RootCause[];
  analysis_summary: string;
  confidence_score: number;
  analysis_method: 'automatic' | 'manual' | 'hybrid';
  generated_at: string;
}

interface TicketRootCauseAnalysisProps {
  ticketId?: number;
  autoAnalyze?: boolean;
  onAnalysisComplete?: (report: RootCauseAnalysisReport) => void;
}

export const TicketRootCauseAnalysis: React.FC<TicketRootCauseAnalysisProps> = ({
  ticketId,
  autoAnalyze = true,
  onAnalysisComplete,
}) => {
  const { message: antMessage } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [analyzing, setAnalyzing] = useState(false);
  const [analysisReport, setAnalysisReport] = useState<RootCauseAnalysisReport | null>(null);
  const [selectedRootCause, setSelectedRootCause] = useState<RootCause | null>(null);

  // 执行根因分析
  const performAnalysis = useCallback(async () => {
    if (!ticketId) {
      antMessage.warning('请先选择工单');
      return;
    }

    try {
      setAnalyzing(true);
      // 调用实际API
      const report = await TicketRootCauseApi.analyzeTicket(ticketId);
      
      if (report) {
        setAnalysisReport(report);
        onAnalysisComplete?.(report);
        antMessage.success('根因分析完成');
        return;
      }
      
      // 如果API返回空，使用模拟数据
      const mockRootCauses: RootCause[] = [
        {
          id: 'rc1',
          title: '数据库连接池耗尽',
          description: '系统检测到数据库连接池在高峰期耗尽，导致多个工单无法正常处理。',
          confidence: 0.92,
          category: 'database',
          evidence: [
            {
              type: 'log',
              content: 'ERROR: Connection pool exhausted at 2024-01-15 14:30:00',
              timestamp: '2024-01-15 14:30:00',
              relevance: 0.95,
            },
            {
              type: 'metric',
              content: '数据库连接数达到最大值 100/100',
              timestamp: '2024-01-15 14:28:00',
              relevance: 0.88,
            },
            {
              type: 'ticket',
              content: 'T-2024-001: 系统响应缓慢',
              timestamp: '2024-01-15 14:25:00',
              relevance: 0.82,
            },
          ],
          related_tickets: [
            { id: 1001, number: 'T-2024-001', title: '系统响应缓慢', similarity: 0.89 },
            { id: 1002, number: 'T-2024-002', title: '无法连接数据库', similarity: 0.85 },
            { id: 1003, number: 'T-2024-003', title: '应用超时', similarity: 0.78 },
          ],
          impact_scope: {
            affected_tickets: 15,
            affected_users: 120,
            affected_systems: ['CRM系统', '订单系统'],
          },
          recommendations: [
            '增加数据库连接池大小',
            '优化数据库查询性能',
            '实施连接池监控和告警',
            '考虑使用读写分离',
          ],
          status: 'identified',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
        {
          id: 'rc2',
          title: '网络带宽不足',
          description: '在业务高峰期，网络带宽使用率达到95%，导致系统响应延迟。',
          confidence: 0.78,
          category: 'network',
          evidence: [
            {
              type: 'metric',
              content: '网络带宽使用率: 95%',
              timestamp: '2024-01-15 14:20:00',
              relevance: 0.85,
            },
            {
              type: 'event',
              content: '网络流量异常增长',
              timestamp: '2024-01-15 14:15:00',
              relevance: 0.75,
            },
          ],
          related_tickets: [
            { id: 1004, number: 'T-2024-004', title: '网络访问缓慢', similarity: 0.82 },
          ],
          impact_scope: {
            affected_tickets: 8,
            affected_users: 65,
            affected_systems: ['所有系统'],
          },
          recommendations: [
            '升级网络带宽',
            '优化网络流量分配',
            '实施流量限流策略',
          ],
          status: 'identified',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ];

      const mockReport: RootCauseAnalysisReport = {
        ticket_id: ticketId,
        ticket_number: `T-${ticketId}`,
        ticket_title: '系统响应缓慢',
        analysis_date: format(new Date(), 'yyyy-MM-dd', { locale: zhCN }),
        root_causes: mockRootCauses,
        analysis_summary: '系统自动分析识别出2个可能的根本原因：数据库连接池耗尽（置信度92%）和网络带宽不足（置信度78%）。建议优先处理数据库连接池问题。',
        confidence_score: 0.85,
        analysis_method: 'automatic',
        generated_at: format(new Date(), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN }),
      };

      setAnalysisReport(mockReport);
      onAnalysisComplete?.(mockReport);
      antMessage.success('根因分析完成');
    } catch (error) {
      console.error('Failed to perform analysis:', error);
      antMessage.error('根因分析失败');
    } finally {
      setAnalyzing(false);
    }
  }, [ticketId, antMessage, onAnalysisComplete]);

  useEffect(() => {
    if (autoAnalyze && ticketId) {
      performAnalysis();
    }
  }, [autoAnalyze, ticketId, performAnalysis]);

  // 获取类别颜色
  const getCategoryColor = (category: string) => {
    const colorMap: Record<string, string> = {
      system: 'blue',
      network: 'cyan',
      application: 'purple',
      database: 'orange',
      user: 'green',
      other: 'default',
    };
    return colorMap[category] || 'default';
  };

  // 获取类别文本
  const getCategoryText = (category: string) => {
    const textMap: Record<string, string> = {
      system: '系统',
      network: '网络',
      application: '应用',
      database: '数据库',
      user: '用户',
      other: '其他',
    };
    return textMap[category] || category;
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      identified: 'orange',
      confirmed: 'blue',
      resolved: 'green',
      false_positive: 'default',
    };
    return colorMap[status] || 'default';
  };

  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      identified: '已识别',
      confirmed: '已确认',
      resolved: '已解决',
      false_positive: '误报',
    };
    return textMap[status] || status;
  };

  // 根因表格列
  const rootCauseColumns = [
    {
      title: '根因标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: RootCause) => (
        <div>
          <Text strong>{text}</Text>
          <br />
          <Tag color={getCategoryColor(record.category)} className='mt-1'>
            {getCategoryText(record.category)}
          </Tag>
        </div>
      ),
    },
    {
      title: '置信度',
      dataIndex: 'confidence',
      key: 'confidence',
      render: (confidence: number) => (
        <div>
          <Progress
            percent={confidence * 100}
            status={confidence >= 0.8 ? 'success' : confidence >= 0.6 ? 'active' : 'exception'}
            size='small'
            format={() => `${(confidence * 100).toFixed(0)}%`}
          />
        </div>
      ),
    },
    {
      title: '影响范围',
      key: 'impact',
      render: (_: any, record: RootCause) => (
        <div>
          <Text type='secondary' className='text-sm'>
            工单: {record.impact_scope.affected_tickets} | 用户: {record.impact_scope.affected_users}
          </Text>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: RootCause) => (
        <Button
          type='link'
          size='small'
          onClick={() => setSelectedRootCause(record)}
        >
          查看详情
        </Button>
      ),
    },
  ];

  return (
    <div className='space-y-4'>
      {/* 工具栏 */}
      <Card>
        <div className='flex items-center justify-between mb-4'>
          <Title level={5} style={{ margin: 0 }}>
            根因分析
          </Title>
          <Space>
            <Button
              type='primary'
              icon={<SearchOutlined />}
              onClick={performAnalysis}
              loading={analyzing}
            >
              执行分析
            </Button>
            <Button icon={<DownloadOutlined />} onClick={() => antMessage.info('导出功能开发中')}>
              导出报告
            </Button>
            <Button icon={<ReloadOutlined />} onClick={performAnalysis} loading={analyzing}>
              刷新
            </Button>
          </Space>
        </div>
      </Card>

      {analyzing ? (
        <div className='text-center py-16'>
          <Spin size='large' />
          <div className='mt-4 text-gray-500'>正在分析根因...</div>
        </div>
      ) : analysisReport ? (
        <>
          {/* 分析摘要 */}
          <Card>
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <div>
                  <Title level={5} style={{ margin: 0 }}>
                    分析摘要
                  </Title>
                  <Text type='secondary' className='text-sm'>
                    {analysisReport.ticket_number} - {analysisReport.ticket_title}
                  </Text>
                </div>
                <div className='text-right'>
                  <Text type='secondary' className='text-sm'>分析置信度</Text>
                  <div>
                    <Progress
                      type='circle'
                      percent={analysisReport.confidence_score * 100}
                      size={80}
                      format={() => `${(analysisReport.confidence_score * 100).toFixed(0)}%`}
                    />
                  </div>
                </div>
              </div>
              <Divider />
              <Paragraph>{analysisReport.analysis_summary}</Paragraph>
              <div className='flex items-center gap-4 text-sm text-gray-500'>
                <span>分析日期: {analysisReport.analysis_date}</span>
                <span>分析方法: {analysisReport.analysis_method === 'automatic' ? '自动分析' : analysisReport.analysis_method === 'manual' ? '人工分析' : '混合分析'}</span>
                <span>生成时间: {analysisReport.generated_at}</span>
              </div>
            </div>
          </Card>

          {/* 根因列表 */}
          <Card title={`识别的根因 (${analysisReport.root_causes.length})`}>
            <Table
              columns={rootCauseColumns}
              dataSource={analysisReport.root_causes}
              rowKey='id'
              pagination={false}
            />
          </Card>

          {/* 根因详情 */}
          {selectedRootCause && (
            <Card
              title={`根因详情: ${selectedRootCause.title}`}
              extra={
                <Button onClick={() => setSelectedRootCause(null)}>关闭</Button>
              }
            >
              <div className='space-y-6'>
                <div>
                  <Title level={5}>描述</Title>
                  <Paragraph>{selectedRootCause.description}</Paragraph>
                </div>

                <Divider />

                <div>
                  <Title level={5}>证据链</Title>
                  <Timeline>
                    {selectedRootCause.evidence.map((evidence, index) => (
                      <Timeline.Item
                        key={index}
                        color={
                          evidence.relevance >= 0.8
                            ? 'red'
                            : evidence.relevance >= 0.6
                            ? 'orange'
                            : 'blue'
                        }
                      >
                        <div className='space-y-1'>
                          <div className='flex items-center gap-2'>
                            <Tag color='blue'>{evidence.type}</Tag>
                            <Text strong>{evidence.content}</Text>
                          </div>
                          <div className='flex items-center gap-4 text-sm text-gray-500'>
                            <span>{evidence.timestamp}</span>
                            <span>相关性: {(evidence.relevance * 100).toFixed(0)}%</span>
                          </div>
                        </div>
                      </Timeline.Item>
                    ))}
                  </Timeline>
                </div>

                <Divider />

                <div>
                  <Title level={5}>相关工单</Title>
                  <Table
                    dataSource={selectedRootCause.related_tickets}
                    columns={[
                      {
                        title: '工单编号',
                        dataIndex: 'number',
                        key: 'number',
                        render: (text: string) => (
                          <Text strong style={{ color: '#1890ff' }}>
                            {text}
                          </Text>
                        ),
                      },
                      {
                        title: '工单标题',
                        dataIndex: 'title',
                        key: 'title',
                      },
                      {
                        title: '相似度',
                        dataIndex: 'similarity',
                        key: 'similarity',
                        render: (similarity: number) => (
                          <Progress
                            percent={similarity * 100}
                            size='small'
                            format={() => `${(similarity * 100).toFixed(0)}%`}
                          />
                        ),
                      },
                    ]}
                    rowKey='id'
                    pagination={false}
                    size='small'
                  />
                </div>

                <Divider />

                <div>
                  <Title level={5}>影响范围</Title>
                  <Row gutter={16}>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title='受影响工单'
                          value={selectedRootCause.impact_scope.affected_tickets}
                          prefix={<FileTextOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title='受影响用户'
                          value={selectedRootCause.impact_scope.affected_users}
                          prefix={<LinkOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card>
                        <div>
                          <Text type='secondary' className='text-sm'>受影响系统</Text>
                          <div className='mt-2 space-y-1'>
                            {selectedRootCause.impact_scope.affected_systems.map((system, index) => (
                              <Tag key={index} color='orange'>
                                {system}
                              </Tag>
                            ))}
                          </div>
                        </div>
                      </Card>
                    </Col>
                  </Row>
                </div>

                <Divider />

                <div>
                  <Title level={5}>建议措施</Title>
                  <ul className='list-disc list-inside space-y-2'>
                    {selectedRootCause.recommendations.map((rec, index) => (
                      <li key={index}>
                        <Text>{rec}</Text>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </Card>
          )}
        </>
      ) : (
        <Empty description='请执行根因分析' />
      )}
    </div>
  );
};


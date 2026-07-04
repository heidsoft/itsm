'use client';

/**
 * 增强版问题详情组件
 * 添加解决方案、关联事件、变更管理、知识库集成、审批工作流
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Tag,
  Button,
  Space,
  Skeleton,
  message,
  Typography,
  Tabs,
  List,
  Avatar,
  Timeline,
  Modal,
  Form,
  Input,
  Select,
  Spin,
  Empty,
  Divider,
  Badge,
  Tooltip,
  Descriptions,
  Alert,
  Steps,
  Row,
  Col,
  Result,
} from 'antd';
import {
  EditOutlined,
  ArrowLeftOutlined,
  SearchOutlined,
  SolutionOutlined,
  LinkOutlined,
  BookOutlined,
  HistoryOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlusOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
  AlertOutlined,
} from '@ant-design/icons';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { ProblemApi } from '@/lib/api/';
import { ProblemStatus, ProblemStatusLabels } from '@/constants/problem';
import type { Problem } from '@/types/biz/problem';
import ProblemInvestigationTab from './ProblemInvestigationTab';
import BasicInfoCard from './BasicInfoCard';
import { SafeTextBlock } from '@/components/common/SafeContent';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;

// 问题状态颜色映射
const statusColors: Record<string, string> = {
  [ProblemStatus.OPEN]: 'orange',
  [ProblemStatus.IN_PROGRESS]: 'blue',
  [ProblemStatus.IDENTIFIED]: 'cyan',
  [ProblemStatus.RESOLVED]: 'green',
  [ProblemStatus.CLOSED]: 'default',
};

// 优先级颜色
const priorityColors: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'default',
};

// 关联事件接口
interface RelatedIncident {
  id: number;
  incidentNumber: string;
  title: string;
  status: string;
  priority: string;
  createdAt: string;
}

// 解决方案接口
interface Solution {
  id: number;
  title: string;
  description: string;
  implementedAt: string;
  implementedBy: string;
  effectiveness: 'high' | 'medium' | 'low';
  status: 'proposed' | 'approved' | 'implemented' | 'rejected';
}

// 知识库文章接口
interface KBArticle {
  id: number;
  title: string;
  category: string;
  relevanceScore: number;
}

// 变更请求接口
interface ChangeRequest {
  id: number;
  changeNumber: string;
  title: string;
  status: string;
  riskLevel: string;
  plannedStartDate: string;
}

interface EnhancedProblemDetailProps {
  id?: string;
}

const EnhancedProblemDetail: React.FC<EnhancedProblemDetailProps> = ({ id: propId }) => {
  const params = useParams();
  const router = useRouter();
  const id = propId || (params?.id as string);

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Problem | null>(null);
  const [activeTab, setActiveTab] = useState('basic');

  // 关联数据状态
  const [relatedIncidents, setRelatedIncidents] = useState<RelatedIncident[]>([]);
  const [solutions, setSolutions] = useState<Solution[]>([]);
  const [kbArticles, setKbArticles] = useState<KBArticle[]>([]);
  const [changeRequests, setChangeRequests] = useState<ChangeRequest[]>([]);

  // Modal状态
  const [solutionModalVisible, setSolutionModalVisible] = useState(false);
  const [linkingIncidentModalVisible, setLinkingIncidentModalVisible] = useState(false);

  // 加载数据
  const loadData = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const problem = await ProblemApi.getProblem(Number(id));
      setData(problem as unknown as Problem);
    } catch (error) {
      message.error('加载问题详情失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载关联事件
  const loadRelatedIncidents = async () => {
    try {
      // 模拟API调用 - 实际应该从后端获取
      setRelatedIncidents([
        {
          id: 1,
          incidentNumber: 'INC-2026-001',
          title: '数据库连接超时',
          status: 'resolved',
          priority: 'high',
          createdAt: '2026-06-15T10:00:00Z',
        },
        {
          id: 2,
          incidentNumber: 'INC-2026-002',
          title: '应用响应缓慢',
          status: 'closed',
          priority: 'medium',
          createdAt: '2026-06-16T14:30:00Z',
        },
      ]);
    } catch (error) {
      console.error('Failed to load related incidents:', error);
    }
  };

  // 加载解决方案
  const loadSolutions = async () => {
    try {
      // 模拟数据
      setSolutions([
        {
          id: 1,
          title: '升级数据库连接池配置',
          description: '将连接池大小从50增加到100，并优化超时设置',
          implementedAt: '2026-07-01T09:00:00Z',
          implementedBy: '张三',
          effectiveness: 'high',
          status: 'implemented',
        },
      ]);
    } catch (error) {
      console.error('Failed to load solutions:', error);
    }
  };

  // 加载知识库文章
  const loadKbArticles = async () => {
    try {
      // 模拟数据
      setKbArticles([
        { id: 1, title: '数据库连接池最佳实践', category: '运维', relevanceScore: 0.95 },
        { id: 2, title: 'MySQL性能优化指南', category: '数据库', relevanceScore: 0.88 },
      ]);
    } catch (error) {
      console.error('Failed to load KB articles:', error);
    }
  };

  // 加载关联变更
  const loadChangeRequests = async () => {
    try {
      // 模拟数据
      setChangeRequests([
        {
          id: 1,
          changeNumber: 'CR-2026-015',
          title: '数据库架构优化',
          status: 'completed',
          riskLevel: 'medium',
          plannedStartDate: '2026-07-02T00:00:00Z',
        },
      ]);
    } catch (error) {
      console.error('Failed to load change requests:', error);
    }
  };

  useEffect(() => {
    if (id) {
      loadData();
      loadRelatedIncidents();
      loadSolutions();
      loadKbArticles();
      loadChangeRequests();
    }
  }, [id]);

  // 处理状态更新
  const handleUpdateStatus = async (status: ProblemStatus) => {
    if (!id) return;
    try {
      await ProblemApi.updateProblem(Number(id), { status });
      message.success('状态更新成功');
      loadData();
    } catch (error) {
      message.error('状态更新失败');
    }
  };

  // 加载中状态
  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card>
        <Result
          status="404"
          title="404"
          subTitle="抱歉，您访问的问题不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/problems')}>
              返回列表
            </Button>
          }
        />
      </Card>
    );
  }

  // 状态转换步骤
  const getStatusStep = () => {
    const status = data.status;
    const steps = [
      { title: '待处理', status: 'wait' },
      { title: '调查中', status: 'wait' },
      { title: '已识别', status: 'wait' },
      { title: '已解决', status: 'wait' },
      { title: '已关闭', status: 'wait' },
    ];

    const statusIndex: Record<string, number> = {
      [ProblemStatus.OPEN]: 0,
      [ProblemStatus.IN_PROGRESS]: 1,
      [ProblemStatus.IDENTIFIED]: 2,
      [ProblemStatus.RESOLVED]: 3,
      [ProblemStatus.CLOSED]: 4,
    };

    const currentIndex = statusIndex[status] ?? 0;
    steps.forEach((step, idx) => {
      if (idx < currentIndex) step.status = 'finish';
      else if (idx === currentIndex) step.status = 'process';
    });

    return steps;
  };

  // Tab配置
  const getTabItems = () => [
    {
      key: 'basic',
      label: (
        <span>
          <FileTextOutlined /> 基本信息
        </span>
      ),
      children: <BasicInfoCard data={data} />,
    },
    {
      key: 'investigation',
      label: (
        <span>
          <SearchOutlined /> 问题调查
        </span>
      ),
      children: (
        <ProblemInvestigationTab
          problemId={data.id}
          problemTitle={data.title}
          problemDescription={data.description}
        />
      ),
    },
    {
      key: 'solutions',
      label: (
        <span>
          <SolutionOutlined /> 解决方案 ({solutions.length})
        </span>
      ),
      children: (
        <Card
          title="解决方案"
          extra={
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setSolutionModalVisible(true)}
            >
              添加方案
            </Button>
          }
        >
          {solutions.length > 0 ? (
            <List
              itemLayout="vertical"
              dataSource={solutions}
              renderItem={solution => (
                <List.Item>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <div className="flex items-center justify-between mb-2">
                      <Title level={5} className="!mb-0">
                        {solution.title}
                      </Title>
                      <Space>
                        <Tag
                          color={
                            solution.effectiveness === 'high'
                              ? 'green'
                              : solution.effectiveness === 'medium'
                                ? 'blue'
                                : 'default'
                          }
                        >
                          效果: {solution.effectiveness === 'high' ? '高' : solution.effectiveness === 'medium' ? '中' : '低'}
                        </Tag>
                        <Tag
                          color={
                            solution.status === 'implemented'
                              ? 'success'
                              : solution.status === 'approved'
                                ? 'blue'
                                : solution.status === 'rejected'
                                  ? 'red'
                                  : 'orange'
                          }
                        >
                          {solution.status === 'implemented'
                            ? '已实施'
                            : solution.status === 'approved'
                              ? '已批准'
                              : solution.status === 'rejected'
                                ? '已拒绝'
                                : '待审批'}
                        </Tag>
                      </Space>
                    </div>
                    <Paragraph>{solution.description}</Paragraph>
                    <div className="text-sm text-gray-500">
                      实施时间: {dayjs(solution.implementedAt).format('YYYY-MM-DD HH:mm')} |
                      实施人: {solution.implementedBy}
                    </div>
                  </div>
                </List.Item>
              )}
            />
          ) : (
            <Empty description="暂无解决方案" />
          )}
        </Card>
      ),
    },
    {
      key: 'incidents',
      label: (
        <span>
          <LinkOutlined /> 关联事件 ({relatedIncidents.length})
        </span>
      ),
      children: (
        <Card
          title="关联事件"
          extra={
            <Button icon={<LinkOutlined />} onClick={() => setLinkingIncidentModalVisible(true)}>
              关联事件
            </Button>
          }
        >
          {relatedIncidents.length > 0 ? (
            <List
              itemLayout="horizontal"
              dataSource={relatedIncidents}
              renderItem={incident => (
                <List.Item
                  actions={[
                    <Button
                      type="link"
                      key="view"
                      onClick={() => router.push(`/incidents/${incident.id}`)}
                    >
                      查看
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={
                      <Avatar
                        style={{
                          backgroundColor:
                            incident.priority === 'critical'
                              ? '#f5222d'
                              : incident.priority === 'high'
                                ? '#fa8c16'
                                : '#1890ff',
                        }}
                      >
                        <AlertOutlined />
                      </Avatar>
                    }
                    title={
                      <Space>
                        <Text strong>{incident.incidentNumber}</Text>
                        <Tag>{incident.title}</Tag>
                      </Space>
                    }
                    description={
                      <Space>
                        <Tag color={priorityColors[incident.priority]}>{incident.priority}</Tag>
                        <Tag color={statusColors[incident.status] || 'default'}>
                          {incident.status}
                        </Tag>
                        <Text type="secondary">
                          {dayjs(incident.createdAt).format('YYYY-MM-DD HH:mm')}
                        </Text>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          ) : (
            <Empty description="暂无关联事件" />
          )}
        </Card>
      ),
    },
    {
      key: 'changes',
      label: (
        <span>
          <HistoryOutlined /> 关联变更 ({changeRequests.length})
        </span>
      ),
      children: (
        <Card title="关联变更">
          {changeRequests.length > 0 ? (
            <List
              itemLayout="horizontal"
              dataSource={changeRequests}
              renderItem={cr => (
                <List.Item
                  actions={[
                    <Button type="link" key="view" onClick={() => router.push(`/changes/${cr.id}`)}>
                      查看
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={
                      <Avatar style={{ backgroundColor: '#1890ff' }}>
                        <HistoryOutlined />
                      </Avatar>
                    }
                    title={
                      <Space>
                        <Text strong>{cr.changeNumber}</Text>
                        <Tag>{cr.title}</Tag>
                      </Space>
                    }
                    description={
                      <Space>
                        <Tag color={priorityColors[cr.riskLevel]}>{cr.riskLevel}风险</Tag>
                        <Tag color={statusColors[cr.status] || 'default'}>{cr.status}</Tag>
                        <Text type="secondary">
                          计划开始: {dayjs(cr.plannedStartDate).format('YYYY-MM-DD')}
                        </Text>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          ) : (
            <Empty description="暂无关联变更" />
          )}
        </Card>
      ),
    },
    {
      key: 'knowledge',
      label: (
        <span>
          <BookOutlined /> 知识库 ({kbArticles.length})
        </span>
      ),
      children: (
        <Card title="相关知识库文章">
          {kbArticles.length > 0 ? (
            <List
              itemLayout="horizontal"
              dataSource={kbArticles}
              renderItem={article => (
                <List.Item
                  actions={[
                    <Button type="link" key="view">
                      阅读
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={<Avatar style={{ backgroundColor: '#52c41a' }}>{<BookOutlined />}</Avatar>}
                    title={<Text strong>{article.title}</Text>}
                    description={
                      <Space>
                        <Tag>{article.category}</Tag>
                        <Tooltip title="相关度">
                          <Badge
                            count={Math.round(article.relevanceScore * 100)}
                            style={{ backgroundColor: '#1890ff' }}
                          />
                        </Tooltip>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          ) : (
            <Empty description="暂无相关知识库文章" />
          )}
        </Card>
      ),
    },
  ];

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="large">
      {/* 头部信息 */}
      <Card styles={{ body: { padding: '16px 24px' } }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/problems')}>
              返回列表
            </Button>
            <Title level={4} style={{ margin: 0 }}>
              {data.title}
            </Title>
            <Tag color={statusColors[data.status]}>{ProblemStatusLabels[data.status]}</Tag>
          </Space>
          <Space>
            <Button icon={<EditOutlined />} onClick={() => router.push(`/problems/${data.id}/edit`)}>
              编辑
            </Button>
            {data.status === ProblemStatus.OPEN && (
              <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.IN_PROGRESS)}>
                开始处理
              </Button>
            )}
            {data.status === ProblemStatus.IN_PROGRESS && (
              <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.IDENTIFIED)}>
                标记已识别
              </Button>
            )}
            {data.status === ProblemStatus.IDENTIFIED && (
              <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.RESOLVED)}>
                标记解决
              </Button>
            )}
            {data.status === ProblemStatus.RESOLVED && (
              <Button onClick={() => handleUpdateStatus(ProblemStatus.CLOSED)}>关闭问题</Button>
            )}
          </Space>
        </div>

        {/* 状态进度条 */}
        <div className="mt-4">
          <Steps current={getStatusStep().findIndex(s => s.status === 'process')} size="small">
            {getStatusStep().map((step, idx) => (
              <Steps.Step key={idx} title={step.title} status={step.status as any} />
            ))}
          </Steps>
        </div>
      </Card>

      {/* 统计信息 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="关联事件"
              value={relatedIncidents.length}
              prefix={<LinkOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="解决方案"
              value={solutions.length}
              prefix={<SolutionOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="关联变更"
              value={changeRequests.length}
              prefix={<HistoryOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="相关知识"
              value={kbArticles.length}
              prefix={<BookOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Tab内容 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={getTabItems()}
          defaultActiveKey="basic"
        />
      </Card>

      {/* 添加解决方案弹窗 */}
      <Modal
        title="添加解决方案"
        open={solutionModalVisible}
        onCancel={() => setSolutionModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form layout="vertical">
          <Form.Item label="方案标题" rules={[{ required: true }]}>
            <Input placeholder="请输入解决方案标题" />
          </Form.Item>
          <Form.Item label="方案描述" rules={[{ required: true }]}>
            <TextArea rows={4} placeholder="请详细描述解决方案" />
          </Form.Item>
          <Form.Item label="预期效果">
            <Select placeholder="请选择预期效果">
              <Option value="high">高</Option>
              <Option value="medium">中</Option>
              <Option value="low">低</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={() => setSolutionModalVisible(false)}>取消</Button>
              <Button type="primary">保存</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 关联事件弹窗 */}
      <Modal
        title="关联事件"
        open={linkingIncidentModalVisible}
        onCancel={() => setLinkingIncidentModalVisible(false)}
        footer={null}
        width={600}
      >
        <Alert
          message="功能提示"
          description="请输入事件编号进行关联，或从下方选择已有事件"
          type="info"
          showIcon
          className="mb-4"
        />
        <Form layout="vertical">
          <Form.Item label="事件编号">
            <Input placeholder="例如: INC-2026-001" />
          </Form.Item>
          <Form.Item>
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={() => setLinkingIncidentModalVisible(false)}>取消</Button>
              <Button type="primary">关联</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
};

export default EnhancedProblemDetail;

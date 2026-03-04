'use client';

/**
 * 问题调查 Tab 组件
 */

import React, { useState, useEffect } from 'react';
import {
  Tabs,
  Card,
  Button,
  Space,
  Table,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  message,
  Timeline,
  Typography,
  Progress,
  Descriptions,
  Divider,
  Alert,
  Empty,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  FileTextOutlined,
  SolutionOutlined,
  ExperimentOutlined,
  LinkOutlined,
  BookOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useParams } from 'next/navigation';

import {
  ProblemInvestigationAPI,
  type ProblemInvestigationSummary,
  type InvestigationStep,
  type RootCauseAnalysis,
  type ProblemSolution,
  type ProblemKnowledgeArticle,
  type CreateStepRequest,
  type CreateRootCauseRequest,
  type CreateSolutionRequest,
  type CreateKnowledgeArticleRequest,
  type SolutionType,
} from '@/lib/api/problem-investigation';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;

// 状态标签映射
const statusColors: Record<string, string> = {
  not_started: 'default',
  in_progress: 'processing',
  on_hold: 'warning',
  completed: 'success',
  cancelled: 'error',
  blocked: 'error',
  pending: 'default',
};

const statusLabels: Record<string, string> = {
  not_started: '未开始',
  in_progress: '进行中',
  on_hold: '暂停',
  completed: '已完成',
  cancelled: '已取消',
  blocked: '已阻塞',
  pending: '待处理',
};

const priorityColors: Record<string, string> = {
  low: 'green',
  medium: 'blue',
  high: 'orange',
  critical: 'red',
};

const confidenceLabels: Record<string, string> = {
  low: '低',
  medium: '中',
  high: '高',
};

const methodLabels: Record<string, string> = {
  '5-whys': '5个为什么',
  fishbone: '鱼骨图',
  timeline: '时间线',
  fault_tree: '故障树',
};

interface ProblemInvestigationTabProps {
  problemId: number;
  problemTitle?: string;
  problemDescription?: string;
}

const ProblemInvestigationTab: React.FC<ProblemInvestigationTabProps> = ({
  problemId,
  problemTitle = '',
  problemDescription = '',
}) => {
  const params = useParams();
  const id = (params?.id as string) || problemId.toString();

  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState<ProblemInvestigationSummary | null>(null);

  // Modal 状态
  const [stepModalOpen, setStepModalOpen] = useState(false);
  const [rootCauseModalOpen, setRootCauseModalOpen] = useState(false);
  const [solutionModalOpen, setSolutionModalOpen] = useState(false);
  const [knowledgeModalOpen, setKnowledgeModalOpen] = useState(false);
  const [editingStep, setEditingStep] = useState<InvestigationStep | null>(null);

  const [stepForm] = Form.useForm();
  const [rootCauseForm] = Form.useForm();
  const [solutionForm] = Form.useForm();
  const [knowledgeForm] = Form.useForm();

  // 加载调查摘要
  const loadSummary = async () => {
    setLoading(true);
    try {
      const data = await ProblemInvestigationAPI.getSummary(Number(id) || problemId);
      setSummary(data);
    } catch (error) {
      console.error('加载调查摘要失败:', error);
      message.error('加载调查摘要失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSummary();
  }, [id, problemId]);

  // 创建调查记录
  const handleCreateInvestigation = async () => {
    try {
      await ProblemInvestigationAPI.createInvestigation({
        problem_id: Number(id) || problemId,
      });
      message.success('创建调查成功');
      loadSummary();
    } catch (error) {
      message.error('创建调查失败');
    }
  };

  // 创建调查步骤
  const handleCreateStep = async (values: {
    step_title: string;
    step_description?: string;
    assigned_to?: number;
    notes?: string;
  }) => {
    try {
      const data: CreateStepRequest = {
        investigation_id: summary?.investigation?.id!,
        step_number: (summary?.steps?.length || 0) + 1,
        step_title: values.step_title,
        step_description: values.step_description || '',
        assigned_to: values.assigned_to,
        notes: values.notes,
      };
      await ProblemInvestigationAPI.createStep(data);
      message.success('创建步骤成功');
      setStepModalOpen(false);
      stepForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error('创建步骤失败');
    }
  };

  // 更新步骤状态
  const handleUpdateStepStatus = async (stepId: number, status: string) => {
    try {
      await ProblemInvestigationAPI.updateStep(stepId, {
        status: status as InvestigationStep['status'],
      });
      message.success('更新成功');
      loadSummary();
    } catch (error) {
      message.error('更新失败');
    }
  };

  // 创建根本原因分析
  const handleCreateRootCause = async (values: {
    analysis_method: string;
    root_cause_description: string;
    contributing_factors?: string;
    evidence?: string;
    confidence_level: 'low' | 'medium' | 'high';
  }) => {
    try {
      const data: CreateRootCauseRequest = {
        problem_id: Number(id) || problemId,
        analysis_method: values.analysis_method,
        root_cause_description: values.root_cause_description,
        contributing_factors: values.contributing_factors || undefined,
        evidence: values.evidence || undefined,
        confidence_level: values.confidence_level,
      };
      await ProblemInvestigationAPI.createRootCause(data);
      message.success('创建成功');
      setRootCauseModalOpen(false);
      rootCauseForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error('创建失败');
    }
  };

  // 创建解决方案
  const handleCreateSolution = async (values: {
    solution_type: SolutionType;
    solution_description: string;
    priority?: string;
    estimated_effort_hours?: number;
    estimated_cost?: number;
    risk_assessment?: string;
  }) => {
    try {
      const data: CreateSolutionRequest = {
        problem_id: Number(id) || problemId,
        solution_type: values.solution_type,
        solution_description: values.solution_description,
        priority: values.priority || 'medium',
        estimated_effort_hours: values.estimated_effort_hours,
        estimated_cost: values.estimated_cost,
        risk_assessment: values.risk_assessment,
      };
      await ProblemInvestigationAPI.createSolution(data);
      message.success('创建成功');
      setSolutionModalOpen(false);
      solutionForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error('创建失败');
    }
  };

  // 沉淀到知识库
  const handleCreateKnowledgeArticle = async (values: {
    article_title: string;
    article_content: string;
    article_type?: string;
    tags?: string[];
  }) => {
    try {
      const data: CreateKnowledgeArticleRequest = {
        problem_id: Number(id) || problemId,
        article_title: values.article_title,
        article_content: values.article_content,
        article_type: values.article_type || '',
        tags: values.tags,
      };
      await ProblemInvestigationAPI.createKnowledgeArticle(data);
      message.success('已沉淀到知识库');
      setKnowledgeModalOpen(false);
      knowledgeForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error('沉淀失败');
    }
  };

  // 调查步骤表格列
  const stepColumns = [
    {
      title: '序号',
      dataIndex: 'step_number',
      key: 'step_number',
      width: 60,
    },
    {
      title: '步骤标题',
      dataIndex: 'step_title',
      key: 'step_title',
    },
    {
      title: '描述',
      dataIndex: 'step_description',
      key: 'step_description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag color={statusColors[status]}>{statusLabels[status]}</Tag>,
    },
    {
      title: '负责人',
      dataIndex: 'assigned_to_name',
      key: 'assigned_to_name',
    },
    {
      title: '完成时间',
      dataIndex: 'completion_date',
      key: 'completion_date',
      render: (date: string) => (date ? dayjs(date).format('YYYY-MM-DD') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: InvestigationStep) => (
        <Space>
          {record.status !== 'completed' && (
            <Button
              size="small"
              type="link"
              onClick={() => handleUpdateStepStatus(record.id, 'completed')}
            >
              完成
            </Button>
          )}
        </Space>
      ),
    },
  ];

  // 解决方案表格列
  const solutionColumns = [
    {
      title: '类型',
      dataIndex: 'solution_type',
      key: 'solution_type',
      render: (type: string) => {
        const colors: Record<string, string> = {
          workaround: 'orange',
          fix: 'blue',
          prevention: 'green',
          process: 'purple',
        };
        const labels: Record<string, string> = {
          workaround: '临时方案',
          fix: '彻底修复',
          prevention: '预防措施',
          process: '流程改进',
        };
        return <Tag color={colors[type]}>{labels[type]}</Tag>;
      },
    },
    {
      title: '描述',
      dataIndex: 'solution_description',
      key: 'solution_description',
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (p: string) => <Tag color={priorityColors[p]}>{p?.toUpperCase()}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusMap: Record<string, { color: string; label: string }> = {
          proposed: { color: 'default', label: '已提议' },
          approved: { color: 'blue', label: '已批准' },
          pending_implementation: { color: 'orange', label: '待实施' },
          in_progress: { color: 'processing', label: '实施中' },
          implemented: { color: 'success', label: '已实施' },
          rejected: { color: 'error', label: '已拒绝' },
        };
        const s = statusMap[status] || { color: 'default', label: status };
        return <Tag color={s.color}>{s.label}</Tag>;
      },
    },
    {
      title: '提议人',
      dataIndex: 'proposed_by_name',
      key: 'proposed_by_name',
    },
  ];

  // Tab 内容
  const tabItems = [
    {
      key: 'overview',
      label: (
        <span>
          <FileTextOutlined /> 调查概览
        </span>
      ),
      children: (
        <Card>
          {summary?.investigation ? (
            <>
              <Descriptions column={2}>
                <Descriptions.Item label="调查状态">
                  <Tag color={statusColors[summary.investigation.status]}>
                    {statusLabels[summary.investigation.status]}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="调查人">
                  {summary.investigation.investigator_name || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="开始日期">
                  {summary.investigation.start_date
                    ? dayjs(summary.investigation.start_date).format('YYYY-MM-DD')
                    : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="预计完成">
                  {summary.investigation.estimated_completion_date
                    ? dayjs(summary.investigation.estimated_completion_date).format('YYYY-MM-DD')
                    : '-'}
                </Descriptions.Item>
              </Descriptions>

              <Divider />

              <Title level={5}>调查概要</Title>
              <Paragraph>{summary.investigation.investigation_summary || '暂无调查概要'}</Paragraph>

              <Divider />

              <Title level={5}>调查进度</Title>
              <Progress
                percent={
                  summary.steps.length > 0
                    ? Math.round(
                        (summary.steps.filter(s => s.status === 'completed').length /
                          summary.steps.length) *
                          100
                      )
                    : 0
                }
                status="active"
              />

              <Divider />

              {summary.root_cause_analysis && (
                <>
                  <Title level={5}>根本原因分析</Title>
                  <Alert
                    type="info"
                    message={`分析方法: ${methodLabels[summary.root_cause_analysis.analysis_method] || summary.root_cause_analysis.analysis_method}`}
                    description={
                      <>
                        <p>
                          <strong>原因描述:</strong>{' '}
                          {summary.root_cause_analysis.root_cause_description}
                        </p>
                        <p>
                          <strong>置信度:</strong>{' '}
                          {confidenceLabels[summary.root_cause_analysis.confidence_level]}
                        </p>
                        {summary.root_cause_analysis.contributing_factors && (
                          <p>
                            <strong>促成因素:</strong>{' '}
                            {summary.root_cause_analysis.contributing_factors}
                          </p>
                        )}
                      </>
                    }
                  />
                </>
              )}

              {summary.solutions.length > 0 && (
                <>
                  <Divider />
                  <Title level={5}>解决方案 ({summary.solutions.length})</Title>
                  <Table
                    size="small"
                    pagination={false}
                    columns={solutionColumns.slice(0, 4)}
                    dataSource={summary.solutions}
                    rowKey="id"
                  />
                </>
              )}
            </>
          ) : (
            <Empty description="暂无调查记录" image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <Button type="primary" onClick={handleCreateInvestigation}>
                开始调查
              </Button>
            </Empty>
          )}
        </Card>
      ),
    },
    {
      key: 'steps',
      label: (
        <span>
          <SolutionOutlined /> 调查步骤 ({summary?.steps.length || 0})
        </span>
      ),
      children: (
        <Card
          title="调查步骤"
          extra={
            summary?.investigation && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setStepModalOpen(true)}>
                添加步骤
              </Button>
            )
          }
        >
          {summary?.steps && summary.steps.length > 0 ? (
            <Table
              columns={stepColumns}
              dataSource={summary.steps}
              rowKey="id"
              pagination={false}
            />
          ) : (
            <Empty
              description={summary?.investigation ? '暂无调查步骤' : '请先开始调查'}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </Card>
      ),
    },
    {
      key: 'root-cause',
      label: (
        <span>
          <ExperimentOutlined /> 根因分析
        </span>
      ),
      children: (
        <Card
          title="根本原因分析"
          extra={
            summary?.investigation &&
            !summary?.root_cause_analysis && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setRootCauseModalOpen(true)}
              >
                开始分析
              </Button>
            )
          }
        >
          {summary?.root_cause_analysis ? (
            <>
              <Descriptions column={2}>
                <Descriptions.Item label="分析方法">
                  {methodLabels[summary.root_cause_analysis.analysis_method] ||
                    summary.root_cause_analysis.analysis_method}
                </Descriptions.Item>
                <Descriptions.Item label="置信度">
                  <Tag color={statusColors[summary.root_cause_analysis.confidence_level]}>
                    {confidenceLabels[summary.root_cause_analysis.confidence_level]}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="分析师" span={2}>
                  {summary.root_cause_analysis.analyst_name || '-'}
                </Descriptions.Item>
              </Descriptions>

              <Divider />

              <Title level={5}>根本原因</Title>
              <Paragraph style={{ fontSize: 16 }}>
                {summary.root_cause_analysis.root_cause_description}
              </Paragraph>

              <Divider />

              <Title level={5}>促成因素</Title>
              <Paragraph>{summary.root_cause_analysis.contributing_factors || '无'}</Paragraph>

              <Divider />

              <Title level={5}>证据支持</Title>
              <Paragraph>{summary.root_cause_analysis.evidence || '无'}</Paragraph>
            </>
          ) : (
            <Empty
              description={summary?.investigation ? '暂无根因分析' : '请先开始调查'}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </Card>
      ),
    },
    {
      key: 'solutions',
      label: (
        <span>
          <CheckCircleOutlined /> 解决方案 ({summary?.solutions.length || 0})
        </span>
      ),
      children: (
        <Card
          title="解决方案"
          extra={
            summary?.investigation && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setSolutionModalOpen(true)}
              >
                添加方案
              </Button>
            )
          }
        >
          {summary?.solutions && summary.solutions.length > 0 ? (
            <Table
              columns={solutionColumns}
              dataSource={summary.solutions}
              rowKey="id"
              pagination={false}
            />
          ) : (
            <Empty
              description={summary?.investigation ? '暂无解决方案' : '请先开始调查'}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </Card>
      ),
    },
    {
      key: 'knowledge',
      label: (
        <span>
          <BookOutlined /> 知识沉淀 ({summary?.knowledge_articles.length || 0})
        </span>
      ),
      children: (
        <Card
          title="知识库文章"
          extra={
            (summary?.root_cause_analysis ||
              (summary?.solutions && summary.solutions.length > 0)) && (
              <Button
                type="primary"
                icon={<BookOutlined />}
                onClick={() => setKnowledgeModalOpen(true)}
              >
                沉淀到知识库
              </Button>
            )
          }
        >
          {summary?.knowledge_articles && summary.knowledge_articles.length > 0 ? (
            <Table
              columns={[
                { title: '标题', dataIndex: 'article_title', key: 'article_title' },
                { title: '类型', dataIndex: 'article_type', key: 'article_type' },
                {
                  title: '标签',
                  dataIndex: 'tags',
                  key: 'tags',
                  render: (tags: string[]) => (
                    <Space wrap>
                      {tags?.map(t => (
                        <Tag key={t}>{t}</Tag>
                      ))}
                    </Space>
                  ),
                },
                { title: '阅读量', dataIndex: 'view_count', key: 'view_count' },
                {
                  title: '状态',
                  dataIndex: 'status',
                  key: 'status',
                  render: (s: string) => (
                    <Tag color={s === 'published' ? 'success' : 'default'}>
                      {s === 'published' ? '已发布' : '草稿'}
                    </Tag>
                  ),
                },
                {
                  title: '创建时间',
                  dataIndex: 'created_at',
                  key: 'created_at',
                  render: (d: string) => dayjs(d).format('YYYY-MM-DD'),
                },
              ]}
              dataSource={summary.knowledge_articles}
              rowKey="id"
              pagination={false}
            />
          ) : (
            <Empty description="暂无知识库文章" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Card>
      ),
    },
    {
      key: 'relationships',
      label: (
        <span>
          <LinkOutlined /> 关联 ({summary?.relationships.length || 0})
        </span>
      ),
      children: (
        <Card title="关联的工单/变更">
          {summary?.relationships && summary.relationships.length > 0 ? (
            <Table
              columns={[
                {
                  title: '类型',
                  dataIndex: 'related_type',
                  key: 'related_type',
                  render: (type: string) => {
                    const colors: Record<string, string> = {
                      ticket: 'blue',
                      change: 'purple',
                      incident: 'red',
                    };
                    const labels: Record<string, string> = {
                      ticket: '工单',
                      change: '变更',
                      incident: '事件',
                    };
                    return <Tag color={colors[type]}>{labels[type]}</Tag>;
                  },
                },
                { title: '标题', dataIndex: 'related_title', key: 'related_title' },
                {
                  title: '关联类型',
                  dataIndex: 'relationship_type',
                  key: 'relationship_type',
                },
                {
                  title: '描述',
                  dataIndex: 'description',
                  key: 'description',
                  ellipsis: true,
                },
              ]}
              dataSource={summary.relationships}
              rowKey="id"
              pagination={false}
            />
          ) : (
            <Empty description="暂无关联记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Card>
      ),
    },
  ];

  return (
    <>
      <Tabs
        items={tabItems}
        defaultActiveKey="overview"
        animated={{ inkBar: true, tabPane: true }}
      />

      {/* 创建调查步骤 Modal */}
      <Modal
        title="添加调查步骤"
        open={stepModalOpen}
        onCancel={() => setStepModalOpen(false)}
        footer={null}
        width={600}
      >
        <Form form={stepForm} layout="vertical" onFinish={handleCreateStep}>
          <Form.Item
            name="step_title"
            label="步骤标题"
            rules={[{ required: true, message: '请输入步骤标题' }]}
          >
            <Input placeholder="请输入步骤标题" />
          </Form.Item>
          <Form.Item
            name="step_description"
            label="步骤描述"
            rules={[{ required: true, message: '请输入步骤描述' }]}
          >
            <TextArea rows={4} placeholder="请输入步骤描述" />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <TextArea rows={2} placeholder="备注信息（可选）" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
              <Button onClick={() => setStepModalOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 创建根因分析 Modal */}
      <Modal
        title="根本原因分析"
        open={rootCauseModalOpen}
        onCancel={() => setRootCauseModalOpen(false)}
        footer={null}
        width={700}
      >
        <Form form={rootCauseForm} layout="vertical" onFinish={handleCreateRootCause}>
          <Form.Item
            name="analysis_method"
            label="分析方法"
            rules={[{ required: true, message: '请选择分析方法' }]}
          >
            <Select placeholder="请选择分析方法">
              <Option value="5-whys">5个为什么 (5-Whys)</Option>
              <Option value="fishbone">鱼骨图 (Fishbone)</Option>
              <Option value="timeline">时间线分析 (Timeline)</Option>
              <Option value="fault_tree">故障树分析 (Fault Tree)</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="root_cause_description"
            label="根本原因"
            rules={[{ required: true, message: '请输入根本原因描述' }]}
          >
            <TextArea rows={4} placeholder="请详细描述问题的根本原因" />
          </Form.Item>
          <Form.Item name="contributing_factors" label="促成因素">
            <TextArea rows={3} placeholder="导致问题发生的其他因素（可选）" />
          </Form.Item>
          <Form.Item name="evidence" label="证据支持">
            <TextArea rows={3} placeholder="支持您分析的证据材料（可选）" />
          </Form.Item>
          <Form.Item
            name="confidence_level"
            label="置信度"
            rules={[{ required: true, message: '请选择置信度' }]}
          >
            <Select placeholder="请选择置信度">
              <Option value="low">低 - 需要进一步验证</Option>
              <Option value="medium">中 - 基本确定</Option>
              <Option value="high">高 - 经过验证确认</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                提交分析
              </Button>
              <Button onClick={() => setRootCauseModalOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 创建解决方案 Modal */}
      <Modal
        title="添加解决方案"
        open={solutionModalOpen}
        onCancel={() => setSolutionModalOpen(false)}
        footer={null}
        width={700}
      >
        <Form form={solutionForm} layout="vertical" onFinish={handleCreateSolution}>
          <Form.Item
            name="solution_type"
            label="方案类型"
            rules={[{ required: true, message: '请选择方案类型' }]}
          >
            <Select placeholder="请选择方案类型">
              <Option value="workaround">临时方案 (Workaround)</Option>
              <Option value="fix">彻底修复 (Fix)</Option>
              <Option value="prevention">预防措施 (Prevention)</Option>
              <Option value="process">流程改进 (Process)</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="solution_description"
            label="方案描述"
            rules={[{ required: true, message: '请输入方案描述' }]}
          >
            <TextArea rows={4} placeholder="请详细描述解决方案" />
          </Form.Item>
          <Form.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder="请选择优先级">
              <Option value="low">低</Option>
              <Option value="medium">中</Option>
              <Option value="high">高</Option>
              <Option value="critical">紧急</Option>
            </Select>
          </Form.Item>
          <Space style={{ width: '100%' }} size="large">
            <Form.Item name="estimated_effort_hours" label="预估工时(小时)">
              <Input type="number" placeholder="0" />
            </Form.Item>
            <Form.Item name="estimated_cost" label="预估成本">
              <Input type="number" placeholder="0" />
            </Form.Item>
          </Space>
          <Form.Item name="risk_assessment" label="风险评估">
            <TextArea rows={2} placeholder="实施该方案的风险（可选）" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                创建方案
              </Button>
              <Button onClick={() => setSolutionModalOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 沉淀到知识库 Modal */}
      <Modal
        title="沉淀到知识库"
        open={knowledgeModalOpen}
        onCancel={() => setKnowledgeModalOpen(false)}
        footer={null}
        width={700}
      >
        <Form form={knowledgeForm} layout="vertical" onFinish={handleCreateKnowledgeArticle}>
          <Form.Item
            name="article_title"
            label="文章标题"
            rules={[{ required: true, message: '请输入文章标题' }]}
          >
            <Input placeholder="请输入文章标题" defaultValue={`[问题] ${problemTitle}`} />
          </Form.Item>
          <Form.Item
            name="article_type"
            label="文章类型"
            rules={[{ required: true, message: '请选择文章类型' }]}
          >
            <Select placeholder="请选择文章类型">
              <Option value="troubleshooting">故障排查指南</Option>
              <Option value="solution">解决方案</Option>
              <Option value="process">操作流程</Option>
              <Option value="prevention">预防措施</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="article_content"
            label="文章内容"
            rules={[{ required: true, message: '请输入文章内容' }]}
          >
            <TextArea
              rows={8}
              placeholder="请输入知识库文章内容"
              defaultValue={`
## 问题描述
${problemDescription || '请描述问题背景'}

## 根本原因
${summary?.root_cause_analysis?.root_cause_description || '请描述根本原因'}

## 解决方案
${summary?.solutions?.map((s: ProblemSolution) => `- ${s.solution_description}`).join('\n') || '请描述解决方案'}

## 预防措施
${
  summary?.solutions
    ?.filter((s: ProblemSolution) => s.solution_type === 'prevention')
    .map((s: ProblemSolution) => `- ${s.solution_description}`)
    .join('\n') || ''
}
              `.trim()}
            />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Select mode="tags" placeholder="添加标签" style={{ width: '100%' }}>
              <Option value="problem">问题</Option>
              <Option value="root-cause">根因</Option>
              <Option value="solution">解决方案</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                沉淀到知识库
              </Button>
              <Button onClick={() => setKnowledgeModalOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default ProblemInvestigationTab;

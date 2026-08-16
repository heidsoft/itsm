'use client';

/**
 * 问题调查 Tab 组件
 */

import React, { useState, useEffect, useMemo } from 'react';
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
  message,
  Typography,
  Progress,
  Descriptions,
  Divider,
  Alert,
  Empty,
} from 'antd';
import { Plus, FileText, ClipboardCheck, FlaskConical, CheckCircle, BookOpen, Link } from 'lucide-react';
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
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Paragraph } = Typography;
const { TextArea } = Input;

interface ProblemInvestigationTabProps {
  problemId: number;
  problemTitle?: string;
  problemDescription?: string;
  initialInnerTab?: string;
}

const ProblemInvestigationTab: React.FC<ProblemInvestigationTabProps> = ({
  problemId,
  problemTitle = '',
  problemDescription = '',
  initialInnerTab = 'overview',
}) => {
  const { t } = useI18n();
  const params = useParams();
  const id = (params?.id as string) || problemId.toString();

  const statusColors: Record<string, string> = useMemo(
    () => ({
      notStarted: 'default',
      inProgress: 'processing',
      onHold: 'warning',
      completed: 'success',
      cancelled: 'error',
      blocked: 'error',
      pending: 'default',
    }),
    []
  );

  const priorityColors: Record<string, string> = useMemo(
    () => ({
      low: 'green',
      medium: 'blue',
      high: 'orange',
      critical: 'red',
    }),
    []
  );

  const getStatusLabel = (status: string): string => {
    const labels: Record<string, string> = {
      notStarted: t('problemInvestigation.statusLabels.notStarted'),
      inProgress: t('problemInvestigation.statusLabels.inProgress'),
      onHold: t('problemInvestigation.statusLabels.onHold'),
      completed: t('problemInvestigation.statusLabels.completed'),
      cancelled: t('problemInvestigation.statusLabels.cancelled'),
      blocked: t('problemInvestigation.statusLabels.blocked'),
      pending: t('problemInvestigation.statusLabels.pending'),
    };
    return labels[status] || status;
  };

  const getConfidenceLabel = (c: string): string => {
    const labels: Record<string, string> = {
      low: t('problemInvestigation.confidenceLabels.low'),
      medium: t('problemInvestigation.confidenceLabels.medium'),
      high: t('problemInvestigation.confidenceLabels.high'),
    };
    return labels[c] || c;
  };

  const getMethodLabel = (m: string): string => {
    const labels: Record<string, string> = {
      '5-whys': t('problemInvestigation.methodLabels.5-whys'),
      fishbone: t('problemInvestigation.methodLabels.fishbone'),
      timeline: t('problemInvestigation.methodLabels.timeline'),
      faultTree: t('problemInvestigation.methodLabels.faultTree'),
    };
    return labels[m] || m;
  };

  const getSolutionTypeLabel = (type: string): string => {
    const colors: Record<string, string> = {
      workaround: 'orange',
      fix: 'blue',
      prevention: 'green',
      process: 'purple',
    };
    const labels: Record<string, string> = {
      workaround: t('problemInvestigation.solutionTypeLabels.workaround'),
      fix: t('problemInvestigation.solutionTypeLabels.fix'),
      prevention: t('problemInvestigation.solutionTypeLabels.prevention'),
      process: t('problemInvestigation.solutionTypeLabels.process'),
    };
    return `<Tag color="${colors[type]}">${labels[type] || type}</Tag>`;
  };

  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState<ProblemInvestigationSummary | null>(null);

  const [stepModalOpen, setStepModalOpen] = useState(false);
  const [rootCauseModalOpen, setRootCauseModalOpen] = useState(false);
  const [solutionModalOpen, setSolutionModalOpen] = useState(false);
  const [knowledgeModalOpen, setKnowledgeModalOpen] = useState(false);

  const [stepForm] = Form.useForm();
  const [rootCauseForm] = Form.useForm();
  const [solutionForm] = Form.useForm();
  const [knowledgeForm] = Form.useForm();

  const loadSummary = async () => {
    setLoading(true);
    try {
      const data = await ProblemInvestigationAPI.getSummary(Number(id) || problemId);
      setSummary(
        data
          ? {
              ...data,
              steps: data.steps ?? [],
              solutions: data.solutions ?? [],
              implementations: data.implementations ?? [],
              relationships: data.relationships ?? [],
              knowledgeArticles: data.knowledgeArticles ?? [],
            }
          : null
      );
    } catch (error) {
      console.error('loadSummary error:', error);
      message.error(t('problemInvestigation.messages.loadSummaryFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSummary();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, problemId]);

  const handleCreateInvestigation = async () => {
    try {
      await ProblemInvestigationAPI.createInvestigation({
        problemId: Number(id) || problemId,
      });
      message.success(t('problemInvestigation.messages.createInvestigationSuccess'));
      loadSummary();
    } catch (error) {
      message.error(t('problemInvestigation.messages.createInvestigationFailed'));
    }
  };

  const handleCreateStep = async (values: {
    stepTitle: string;
    stepDescription?: string;
    assignedTo?: number;
    notes?: string;
  }) => {
    try {
      const data: CreateStepRequest = {
        investigationId: summary?.investigation?.id!,
        stepNumber: (summary?.steps?.length || 0) + 1,
        stepTitle: values.stepTitle,
        stepDescription: values.stepDescription || '',
        assignedTo: values.assignedTo,
        notes: values.notes,
      };
      await ProblemInvestigationAPI.createStep(data);
      message.success(t('problemInvestigation.messages.createStepSuccess'));
      setStepModalOpen(false);
      stepForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error(t('problemInvestigation.messages.createStepFailed'));
    }
  };

  const handleUpdateStepStatus = async (stepId: number, status: string) => {
    try {
      await ProblemInvestigationAPI.updateStep(stepId, {
        status: status as InvestigationStep['status'],
      });
      message.success(t('problemInvestigation.messages.updateSuccess'));
      loadSummary();
    } catch (error) {
      message.error(t('problemInvestigation.messages.updateFailed'));
    }
  };

  const handleCreateRootCause = async (values: {
    analysisMethod: string;
    rootCauseDescription: string;
    contributingFactors?: string;
    evidence?: string;
    confidenceLevel: 'low' | 'medium' | 'high';
  }) => {
    try {
      const data: CreateRootCauseRequest = {
        problemId: Number(id) || problemId,
        analysisMethod: values.analysisMethod,
        rootCauseDescription: values.rootCauseDescription,
        contributingFactors: values.contributingFactors || undefined,
        evidence: values.evidence || undefined,
        confidenceLevel: values.confidenceLevel,
      };
      await ProblemInvestigationAPI.createRootCause(data);
      message.success(t('problemInvestigation.messages.createSuccess'));
      setRootCauseModalOpen(false);
      rootCauseForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error(t('problemInvestigation.messages.createFailed'));
    }
  };

  const handleCreateSolution = async (values: {
    solutionType: SolutionType;
    solutionDescription: string;
    priority?: string;
    estimatedEffortHours?: number;
    estimatedCost?: number;
    riskAssessment?: string;
  }) => {
    try {
      const data: CreateSolutionRequest = {
        problemId: Number(id) || problemId,
        solutionType: values.solutionType,
        solutionDescription: values.solutionDescription,
        priority: values.priority || 'medium',
        estimatedEffortHours: values.estimatedEffortHours,
        estimatedCost: values.estimatedCost,
        riskAssessment: values.riskAssessment,
      };
      await ProblemInvestigationAPI.createSolution(data);
      message.success(t('problemInvestigation.messages.createSuccess'));
      setSolutionModalOpen(false);
      solutionForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error(t('problemInvestigation.messages.createFailed'));
    }
  };

  const handleCreateKnowledgeArticle = async (values: {
    articleTitle: string;
    articleContent: string;
    articleType?: string;
    tags?: string[];
  }) => {
    try {
      const data: CreateKnowledgeArticleRequest = {
        problemId: Number(id) || problemId,
        articleTitle: values.articleTitle,
        articleContent: values.articleContent,
        articleType: values.articleType || '',
        tags: values.tags,
      };
      await ProblemInvestigationAPI.createKnowledgeArticle(data);
      message.success(t('problemInvestigation.messages.knowledgeDeposited'));
      setKnowledgeModalOpen(false);
      knowledgeForm.resetFields();
      loadSummary();
    } catch (error) {
      message.error(t('problemInvestigation.messages.knowledgeDepositFailed'));
    }
  };

  const stepColumns = useMemo(
    () => [
      { title: t('problemInvestigation.columns.stepNumber'), dataIndex: 'stepNumber', key: 'stepNumber', width: 60 },
      { title: t('problemInvestigation.columns.stepTitle'), dataIndex: 'stepTitle', key: 'stepTitle' },
      {
        title: t('problemInvestigation.columns.description'),
        dataIndex: 'stepDescription',
        key: 'stepDescription',
        ellipsis: true,
      },
      {
        title: t('problemInvestigation.columns.status'),
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => <Tag color={statusColors[status]}>{getStatusLabel(status)}</Tag>,
      },
      { title: t('problemInvestigation.columns.assignee'), dataIndex: 'assignedToName', key: 'assignedToName' },
      {
        title: t('problemInvestigation.columns.completionDate'),
        dataIndex: 'completionDate',
        key: 'completionDate',
        render: (date: string) => (date ? dayjs(date).format('YYYY-MM-DD') : '-'),
      },
      {
        title: t('problemInvestigation.columns.action'),
        key: 'action',
        render: (_: unknown, record: InvestigationStep) => (
          <Space>
            {record.status !== 'completed' && (
              <Button
                size="small"
                type="link"
                onClick={() => handleUpdateStepStatus(record.id, 'completed')}
              >
                {t('problemInvestigation.messages.complete')}
              </Button>
            )}
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [statusColors]
  );

  const renderSolutionTypeTag = (type: string) => {
    const colors: Record<string, string> = {
      workaround: 'orange',
      fix: 'blue',
      prevention: 'green',
      process: 'purple',
    };
    const labels: Record<string, string> = {
      workaround: t('problemInvestigation.solutionTypeLabels.workaround'),
      fix: t('problemInvestigation.solutionTypeLabels.fix'),
      prevention: t('problemInvestigation.solutionTypeLabels.prevention'),
      process: t('problemInvestigation.solutionTypeLabels.process'),
    };
    return <Tag color={colors[type]}>{labels[type] || type}</Tag>;
  };

  const solutionColumns = useMemo(
    () => [
      {
        title: t('problemInvestigation.columns.solutionType'),
        dataIndex: 'solutionType',
        key: 'solutionType',
        render: renderSolutionTypeTag,
      },
      {
        title: t('problemInvestigation.columns.description'),
        dataIndex: 'solutionDescription',
        key: 'solutionDescription',
        ellipsis: true,
      },
      {
        title: t('problemInvestigation.columns.priority'),
        dataIndex: 'priority',
        key: 'priority',
        render: (p: string) => <Tag color={priorityColors[p]}>{p?.toUpperCase()}</Tag>,
      },
      {
        title: t('problemInvestigation.columns.status'),
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => {
          const statusMap: Record<string, { color: string; labelKey: string }> = {
            proposed: { color: 'default', labelKey: 'problemInvestigation.solutionStatusLabels.proposed' },
            approved: { color: 'blue', labelKey: 'problemInvestigation.solutionStatusLabels.approved' },
            pendingImplementation: { color: 'orange', labelKey: 'problemInvestigation.solutionStatusLabels.pendingImplementation' },
            inProgress: { color: 'processing', labelKey: 'problemInvestigation.solutionStatusLabels.inProgress' },
            implemented: { color: 'success', labelKey: 'problemInvestigation.solutionStatusLabels.implemented' },
            rejected: { color: 'error', labelKey: 'problemInvestigation.solutionStatusLabels.rejected' },
          };
          const s = statusMap[status];
          return <Tag color={s?.color || 'default'}>{s ? t(s.labelKey) : status}</Tag>;
        },
      },
      { title: t('problemInvestigation.columns.proposedBy'), dataIndex: 'proposedByName', key: 'proposedByName' },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [priorityColors]
  );

  const tabItems = [
    {
      key: 'overview',
      label: (
        <span>
          <FileText /> {t('problemInvestigation.tabs.overview')}
        </span>
      ),
      children: (
        <Card>
          {summary?.investigation ? (
            <>
              <Descriptions column={2}>
                <Descriptions.Item label={t('problemInvestigation.overview.investigationStatus')}>
                  <Tag color={statusColors[summary.investigation.status]}>
                    {getStatusLabel(summary.investigation.status)}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('problemInvestigation.overview.investigator')}>
                  {summary.investigation.investigatorName || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('problemInvestigation.overview.startDate')}>
                  {summary.investigation.startDate
                    ? dayjs(summary.investigation.startDate).format('YYYY-MM-DD')
                    : '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('problemInvestigation.overview.estimatedCompletion')}>
                  {summary.investigation.estimatedCompletionDate
                    ? dayjs(summary.investigation.estimatedCompletionDate).format('YYYY-MM-DD')
                    : '-'}
                </Descriptions.Item>
              </Descriptions>

              <Divider />

              <Title level={5}>{t('problemInvestigation.overview.summary')}</Title>
              <Paragraph>
                {summary.investigation.investigationSummary || t('problemInvestigation.overview.summaryEmpty')}
              </Paragraph>

              <Divider />

              <Title level={5}>{t('problemInvestigation.overview.progress')}</Title>
              <Progress
                percent={
                  summary.steps.length > 0
                    ? Math.round(
                        (summary.steps.filter((s) => s.status === 'completed').length /
                          summary.steps.length) *
                          100
                      )
                    : 0
                }
                status="active"
              />

              <Divider />

              {summary.rootCauseAnalysis && (
                <>
                  <Title level={5}>{t('problemInvestigation.overview.rootCauseAnalysis')}</Title>
                  <Alert
                    type="info"
                    message={t('problemInvestigation.overview.analysisMethodLabel', {
                      method: getMethodLabel(summary.rootCauseAnalysis.analysisMethod),
                    })}
                    description={
                      <>
                        <p>
                          <strong>{t('problemInvestigation.overview.rootCauseDescLabel')}</strong>{' '}
                          {summary.rootCauseAnalysis.rootCauseDescription}
                        </p>
                        <p>
                          <strong>{t('problemInvestigation.overview.confidenceLabel')}</strong>{' '}
                          {getConfidenceLabel(summary.rootCauseAnalysis.confidenceLevel)}
                        </p>
                        {summary.rootCauseAnalysis.contributingFactors && (
                          <p>
                            <strong>{t('problemInvestigation.overview.contributingFactorsLabel')}</strong>{' '}
                            {summary.rootCauseAnalysis.contributingFactors}
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
                  <Title level={5}>
                    {t('problemInvestigation.overview.solutionsTitle', { count: summary.solutions.length })}
                  </Title>
                  <Table
                    size="small"
                    pagination={false}
                    scroll={{ x: 'max-content' }}
                    columns={solutionColumns.slice(0, 4)}
                    dataSource={summary.solutions}
                    rowKey="id"
                  />
                </>
              )}
            </>
          ) : (
            <Empty
              description={t('problemInvestigation.overview.noInvestigation')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              <Button type="primary" onClick={handleCreateInvestigation}>
                {t('problemInvestigation.overview.startInvestigation')}
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
          <ClipboardCheck />{' '}
          {t('problemInvestigation.tabs.steps', { count: summary?.steps.length || 0 })}
        </span>
      ),
      children: (
        <Card
          title={t('problemInvestigation.steps.title')}
          extra={
            summary?.investigation && (
              <Button type="primary" icon={<Plus />} onClick={() => setStepModalOpen(true)}>
                {t('problemInvestigation.steps.addStep')}
              </Button>
            )
          }
        >
          {summary?.steps && summary.steps.length > 0 ? (
            <Table
              columns={stepColumns}
              dataSource={summary.steps}
              rowKey="id"
              scroll={{ x: 'max-content' }}
              pagination={false}
            />
          ) : (
            <Empty
              description={
                summary?.investigation
                  ? t('problemInvestigation.steps.emptyWithInvestigation')
                  : t('problemInvestigation.steps.emptyWithoutInvestigation')
              }
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
          <FlaskConical /> {t('problemInvestigation.tabs.rootCause')}
        </span>
      ),
      children: (
        <Card
          title={t('problemInvestigation.rootCause.title')}
          extra={
            summary?.investigation &&
            !summary?.rootCauseAnalysis && (
              <Button
                type="primary"
                icon={<Plus />}
                onClick={() => setRootCauseModalOpen(true)}
              >
                {t('problemInvestigation.rootCause.startAnalysis')}
              </Button>
            )
          }
        >
          {summary?.rootCauseAnalysis ? (
            <>
              <Descriptions column={2}>
                <Descriptions.Item label={t('problemInvestigation.rootCause.analysisMethod')}>
                  {getMethodLabel(summary.rootCauseAnalysis.analysisMethod) ||
                    summary.rootCauseAnalysis.analysisMethod}
                </Descriptions.Item>
                <Descriptions.Item label={t('problemInvestigation.rootCause.confidence')}>
                  <Tag color={statusColors[summary.rootCauseAnalysis.confidenceLevel]}>
                    {getConfidenceLabel(summary.rootCauseAnalysis.confidenceLevel)}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('problemInvestigation.rootCause.analyst')} span={2}>
                  {summary.rootCauseAnalysis.analystName || '-'}
                </Descriptions.Item>
              </Descriptions>

              <Divider />

              <Title level={5}>{t('problemInvestigation.rootCause.rootCause')}</Title>
              <Paragraph style={{ fontSize: 16 }}>
                {summary.rootCauseAnalysis.rootCauseDescription}
              </Paragraph>

              <Divider />

              <Title level={5}>{t('problemInvestigation.rootCause.contributingFactors')}</Title>
              <Paragraph>
                {summary.rootCauseAnalysis.contributingFactors || t('problemInvestigation.rootCause.emptyValue')}
              </Paragraph>

              <Divider />

              <Title level={5}>{t('problemInvestigation.rootCause.evidence')}</Title>
              <Paragraph>
                {summary.rootCauseAnalysis.evidence || t('problemInvestigation.rootCause.emptyValue')}
              </Paragraph>
            </>
          ) : (
            <Empty
              description={
                summary?.investigation
                  ? t('problemInvestigation.rootCause.emptyWithInvestigation')
                  : t('problemInvestigation.rootCause.emptyWithoutInvestigation')
              }
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
          <CheckCircle />{' '}
          {t('problemInvestigation.tabs.solutions', { count: summary?.solutions.length || 0 })}
        </span>
      ),
      children: (
        <Card
          title={t('problemInvestigation.solutions.title')}
          extra={
            summary?.investigation && (
              <Button
                type="primary"
                icon={<Plus />}
                onClick={() => setSolutionModalOpen(true)}
              >
                {t('problemInvestigation.solutions.addSolution')}
              </Button>
            )
          }
        >
          {summary?.solutions && summary.solutions.length > 0 ? (
            <Table
              columns={solutionColumns}
              dataSource={summary.solutions}
              rowKey="id"
              scroll={{ x: 'max-content' }}
              pagination={false}
            />
          ) : (
            <Empty
              description={
                summary?.investigation
                  ? t('problemInvestigation.solutions.emptyWithInvestigation')
                  : t('problemInvestigation.solutions.emptyWithoutInvestigation')
              }
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
          <BookOpen />{' '}
          {t('problemInvestigation.tabs.knowledge', { count: summary?.knowledgeArticles.length || 0 })}
        </span>
      ),
      children: (
        <Card
          title={t('problemInvestigation.knowledge.title')}
          extra={
            (summary?.rootCauseAnalysis ||
              (summary?.solutions && summary.solutions.length > 0)) && (
              <Button
                type="primary"
                icon={<BookOpen />}
                onClick={() => setKnowledgeModalOpen(true)}
              >
                {t('problemInvestigation.knowledge.depositToKb')}
              </Button>
            )
          }
        >
          {summary?.knowledgeArticles && summary.knowledgeArticles.length > 0 ? (
            <Table
              columns={[
                { title: t('problemInvestigation.columns.articleTitle'), dataIndex: 'articleTitle', key: 'articleTitle' },
                { title: t('problemInvestigation.columns.articleType'), dataIndex: 'articleType', key: 'articleType' },
                {
                  title: t('problemInvestigation.columns.tags'),
                  dataIndex: 'tags',
                  key: 'tags',
                  render: (tags: string[]) => (
                    <Space wrap>
                      {tags?.map((tag) => (
                        <Tag key={tag}>{tag}</Tag>
                      ))}
                    </Space>
                  ),
                },
                { title: t('problemInvestigation.columns.viewCount'), dataIndex: 'viewCount', key: 'viewCount' },
                {
                  title: t('problemInvestigation.columns.status'),
                  dataIndex: 'status',
                  key: 'status',
                  render: (s: string) => (
                    <Tag color={s === 'published' ? 'success' : 'default'}>
                      {s === 'published'
                        ? t('problemInvestigation.knowledgeArticleStatus.published')
                        : t('problemInvestigation.knowledgeArticleStatus.draft')}
                    </Tag>
                  ),
                },
                {
                  title: t('problemInvestigation.columns.createdAt'),
                  dataIndex: 'createdAt',
                  key: 'createdAt',
                  render: (d: string) => dayjs(d).format('YYYY-MM-DD'),
                },
              ]}
              dataSource={summary.knowledgeArticles}
              rowKey="id"
              pagination={false}
            />
          ) : (
            <Empty
              description={t('problemInvestigation.knowledge.empty')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </Card>
      ),
    },
    {
      key: 'relationships',
      label: (
        <span>
          <Link /> {t('problemInvestigation.tabs.relationships', { count: summary?.relationships.length || 0 })}
        </span>
      ),
      children: (
        <Card title={t('problemInvestigation.relationships.title')}>
          {summary?.relationships && summary.relationships.length > 0 ? (
            <Table
              columns={[
                {
                  title: t('problemInvestigation.columns.relatedType'),
                  dataIndex: 'relatedType',
                  key: 'relatedType',
                  render: (type: string) => {
                    const colors: Record<string, string> = {
                      ticket: 'blue',
                      change: 'purple',
                      incident: 'red',
                    };
                    const labels: Record<string, string> = {
                      ticket: t('problemInvestigation.relatedTypeLabels.ticket'),
                      change: t('problemInvestigation.relatedTypeLabels.change'),
                      incident: t('problemInvestigation.relatedTypeLabels.incident'),
                    };
                    return <Tag color={colors[type]}>{labels[type] || type}</Tag>;
                  },
                },
                { title: t('problemInvestigation.columns.title'), dataIndex: 'relatedTitle', key: 'relatedTitle' },
                {
                  title: t('problemInvestigation.columns.relationshipType'),
                  dataIndex: 'relationshipType',
                  key: 'relationshipType',
                },
                {
                  title: t('problemInvestigation.columns.description'),
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
            <Empty
              description={t('problemInvestigation.relationships.empty')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </Card>
      ),
    },
  ];

  // keep helper used to avoid unused-var lint when getSolutionTypeLabel is referenced only via Tag template elsewhere
  void getSolutionTypeLabel;

  return (
    <>
      <Tabs
        items={tabItems}
        defaultActiveKey={initialInnerTab}
        animated={{ inkBar: true, tabPane: true }}
      />

      {/* 创建调查步骤 Modal */}
      <Modal
        title={t('problemInvestigation.modals.addStepTitle')}
        open={stepModalOpen}
        onCancel={() => setStepModalOpen(false)}
        footer={null}
        width={600}
        destroyOnHidden
      >
        <Form form={stepForm} layout="vertical" onFinish={handleCreateStep}>
          <Form.Item
            name="stepTitle"
            label={t('problemInvestigation.modals.stepTitle')}
            rules={[{ required: true, message: t('problemInvestigation.modals.stepTitleRequired') }]}
          >
            <Input placeholder={t('problemInvestigation.modals.stepTitlePlaceholder')} />
          </Form.Item>
          <Form.Item
            name="stepDescription"
            label={t('problemInvestigation.modals.stepDescription')}
            rules={[{ required: true, message: t('problemInvestigation.modals.stepDescriptionRequired') }]}
          >
            <TextArea rows={4} placeholder={t('problemInvestigation.modals.stepDescriptionPlaceholder')} />
          </Form.Item>
          <Form.Item name="notes" label={t('problemInvestigation.modals.notes')}>
            <TextArea rows={2} placeholder={t('problemInvestigation.modals.notesPlaceholder')} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('common.create')}
              </Button>
              <Button onClick={() => setStepModalOpen(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 创建根因分析 Modal */}
      <Modal
        title={t('problemInvestigation.modals.rootCauseTitle')}
        open={rootCauseModalOpen}
        onCancel={() => setRootCauseModalOpen(false)}
        footer={null}
        width={700}
        destroyOnHidden
      >
        <Form form={rootCauseForm} layout="vertical" onFinish={handleCreateRootCause}>
          <Form.Item
            name="analysisMethod"
            label={t('problemInvestigation.rootCause.analysisMethod')}
            rules={[{ required: true, message: t('problemInvestigation.modals.analysisMethodRequired') }]}
          >
            <Select
              placeholder={t('problemInvestigation.modals.analysisMethodPlaceholder')}
              options={[
                { value: '5-whys', label: t('problemInvestigation.methodOptions.5-whys') },
                { value: 'fishbone', label: t('problemInvestigation.methodOptions.fishbone') },
                { value: 'timeline', label: t('problemInvestigation.methodOptions.timeline') },
                { value: 'fault_tree', label: t('problemInvestigation.methodOptions.fault_tree') },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="rootCauseDescription"
            label={t('problemInvestigation.modals.rootCauseDescription')}
            rules={[{ required: true, message: t('problemInvestigation.modals.rootCauseDescriptionRequired') }]}
          >
            <TextArea rows={4} placeholder={t('problemInvestigation.modals.rootCauseDescriptionPlaceholder')} />
          </Form.Item>
          <Form.Item name="contributingFactors" label={t('problemInvestigation.rootCause.contributingFactors')}>
            <TextArea rows={3} placeholder={t('problemInvestigation.modals.contributingFactorsPlaceholder')} />
          </Form.Item>
          <Form.Item name="evidence" label={t('problemInvestigation.rootCause.evidence')}>
            <TextArea rows={3} placeholder={t('problemInvestigation.modals.evidencePlaceholder')} />
          </Form.Item>
          <Form.Item
            name="confidenceLevel"
            label={t('problemInvestigation.rootCause.confidence')}
            rules={[{ required: true, message: t('problemInvestigation.modals.confidenceRequired') }]}
          >
            <Select
              placeholder={t('problemInvestigation.modals.confidencePlaceholder')}
              options={[
                { value: 'low', label: t('problemInvestigation.confidenceOptions.low') },
                { value: 'medium', label: t('problemInvestigation.confidenceOptions.medium') },
                { value: 'high', label: t('problemInvestigation.confidenceOptions.high') },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('problemInvestigation.modals.submitAnalysis')}
              </Button>
              <Button onClick={() => setRootCauseModalOpen(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 创建解决方案 Modal */}
      <Modal
        title={t('problemInvestigation.modals.addSolutionTitle')}
        open={solutionModalOpen}
        onCancel={() => setSolutionModalOpen(false)}
        footer={null}
        width={700}
      >
        <Form form={solutionForm} layout="vertical" onFinish={handleCreateSolution}>
          <Form.Item
            name="solutionType"
            label={t('problemInvestigation.columns.solutionType')}
            rules={[{ required: true, message: t('problemInvestigation.modals.solutionTypeRequired') }]}
          >
            <Select
              placeholder={t('problemInvestigation.modals.solutionTypePlaceholder')}
              options={[
                { value: 'workaround', label: t('problemInvestigation.solutionTypeOptions.workaround') },
                { value: 'fix', label: t('problemInvestigation.solutionTypeOptions.fix') },
                { value: 'prevention', label: t('problemInvestigation.solutionTypeOptions.prevention') },
                { value: 'process', label: t('problemInvestigation.solutionTypeOptions.process') },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="solutionDescription"
            label={t('problemInvestigation.modals.solutionDescription')}
            rules={[{ required: true, message: t('problemInvestigation.modals.solutionDescriptionRequired') }]}
          >
            <TextArea rows={4} placeholder={t('problemInvestigation.modals.solutionDescriptionPlaceholder')} />
          </Form.Item>
          <Form.Item
            name="priority"
            label={t('problemInvestigation.columns.priority')}
            rules={[{ required: true, message: t('problemInvestigation.modals.priorityRequired') }]}
          >
            <Select
              placeholder={t('problemInvestigation.modals.priorityPlaceholder')}
              options={[
                { value: 'low', label: t('problemInvestigation.confidenceLabels.low') },
                { value: 'medium', label: t('problemInvestigation.confidenceLabels.medium') },
                { value: 'high', label: t('problemInvestigation.confidenceLabels.high') },
                { value: 'critical', label: t('problemInvestigation.knowledgeArticleStatus.draft') === '草稿' ? '紧急' : 'Critical' },
              ]}
            />
          </Form.Item>
          <Space style={{ width: '100%' }} size="large">
            <Form.Item name="estimatedEffortHours" label={t('problemInvestigation.modals.estimatedEffortHours')}>
              <Input type="number" placeholder="0" />
            </Form.Item>
            <Form.Item name="estimatedCost" label={t('problemInvestigation.modals.estimatedCost')}>
              <Input type="number" placeholder="0" />
            </Form.Item>
          </Space>
          <Form.Item name="riskAssessment" label={t('problemInvestigation.modals.riskAssessment')}>
            <TextArea rows={2} placeholder={t('problemInvestigation.modals.riskAssessmentPlaceholder')} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('problemInvestigation.modals.createSolution')}
              </Button>
              <Button onClick={() => setSolutionModalOpen(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 沉淀到知识库 Modal */}
      <Modal
        title={t('problemInvestigation.modals.depositKbTitle')}
        open={knowledgeModalOpen}
        onCancel={() => setKnowledgeModalOpen(false)}
        footer={null}
        width={700}
      >
        <Form form={knowledgeForm} layout="vertical" onFinish={handleCreateKnowledgeArticle}>
          <Form.Item
            name="articleTitle"
            label={t('problemInvestigation.modals.articleTitle')}
            rules={[{ required: true, message: t('problemInvestigation.modals.articleTitleRequired') }]}
          >
            <Input
              placeholder={t('problemInvestigation.modals.articleTitlePlaceholder')}
              defaultValue={`${t('problemInvestigation.modals.articleTitleDefaultPrefix')}${problemTitle}`}
            />
          </Form.Item>
          <Form.Item
            name="articleType"
            label={t('problemInvestigation.modals.articleType')}
            rules={[{ required: true, message: t('problemInvestigation.modals.articleTypeRequired') }]}
          >
            <Select
              placeholder={t('problemInvestigation.modals.articleTypePlaceholder')}
              options={[
                {
                  value: 'troubleshooting',
                  label: t('problemInvestigation.knowledgeArticleTypeOptions.troubleshooting'),
                },
                { value: 'solution', label: t('problemInvestigation.knowledgeArticleTypeOptions.solution') },
                { value: 'process', label: t('problemInvestigation.knowledgeArticleTypeOptions.process') },
                { value: 'prevention', label: t('problemInvestigation.knowledgeArticleTypeOptions.prevention') },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="articleContent"
            label={t('problemInvestigation.modals.articleContent')}
            rules={[{ required: true, message: t('problemInvestigation.modals.articleContentRequired') }]}
          >
            <TextArea
              rows={8}
              placeholder={t('problemInvestigation.modals.articleContentPlaceholder')}
              defaultValue={`
${t('problemInvestigation.modals.articleTemplate', {
  description: problemDescription || t('problemInvestigation.modals.articleTemplateProblemDesc'),
  rootCause:
    summary?.rootCauseAnalysis?.rootCauseDescription ||
    t('problemInvestigation.modals.articleTemplateRootCause'),
  solutions:
    summary?.solutions?.map((s: ProblemSolution) => `- ${s.solutionDescription}`).join('\n') ||
    t('problemInvestigation.modals.articleTemplateSolutions'),
  preventions:
    summary?.solutions
      ?.filter((s: ProblemSolution) => s.solutionType === 'prevention')
      .map((s: ProblemSolution) => `- ${s.solutionDescription}`)
      .join('\n') || '',
})}
              `.trim()}
            />
          </Form.Item>
          <Form.Item name="tags" label={t('problemInvestigation.modals.tags')}>
            <Select
              mode="tags"
              placeholder={t('problemInvestigation.modals.tagsPlaceholder')}
              style={{ width: '100%' }}
              options={[
                { value: 'problem', label: t('problemInvestigation.knowledgeTagLabels.problem') },
                { value: 'root-cause', label: t('problemInvestigation.knowledgeTagLabels.root-cause') },
                { value: 'solution', label: t('problemInvestigation.knowledgeTagLabels.solution') },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('problemInvestigation.modals.depositSubmit')}
              </Button>
              <Button onClick={() => setKnowledgeModalOpen(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

export default ProblemInvestigationTab;
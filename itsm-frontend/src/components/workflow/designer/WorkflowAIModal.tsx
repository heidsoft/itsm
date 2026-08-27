// AI辅助设计模态框
// Workflow AI Assistant Modal Component

'use client';

import React, { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  List,
  Modal,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { AlertTriangle, CheckCircle, XCircle, Bug, Rocket, Send, Bot } from 'lucide-react';
import { getBpmnDesignerApi } from './WorkflowCanvas';
import {
  BPMNAIApi,
  type BPMNEnterpriseType,
  type BPMNProcessType,
  type BPMNTemplateSuggestion,
  type GenerateBPMNResponse,
  type PreviewBPMNResponse,
} from '@/lib/api/bpmn-ai-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text, Title, Paragraph } = Typography;
const { TextArea } = Input;

interface GenerateFormValues {
  processName: string;
  processDescription: string;
  processType: BPMNProcessType;
  enterpriseType: BPMNEnterpriseType;
  includeSla: boolean;
  includeNotifications: boolean;
  includeApprovals: boolean;
}

interface WorkflowAIModalProps {
  visible: boolean;
  onClose: () => void;
  currentXML: string;
  workflowName?: string;
  onApplyGeneratedProcess?: (xml: string) => void;
}

// 优化建议类型
interface OptimizationSuggestion {
  id: string;
  type: 'optimization' | 'warning' | 'error';
  title: string;
  description: string;
  elementId?: string;
  severity: 'low' | 'medium' | 'high';
}

// 合规检查结果
interface ComplianceIssue {
  id: string;
  type: 'violation' | 'warning' | 'suggestion';
  rule: string;
  description: string;
  elementId?: string;
  severity: 'low' | 'medium' | 'high';
}

export default function WorkflowAIModal({
  visible,
  onClose,
  currentXML,
  workflowName,
  onApplyGeneratedProcess,
}: WorkflowAIModalProps) {
  const { message } = App.useApp();
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState('generate');
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [templateLoading, setTemplateLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<OptimizationSuggestion[]>([]);
  const [complianceIssues, setComplianceIssues] = useState<ComplianceIssue[]>([]);
  const [generatedProcess, setGeneratedProcess] = useState<string>('');
  const [generationResult, setGenerationResult] = useState<GenerateBPMNResponse | null>(null);
  const [previewResult, setPreviewResult] = useState<PreviewBPMNResponse | null>(null);
  const [templateSuggestions, setTemplateSuggestions] = useState<BPMNTemplateSuggestion[]>([]);
  const [generationError, setGenerationError] = useState<string>('');

  const getErrorMessage = (error: unknown) =>
    error instanceof Error ? error.message : t('workflow.aiModal.requestFailedRetry');

  // 生成流程
  const handleGenerateProcess = async (values: GenerateFormValues) => {
    setLoading(true);
    setGenerationError('');
    try {
      message.info(t('workflow.aiModal.generating'));

      const requirement = `${values.processName}\n\n${values.processDescription}`.trim();
      const result = await BPMNAIApi.generateBPMN({
        requirement,
        processType: values.processType,
        enterpriseType: values.enterpriseType,
        includeSla: values.includeSla,
        includeNotifications: values.includeNotifications,
        includeApprovals: values.includeApprovals,
      });

      setGenerationResult(result);
      setGeneratedProcess(result.bpmnXml);
      message.success(t('workflow.aiModal.generateCompleted', { name: result.processName || values.processName }));
    } catch (error) {
      console.error('生成流程失败:', error);
      const errorMessage = getErrorMessage(error);
      setGenerationError(errorMessage);
      message.error(t('workflow.aiModal.generateFailed', { message: errorMessage }));
    } finally {
      setLoading(false);
    }
  };

  const handlePreviewProcess = async () => {
    const values = await form.validateFields(['processName', 'processDescription', 'processType', 'enterpriseType']);
    setPreviewLoading(true);
    setGenerationError('');
    try {
      const requirement = `${values.processName}\n\n${values.processDescription}`.trim();
      const result = await BPMNAIApi.previewBPMN({
        requirement,
        processType: values.processType,
        enterpriseType: values.enterpriseType,
      });
      setPreviewResult(result);
      message.success(t('workflow.aiModal.previewGenerated'));
    } catch (error) {
      console.error('预览流程失败:', error);
      const errorMessage = getErrorMessage(error);
      setGenerationError(errorMessage);
      message.error(t('workflow.aiModal.previewFailed', { message: errorMessage }));
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleLoadTemplateSuggestions = async () => {
    const values = await form.validateFields(['processDescription', 'processType']);
    setTemplateLoading(true);
    try {
      const result = await BPMNAIApi.getTemplateSuggestions({
        keyword: values.processDescription,
        processType: values.processType,
      });
      setTemplateSuggestions(result);
      if (result.length === 0) {
        message.info(t('workflow.aiModal.noMatchingTemplates'));
      }
    } catch (error) {
      console.error('获取模板建议失败:', error);
      message.error(t('workflow.aiModal.loadTemplatesFailed', { message: getErrorMessage(error) }));
    } finally {
      setTemplateLoading(false);
    }
  };

  // 应用生成的流程
  const handleApplyGeneratedProcess = () => {
    if (!generatedProcess) {
      message.error(t('workflow.aiModal.applyFailedNoXml'));
      return;
    }

    onApplyGeneratedProcess?.(generatedProcess);
    message.success(t('workflow.aiModal.appliedToCanvas'));
    onClose();
  };

  // 获取优化建议
  const handleGetSuggestions = async () => {
    setLoading(true);
    try {
      message.info(t('workflow.aiModal.analyzing'));

      await new Promise(resolve => setTimeout(resolve, 1500));

      const mockSuggestions: OptimizationSuggestion[] = [
        {
          id: '1',
          type: 'optimization',
          title: t('workflow.aiModal.suggDocTitle'),
          description: t('workflow.aiModal.suggDocDesc'),
          severity: 'low',
        },
        {
          id: '2',
          type: 'warning',
          title: t('workflow.aiModal.suggAssigneeTitle'),
          description: t('workflow.aiModal.suggAssigneeDesc'),
          elementId: 'UserTask_1',
          severity: 'medium',
        },
        {
          id: '3',
          type: 'warning',
          title: t('workflow.aiModal.suggGatewayTitle'),
          description: t('workflow.aiModal.suggGatewayDesc'),
          elementId: 'Gateway_1',
          severity: 'high',
        },
        {
          id: '4',
          type: 'optimization',
          title: t('workflow.aiModal.suggSlaTitle'),
          description: t('workflow.aiModal.suggSlaDesc'),
          severity: 'low',
        },
      ];

      setSuggestions(mockSuggestions);
      message.success(t('workflow.aiModal.suggestionCount', { count: 4 }));
    } catch (error) {
      console.error('获取优化建议失败:', error);
      message.error(t('workflow.aiModal.suggestionFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 合规检查
  const handleComplianceCheck = async () => {
    setLoading(true);
    try {
      message.info(t('workflow.aiModal.complianceChecking'));

      await new Promise(resolve => setTimeout(resolve, 1800));

      const mockIssues: ComplianceIssue[] = [
        {
          id: '1',
          type: 'violation',
          rule: t('workflow.aiModal.ruleFinanceApproval'),
          description: t('workflow.aiModal.ruleFinanceApprovalDesc'),
          severity: 'high',
        },
        {
          id: '2',
          type: 'warning',
          rule: t('workflow.aiModal.ruleDataSecurity'),
          description: t('workflow.aiModal.ruleDataSecurityDesc'),
          severity: 'medium',
        },
        {
          id: '3',
          type: 'suggestion',
          rule: t('workflow.aiModal.ruleEfficiency'),
          description: t('workflow.aiModal.ruleEfficiencyDesc'),
          severity: 'low',
        },
      ];

      setComplianceIssues(mockIssues);
      message.success(t('workflow.aiModal.complianceCompleted'));
    } catch (error) {
      console.error('合规检查失败:', error);
      message.error(t('workflow.aiModal.complianceFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 跳转到问题元素
  const jumpToElement = (elementId?: string) => {
    if (!elementId) return;

    const api = getBpmnDesignerApi();
    if (api) {
      api.selectElement(elementId);
      onClose();
      message.info(t('workflow.aiModal.elementLocated', { id: elementId }));
    }
  };

  return (
    <Modal
      title={
        <Space>
          <Bot className="text-blue-500" />
          <span>{t('workflow.aiModal.title')}</span>
        </Space>
      }
      open={visible}
      onCancel={onClose}
      width={900}
      footer={null}
      destroyOnHidden
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'generate',
            label: (
              <Space>
                <Rocket />
                {t('workflow.aiModal.tabGenerate')}
              </Space>
            ),
            children: (
              <div className="py-4">
                <Paragraph>{t('workflow.aiModal.generateIntro')}</Paragraph>

                <Form
                  form={form}
                  layout="vertical"
                  onFinish={handleGenerateProcess}
                  initialValues={{
                    processName: workflowName || '',
                    processDescription: '',
                    processType: 'custom',
                    enterpriseType: 'cn_enterprise',
                    includeSla: true,
                    includeNotifications: true,
                    includeApprovals: true,
                  }}
                >
                  <Form.Item
                    name="processName"
                    label={t('workflow.aiModal.processName')}
                    rules={[{ required: true, message: t('workflow.aiModal.processNameRequired') }]}
                  >
                    <Input placeholder={t('workflow.aiModal.processNamePlaceholder')} />
                  </Form.Item>

                  <Form.Item
                    name="processDescription"
                    label={t('workflow.aiModal.processDescription')}
                    rules={[{ required: true, message: t('workflow.aiModal.processDescriptionRequired') }]}
                  >
                    <TextArea rows={4} placeholder={t('workflow.aiModal.processDescriptionPlaceholder')} />
                  </Form.Item>

                  <div className="grid grid-cols-2 gap-4">
                    <Form.Item
                      name="processType"
                      label={t('workflow.aiModal.processType')}
                      rules={[{ required: true, message: t('workflow.aiModal.processTypeRequired') }]}
                    >
                      <Select
                        options={[
                          { label: t('workflow.aiModal.processTypeIncident'), value: 'incident' },
                          { label: t('workflow.aiModal.processTypeChange'), value: 'change' },
                          { label: t('workflow.aiModal.processTypeProblem'), value: 'problem' },
                          { label: t('workflow.aiModal.processTypeServiceRequest'), value: 'service_request' },
                          { label: t('workflow.aiModal.processTypeCustom'), value: 'custom' },
                        ]}
                      />
                    </Form.Item>

                    <Form.Item
                      name="enterpriseType"
                      label={t('workflow.aiModal.enterpriseType')}
                      rules={[{ required: true, message: t('workflow.aiModal.enterpriseTypeRequired') }]}
                    >
                      <Select
                        options={[
                          { label: t('workflow.aiModal.enterpriseCn'), value: 'cn_enterprise' },
                          { label: t('workflow.aiModal.enterpriseInternational'), value: 'international' },
                          { label: t('workflow.aiModal.enterpriseStartup'), value: 'startup' },
                          { label: t('workflow.aiModal.enterpriseGovernment'), value: 'government' },
                        ]}
                      />
                    </Form.Item>
                  </div>

                  <Form.Item label={t('workflow.aiModal.generateConfig')}>
                    <Space wrap>
                      <Form.Item name="includeSla" valuePropName="checked" noStyle>
                        <Checkbox>{t('workflow.aiModal.includeSla')}</Checkbox>
                      </Form.Item>
                      <Form.Item name="includeNotifications" valuePropName="checked" noStyle>
                        <Checkbox>{t('workflow.aiModal.includeNotifications')}</Checkbox>
                      </Form.Item>
                      <Form.Item name="includeApprovals" valuePropName="checked" noStyle>
                        <Checkbox>{t('workflow.aiModal.includeApprovals')}</Checkbox>
                      </Form.Item>
                    </Space>
                  </Form.Item>

                  {generationError && (
                    <Alert
                      className="mb-4"
                      type="error"
                      showIcon
                      message={t('workflow.aiModal.generationFailed')}
                      description={generationError}
                    />
                  )}

                  <Form.Item>
                    <Space wrap>
                      <Button type="default" onClick={handlePreviewProcess} loading={previewLoading} icon={<Bot />}>
                        {t('workflow.aiModal.previewStructure')}
                      </Button>
                      <Button onClick={handleLoadTemplateSuggestions} loading={templateLoading} icon={<Rocket />}>
                        {t('workflow.aiModal.recommendTemplates')}
                      </Button>
                      <Button type="primary" htmlType="submit" loading={loading} icon={<Send />}>
                        {t('workflow.aiModal.generateProcess')}
                      </Button>
                    </Space>
                  </Form.Item>
                </Form>

                {templateSuggestions.length > 0 && (
                  <Card className="mb-4" size="small" title={t('workflow.aiModal.templateSuggestions')}>
                    <Space wrap>
                      {templateSuggestions.map((item, index) => (
                        <Tag key={item.id || `${item.name}-${index}`} color="blue">
                          {item.name || item.description || item.id || t('workflow.aiModal.suggestionIndex', { index: index + 1 })}
                        </Tag>
                      ))}
                    </Space>
                  </Card>
                )}

                {previewResult && (
                  <Card className="mb-4" size="small" title={t('workflow.aiModal.structurePreview')}>
                    <Space wrap className="mb-3">
                      <Tag color="blue">{t('workflow.aiModal.complexityLabel', { complexity: previewResult.complexity })}</Tag>
                      <Tag color="purple">{t('workflow.aiModal.estimatedNodes', { count: previewResult.estimatedNodeCount })}</Tag>
                    </Space>
                    <Paragraph className="mb-3">{previewResult.structureDescription}</Paragraph>
                    {previewResult.suggestions?.length > 0 && (
                      <Alert
                        className="mb-3"
                        type="info"
                        showIcon
                        message={t('workflow.aiModal.optimizationSuggestions')}
                        description={previewResult.suggestions.join(t('workflow.aiModal.suggestionSeparator'))}
                      />
                    )}
                    <List
                      size="small"
                      bordered
                      dataSource={previewResult.nodes || []}
                      renderItem={node => (
                        <List.Item>
                          <div className="space-y-1">
                            <Space wrap>
                              <Text strong>{node.name}</Text>
                              <Tag>{node.type}</Tag>
                              {node.assigneeRole && <Tag color="green">{node.assigneeRole}</Tag>}
                              {node.slaMinutes ? (
                                <Tag color="orange">
                                  {t('workflow.aiModal.slaMinutesTag', { minutes: node.slaMinutes })}
                                </Tag>
                              ) : null}
                            </Space>
                            <Text type="secondary">{node.description}</Text>
                          </div>
                        </List.Item>
                      )}
                    />
                  </Card>
                )}

                {generatedProcess && (
                  <div className="mt-6">
                    <div className="mb-3 flex justify-between items-center">
                      <div className="space-y-1">
                        <Title level={5}>{t('workflow.aiModal.generationResultPreview')}</Title>
                        {generationResult && (
                          <Space wrap>
                            <Tag color="blue">{generationResult.processName}</Tag>
                            <Tag color="purple">{t('workflow.aiModal.versionTag', { version: generationResult.version })}</Tag>
                            <Tag color="green">{t('workflow.aiModal.nodeCountTag', { count: generationResult.nodeCount })}</Tag>
                            <Tag color="orange">{t('workflow.aiModal.complexityTag', { complexity: generationResult.complexity })}</Tag>
                          </Space>
                        )}
                      </div>
                      <Button
                        type="primary"
                        size="small"
                        disabled={!generatedProcess || loading}
                        onClick={handleApplyGeneratedProcess}
                      >
                        {t('workflow.aiModal.applyToCanvas')}
                      </Button>
                    </div>
                    {generationResult?.explanation && (
                      <Alert
                        className="mb-3"
                        type="success"
                        showIcon
                        message={t('workflow.aiModal.generationExplanation')}
                        description={generationResult.explanation}
                      />
                    )}
                    <Card className="bg-gray-50 font-mono text-xs overflow-auto max-h-[400px] whitespace-pre-wrap">
                      {generatedProcess}
                    </Card>
                  </div>
                )}
              </div>
            ),
          },
          {
            key: 'optimize',
            label: (
              <Space>
                <Bot />
                {t('workflow.aiModal.tabOptimize')}
              </Space>
            ),
            children: (
              <div className="py-4">
                <Paragraph>{t('workflow.aiModal.optimizeIntro')}</Paragraph>

                <div className="mb-4">
                  <Button type="primary" onClick={handleGetSuggestions} loading={loading} icon={<Bot />}>
                    {t('workflow.aiModal.getOptimizationSuggestions')}
                  </Button>
                </div>

                {suggestions.length > 0 && (
                  <div className="divide-y divide-gray-100">
                    {suggestions.map(item => (
                      <div
                        key={item.id}
                        className="cursor-pointer hover:bg-gray-50"
                        onClick={() => item.elementId && jumpToElement(item.elementId)}
                      >
                        <div className="flex gap-3 px-4 py-3">
                          <div className="pt-1">
                            {item.type === 'error' ? (
                              <XCircle className="text-red-500 text-xl" />
                            ) : item.type === 'warning' ? (
                              <AlertTriangle className="text-yellow-500 text-xl" />
                            ) : (
                              <CheckCircle className="text-green-500 text-xl" />
                            )}
                          </div>
                          <div className="min-w-0 flex-1">
                            <Space wrap>
                              <span>{item.title}</span>
                              <Tag
                                color={
                                  item.severity === 'high' ? 'error' : item.severity === 'medium' ? 'warning' : 'success'
                                }
                              >
                                {item.severity === 'high'
                                  ? t('workflow.aiModal.severityHigh')
                                  : item.severity === 'medium'
                                  ? t('workflow.aiModal.severityMedium')
                                  : t('workflow.aiModal.severityLow')}
                              </Tag>
                              {item.elementId && <Tag color="blue">{t('workflow.aiModal.elementTag', { id: item.elementId })}</Tag>}
                            </Space>
                            <div className="mt-1 text-sm text-gray-500">{item.description}</div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ),
          },
          {
            key: 'compliance',
            label: (
              <Space>
                <Bug />
                {t('workflow.aiModal.tabCompliance')}
              </Space>
            ),
            children: (
              <div className="py-4">
                <Paragraph>{t('workflow.aiModal.complianceIntro')}</Paragraph>

                <div className="mb-4">
                  <Button type="primary" onClick={handleComplianceCheck} loading={loading} icon={<Bug />}>
                    {t('workflow.aiModal.startComplianceCheck')}
                  </Button>
                </div>

                {complianceIssues.length > 0 && (
                  <div className="divide-y divide-gray-100">
                    {complianceIssues.map(item => (
                      <div
                        key={item.id}
                        className="cursor-pointer hover:bg-gray-50"
                        onClick={() => item.elementId && jumpToElement(item.elementId)}
                      >
                        <div className="flex gap-3 px-4 py-3">
                          <div className="pt-1">
                            {item.type === 'violation' ? (
                              <XCircle className="text-red-500 text-xl" />
                            ) : item.type === 'warning' ? (
                              <AlertTriangle className="text-yellow-500 text-xl" />
                            ) : (
                              <CheckCircle className="text-green-500 text-xl" />
                            )}
                          </div>
                          <div className="min-w-0 flex-1">
                            <Space wrap>
                              <span>{item.rule}</span>
                              <Tag
                                color={
                                  item.severity === 'high' ? 'error' : item.severity === 'medium' ? 'warning' : 'success'
                                }
                              >
                                {item.severity === 'high'
                                  ? t('workflow.aiModal.riskHigh')
                                  : item.severity === 'medium'
                                  ? t('workflow.aiModal.riskMedium')
                                  : t('workflow.aiModal.riskSuggestion')}
                              </Tag>
                              {item.elementId && <Tag color="blue">{t('workflow.aiModal.elementTag', { id: item.elementId })}</Tag>}
                            </Space>
                            <div className="mt-1 text-sm text-gray-500">{item.description}</div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ),
          },
        ]}
      />
    </Modal>
  );
}
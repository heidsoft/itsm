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
    error instanceof Error ? error.message : '请求失败，请稍后重试';

  // 生成流程
  const handleGenerateProcess = async (values: GenerateFormValues) => {
    setLoading(true);
    setGenerationError('');
    try {
      message.info('AI正在生成流程，请稍候...');

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
      message.success(`流程生成完成：${result.processName || values.processName}`);
    } catch (error) {
      console.error('生成流程失败:', error);
      const errorMessage = getErrorMessage(error);
      setGenerationError(errorMessage);
      message.error(`生成流程失败：${errorMessage}`);
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
      message.success('流程结构预览已生成');
    } catch (error) {
      console.error('预览流程失败:', error);
      const errorMessage = getErrorMessage(error);
      setGenerationError(errorMessage);
      message.error(`预览流程失败：${errorMessage}`);
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
        message.info('暂无匹配的模板建议');
      }
    } catch (error) {
      console.error('获取模板建议失败:', error);
      message.error(`获取模板建议失败：${getErrorMessage(error)}`);
    } finally {
      setTemplateLoading(false);
    }
  };

  // 应用生成的流程
  const handleApplyGeneratedProcess = () => {
    if (!generatedProcess) {
      message.error('应用失败，未找到生成的流程XML');
      return;
    }

    onApplyGeneratedProcess?.(generatedProcess);
    message.success('已应用生成的流程');
    onClose();
  };

  // 获取优化建议
  const handleGetSuggestions = async () => {
    setLoading(true);
    try {
      message.info('AI正在分析流程，请稍候...');
      
      // 模拟延迟
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // 模拟建议结果
      const mockSuggestions: OptimizationSuggestion[] = [
        {
          id: '1',
          type: 'optimization',
          title: '添加流程说明文档',
          description: '建议为关键节点添加说明文档，帮助审批人员了解操作要求',
          severity: 'low'
        },
        {
          id: '2',
          type: 'warning',
          title: '用户任务未配置审批人',
          description: '检测到有3个用户任务未配置审批人或候选组，部署后可能无法正常运行',
          elementId: 'UserTask_1',
          severity: 'medium'
        },
        {
          id: '3',
          type: 'warning',
          title: '网关缺少默认分支',
          description: '排他网关"金额是否大于10000"未配置默认分支，可能导致流程卡住',
          elementId: 'Gateway_1',
          severity: 'high'
        },
        {
          id: '4',
          type: 'optimization',
          title: '建议添加SLA配置',
          description: '为关键审批节点配置SLA超时提醒，提升流程执行效率',
          severity: 'low'
        }
      ];
      
      setSuggestions(mockSuggestions);
      message.success('分析完成，共找到4条建议');
    } catch (error) {
      console.error('获取优化建议失败:', error);
      message.error('获取优化建议失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  // 合规检查
  const handleComplianceCheck = async () => {
    setLoading(true);
    try {
      message.info('AI正在进行合规检查，请稍候...');
      
      // 模拟延迟
      await new Promise(resolve => setTimeout(resolve, 1800));
      
      // 模拟检查结果
      const mockIssues: ComplianceIssue[] = [
        {
          id: '1',
          type: 'violation',
          rule: '财务审批流程规范',
          description: '金额大于50000的流程必须包含CEO审批节点，当前流程缺失',
          severity: 'high'
        },
        {
          id: '2',
          type: 'warning',
          rule: '数据安全规范',
          description: '流程中包含敏感数据处理，建议添加数据脱敏配置',
          severity: 'medium'
        },
        {
          id: '3',
          type: 'suggestion',
          rule: '流程效率优化',
          description: '当前流程有4个串行审批节点，建议考虑并行审批以提升效率',
          severity: 'low'
        }
      ];
      
      setComplianceIssues(mockIssues);
      message.success('合规检查完成');
    } catch (error) {
      console.error('合规检查失败:', error);
      message.error('合规检查失败，请重试');
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
      message.info(`已定位到元素 ${elementId}`);
    }
  };

  return (
    <Modal
      title={
        <Space>
          <Bot className="text-blue-500" />
          <span>AI工作流助手</span>
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
                生成流程
              </Space>
            ),
            children: (
          <div className="py-4">
            <Paragraph>
              描述您需要的工作流场景，AI将自动为您生成符合BPMN 2.0规范的流程定义。
            </Paragraph>
            
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
                label="流程名称"
                rules={[{ required: true, message: '请输入流程名称' }]}
              >
                <Input placeholder="例如：费用报销流程" />
              </Form.Item>

              <Form.Item
                name="processDescription"
                label="流程描述"
                rules={[{ required: true, message: '请描述您需要的流程' }]}
              >
                <TextArea
                  rows={4}
                  placeholder="例如：员工提交报销申请，部门经理审批，金额大于5000需要总经理审批，最后财务打款。超过3天未审批自动提醒。"
                />
              </Form.Item>

              <div className="grid grid-cols-2 gap-4">
                <Form.Item
                  name="processType"
                  label="流程类型"
                  rules={[{ required: true, message: '请选择流程类型' }]}
                >
                  <Select
                    options={[
                      { label: '事件管理', value: 'incident' },
                      { label: '变更管理', value: 'change' },
                      { label: '问题管理', value: 'problem' },
                      { label: '服务请求', value: 'service_request' },
                      { label: '自定义流程', value: 'custom' },
                    ]}
                  />
                </Form.Item>

                <Form.Item
                  name="enterpriseType"
                  label="企业类型"
                  rules={[{ required: true, message: '请选择企业类型' }]}
                >
                  <Select
                    options={[
                      { label: '国内企业', value: 'cn_enterprise' },
                      { label: '国际企业', value: 'international' },
                      { label: '创业公司', value: 'startup' },
                      { label: '政府/事业单位', value: 'government' },
                    ]}
                  />
                </Form.Item>
              </div>

              <Form.Item label="生成配置">
                <Space wrap>
                  <Form.Item name="includeSla" valuePropName="checked" noStyle>
                    <Checkbox>包含SLA配置</Checkbox>
                  </Form.Item>
                  <Form.Item name="includeNotifications" valuePropName="checked" noStyle>
                    <Checkbox>包含通知配置</Checkbox>
                  </Form.Item>
                  <Form.Item name="includeApprovals" valuePropName="checked" noStyle>
                    <Checkbox>包含审批节点</Checkbox>
                  </Form.Item>
                </Space>
              </Form.Item>

              {generationError && (
                <Alert
                  className="mb-4"
                  type="error"
                  showIcon
                  title="AI流程生成请求失败"
                  description={generationError}
                />
              )}

              <Form.Item>
                <Space wrap>
                  <Button
                    type="default"
                    onClick={handlePreviewProcess}
                    loading={previewLoading}
                    icon={<Bot />}
                  >
                    预览结构
                  </Button>
                  <Button
                    onClick={handleLoadTemplateSuggestions}
                    loading={templateLoading}
                    icon={<Rocket />}
                  >
                    推荐模板
                  </Button>
                  <Button type="primary" htmlType="submit" loading={loading} icon={<Send />}>
                    生成流程
                  </Button>
                </Space>
              </Form.Item>
            </Form>

            {templateSuggestions.length > 0 && (
              <Card className="mb-4" size="small" title="模板建议">
                <Space wrap>
                  {templateSuggestions.map((item, index) => (
                    <Tag key={item.id || `${item.name}-${index}`} color="blue">
                      {item.name || item.description || item.id || `建议 ${index + 1}`}
                    </Tag>
                  ))}
                </Space>
              </Card>
            )}

            {previewResult && (
              <Card className="mb-4" size="small" title="结构预览">
                <Space wrap className="mb-3">
                  <Tag color="blue">复杂度: {previewResult.complexity}</Tag>
                  <Tag color="purple">预计节点: {previewResult.estimatedNodeCount}</Tag>
                </Space>
                <Paragraph className="mb-3">{previewResult.structureDescription}</Paragraph>
                {previewResult.suggestions?.length > 0 && (
                  <Alert
                    className="mb-3"
                    type="info"
                    showIcon
                    title="优化建议"
                    description={previewResult.suggestions.join('；')}
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
                          {node.slaMinutes ? <Tag color="orange">SLA {node.slaMinutes}分钟</Tag> : null}
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
                    <Title level={5}>生成结果预览</Title>
                    {generationResult && (
                      <Space wrap>
                        <Tag color="blue">{generationResult.processName}</Tag>
                        <Tag color="purple">版本 {generationResult.version}</Tag>
                        <Tag color="green">节点 {generationResult.nodeCount}</Tag>
                        <Tag color="orange">复杂度 {generationResult.complexity}</Tag>
                      </Space>
                    )}
                  </div>
                  <Button
                    type="primary"
                    size="small"
                    disabled={!generatedProcess || loading}
                    onClick={handleApplyGeneratedProcess}
                  >
                    应用到画布
                  </Button>
                </div>
                {generationResult?.explanation && (
                  <Alert
                    className="mb-3"
                    type="success"
                    showIcon
                    title="生成说明"
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
                优化建议
              </Space>
            ),
            children: (
          <div className="py-4">
            <Paragraph>
              AI将分析您当前的流程设计，提供优化建议和潜在问题检测。
            </Paragraph>
            
            <div className="mb-4">
              <Button type="primary" onClick={handleGetSuggestions} loading={loading} icon={<Bot />}>
                获取优化建议
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
                        {
                        item.type === 'error' ? (
                          <XCircle className="text-red-500 text-xl" />
                        ) : item.type === 'warning' ? (
                          <AlertTriangle className="text-yellow-500 text-xl" />
                        ) : (
                          <CheckCircle className="text-green-500 text-xl" />
                        )
                        }
                      </div>
                      <div className="min-w-0 flex-1">
                        <Space wrap>
                          <span>{item.title}</span>
                          <Tag color={
                            item.severity === 'high' ? 'error' :
                            item.severity === 'medium' ? 'warning' : 'success'
                          }>
                            {item.severity === 'high' ? '高优先级' :
                             item.severity === 'medium' ? '中优先级' : '低优先级'}
                          </Tag>
                          {item.elementId && (
                            <Tag color="blue">
                              元素: {item.elementId}
                            </Tag>
                          )}
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
                合规检查
              </Space>
            ),
            children: (
          <div className="py-4">
            <Paragraph>
              基于企业流程规范和最佳实践，检查当前流程是否符合合规要求。
            </Paragraph>
            
            <div className="mb-4">
              <Button type="primary" onClick={handleComplianceCheck} loading={loading} icon={<Bug />}>
                开始合规检查
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
                        {
                        item.type === 'violation' ? (
                          <XCircle className="text-red-500 text-xl" />
                        ) : item.type === 'warning' ? (
                          <AlertTriangle className="text-yellow-500 text-xl" />
                        ) : (
                          <CheckCircle className="text-green-500 text-xl" />
                        )
                        }
                      </div>
                      <div className="min-w-0 flex-1">
                        <Space wrap>
                          <span>{item.rule}</span>
                          <Tag color={
                            item.severity === 'high' ? 'error' :
                            item.severity === 'medium' ? 'warning' : 'success'
                          }>
                            {item.severity === 'high' ? '高风险' :
                             item.severity === 'medium' ? '中风险' : '建议'}
                          </Tag>
                          {item.elementId && (
                            <Tag color="blue">
                              元素: {item.elementId}
                            </Tag>
                          )}
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

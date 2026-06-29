// AI辅助设计模态框
// Workflow AI Assistant Modal Component

'use client';

import React, { useState } from 'react';
import { Modal, Tabs, Form, Input, Button, Space, Typography, List, Card, Tag, message } from 'antd';
import { 
  RocketOutlined, 
  RobotOutlined, 
  BugOutlined, 
  SendOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import TextArea from 'antd/es/input/TextArea';
import { getBpmnDesignerApi } from './WorkflowCanvas';

const { Text, Title, Paragraph } = Typography;
const { TabPane } = Tabs;

interface WorkflowAIModalProps {
  visible: boolean;
  onClose: () => void;
  currentXML: string;
  workflowName?: string;
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
  workflowName
}: WorkflowAIModalProps) {
  const [activeTab, setActiveTab] = useState('generate');
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<OptimizationSuggestion[]>([]);
  const [complianceIssues, setComplianceIssues] = useState<ComplianceIssue[]>([]);
  const [generatedProcess, setGeneratedProcess] = useState<string>('');

  // 生成流程
  const handleGenerateProcess = async (values: any) => {
    setLoading(true);
    try {
      // TODO: 调用AI生成流程API
      message.info('AI正在生成流程，请稍候...');
      
      // 模拟生成延迟
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // 模拟生成结果
      const generatedXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  id="Definitions_1"
                  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true" name="${values.processName || 'AI生成的流程'}">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_1" name="提交申请">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_2" name="部门经理审批">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_3" name="财务审批">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_1" name="金额是否大于10000">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:userTask id="UserTask_4" name="总经理审批">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="审批通过">
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:incoming>Flow_6</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="UserTask_2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_2" targetRef="UserTask_3" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="UserTask_3" targetRef="Gateway_1" />
    <bpmn:sequenceFlow id="Flow_5" name="金额>10000" sourceRef="Gateway_1" targetRef="UserTask_4">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">
        <![CDATA[${amount > 10000}]]>
      </bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_6" name="金额<=10000" sourceRef="Gateway_1" targetRef="EndEvent_1" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="UserTask_4" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`;
      
      setGeneratedProcess(generatedXML);
      message.success('流程生成完成');
    } catch (error) {
      console.error('生成流程失败:', error);
      message.error('生成流程失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  // 应用生成的流程
  const handleApplyGeneratedProcess = () => {
    const api = getBpmnDesignerApi();
    if (!api || !generatedProcess) {
      message.error('应用失败，设计器未就绪');
      return;
    }

    // TODO: 替换当前画布的XML
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
          <RobotOutlined className="text-blue-500" />
          <span>AI工作流助手</span>
        </Space>
      }
      open={visible}
      onCancel={onClose}
      width={900}
      footer={null}
      destroyOnClose
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab={
          <Space>
            <RocketOutlined />
            生成流程
          </Space>
        } key="generate">
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
                processDescription: ''
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

              <Form.Item>
                <Button type="primary" htmlType="submit" loading={loading} icon={<SendOutlined />}>
                  生成流程
                </Button>
              </Form.Item>
            </Form>

            {generatedProcess && (
              <div className="mt-6">
                <div className="mb-3 flex justify-between items-center">
                  <Title level={5}>生成结果预览</Title>
                  <Button type="primary" size="small" onClick={handleApplyGeneratedProcess}>
                    应用到画布
                  </Button>
                </div>
                <Card className="bg-gray-50 font-mono text-xs overflow-auto max-h-[400px] whitespace-pre-wrap">
                  {generatedProcess}
                </Card>
              </div>
            )}
          </div>
        </TabPane>

        <TabPane tab={
          <Space>
            <RobotOutlined />
            优化建议
          </Space>
        } key="optimize">
          <div className="py-4">
            <Paragraph>
              AI将分析您当前的流程设计，提供优化建议和潜在问题检测。
            </Paragraph>
            
            <div className="mb-4">
              <Button type="primary" onClick={handleGetSuggestions} loading={loading} icon={<RobotOutlined />}>
                获取优化建议
              </Button>
            </div>

            {suggestions.length > 0 && (
              <List
                dataSource={suggestions}
                renderItem={item => (
                  <List.Item
                    className="cursor-pointer hover:bg-gray-50"
                    onClick={() => item.elementId && jumpToElement(item.elementId)}
                  >
                    <List.Item.Meta
                      avatar={
                        item.type === 'error' ? (
                          <CloseCircleOutlined className="text-red-500 text-xl" />
                        ) : item.type === 'warning' ? (
                          <WarningOutlined className="text-yellow-500 text-xl" />
                        ) : (
                          <CheckCircleOutlined className="text-green-500 text-xl" />
                        )
                      }
                      title={
                        <Space>
                          <span>{item.title}</span>
                          <Tag color={
                            item.severity === 'high' ? 'error' :
                            item.severity === 'medium' ? 'warning' : 'success'
                          } size="small">
                            {item.severity === 'high' ? '高优先级' :
                             item.severity === 'medium' ? '中优先级' : '低优先级'}
                          </Tag>
                          {item.elementId && (
                            <Tag color="blue" size="small">
                              元素: {item.elementId}
                            </Tag>
                          )}
                        </Space>
                      }
                      description={item.description}
                    />
                  </List.Item>
                )}
              />
            )}
          </div>
        </TabPane>

        <TabPane tab={
          <Space>
            <BugOutlined />
            合规检查
          </Space>
        } key="compliance">
          <div className="py-4">
            <Paragraph>
              基于企业流程规范和最佳实践，检查当前流程是否符合合规要求。
            </Paragraph>
            
            <div className="mb-4">
              <Button type="primary" onClick={handleComplianceCheck} loading={loading} icon={<BugOutlined />}>
                开始合规检查
              </Button>
            </div>

            {complianceIssues.length > 0 && (
              <List
                dataSource={complianceIssues}
                renderItem={item => (
                  <List.Item
                    className="cursor-pointer hover:bg-gray-50"
                    onClick={() => item.elementId && jumpToElement(item.elementId)}
                  >
                    <List.Item.Meta
                      avatar={
                        item.type === 'violation' ? (
                          <CloseCircleOutlined className="text-red-500 text-xl" />
                        ) : item.type === 'warning' ? (
                          <WarningOutlined className="text-yellow-500 text-xl" />
                        ) : (
                          <CheckCircleOutlined className="text-green-500 text-xl" />
                        )
                      }
                      title={
                        <Space>
                          <span>{item.rule}</span>
                          <Tag color={
                            item.severity === 'high' ? 'error' :
                            item.severity === 'medium' ? 'warning' : 'success'
                          } size="small">
                            {item.severity === 'high' ? '高风险' :
                             item.severity === 'medium' ? '中风险' : '建议'}
                          </Tag>
                          {item.elementId && (
                            <Tag color="blue" size="small">
                              元素: {item.elementId}
                            </Tag>
                          )}
                        </Space>
                      }
                      description={item.description}
                    />
                  </List.Item>
                )}
              />
            )}
          </div>
        </TabPane>
      </Tabs>
    </Modal>
  );
}

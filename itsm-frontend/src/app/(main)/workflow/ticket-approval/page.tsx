"use client";

import React, { useState, useEffect, useCallback } from "react";
import {
  Card,
  Button,
  Space,
  Typography,
  Alert,
  Modal,
  Form,
  Input,
  Select,
  Row,
  Col,
  Statistic,
  Tag,
  Badge,
  List,
  Tooltip,
  Divider,
  Steps,
  Timeline,
  App,
  message,
} from "antd";
import {
  ArrowLeft,
  Save,
  Play,
  Eye,
  Download,
  Upload,
  GitBranch,
  Users,
  Clock,
  CheckCircle,
  AlertCircle,
  Settings,
  FileText,
  Zap,
} from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import BPMNDesigner from "@/components/workflow/BPMNDesigner";
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { Option } = Select;

// 获取节点类型颜色
const getNodeTypeColor = (type: string) => {
  switch (type) {
    case "start":
      return "green";
    case "approval":
      return "blue";
    case "condition":
      return "gold";
    case "action":
      return "purple";
    case "end":
      return "red";
    default:
      return "default";
  }
};

// 获取节点类型名称（将在组件内使用国际化）
const getNodeTypeName = (type: string, t: (key: string) => string) => {
  switch (type) {
    case "start":
      return t('workflow.nodeTypeStart');
    case "approval":
      return t('workflow.nodeTypeApproval');
    case "condition":
      return t('workflow.nodeTypeCondition');
    case "action":
      return t('workflow.nodeTypeAction');
    case "end":
      return t('workflow.nodeTypeEnd');
    default:
      return t('workflow.nodeTypeUnknown');
  }
};

interface TicketApprovalWorkflow {
  id?: number;
  name: string;
  description: string;
  type: string;
  nodes: any[];
  metadata: {
    version: string;
    lastModified: string;
    nodeCount: number;
    approvalCount: number;
  };
}

// 默认工单审批流程 BPMN XML
const getDefaultBPMNXML = () => {
  return '<?xml version="1.0" encoding="UTF-8"?>' +
'<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" ' +
'  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" ' +
'  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" ' +
'  xmlns:di="http://www.omg.org/spec/DD/20100524/DI" ' +
'  id="ticket_approval" targetNamespace="http://bpmn.io/schema/bpmn">' +
'  <bpmn:process id="TicketApproval" name="工单审批流程" isExecutable="true">' +
'    <bpmn:startEvent id="StartEvent_1" name="工单提交">' +
'      <bpmn:outgoing>Flow_1</bpmn:outgoing>' +
'    </bpmn:startEvent>' +
'    <bpmn:userTask id="Task_approval" name="审批">' +
'      <bpmn:incoming>Flow_1</bpmn:incoming>' +
'      <bpmn:outgoing>Flow_2</bpmn:outgoing>' +
'    </bpmn:userTask>' +
'    <bpmn:exclusiveGateway id="Gateway_1" name="是否通过">' +
'      <bpmn:incoming>Flow_2</bpmn:incoming>' +
'      <bpmn:outgoing>Flow_approved</bpmn:outgoing>' +
'      <bpmn:outgoing>Flow_rejected</bpmn:outgoing>' +
'    </bpmn:exclusiveGateway>' +
'    <bpmn:endEvent id="EndEvent_approved" name="已完成">' +
'      <bpmn:incoming>Flow_approved</bpmn:incoming>' +
'    </bpmn:endEvent>' +
'    <bpmn:endEvent id="EndEvent_rejected" name="已拒绝">' +
'      <bpmn:incoming>Flow_rejected</bpmn:incoming>' +
'    </bpmn:endEvent>' +
'    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_approval"/>' +
'    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_approval" targetRef="Gateway_1"/>' +
'    <bpmn:sequenceFlow id="Flow_approved" sourceRef="Gateway_1" targetRef="EndEvent_approved">' +
'      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${approved == true}</bpmn:conditionExpression>' +
'    </bpmn:sequenceFlow>' +
'    <bpmn:sequenceFlow id="Flow_rejected" sourceRef="Gateway_1" targetRef="EndEvent_rejected">' +
'      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${approved == false}</bpmn:conditionExpression>' +
'    </bpmn:sequenceFlow>' +
'  </bpmn:process>' +
'  <bpmndi:BPMNDiagram id="BPMNDiagram_1">' +
'    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="TicketApproval"/>' +
'  </bpmndi:BPMNDiagram>' +
'</bpmn:definitions>';
};

const TicketApprovalWorkflowPage = () => {
  const { t } = useI18n();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { modal } = App.useApp();
  const [workflow, setWorkflow] = useState<TicketApprovalWorkflow | null>(null);
  const [loading, setLoading] = useState(false);
  const [saveModalVisible, setSaveModalVisible] = useState(false);
  const [currentXML, setCurrentXML] = useState(getDefaultBPMNXML());
  const [hasChanges, setHasChanges] = useState(false);
  const [form] = Form.useForm();

  // 从URL参数获取工作流ID（如果有）
  const workflowId = searchParams.get('id');

  // 加载已有工作流（如果指定了ID）
  useEffect(() => {
    if (workflowId) {
      loadWorkflow(workflowId);
    }
  }, [workflowId]);

  const loadWorkflow = async (id: string) => {
    try {
      const { WorkflowAPI } = await import('@/lib/api/workflow-api');
      const response = await WorkflowAPI.getProcessDefinition(id) as any;

      if (response?.bpmn_xml) {
        let xmlContent = response.bpmn_xml;
        // 尝试Base64解码
        if (!xmlContent.trim().startsWith('<?xml')) {
          try {
            xmlContent = atob(xmlContent);
          } catch (e) {
            console.warn('XML Base64 decode failed');
          }
        }
        setCurrentXML(xmlContent);
      }

      setWorkflow({
        id: parseInt(id),
        name: response.name || t('workflow.ticketApprovalProcess'),
        description: response.description || '',
        type: response.category || 'approval',
        nodes: [],
        metadata: {
          version: response.version || '1.0.0',
          lastModified: response.updated_at || new Date().toISOString(),
          nodeCount: 0,
          approvalCount: 0,
        },
      });
    } catch (error) {
      console.error('加载工作流失败:', error);
      message.error(t('workflow.loadFailed') || '加载工作流失败');
    }
  };

  // BPMN XML 变化处理
  const handleBPMNChange = useCallback((xml: string) => {
    setCurrentXML(xml);
    setHasChanges(true);
    setWorkflow(prev => prev ? {
      ...prev,
      metadata: {
        ...prev.metadata,
        lastModified: new Date().toISOString(),
      }
    } : null);
  }, []);

  // 保存工作流
  const handleSave = async (xml: string) => {
    setSaveModalVisible(true);
    setCurrentXML(xml);
  };

  const handleSaveWorkflow = (workflowData: any) => {
    console.log("保存工作流数据:", workflowData);

    // 显示保存模态框
    setSaveModalVisible(true);
    setWorkflow(workflowData);

    // 设置表单初始值
    form.setFieldsValue({
      name: workflowData.name || t('workflow.ticketApprovalProcess'),
      description: workflowData.description || t('workflow.ticketApprovalDescription'),
      category: t('workflow.approvalProcess'),
      version: workflowData.metadata?.version || "1.0.0",
    });
  };

  const handleFinalSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const { WorkflowAPI } = await import('@/lib/api/workflow-api');

      // 构建保存数据
      const workflowData = {
        code: workflow?.id ? `ticket_approval_${workflow.id}` : `ticket_approval_${Date.now()}`,
        name: values.name || t('workflow.ticketApprovalProcess'),
        description: values.description || '',
        type: values.category || 'approval',
        bpmn_xml: currentXML,
      };

      if (workflow?.id) {
        // 更新已有工作流
        await WorkflowAPI.updateWorkflow(workflow.id.toString(), workflowData);
      } else {
        // 创建新工作流
        await WorkflowAPI.createWorkflow(workflowData);
      }

      message.success(t('workflow.ticketApprovalSaveSuccess'));
      setSaveModalVisible(false);

      // 保存成功后返回工作流列表
      setTimeout(() => {
        router.push("/workflow");
      }, 1000);
    } catch (error) {
      console.error('保存工作流失败:', error);
      message.error(t('workflow.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    modal.confirm({
      title: t('workflow.confirmLeave'),
      content: t('workflow.leaveConfirmation'),
      okText: t('workflow.confirmLeave'),
      cancelText: t('workflow.continueEdit'),
      okType: "danger",
      onOk: () => {
        router.push("/workflow");
      },
    });
  };

  const handlePreview = () => {
    if (!workflow) {
      message.warning(t('workflow.designWorkflowFirst'));
      return;
    }

    // 创建预览用的流程步骤
    const previewSteps =
      workflow.nodes?.map((node: any, index: number) => {
        const status: 'process' | 'wait' = index === 0 ? 'process' : 'wait';
        return {
          title: node.name,
          description: node.description,
          status,
          icon:
            node.type === 'start' ? (
              <Play className="w-4 h-4" />
            ) : node.type === 'approval' ? (
              <Users className="w-4 h-4" />
            ) : node.type === 'condition' ? (
              <GitBranch className="w-4 h-4" />
            ) : node.type === 'action' ? (
              <Zap className="w-4 h-4" />
            ) : node.type === 'end' ? (
              <CheckCircle className="w-4 h-4" />
            ) : null,
        };
      }) || [];

    modal.info({
      title: (
        <div className="flex items-center gap-2">
          <Eye className="w-5 h-5" />
          {t('workflow.workflowPreview')}
        </div>
      ),
      width: 800,
      content: (
        <div className="space-y-6">
          <Alert
            message={t('workflow.workflowOverview')}
            description={t('workflow.workflowOverviewDescription', { 
              total: workflow.nodes?.length || 0,
              approval: workflow.nodes?.filter((n: any) => n.type === "approval").length || 0
            })}
            type="info"
            showIcon
          />

          <div>
            <Title level={5}>{t('workflow.processSteps')}</Title>
            <Steps
              direction="vertical"
              size="small"
              current={0}
              items={previewSteps}
            />
          </div>

          <div>
            <Title level={5}>{t('workflow.nodeDetails')}</Title>
            <List
              size="small"
              dataSource={workflow.nodes || []}
              renderItem={(node: any, index: number) => (
                <List.Item>
                  <div className="flex items-center gap-3 w-full">
                    <Badge count={index + 1} color="blue" />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <Text strong>{node.name}</Text>
                        <Tag color={getNodeTypeColor(node.type)}>
                          {getNodeTypeName(node.type, t)}
                        </Tag>
                      </div>
                      {node.description && (
                        <Text type="secondary" className="text-sm">
                          {node.description}
                        </Text>
                      )}
                      {node.type === "approval" && node.config?.approvers && (
                        <div className="mt-1">
                          <Tag color="blue" className="text-xs">
                            {t('workflow.approversCount', { count: node.config.approvers.value?.length || 0 })}
                          </Tag>
                          <Tag color="orange" className="text-xs">
                            {t('workflow.timeoutHours', { hours: node.config.timeout || 24 })}
                          </Tag>
                        </div>
                      )}
                    </div>
                  </div>
                </List.Item>
              )}
            />
          </div>
        </div>
      ),
    });
  };

  // getNodeTypeName 已在文件顶部定义，使用国际化版本

  return (
    <div className="h-screen flex flex-col">
      {/* 页面头部 */}
      <div className="border-b border-gray-200 bg-white">
        <div className="px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Button
                icon={<ArrowLeft className="w-4 h-4" />}
                onClick={handleCancel}
              >
                {t('workflow.backToWorkflowList')}
              </Button>
              <Divider type="vertical" />
              <div>
                <Title level={3} className="!mb-1">
                  <GitBranch className="inline-block w-6 h-6 mr-2" />
                  {t('workflow.ticketApprovalDesigner')}
                </Title>
                <Text type="secondary">
                  {t('workflow.ticketApprovalDesignerDescription')}
                </Text>
              </div>
            </div>

            <Space>
              <Button
                icon={<Eye className="w-4 h-4" />}
                onClick={handlePreview}
                disabled={!workflow}
              >
                {t('workflow.preview')}
              </Button>
              <Button
                icon={<Download className="w-4 h-4" />}
                disabled={!workflow}
              >
                {t('workflow.export')}
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                disabled={!workflow}
                onClick={() => setSaveModalVisible(true)}
              >
                {t('workflow.saveWorkflow')}
              </Button>
            </Space>
          </div>
        </div>
      </div>

      {/* BPMN 设计器区域 */}
      <div className="flex-1 bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        <BPMNDesigner
          xml={currentXML}
          onSave={handleSave}
          onChange={handleBPMNChange}
          height="100%"
        />
      </div>

      {/* 保存确认模态框 */}
      <Modal
        title={t('workflow.saveTicketApprovalWorkflow')}
        open={saveModalVisible}
        onOk={handleFinalSave}
        onCancel={() => setSaveModalVisible(false)}
        width={600}
        confirmLoading={loading}
        okText={t('workflow.confirmSave')}
        cancelText={t('workflow.cancel')}
      >
        <div className="space-y-6">
          <Alert
            message={t('workflow.saveConfirmation')}
            description={t('workflow.saveConfirmationDescription')}
            type="info"
            showIcon
          />

          {workflow && (
            <Card size="small" title={t('workflow.processStatistics')}>
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic
                    title={t('workflow.totalNodes')}
                    value={workflow.metadata?.nodeCount || 0}
                    prefix={<GitBranch className="w-4 h-4" />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title={t('workflow.approvalNodes')}
                    value={workflow.metadata?.approvalCount || 0}
                    prefix={<Users className="w-4 h-4" />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title={t('workflow.conditionNodes')}
                    value={
                      workflow.nodes?.filter((n: any) => n.type === "condition")
                        .length || 0
                    }
                    prefix={<Settings className="w-4 h-4" />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title={t('workflow.actionNodes')}
                    value={
                      workflow.nodes?.filter((n: any) => n.type === "action")
                        .length || 0
                    }
                    prefix={<Zap className="w-4 h-4" />}
                  />
                </Col>
              </Row>
            </Card>
          )}

          <Form form={form} layout="vertical">
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  label={t('workflow.processName')}
                  name="name"
                  rules={[{ required: true, message: t('workflow.processNameRequired') }]}
                >
                  <Input placeholder={t('workflow.processNamePlaceholder')} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={t('workflow.processCategory')}
                  name="category"
                  rules={[{ required: true, message: t('workflow.processCategoryRequired') }]}
                >
                  <Select placeholder={t('workflow.processCategoryPlaceholder')}>
                    <Option value={t('workflow.approvalProcess')}>{t('workflow.approvalProcess')}</Option>
                    <Option value={t('workflow.ticketProcess')}>{t('workflow.ticketProcess')}</Option>
                    <Option value={t('workflow.incidentProcess')}>{t('workflow.incidentProcess')}</Option>
                    <Option value={t('workflow.changeProcess')}>{t('workflow.changeProcess')}</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label={t('workflow.processDescription')}
              name="description"
              rules={[{ required: true, message: t('workflow.processDescriptionRequired') }]}
            >
              <Input.TextArea rows={3} placeholder={t('workflow.processDescriptionPlaceholder')} />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  label={t('workflow.version')}
                  name="version"
                  rules={[{ required: true, message: t('workflow.versionRequired') }]}
                >
                  <Input placeholder={t('workflow.versionPlaceholder')} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label={t('workflow.initialStatus')} name="status" initialValue="draft">
                  <Select disabled>
                    <Option value="draft">{t('workflow.draft')}</Option>
                    <Option value="active">{t('workflow.statusEnabled')}</Option>
                    <Option value="inactive">{t('workflow.statusDisabled')}</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>
          </Form>
        </div>
      </Modal>
    </div>
  );
};

export default TicketApprovalWorkflowPage;

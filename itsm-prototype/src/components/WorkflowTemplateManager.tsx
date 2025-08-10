"use client";

import React, { useState, useEffect } from "react";
import {
  Modal,
  Form,
  Input,
  Select,
  Button,
  Space,
  message,
  Card,
  Row,
  Col,
  Tag,
  Typography,
  Divider,
  Upload,
  Tooltip,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CopyOutlined,
  DownloadOutlined,
  UploadOutlined,
  EyeOutlined,
} from "@ant-design/icons";
import { WorkflowAPI } from "../app/lib/workflow-api";

const { Option } = Select;
const { TextArea } = Input;
const { Title, Text } = Typography;

interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  xml: string;
  thumbnail?: string;
  tags: string[];
  created_at: string;
  updated_at: string;
  usage_count: number;
  is_public: boolean;
}

interface WorkflowTemplateManagerProps {
  visible: boolean;
  onClose: () => void;
  onSelectTemplate: (template: WorkflowTemplate) => void;
}

const WorkflowTemplateManager: React.FC<WorkflowTemplateManagerProps> = ({
  visible,
  onClose,
  onSelectTemplate,
}) => {
  const [templates, setTemplates] = useState<WorkflowTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedTemplate, setSelectedTemplate] =
    useState<WorkflowTemplate | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showPreviewModal, setShowPreviewModal] = useState(false);
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // 预定义的工作流模板
  const defaultTemplates: WorkflowTemplate[] = [
    {
      id: "simple-approval",
      name: "简单审批流程",
      description: "包含开始、审批、结束三个节点的简单审批流程",
      category: "approval",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_1" name="审批">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_2</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_1_di" bpmnElement="UserTask_1">
        <dc:Bounds x="240" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="392" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="240" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="340" y="120" />
        <di:waypoint x="392" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`,
      tags: ["审批", "简单", "基础"],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      usage_count: 15,
      is_public: true,
    },
    {
      id: "it-incident-process",
      name: "IT事件处理流程",
      description: "完整的IT事件处理工作流，包含分类、分配、处理、验证等步骤",
      category: "it_service",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ITIncidentProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="事件报告">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="ClassifyTask" name="事件分类">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="PriorityGateway" name="优先级判断">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="HighPriorityTask" name="高优先级处理">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="NormalPriorityTask" name="标准处理">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:parallelGateway id="MergeGateway" name="合并网关">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:incoming>Flow_6</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:userTask id="ValidationTask" name="验证解决">
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:outgoing>Flow_8</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="事件关闭">
      <bpmn:incoming>Flow_8</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ClassifyTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ClassifyTask" targetRef="PriorityGateway" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="PriorityGateway" targetRef="HighPriorityTask" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="PriorityGateway" targetRef="NormalPriorityTask" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="HighPriorityTask" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="NormalPriorityTask" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="MergeGateway" targetRef="ValidationTask" />
    <bpmn:sequenceFlow id="Flow_8" sourceRef="ValidationTask" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
      tags: ["IT服务", "事件处理", "复杂流程"],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      usage_count: 8,
      is_public: true,
    },
    {
      id: "change-management",
      name: "变更管理流程",
      description:
        "IT变更管理标准流程，包含变更申请、评估、审批、实施、验证等步骤",
      category: "change_management",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ChangeManagementProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="变更申请">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="AssessmentTask" name="变更评估">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="RiskGateway" name="风险评估">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="LowRiskApproval" name="低风险审批">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="HighRiskApproval" name="高风险审批">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:parallelGateway id="MergeGateway" name="合并审批">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:incoming>Flow_6</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:userTask id="ImplementationTask" name="变更实施">
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:outgoing>Flow_8</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="VerificationTask" name="变更验证">
      <bpmn:incoming>Flow_8</bpmn:incoming>
      <bpmn:outgoing>Flow_9</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="变更完成">
      <bpmn:incoming>Flow_9</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="AssessmentTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="AssessmentTask" targetRef="RiskGateway" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="RiskGateway" targetRef="LowRiskApproval" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="RiskGateway" targetRef="HighRiskApproval" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="LowRiskApproval" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="HighRiskApproval" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="MergeGateway" targetRef="ImplementationTask" />
    <bpmn:sequenceFlow id="Flow_8" sourceRef="ImplementationTask" targetRef="VerificationTask" />
    <bpmn:sequenceFlow id="Flow_9" sourceRef="VerificationTask" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
      tags: ["变更管理", "ITIL", "复杂流程"],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      usage_count: 12,
      is_public: true,
    },
  ];

  useEffect(() => {
    if (visible) {
      loadTemplates();
    }
  }, [visible]);

  const loadTemplates = async () => {
    setLoading(true);
    try {
      // 这里应该调用API获取模板列表
      // 暂时使用默认模板
      setTemplates(defaultTemplates);
    } catch (error) {
      console.error("加载模板失败:", error);
      message.error("加载模板失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTemplate = async (values: any) => {
    try {
      // 这里应该调用API创建模板
      const newTemplate: WorkflowTemplate = {
        id: `template-${Date.now()}`,
        ...values,
        tags: values.tags || [],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        usage_count: 0,
        is_public: false,
      };

      setTemplates([...templates, newTemplate]);
      setShowCreateModal(false);
      form.resetFields();
      message.success("模板创建成功");
    } catch (error) {
      console.error("创建模板失败:", error);
      message.error("创建模板失败");
    }
  };

  const handleEditTemplate = async (values: any) => {
    if (!selectedTemplate) return;

    try {
      // 这里应该调用API更新模板
      const updatedTemplates = templates.map((template) =>
        template.id === selectedTemplate.id
          ? { ...template, ...values, updated_at: new Date().toISOString() }
          : template
      );

      setTemplates(updatedTemplates);
      setShowEditModal(false);
      editForm.resetFields();
      setSelectedTemplate(null);
      message.success("模板更新成功");
    } catch (error) {
      console.error("更新模板失败:", error);
      message.error("更新模板失败");
    }
  };

  const handleDeleteTemplate = async (templateId: string) => {
    try {
      // 这里应该调用API删除模板
      const updatedTemplates = templates.filter(
        (template) => template.id !== templateId
      );
      setTemplates(updatedTemplates);
      message.success("模板删除成功");
    } catch (error) {
      console.error("删除模板失败:", error);
      message.error("删除模板失败");
    }
  };

  const handleCopyTemplate = (template: WorkflowTemplate) => {
    const copiedTemplate: WorkflowTemplate = {
      ...template,
      id: `copy-${Date.now()}`,
      name: `${template.name} (副本)`,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      usage_count: 0,
    };

    setTemplates([...templates, copiedTemplate]);
    message.success("模板复制成功");
  };

  const handleExportTemplate = (template: WorkflowTemplate) => {
    const blob = new Blob([template.xml], { type: "application/xml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${template.name}.bpmn`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    message.success("模板导出成功");
  };

  const handleImportTemplate = (file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const xml = e.target?.result as string;
      const importedTemplate: WorkflowTemplate = {
        id: `imported-${Date.now()}`,
        name: file.name.replace(".bpmn", "").replace(".xml", ""),
        description: "从文件导入的模板",
        category: "imported",
        xml,
        tags: ["导入"],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        usage_count: 0,
        is_public: false,
      };

      setTemplates([...templates, importedTemplate]);
      message.success("模板导入成功");
    };
    reader.readAsText(file);
  };

  const handleSelectTemplate = (template: WorkflowTemplate) => {
    onSelectTemplate(template);
    onClose();
  };

  const categories = [
    { value: "approval", label: "审批流程" },
    { value: "it_service", label: "IT服务" },
    { value: "change_management", label: "变更管理" },
    { value: "incident_management", label: "事件管理" },
    { value: "problem_management", label: "问题管理" },
    { value: "general", label: "通用流程" },
    { value: "imported", label: "导入模板" },
  ];

  return (
    <>
      <Modal
        title="工作流模板管理"
        open={visible}
        onCancel={onClose}
        width={1200}
        footer={null}
        destroyOnClose
      >
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setShowCreateModal(true)}
            >
              创建模板
            </Button>
            <Upload
              accept=".bpmn,.xml"
              showUploadList={false}
              beforeUpload={(file) => {
                handleImportTemplate(file);
                return false;
              }}
            >
              <Button icon={<UploadOutlined />}>导入模板</Button>
            </Upload>
          </Space>
        </div>

        <Row gutter={[16, 16]}>
          {templates.map((template) => (
            <Col key={template.id} xs={24} sm={12} lg={8}>
              <Card
                hoverable
                actions={[
                  <Tooltip key="view" title="预览">
                    <EyeOutlined
                      onClick={() => {
                        setSelectedTemplate(template);
                        setShowPreviewModal(true);
                      }}
                    />
                  </Tooltip>,
                  <Tooltip key="edit" title="编辑">
                    <EditOutlined
                      onClick={() => {
                        setSelectedTemplate(template);
                        editForm.setFieldsValue(template);
                        setShowEditModal(true);
                      }}
                    />
                  </Tooltip>,
                  <Tooltip key="copy" title="复制">
                    <CopyOutlined
                      onClick={() => handleCopyTemplate(template)}
                    />
                  </Tooltip>,
                  <Tooltip key="export" title="导出">
                    <DownloadOutlined
                      onClick={() => handleExportTemplate(template)}
                    />
                  </Tooltip>,
                  <Tooltip key="delete" title="删除">
                    <DeleteOutlined
                      style={{ color: "#ff4d4f" }}
                      onClick={() => handleDeleteTemplate(template.id)}
                    />
                  </Tooltip>,
                ]}
              >
                <Card.Meta
                  title={
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                      }}
                    >
                      <span>{template.name}</span>
                      <Tag color={template.is_public ? "blue" : "orange"}>
                        {template.is_public ? "公开" : "私有"}
                      </Tag>
                    </div>
                  }
                  description={
                    <div>
                      <Text type="secondary">{template.description}</Text>
                      <div style={{ marginTop: 8 }}>
                        <Tag color="blue">{template.category}</Tag>
                        <Text type="secondary" style={{ marginLeft: 8 }}>
                          使用 {template.usage_count} 次
                        </Text>
                      </div>
                      <div style={{ marginTop: 8 }}>
                        {template.tags.map((tag) => (
                          <Tag key={tag} size="small">
                            {tag}
                          </Tag>
                        ))}
                      </div>
                    </div>
                  }
                />
                <div style={{ marginTop: 16 }}>
                  <Button
                    type="primary"
                    block
                    onClick={() => handleSelectTemplate(template)}
                  >
                    使用此模板
                  </Button>
                </div>
              </Card>
            </Col>
          ))}
        </Row>

        {/* 创建模板模态框 */}
        <Modal
          title="创建模板"
          open={showCreateModal}
          onCancel={() => setShowCreateModal(false)}
          onOk={() => form.submit()}
          destroyOnClose
        >
          <Form form={form} onFinish={handleCreateTemplate} layout="vertical">
            <Form.Item
              name="name"
              label="模板名称"
              rules={[{ required: true, message: "请输入模板名称" }]}
            >
              <Input placeholder="请输入模板名称" />
            </Form.Item>
            <Form.Item
              name="description"
              label="模板描述"
              rules={[{ required: true, message: "请输入模板描述" }]}
            >
              <TextArea rows={3} placeholder="请输入模板描述" />
            </Form.Item>
            <Form.Item
              name="category"
              label="模板分类"
              rules={[{ required: true, message: "请选择模板分类" }]}
            >
              <Select placeholder="请选择模板分类">
                {categories.map((cat) => (
                  <Option key={cat.value} value={cat.value}>
                    {cat.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item
              name="xml"
              label="BPMN XML"
              rules={[{ required: true, message: "请输入BPMN XML" }]}
            >
              <TextArea rows={10} placeholder="请输入BPMN XML内容" />
            </Form.Item>
            <Form.Item name="tags" label="标签">
              <Select
                mode="tags"
                placeholder="请输入标签"
                style={{ width: "100%" }}
              />
            </Form.Item>
          </Form>
        </Modal>

        {/* 编辑模板模态框 */}
        <Modal
          title="编辑模板"
          open={showEditModal}
          onCancel={() => setShowEditModal(false)}
          onOk={() => editForm.submit()}
          destroyOnClose
        >
          <Form form={editForm} onFinish={handleEditTemplate} layout="vertical">
            <Form.Item
              name="name"
              label="模板名称"
              rules={[{ required: true, message: "请输入模板名称" }]}
            >
              <Input placeholder="请输入模板名称" />
            </Form.Item>
            <Form.Item
              name="description"
              label="模板描述"
              rules={[{ required: true, message: "请输入模板描述" }]}
            >
              <TextArea rows={3} placeholder="请输入模板描述" />
            </Form.Item>
            <Form.Item
              name="category"
              label="模板分类"
              rules={[{ required: true, message: "请选择模板分类" }]}
            >
              <Select placeholder="请选择模板分类">
                {categories.map((cat) => (
                  <Option key={cat.value} value={cat.value}>
                    {cat.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item
              name="xml"
              label="BPMN XML"
              rules={[{ required: true, message: "请输入BPMN XML" }]}
            >
              <TextArea rows={10} placeholder="请输入BPMN XML内容" />
            </Form.Item>
            <Form.Item name="tags" label="标签">
              <Select
                mode="tags"
                placeholder="请输入标签"
                style={{ width: "100%" }}
              />
            </Form.Item>
          </Form>
        </Modal>

        {/* 预览模板模态框 */}
        <Modal
          title={`预览模板: ${selectedTemplate?.name}`}
          open={showPreviewModal}
          onCancel={() => setShowPreviewModal(false)}
          footer={[
            <Button key="close" onClick={() => setShowPreviewModal(false)}>
              关闭
            </Button>,
            <Button
              key="use"
              type="primary"
              onClick={() =>
                selectedTemplate && handleSelectTemplate(selectedTemplate)
              }
            >
              使用此模板
            </Button>,
          ]}
          width={800}
          destroyOnClose
        >
          {selectedTemplate && (
            <div>
              <div style={{ marginBottom: 16 }}>
                <Text strong>描述:</Text> {selectedTemplate.description}
              </div>
              <div style={{ marginBottom: 16 }}>
                <Text strong>分类:</Text> {selectedTemplate.category}
              </div>
              <div style={{ marginBottom: 16 }}>
                <Text strong>标签:</Text>
                {selectedTemplate.tags.map((tag) => (
                  <Tag key={tag} style={{ marginLeft: 8 }}>
                    {tag}
                  </Tag>
                ))}
              </div>
              <Divider />
              <div>
                <Text strong>BPMN XML:</Text>
                <pre
                  style={{
                    background: "#f5f5f5",
                    padding: 16,
                    borderRadius: 4,
                    maxHeight: 400,
                    overflow: "auto",
                    fontSize: 12,
                  }}
                >
                  {selectedTemplate.xml}
                </pre>
              </div>
            </div>
          )}
        </Modal>
      </Modal>
    </>
  );
};

export default WorkflowTemplateManager;

"use client";

import React, { useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Layout,
  Button,
  Space,
  message,
  Typography,
  Row,
  Col,
  Tag,
  Card,
  Tabs,
  Select,
  Input,
  Form,
  Modal,
  Timeline,
  Badge,
  Alert,
} from "antd";
import {
  ArrowLeft,
  Save,
  PlayCircle,
  GitBranch,
  Settings,
  Eye,
  Edit3,
  CheckCircle,
  Clock,
  AlertCircle,
} from "lucide-react";
import EnhancedBPMNDesigner from "../../components/EnhancedBPMNDesigner";
import { WorkflowAPI } from "../../lib/workflow-api";

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;

interface WorkflowDesignerPageProps {
  params: { id?: string };
}

interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  version: string;
  category: string;
  status: "draft" | "active" | "inactive" | "archived";
  xml: string;
  created_at: string;
  updated_at: string;
  created_by: string;
  tags: string[];
  approval_config?: ApprovalConfig;
  variables?: WorkflowVariable[];
  sla_config?: SLAConfig;
}

interface ApprovalConfig {
  require_approval: boolean;
  approval_type: "single" | "parallel" | "sequential" | "conditional";
  approvers: string[];
  auto_approve_roles: string[];
  escalation_rules: EscalationRule[];
}

interface EscalationRule {
  level: number;
  timeout_hours: number;
  escalate_to: string[];
  action: "notify" | "auto_approve" | "escalate";
}

interface WorkflowVariable {
  name: string;
  type: "string" | "number" | "boolean" | "date" | "object";
  required: boolean;
  default_value?: string | number | boolean | Date | Record<string, unknown>;
  description: string;
}

interface SLAConfig {
  response_time_hours: number;
  resolution_time_hours: number;
  business_hours_only: boolean;
  exclude_weekends: boolean;
  exclude_holidays: boolean;
}

interface WorkflowVersion {
  id: string;
  version: string;
  status: "draft" | "active" | "archived";
  created_at: string;
  created_by: string;
  change_log: string;
  xml: string;
}

const WorkflowDesignerPage: React.FC<WorkflowDesignerPageProps> = ({
  params,
}) => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [form] = Form.useForm();

  const [workflow, setWorkflow] = useState<WorkflowDefinition | null>(null);
  const [saving, setSaving] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [currentXML, setCurrentXML] = useState("");
  const [hasChanges, setHasChanges] = useState(false);
  const [activeTab, setActiveTab] = useState("designer");
  const [showVersionModal, setShowVersionModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [workflowVersions, setWorkflowVersions] = useState<WorkflowVersion[]>(
    []
  );
  const [approvalConfig, setApprovalConfig] = useState<ApprovalConfig>({
    require_approval: true,
    approval_type: "sequential",
    approvers: [],
    auto_approve_roles: [],
    escalation_rules: [],
  });

  // 从URL参数获取工作流ID
  const workflowId = params.id || searchParams.get("id");

  useEffect(() => {
    if (workflowId) {
      loadWorkflow(workflowId);
      loadWorkflowVersions(workflowId);
    } else {
      // 创建新工作流
      const newWorkflow: WorkflowDefinition = {
        id: "new",
        name: "新工作流",
        description: "",
        version: "1.0.0",
        category: "general",
        status: "draft",
        xml: getDefaultBPMNXML(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        created_by: "当前用户",
        tags: [],
        approval_config: approvalConfig,
        variables: [],
        sla_config: {
          response_time_hours: 24,
          resolution_time_hours: 72,
          business_hours_only: true,
          exclude_weekends: true,
          exclude_holidays: true,
        },
      };
      setWorkflow(newWorkflow);
      setCurrentXML(getDefaultBPMNXML());
    }
  }, [workflowId]);

  const loadWorkflow = async (id: string) => {
    if (id === "new") return;

    try {
      // 使用新的BPMN API
      const response = await WorkflowAPI.getProcessDefinition(id);
      setWorkflow({
        id: response.key,
        name: response.name,
        description: response.description || "",
        version: response.version.toString(),
        category: response.category || "general",
        status: response.is_active ? "active" : "inactive",
        xml: "", // BPMN XML需要从部署的资源中获取
        created_at: response.created_at,
        updated_at: response.updated_at,
        created_by: "系统", // 从用户信息中获取
        tags: [],
        approval_config: approvalConfig,
        variables: [],
        sla_config: {
          response_time_hours: 24,
          resolution_time_hours: 72,
          business_hours_only: true,
          exclude_weekends: true,
          exclude_holidays: true,
        },
      });
      // 这里需要从部署资源中获取BPMN XML
      setCurrentXML(getDefaultBPMNXML());
    } catch (error) {
      console.error("加载工作流失败:", error);
      message.error("加载工作流失败");
    }
  };

  const loadWorkflowVersions = async (id: string) => {
    if (id === "new") return;

    try {
      const versions = await WorkflowAPI.getProcessVersions(id);
      setWorkflowVersions(versions);
    } catch (error) {
      console.error("加载工作流版本失败:", error);
    }
  };

  const getDefaultBPMNXML = () => {
    return `<?xml version="1.0" encoding="UTF-8"?>
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
    <bpmn:userTask id="UserTask_1" name="提交申请">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_2" name="审核">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_3</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="UserTask_2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_2" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="158" y="145" width="24" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_1_di" bpmnElement="UserTask_1">
        <dc:Bounds x="240" y="80" width="100" height="80" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="255" y="105" width="70" height="30" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_2_di" bpmnElement="UserTask_2">
        <dc:Bounds x="400" y="80" width="100" height="80" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="425" y="105" width="50" height="30" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="552" y="102" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="558" y="145" width="24" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="240" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="340" y="120" />
        <di:waypoint x="400" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_3_di" bpmnElement="Flow_3">
        <di:waypoint x="500" y="120" />
        <di:waypoint x="552" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;
  };

  const handleSave = async (xml: string) => {
    if (!workflow) return;

    setSaving(true);
    try {
      if (workflow.id === "new") {
        // 创建新工作流 - 使用新的BPMN API
        const response = await WorkflowAPI.createProcessDefinition({
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmn_xml: xml,
          tenant_id: 1, // 从当前用户信息中获取
        });

        // 更新工作流ID和状态
        setWorkflow((prev) =>
          prev
            ? {
                ...prev,
                id: response.key,
                status: "draft",
              }
            : null
        );

        message.success("工作流创建成功");
      } else {
        // 更新现有工作流
        await WorkflowAPI.updateProcessDefinition(workflow.id, {
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmn_xml: xml,
        });

        message.success("工作流更新成功");
      }

      setCurrentXML(xml);
      setHasChanges(false);
    } catch (error) {
      console.error("保存工作流失败:", error);
      message.error("保存工作流失败");
    } finally {
      setSaving(false);
    }
  };

  const handleDeploy = async () => {
    if (!workflow || !currentXML) return;

    setDeploying(true);
    try {
      // 部署工作流
      await WorkflowAPI.deployProcessDefinition(workflow.id, {
        bpmn_xml: currentXML,
        activate: true,
      });

      // 更新状态
      setWorkflow((prev) =>
        prev
          ? {
              ...prev,
              status: "active",
            }
          : null
      );

      message.success("工作流部署成功");
    } catch (error) {
      console.error("部署工作流失败:", error);
      message.error("部署工作流失败");
    } finally {
      setDeploying(false);
    }
  };

  const handleCreateVersion = async () => {
    if (!workflow) return;

    try {
      const newVersion = await WorkflowAPI.createProcessVersion(workflow.id, {
        version: `${parseFloat(workflow.version) + 0.1}`.slice(0, 3),
        change_log: "创建新版本",
        bpmn_xml: currentXML,
      });

      setWorkflowVersions([...workflowVersions, newVersion]);
      setWorkflow((prev) =>
        prev
          ? {
              ...prev,
              version: newVersion.version,
            }
          : null
      );

      message.success("新版本创建成功");
      setShowVersionModal(false);
    } catch (error) {
      console.error("创建版本失败:", error);
      message.error("创建版本失败");
    }
  };

  const handleSwitchVersion = async (versionId: string) => {
    try {
      const version = workflowVersions.find((v) => v.id === versionId);
      if (version) {
        setCurrentXML(version.xml);
        setWorkflow((prev) =>
          prev
            ? {
                ...prev,
                version: version.version,
              }
            : null
        );
        message.success(`已切换到版本 ${version.version}`);
      }
    } catch (error) {
      console.error("切换版本失败:", error);
      message.error("切换版本失败");
    }
  };

  const handleSaveSettings = async () => {
    try {
      const values = await form.validateFields();

      setWorkflow((prev) =>
        prev
          ? {
              ...prev,
              approval_config: values.approval_config,
              sla_config: values.sla_config,
            }
          : null
      );

      message.success("设置保存成功");
      setShowSettingsModal(false);
    } catch (error) {
      console.error("保存设置失败:", error);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "active":
        return "success";
      case "draft":
        return "processing";
      case "inactive":
        return "default";
      case "archived":
        return "warning";
      default:
        return "default";
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle className="w-4 h-4" />;
      case "draft":
        return <Edit3 className="w-4 h-4" />;
      case "inactive":
        return <Clock className="w-4 h-4" />;
      case "archived":
        return <AlertCircle className="w-4 h-4" />;
      default:
        return <Clock className="w-4 h-4" />;
    }
  };

  return (
    <Layout style={{ height: "100vh" }}>
      <Header
        style={{
          background: "#fff",
          padding: "0 24px",
          borderBottom: "1px solid #f0f0f0",
        }}
      >
        <div className="flex items-center justify-between h-full">
          <div className="flex items-center">
            <Button
              type="text"
              icon={<ArrowLeft className="w-4 h-4" />}
              onClick={() => router.back()}
              className="mr-4"
            >
              返回
            </Button>
            <div>
              <Title level={4} className="!mb-0">
                {workflow?.name || "工作流设计器"}
              </Title>
              <div className="flex items-center gap-2">
                <Tag color={getStatusColor(workflow?.status || "draft")}>
                  {getStatusIcon(workflow?.status || "draft")}
                  {workflow?.status === "active"
                    ? "已激活"
                    : workflow?.status === "draft"
                    ? "草稿"
                    : workflow?.status === "inactive"
                    ? "未激活"
                    : "已归档"}
                </Tag>
                <Text type="secondary">版本 {workflow?.version}</Text>
                {workflow?.category && (
                  <Tag color="blue">{workflow.category}</Tag>
                )}
              </div>
            </div>
          </div>

          <Space>
            <Button
              icon={<GitBranch className="w-4 h-4" />}
              onClick={() => setShowVersionModal(true)}
            >
              版本管理
            </Button>
            <Button
              icon={<Settings className="w-4 h-4" />}
              onClick={() => setShowSettingsModal(true)}
            >
              流程设置
            </Button>
            <Button
              icon={<Save className="w-4 h-4" />}
              loading={saving}
              onClick={() => handleSave(currentXML)}
              disabled={!hasChanges}
            >
              保存
            </Button>
            <Button
              type="primary"
              icon={<PlayCircle className="w-4 h-4" />}
              loading={deploying}
              onClick={handleDeploy}
              disabled={!workflow || workflow.status === "active"}
            >
              部署
            </Button>
          </Space>
        </div>
      </Header>

      <Layout>
        <Content style={{ padding: "24px" }}>
          <Tabs activeKey={activeTab} onChange={setActiveTab}>
            <TabPane tab="流程设计" key="designer">
              <div style={{ height: "calc(100vh - 200px)" }}>
                <EnhancedBPMNDesigner
                  xml={currentXML}
                  onSave={handleSave}
                  onChange={(xml) => {
                    setCurrentXML(xml);
                    setHasChanges(true);
                  }}
                />
              </div>
            </TabPane>

            <TabPane tab="版本历史" key="versions">
              <Card>
                <div className="flex justify-between items-center mb-4">
                  <Title level={5}>版本历史</Title>
                  <Button
                    type="primary"
                    icon={<GitBranch className="w-4 h-4" />}
                    onClick={() => setShowVersionModal(true)}
                  >
                    创建新版本
                  </Button>
                </div>

                <Timeline>
                  {workflowVersions.map((version) => (
                    <Timeline.Item
                      key={version.id}
                      dot={
                        <Badge
                          status={
                            version.status === "active" ? "success" : "default"
                          }
                          text={version.status === "active" ? "当前" : ""}
                        />
                      }
                    >
                      <div className="flex justify-between items-center">
                        <div>
                          <Text strong>版本 {version.version}</Text>
                          <div className="text-sm text-gray-500">
                            {version.change_log}
                          </div>
                          <div className="text-xs text-gray-400">
                            {new Date(version.created_at).toLocaleString()} -{" "}
                            {version.created_by}
                          </div>
                        </div>
                        <Space>
                          <Button
                            size="small"
                            icon={<Eye className="w-3 h-3" />}
                            onClick={() => handleSwitchVersion(version.id)}
                          >
                            查看
                          </Button>
                          {version.status !== "active" && (
                            <Button
                              size="small"
                              type="primary"
                              onClick={() => handleSwitchVersion(version.id)}
                            >
                              切换到此版本
                            </Button>
                          )}
                        </Space>
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              </Card>
            </TabPane>

            <TabPane tab="流程配置" key="config">
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Card title="审批配置" className="h-full">
                    <div className="space-y-4">
                      <div>
                        <Text strong>审批类型</Text>
                        <div className="mt-2">
                          <Select
                            value={approvalConfig.approval_type}
                            onChange={(value) =>
                              setApprovalConfig((prev) => ({
                                ...prev,
                                approval_type: value,
                              }))
                            }
                            style={{ width: "100%" }}
                          >
                            <Option value="single">单人审批</Option>
                            <Option value="parallel">并行审批</Option>
                            <Option value="sequential">串行审批</Option>
                            <Option value="conditional">条件审批</Option>
                          </Select>
                        </div>
                      </div>

                      <div>
                        <Text strong>审批人</Text>
                        <div className="mt-2">
                          <Select
                            mode="multiple"
                            placeholder="选择审批人"
                            value={approvalConfig.approvers}
                            onChange={(value) =>
                              setApprovalConfig((prev) => ({
                                ...prev,
                                approvers: value,
                              }))
                            }
                            style={{ width: "100%" }}
                          >
                            <Option value="user1">张三</Option>
                            <Option value="user2">李四</Option>
                            <Option value="user3">王五</Option>
                          </Select>
                        </div>
                      </div>

                      <div>
                        <Text strong>自动审批角色</Text>
                        <div className="mt-2">
                          <Select
                            mode="multiple"
                            placeholder="选择角色"
                            value={approvalConfig.auto_approve_roles}
                            onChange={(value) =>
                              setApprovalConfig((prev) => ({
                                ...prev,
                                auto_approve_roles: value,
                              }))
                            }
                            style={{ width: "100%" }}
                          >
                            <Option value="admin">管理员</Option>
                            <Option value="manager">经理</Option>
                            <Option value="supervisor">主管</Option>
                          </Select>
                        </div>
                      </div>
                    </div>
                  </Card>
                </Col>

                <Col span={12}>
                  <Card title="SLA配置" className="h-full">
                    <div className="space-y-4">
                      <div>
                        <Text strong>响应时间</Text>
                        <div className="mt-2">
                          <Input
                            type="number"
                            suffix="小时"
                            value={workflow?.sla_config?.response_time_hours}
                            onChange={(e) =>
                              setWorkflow((prev) =>
                                prev
                                  ? {
                                      ...prev,
                                      sla_config: {
                                        ...prev.sla_config!,
                                        response_time_hours:
                                          parseInt(e.target.value) || 24,
                                      },
                                    }
                                  : null
                              )
                            }
                          />
                        </div>
                      </div>

                      <div>
                        <Text strong>解决时间</Text>
                        <div className="mt-2">
                          <Input
                            type="number"
                            suffix="小时"
                            value={workflow?.sla_config?.resolution_time_hours}
                            onChange={(e) =>
                              setWorkflow((prev) =>
                                prev
                                  ? {
                                      ...prev,
                                      sla_config: {
                                        ...prev.sla_config!,
                                        resolution_time_hours:
                                          parseInt(e.target.value) || 72,
                                      },
                                    }
                                  : null
                              )
                            }
                          />
                        </div>
                      </div>

                      <div>
                        <Text strong>工作时间设置</Text>
                        <div className="mt-2 space-y-2">
                          <div>
                            <input
                              type="checkbox"
                              checked={
                                workflow?.sla_config?.business_hours_only
                              }
                              onChange={(e) =>
                                setWorkflow((prev) =>
                                  prev
                                    ? {
                                        ...prev,
                                        sla_config: {
                                          ...prev.sla_config!,
                                          business_hours_only: e.target.checked,
                                        },
                                      }
                                    : null
                                )
                              }
                            />
                            <Text className="ml-2">仅工作时间</Text>
                          </div>
                          <div>
                            <input
                              type="checkbox"
                              checked={workflow?.sla_config?.exclude_weekends}
                              onChange={(e) =>
                                setWorkflow((prev) =>
                                  prev
                                    ? {
                                        ...prev,
                                        sla_config: {
                                          ...prev.sla_config!,
                                          exclude_weekends: e.target.checked,
                                        },
                                      }
                                    : null
                                )
                              }
                            />
                            <Text className="ml-2">排除周末</Text>
                          </div>
                        </div>
                      </div>
                    </div>
                  </Card>
                </Col>
              </Row>
            </TabPane>
          </Tabs>
        </Content>
      </Layout>

      {/* 版本管理模态框 */}
      <Modal
        title="创建新版本"
        open={showVersionModal}
        onOk={handleCreateVersion}
        onCancel={() => setShowVersionModal(false)}
        okText="创建"
        cancelText="取消"
      >
        <div className="space-y-4">
          <Alert
            message="版本管理"
            description="创建新版本将保存当前的设计状态，不会影响已部署的版本。"
            type="info"
            showIcon
          />
          <div>
            <Text strong>当前版本</Text>
            <div className="mt-1">
              <Tag color="blue">{workflow?.version}</Tag>
            </div>
          </div>
          <div>
            <Text strong>新版本号</Text>
            <div className="mt-1">
              <Tag color="green">
                {workflow
                  ? `${parseFloat(workflow.version) + 0.1}`.slice(0, 3)
                  : "1.1"}
              </Tag>
            </div>
          </div>
        </div>
      </Modal>

      {/* 流程设置模态框 */}
      <Modal
        title="流程设置"
        open={showSettingsModal}
        onOk={handleSaveSettings}
        onCancel={() => setShowSettingsModal(false)}
        width={800}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" initialValues={workflow || {}}>
          <Tabs>
            <TabPane tab="审批配置" key="approval">
              <Form.Item
                label="审批类型"
                name={["approval_config", "approval_type"]}
                rules={[{ required: true, message: "请选择审批类型" }]}
              >
                <Select>
                  <Option value="single">单人审批</Option>
                  <Option value="parallel">并行审批</Option>
                  <Option value="sequential">串行审批</Option>
                  <Option value="conditional">条件审批</Option>
                </Select>
              </Form.Item>

              <Form.Item label="审批人" name={["approval_config", "approvers"]}>
                <Select mode="multiple" placeholder="选择审批人">
                  <Option value="user1">张三</Option>
                  <Option value="user2">李四</Option>
                  <Option value="user3">王五</Option>
                </Select>
              </Form.Item>
            </TabPane>

            <TabPane tab="SLA配置" key="sla">
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    label="响应时间(小时)"
                    name={["sla_config", "response_time_hours"]}
                  >
                    <Input type="number" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    label="解决时间(小时)"
                    name={["sla_config", "resolution_time_hours"]}
                  >
                    <Input type="number" />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                label="工作时间设置"
                name={["sla_config", "business_hours_only"]}
                valuePropName="checked"
              >
                <input type="checkbox" />
                <Text className="ml-2">仅工作时间</Text>
              </Form.Item>
            </TabPane>
          </Tabs>
        </Form>
      </Modal>
    </Layout>
  );
};

export default WorkflowDesignerPage;

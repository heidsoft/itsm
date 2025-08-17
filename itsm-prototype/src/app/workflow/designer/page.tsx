"use client";

import React, { useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Layout,
  Button,
  Space,
  message,
  Typography,
  Divider,
  Row,
  Col,
  Tag,
  App,
} from "antd";
import { ArrowLeft, Save, PlayCircle, Download, Upload } from "lucide-react";
import EnhancedBPMNDesigner from "../../components/EnhancedBPMNDesigner";
import { WorkflowAPI } from "../../lib/workflow-api";

const { Header, Content, Sider } = Layout;
const { Title, Text } = Typography;

interface WorkflowDesignerPageProps {
  params: { id?: string };
}

interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  version: string;
  category: string;
  status: "draft" | "active" | "inactive";
  xml: string;
  created_at: string;
  updated_at: string;
  created_by: string;
  tags: string[];
}

const WorkflowDesignerPage: React.FC<WorkflowDesignerPageProps> = ({
  params,
}) => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { modal } = App.useApp();

  const [workflow, setWorkflow] = useState<WorkflowDefinition | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [currentXML, setCurrentXML] = useState("");
  const [hasChanges, setHasChanges] = useState(false);

  // 从URL参数获取工作流ID
  const workflowId = params.id || searchParams.get("id");

  useEffect(() => {
    if (workflowId) {
      loadWorkflow(workflowId);
    } else {
      // 创建新工作流
      setWorkflow({
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
      });
      setCurrentXML(getDefaultBPMNXML());
    }
  }, [workflowId]);

  const loadWorkflow = async (id: string) => {
    if (id === "new") return;

    setLoading(true);
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
      });
      // 这里需要从部署资源中获取BPMN XML
      setCurrentXML(getDefaultBPMNXML());
    } catch (error) {
      console.error("加载工作流失败:", error);
      message.error("加载工作流失败");
    } finally {
      setLoading(false);
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
                xml,
                updated_at: response.updated_at,
              }
            : null
        );

        // 更新URL，添加新创建的工作流ID
        router.replace(`/workflow/designer?id=${response.key}`);

        message.success("工作流创建成功");
      } else {
        // 更新现有工作流 - 使用新的BPMN API
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

      // 更新工作流状态
      setWorkflow((prev) =>
        prev ? { ...prev, xml, updated_at: new Date().toISOString() } : null
      );
    } catch (error) {
      console.error("保存失败:", error);
      message.error("保存失败: " + (error as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDeploy = async (xml: string) => {
    if (!workflow) return;

    setDeploying(true);
    try {
      // 先保存工作流
      await handleSave(xml);

      // 然后部署工作流 - 使用新的BPMN API
      await WorkflowAPI.setProcessDefinitionActive(workflow.id, true);

      message.success("工作流部署成功");

      // 更新工作流状态
      setWorkflow((prev) =>
        prev ? { ...prev, status: "active" as const } : null
      );
    } catch (error) {
      console.error("部署失败:", error);
      message.error("部署失败: " + (error as Error).message);
    } finally {
      setDeploying(false);
    }
  };

  const handleBack = () => {
    if (hasChanges) {
      modal.confirm({
        title: "确认离开",
        content: "当前有未保存的更改，确定要离开吗？",
        onOk: () => router.push("/workflow"),
      });
    } else {
      router.push("/workflow");
    }
  };

  const handleExport = () => {
    const blob = new Blob([currentXML], { type: "application/xml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${workflow?.name || "workflow"}.bpmn`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleImport = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".bpmn,.xml";
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (file) {
        const reader = new FileReader();
        reader.onload = (e) => {
          const xml = e.target?.result as string;
          setCurrentXML(xml);
          setHasChanges(true);
          message.success("工作流导入成功");
        };
        reader.readAsText(file);
      }
    };
    input.click();
  };

  if (loading) {
    return (
      <Layout style={{ minHeight: "100vh" }}>
        <Content style={{ padding: "50px", textAlign: "center" }}>
          <div>加载中...</div>
        </Content>
      </Layout>
    );
  }

  if (!workflow) {
    return (
      <Layout style={{ minHeight: "100vh" }}>
        <Content style={{ padding: "50px", textAlign: "center" }}>
          <div>工作流不存在</div>
        </Content>
      </Layout>
    );
  }

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Header
        style={{
          background: "#fff",
          padding: "0 24px",
          borderBottom: "1px solid #f0f0f0",
        }}
      >
        <Row align="middle" style={{ height: "100%" }}>
          <Col flex="auto">
            <Space>
              <Button icon={<ArrowLeft />} onClick={handleBack} type="text">
                返回
              </Button>
              <Divider type="vertical" />
              <Title level={4} style={{ margin: 0 }}>
                {workflow.name}
              </Title>
              <Tag color={workflow.status === "active" ? "green" : "orange"}>
                {workflow.status === "active" ? "已激活" : "草稿"}
              </Tag>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button icon={<Upload />} onClick={handleImport}>
                导入
              </Button>
              <Button icon={<Download />} onClick={handleExport}>
                导出
              </Button>
              <Button
                icon={<Save />}
                type="primary"
                loading={saving}
                onClick={() => handleSave(currentXML)}
              >
                保存
              </Button>
              <Button
                icon={<PlayCircle />}
                type="primary"
                loading={deploying}
                onClick={() => handleDeploy(currentXML)}
                disabled={workflow.status === "active"}
              >
                部署
              </Button>
            </Space>
          </Col>
        </Row>
      </Header>

      <Layout>
        <Sider
          width={300}
          style={{ background: "#fff", borderRight: "1px solid #f0f0f0" }}
        >
          <div style={{ padding: "16px" }}>
            <Title level={5}>工作流信息</Title>
            <div style={{ marginBottom: "16px" }}>
              <Text strong>版本:</Text> {workflow.version}
            </div>
            <div style={{ marginBottom: "16px" }}>
              <Text strong>分类:</Text> {workflow.category}
            </div>
            <div style={{ marginBottom: "16px" }}>
              <Text strong>创建者:</Text> {workflow.created_by}
            </div>
            <div style={{ marginBottom: "16px" }}>
              <Text strong>创建时间:</Text>{" "}
              {new Date(workflow.created_at).toLocaleString()}
            </div>
            <div style={{ marginBottom: "16px" }}>
              <Text strong>更新时间:</Text>{" "}
              {new Date(workflow.updated_at).toLocaleString()}
            </div>

            {workflow.tags.length > 0 && (
              <div style={{ marginBottom: "16px" }}>
                <Text strong>标签:</Text>
                <div style={{ marginTop: "8px" }}>
                  {workflow.tags.map((tag) => (
                    <Tag key={tag} style={{ marginBottom: "4px" }}>
                      {tag}
                    </Tag>
                  ))}
                </div>
              </div>
            )}

            {workflow.description && (
              <div style={{ marginBottom: "16px" }}>
                <Text strong>描述:</Text>
                <div style={{ marginTop: "8px", color: "#666" }}>
                  {workflow.description}
                </div>
              </div>
            )}
          </div>
        </Sider>

        <Content style={{ padding: "0" }}>
          <EnhancedBPMNDesigner
            initialXML={currentXML}
            onSave={handleSave}
            onDeploy={handleDeploy}
            height={800}
            showPropertiesPanel={true}
          />
        </Content>
      </Layout>
    </Layout>
  );
};

export default WorkflowDesignerPage;

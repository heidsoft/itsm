"use client";

import React, { useState, useEffect } from "react";
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
import { useRouter } from "next/navigation";
import TicketApprovalWorkflowDesigner from "../../components/TicketApprovalWorkflowDesigner";

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

// 获取节点类型名称
const getNodeTypeName = (type: string) => {
  switch (type) {
    case "start":
      return "开始";
    case "approval":
      return "审批";
    case "condition":
      return "条件";
    case "action":
      return "动作";
    case "end":
      return "结束";
    default:
      return "未知";
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

const TicketApprovalWorkflowPage = () => {
  const router = useRouter();
  const { modal } = App.useApp();
  const [workflow, setWorkflow] = useState<TicketApprovalWorkflow | null>(null);
  const [loading, setLoading] = useState(false);
  const [saveModalVisible, setSaveModalVisible] = useState(false);
  const [form] = Form.useForm();

  const handleSaveWorkflow = (workflowData: any) => {
    console.log("保存工作流数据:", workflowData);

    // 显示保存模态框
    setSaveModalVisible(true);
    setWorkflow(workflowData);

    // 设置表单初始值
    form.setFieldsValue({
      name: workflowData.name || "工单审批流程",
      description: workflowData.description || "标准工单审批流程",
      category: "审批流程",
      version: workflowData.metadata?.version || "1.0.0",
    });
  };

  const handleFinalSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // 模拟保存API调用
      await new Promise((resolve) => setTimeout(resolve, 2000));

      const finalWorkflow = {
        ...workflow,
        ...values,
        id: Date.now(),
        status: "draft",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        instances_count: 0,
        running_instances: 0,
        created_by: "当前用户",
      };

      console.log("最终保存的工作流:", finalWorkflow);

      message.success("工单审批流程保存成功！");
      setSaveModalVisible(false);

      // 保存成功后返回工作流列表
      setTimeout(() => {
        router.push("/workflow");
      }, 1000);
    } catch (error) {
      message.error("保存失败，请检查必填项");
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    modal.confirm({
      title: "确认离开",
      content: "离开此页面将丢失未保存的更改，确定要离开吗？",
      okText: "确定离开",
      cancelText: "继续编辑",
      okType: "danger",
      onOk: () => {
        router.push("/workflow");
      },
    });
  };

  const handlePreview = () => {
    if (!workflow) {
      message.warning("请先设计工作流程");
      return;
    }

    // 创建预览用的流程步骤
    const previewSteps =
      workflow.nodes?.map((node: any, index: number) => ({
        title: node.name,
        description: node.description,
        status: index === 0 ? "process" : "wait",
        icon:
          node.type === "start" ? (
            <Play className="w-4 h-4" />
          ) : node.type === "approval" ? (
            <Users className="w-4 h-4" />
          ) : node.type === "condition" ? (
            <GitBranch className="w-4 h-4" />
          ) : node.type === "action" ? (
            <Zap className="w-4 h-4" />
          ) : node.type === "end" ? (
            <CheckCircle className="w-4 h-4" />
          ) : null,
      })) || [];

    modal.info({
      title: (
        <div className="flex items-center gap-2">
          <Eye className="w-5 h-5" />
          工作流预览
        </div>
      ),
      width: 800,
      content: (
        <div className="space-y-6">
          <Alert
            message="工作流程概览"
            description={`该工作流包含 ${
              workflow.nodes?.length || 0
            } 个节点，其中 ${
              workflow.nodes?.filter((n: any) => n.type === "approval")
                .length || 0
            } 个审批节点`}
            type="info"
            showIcon
          />

          <div>
            <Title level={5}>流程步骤</Title>
            <Steps
              direction="vertical"
              size="small"
              current={0}
              items={previewSteps}
            />
          </div>

          <div>
            <Title level={5}>节点详情</Title>
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
                          {getNodeTypeName(node.type)}
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
                            {node.config.approvers.value?.length || 0} 审批人
                          </Tag>
                          <Tag color="orange" className="text-xs">
                            {node.config.timeout || 24}小时
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

  const getNodeTypeName = (type: string) => {
    const names: Record<string, string> = {
      start: "开始",
      approval: "审批",
      condition: "条件",
      action: "动作",
      end: "结束",
    };
    return names[type] || type;
  };

  const getNodeTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      start: "green",
      approval: "blue",
      condition: "orange",
      action: "purple",
      end: "red",
    };
    return colors[type] || "default";
  };

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
                返回工作流列表
              </Button>
              <Divider type="vertical" />
              <div>
                <Title level={3} className="!mb-1">
                  <GitBranch className="inline-block w-6 h-6 mr-2" />
                  工单审批流程设计器
                </Title>
                <Text type="secondary">
                  设计和配置工单的审批流程，支持多级审批和条件分支
                </Text>
              </div>
            </div>

            <Space>
              <Button
                icon={<Eye className="w-4 h-4" />}
                onClick={handlePreview}
                disabled={!workflow}
              >
                预览
              </Button>
              <Button
                icon={<Download className="w-4 h-4" />}
                disabled={!workflow}
              >
                导出
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                disabled={!workflow}
                onClick={() => setSaveModalVisible(true)}
              >
                保存流程
              </Button>
            </Space>
          </div>
        </div>
      </div>

      {/* 设计器区域 */}
      <div className="flex-1">
        <TicketApprovalWorkflowDesigner
          workflow={workflow}
          onSave={handleSaveWorkflow}
          onCancel={handleCancel}
        />
      </div>

      {/* 保存确认模态框 */}
      <Modal
        title="保存工单审批流程"
        open={saveModalVisible}
        onOk={handleFinalSave}
        onCancel={() => setSaveModalVisible(false)}
        width={600}
        confirmLoading={loading}
        okText="确认保存"
        cancelText="取消"
      >
        <div className="space-y-6">
          <Alert
            message="保存确认"
            description="请确认工作流的基本信息，保存后将可以在工作流管理中查看和部署"
            type="info"
            showIcon
          />

          {workflow && (
            <Card size="small" title="流程统计">
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic
                    title="总节点"
                    value={workflow.metadata?.nodeCount || 0}
                    prefix={<GitBranch className="w-4 h-4" />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="审批节点"
                    value={workflow.metadata?.approvalCount || 0}
                    prefix={<Users className="w-4 h-4" />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="条件节点"
                    value={
                      workflow.nodes?.filter((n: any) => n.type === "condition")
                        .length || 0
                    }
                    prefix={<Settings className="w-4 h-4" />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="动作节点"
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
                  label="流程名称"
                  name="name"
                  rules={[{ required: true, message: "请输入流程名称" }]}
                >
                  <Input placeholder="输入流程名称" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label="流程分类"
                  name="category"
                  rules={[{ required: true, message: "请选择流程分类" }]}
                >
                  <Select placeholder="选择流程分类">
                    <Option value="审批流程">审批流程</Option>
                    <Option value="工单流程">工单流程</Option>
                    <Option value="事件流程">事件流程</Option>
                    <Option value="变更流程">变更流程</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label="流程描述"
              name="description"
              rules={[{ required: true, message: "请输入流程描述" }]}
            >
              <Input.TextArea rows={3} placeholder="描述此工作流的用途和特点" />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  label="版本号"
                  name="version"
                  rules={[{ required: true, message: "请输入版本号" }]}
                >
                  <Input placeholder="如: 1.0.0" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="初始状态" name="status" initialValue="draft">
                  <Select disabled>
                    <Option value="draft">草稿</Option>
                    <Option value="active">启用</Option>
                    <Option value="inactive">停用</Option>
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

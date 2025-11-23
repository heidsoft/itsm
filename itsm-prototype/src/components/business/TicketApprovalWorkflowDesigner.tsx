"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Tabs,
  Typography,
  Alert,
  Row,
  Col,
  Tag,
  Divider,
  Tooltip,
  Badge,
  Switch,
  InputNumber,
  Radio,
  App,
} from "antd";
import {
  Save,
  Play,
  Eye,
  Trash2,
  Edit,
  Users,
  CheckCircle,
  AlertCircle,
  GitBranch,
  Zap,
  FileText,
  Clock,
  Settings,
} from "lucide-react";

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;

interface ApprovalNode {
  id: string;
  type: "start" | "approval" | "condition" | "action" | "end";
  name: string;
  description?: string;
  position: { x: number; y: number };
  config: {
    // 审批节点配置
    approvers?: {
      type: "user" | "role" | "group";
      value: string[];
      mode: "all" | "any" | "sequential";
    };
    // 条件节点配置
    conditions?: {
      field: string;
      operator: string;
      value: string;
    }[];
    // 动作节点配置
    actions?: {
      type: "notification" | "assignment" | "escalation" | "custom";
      config: Record<string, unknown>;
    }[];
    // 通用配置
    timeout?: number;
    escalation?: {
      enabled: boolean;
      timeout: number;
      target: string[];
    };
  };
  connections: string[]; // 连接的下一个节点ID
}

interface TicketApprovalWorkflowDesignerProps {
  workflow?: unknown;
  onSave: (data: unknown) => void;
  onCancel: () => void;
}

// 预定义的工单审批流程模板
const TICKET_APPROVAL_TEMPLATES = {
  simple: {
    name: "简单审批流程",
    description: "单级审批，适用于一般工单",
    nodes: [
      {
        id: "start",
        type: "start",
        name: "开始",
        description: "工单提交",
        position: { x: 100, y: 200 },
        config: {},
        connections: ["approval1"],
      },
      {
        id: "approval1",
        type: "approval",
        name: "直属主管审批",
        description: "工单的直属主管进行审批",
        position: { x: 300, y: 200 },
        config: {
          approvers: {
            type: "role",
            value: ["direct_manager"],
            mode: "any",
          },
          timeout: 24,
          escalation: {
            enabled: true,
            timeout: 48,
            target: ["department_manager"],
          },
        },
        connections: ["end"],
      },
      {
        id: "end",
        type: "end",
        name: "结束",
        description: "审批完成",
        position: { x: 500, y: 200 },
        config: {},
        connections: [],
      },
    ],
  },
  complex: {
    name: "复杂审批流程",
    description: "多级审批，包含条件判断和并行审批",
    nodes: [
      {
        id: "start",
        type: "start",
        name: "开始",
        description: "工单提交",
        position: { x: 100, y: 200 },
        config: {},
        connections: ["condition1"],
      },
      {
        id: "condition1",
        type: "condition",
        name: "金额判断",
        description: "根据工单金额决定审批流程",
        position: { x: 300, y: 200 },
        config: {
          conditions: [
            {
              field: "amount",
              operator: ">=",
              value: "10000",
            },
          ],
        },
        connections: ["approval1", "approval2"],
      },
      {
        id: "approval1",
        type: "approval",
        name: "直属主管审批",
        description: "金额小于10000，直属主管审批",
        position: { x: 500, y: 150 },
        config: {
          approvers: {
            type: "role",
            value: ["direct_manager"],
            mode: "any",
          },
          timeout: 24,
        },
        connections: ["end"],
      },
      {
        id: "approval2",
        type: "approval",
        name: "部门经理审批",
        description: "金额大于等于10000，需要部门经理审批",
        position: { x: 500, y: 250 },
        config: {
          approvers: {
            type: "role",
            value: ["department_manager"],
            mode: "any",
          },
          timeout: 48,
          escalation: {
            enabled: true,
            timeout: 72,
            target: ["finance_manager"],
          },
        },
        connections: ["approval3"],
      },
      {
        id: "approval3",
        type: "approval",
        name: "财务审批",
        description: "高金额工单需要财务审批",
        position: { x: 700, y: 250 },
        config: {
          approvers: {
            type: "role",
            value: ["finance_manager"],
            mode: "any",
          },
          timeout: 24,
        },
        connections: ["end"],
      },
      {
        id: "end",
        type: "end",
        name: "结束",
        description: "审批完成",
        position: { x: 900, y: 200 },
        config: {},
        connections: [],
      },
    ],
  },
};

const TicketApprovalWorkflowDesigner: React.FC<
  TicketApprovalWorkflowDesignerProps
> = ({ workflow, onSave, onCancel }) => {
  const { message } = App.useApp();
  const [nodes, setNodes] = useState<ApprovalNode[]>([]);
  const [selectedNode, setSelectedNode] = useState<ApprovalNode | null>(null);
  const [activeTab, setActiveTab] = useState("design");
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [nodeConfigModalVisible, setNodeConfigModalVisible] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    if (workflow && typeof workflow === "object" && "nodes" in workflow) {
      setNodes(workflow.nodes as ApprovalNode[]);
    } else {
      // 如果没有现有工作流，显示模板选择
      setTemplateModalVisible(true);
    }
  }, [workflow]);

  const loadTemplate = (templateKey: string) => {
    const template =
      TICKET_APPROVAL_TEMPLATES[
        templateKey as keyof typeof TICKET_APPROVAL_TEMPLATES
      ];
    if (template) {
      setNodes(template.nodes as ApprovalNode[]);
      setTemplateModalVisible(false);
      message.success(`已加载模板：${template.name}`);
    }
  };

  const addNode = (
    type: ApprovalNode["type"],
    position: { x: number; y: number }
  ) => {
    const newNode: ApprovalNode = {
      id: `${type}_${Date.now()}`,
      type,
      name: getNodeTypeName(type),
      description: "",
      position,
      config: getDefaultNodeConfig(type),
      connections: [],
    };

    setNodes((prev) => [...prev, newNode]);
    setSelectedNode(newNode);
    setNodeConfigModalVisible(true);
  };

  const getNodeTypeName = (type: ApprovalNode["type"]) => {
    const names = {
      start: "开始",
      approval: "审批节点",
      condition: "条件判断",
      action: "执行动作",
      end: "结束",
    };
    return names[type];
  };

  const getDefaultNodeConfig = (type: ApprovalNode["type"]) => {
    switch (type) {
      case "approval":
        return {
          approvers: {
            type: "role" as const,
            value: [],
            mode: "any" as const,
          },
          timeout: 24,
          escalation: {
            enabled: false,
            timeout: 48,
            target: [],
          },
        };
      case "condition":
        return {
          conditions: [
            {
              field: "",
              operator: "==",
              value: "",
            },
          ],
        };
      case "action":
        return {
          actions: [],
        };
      default:
        return {};
    }
  };

  const updateNode = (nodeId: string, updates: Partial<ApprovalNode>) => {
    setNodes((prev) =>
      prev.map((node) => (node.id === nodeId ? { ...node, ...updates } : node))
    );
  };

  const deleteNode = (nodeId: string) => {
    setNodes((prev) => {
      // 删除节点并清理所有指向该节点的连接
      const filteredNodes = prev.filter((node) => node.id !== nodeId);
      return filteredNodes.map((node) => ({
        ...node,
        connections: node.connections.filter((connId) => connId !== nodeId),
      }));
    });
    if (selectedNode?.id === nodeId) {
      setSelectedNode(null);
    }
  };

  const getNodeIcon = (type: ApprovalNode["type"]) => {
    const icons = {
      start: <Play className="w-4 h-4" />,
      approval: <Users className="w-4 h-4" />,
      condition: <GitBranch className="w-4 h-4" />,
      action: <Zap className="w-4 h-4" />,
      end: <CheckCircle className="w-4 h-4" />,
    };
    return icons[type];
  };

  const getNodeColor = (type: ApprovalNode["type"]) => {
    const colors = {
      start: "#52c41a",
      approval: "#1890ff",
      condition: "#faad14",
      action: "#722ed1",
      end: "#f5222d",
    };
    return colors[type];
  };

  const handleSave = () => {
    // 验证工作流
    if (nodes.length === 0) {
      message.error("工作流不能为空");
      return;
    }

    const startNodes = nodes.filter((node) => node.type === "start");
    const endNodes = nodes.filter((node) => node.type === "end");

    if (startNodes.length !== 1) {
      message.error("工作流必须有且仅有一个开始节点");
      return;
    }

    if (endNodes.length === 0) {
      message.error("工作流必须至少有一个结束节点");
      return;
    }

    const workflowData = {
      type: "ticket_approval",
      name: "工单审批流程",
      description: "工单审批流程配置",
      nodes,
      metadata: {
        version: "1.0.0",
        lastModified: new Date().toISOString(),
        nodeCount: nodes.length,
        approvalCount: nodes.filter((n) => n.type === "approval").length,
      },
    };

    onSave(workflowData);
    message.success("工作流保存成功");
  };

  const handleNodeConfigSave = () => {
    const values = form.getFieldsValue();
    if (selectedNode) {
      updateNode(selectedNode.id, {
        name: values.name,
        description: values.description,
        config: values.config,
      });
      setNodeConfigModalVisible(false);
      setSelectedNode(null);
      form.resetFields();
      message.success("节点配置已保存");
    }
  };

  const renderNode = (node: ApprovalNode) => {
    const isSelected = selectedNode?.id === node.id;
    const nodeStyle = {
      start: "bg-gradient-to-r from-green-400 to-green-600",
      approval: "bg-gradient-to-r from-blue-400 to-blue-600",
      condition: "bg-gradient-to-r from-yellow-400 to-yellow-600",
      action: "bg-gradient-to-r from-purple-400 to-purple-600",
      end: "bg-gradient-to-r from-red-400 to-red-600",
    };

    return (
      <div
        key={node.id}
        className={`absolute border-2 rounded-xl p-4 cursor-pointer transition-all duration-300 hover:scale-105 shadow-md ${
          isSelected
            ? "border-blue-500 shadow-xl ring-2 ring-blue-200"
            : "border-gray-200 hover:border-gray-300 hover:shadow-lg"
        }`}
        style={{
          left: node.position.x,
          top: node.position.y,
          backgroundColor: "white",
          minWidth: 140,
          maxWidth: 200,
        }}
        onClick={() => setSelectedNode(node)}
        onDoubleClick={() => {
          setSelectedNode(node);
          form.setFieldsValue({
            name: node.name,
            description: node.description,
            config: node.config,
          });
          setNodeConfigModalVisible(true);
        }}
      >
        {/* 节点头部 */}
        <div
          className={`flex items-center gap-2 mb-2 p-2 rounded-lg text-white ${
            nodeStyle[node.type as keyof typeof nodeStyle]
          }`}
        >
          <div className="w-5 h-5 flex items-center justify-center">
            {getNodeIcon(node.type)}
          </div>
          <Text strong className="text-sm text-white">
            {node.name}
          </Text>
        </div>

        {/* 节点描述 */}
        {node.description && (
          <div className="mb-2">
            <Text type="secondary" className="text-xs">
              {node.description}
            </Text>
          </div>
        )}

        {/* 审批节点特殊显示 */}
        {node.type === "approval" && node.config.approvers && (
          <div className="flex flex-wrap gap-1">
            <Tag color="blue" className="text-xs">
              <Users className="w-3 h-3 inline mr-1" />
              {node.config.approvers.value?.length || 0} 审批人
            </Tag>
            <Tag color="orange" className="text-xs">
              <Clock className="w-3 h-3 inline mr-1" />
              {node.config.timeout || 24}h
            </Tag>
          </div>
        )}

        {/* 条件节点特殊显示 */}
        {node.type === "condition" && node.config.conditions && (
          <div>
            <Tag color="gold" className="text-xs">
              {node.config.conditions.length || 0} 条件
            </Tag>
          </div>
        )}

        {/* 动作节点特殊显示 */}
        {node.type === "action" && node.config.actions && (
          <div>
            <Tag color="purple" className="text-xs">
              {node.config.actions.length || 0} 动作
            </Tag>
          </div>
        )}
      </div>
    );
  };

  const renderConnections = () => {
    return nodes.map((node) =>
      node.connections.map((targetId) => {
        const targetNode = nodes.find((n) => n.id === targetId);
        if (!targetNode) return null;

        const startX = node.position.x + 70; // 节点中心点
        const startY = node.position.y + 50;
        const endX = targetNode.position.x + 70;
        const endY = targetNode.position.y + 50;

        // 计算中间控制点，创建贝塞尔曲线
        const midX = (startX + endX) / 2;
        const controlY = startY < endY ? startY + 50 : startY - 50;

        const pathData = `M ${startX} ${startY} Q ${midX} ${controlY} ${endX} ${endY}`;

        return (
          <svg
            key={`${node.id}-${targetId}`}
            className="absolute pointer-events-none"
            style={{
              left: 0,
              top: 0,
              width: "100%",
              height: "100%",
            }}
          >
            <defs>
              <marker
                id="arrowhead-blue"
                markerWidth="12"
                markerHeight="8"
                refX="10"
                refY="4"
                orient="auto"
                markerUnits="strokeWidth"
              >
                <path
                  d="M0,0 L0,8 L12,4 z"
                  fill="#3b82f6"
                  stroke="#3b82f6"
                  strokeWidth="1"
                />
              </marker>
              <filter id="glow">
                <feGaussianBlur stdDeviation="2" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            </defs>
            <path
              d={pathData}
              stroke="#3b82f6"
              strokeWidth="3"
              fill="none"
              markerEnd="url(#arrowhead-blue)"
              filter="url(#glow)"
              className="drop-shadow-sm"
            />
          </svg>
        );
      })
    );
  };

  return (
    <div className="h-full flex flex-col">
      {/* 工具栏 */}
      <div className="border-b border-gray-200 p-4 bg-white">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Space>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                onClick={handleSave}
              >
                保存工作流
              </Button>
              <Button
                icon={<FileText className="w-4 h-4" />}
                onClick={() => setTemplateModalVisible(true)}
              >
                加载模板
              </Button>
              <Button icon={<Eye className="w-4 h-4" />}>预览</Button>
            </Space>

            <Divider type="vertical" />

            <Space>
              <Tooltip title="添加审批节点">
                <Button
                  icon={<Users className="w-4 h-4" />}
                  onClick={() => addNode("approval", { x: 200, y: 200 })}
                >
                  审批
                </Button>
              </Tooltip>
              <Tooltip title="添加条件判断">
                <Button
                  icon={<GitBranch className="w-4 h-4" />}
                  onClick={() => addNode("condition", { x: 200, y: 300 })}
                >
                  条件
                </Button>
              </Tooltip>
              <Tooltip title="添加执行动作">
                <Button
                  icon={<Zap className="w-4 h-4" />}
                  onClick={() => addNode("action", { x: 200, y: 400 })}
                >
                  动作
                </Button>
              </Tooltip>
            </Space>
          </div>

          <Space>
            <Button onClick={onCancel}>取消</Button>
          </Space>
        </div>
      </div>

      {/* 主要内容区域 */}
      <div className="flex-1 flex">
        {/* 设计画布 */}
        <div className="flex-1 relative bg-gray-50 overflow-auto">
          <div
            className="relative w-full h-full min-w-[1200px] min-h-[800px]"
            onClick={(e) => {
              if (e.target === e.currentTarget) {
                setSelectedNode(null);
              }
            }}
          >
            {renderConnections()}
            {nodes.map(renderNode)}

            {/* 网格背景 */}
            <div
              className="absolute inset-0 pointer-events-none opacity-20"
              style={{
                backgroundImage: `
                  radial-gradient(circle, #999 1px, transparent 1px)
                `,
                backgroundSize: "20px 20px",
              }}
            />
          </div>
        </div>

        {/* 右侧属性面板 */}
        <div className="w-80 border-l border-gray-200 bg-white overflow-hidden">
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            className="h-full"
            items={[
              {
                key: "properties",
                label: (
                  <span className="flex items-center gap-2">
                    <Settings className="w-4 h-4" />
                    节点属性
                  </span>
                ),
                children: (
                  <div className="p-4 h-full overflow-y-auto">
                    {selectedNode ? (
                      <div className="space-y-4">
                        {/* 节点基本信息 */}
                        <Card size="small" className="border-0 bg-gray-50">
                          <div className="flex items-center gap-3 mb-3">
                            <div
                              className="w-10 h-10 rounded-full flex items-center justify-center text-white"
                              style={{
                                background: `linear-gradient(135deg, ${getNodeColor(
                                  selectedNode.type
                                )}dd, ${getNodeColor(selectedNode.type)})`,
                              }}
                            >
                              {getNodeIcon(selectedNode.type)}
                            </div>
                            <div>
                              <Title level={5} className="!mb-0">
                                {selectedNode.name}
                              </Title>
                              <Text type="secondary" className="text-xs">
                                {selectedNode.type === "start" && "流程起点"}
                                {selectedNode.type === "approval" && "审批节点"}
                                {selectedNode.type === "condition" &&
                                  "条件分支"}
                                {selectedNode.type === "action" && "动作执行"}
                                {selectedNode.type === "end" && "流程终点"}
                              </Text>
                            </div>
                          </div>
                          {selectedNode.description && (
                            <Text type="secondary" className="text-sm">
                              {selectedNode.description}
                            </Text>
                          )}
                        </Card>

                        {/* 操作按钮 */}
                        <div className="grid grid-cols-2 gap-2">
                          <Button
                            type="primary"
                            icon={<Edit className="w-4 h-4" />}
                            onClick={() => {
                              form.setFieldsValue({
                                name: selectedNode.name,
                                description: selectedNode.description,
                                config: selectedNode.config,
                              });
                              setNodeConfigModalVisible(true);
                            }}
                            className="h-10"
                          >
                            编辑配置
                          </Button>
                          <Button
                            danger
                            icon={<Trash2 className="w-4 h-4" />}
                            onClick={() => deleteNode(selectedNode.id)}
                            className="h-10"
                          >
                            删除节点
                          </Button>
                        </div>

                        {/* 审批节点详细配置 */}
                        {selectedNode.type === "approval" && (
                          <Card
                            size="small"
                            title="审批配置"
                            className="border border-blue-200"
                          >
                            <div className="space-y-3">
                              <Row justify="space-between" align="middle">
                                <Col>
                                  <Text className="flex items-center gap-1">
                                    <Users className="w-4 h-4" />
                                    审批人数量
                                  </Text>
                                </Col>
                                <Col>
                                  <Badge
                                    count={
                                      selectedNode.config.approvers?.value
                                        ?.length || 0
                                    }
                                    style={{ backgroundColor: "#1890ff" }}
                                  />
                                </Col>
                              </Row>
                              <Row justify="space-between" align="middle">
                                <Col>
                                  <Text className="flex items-center gap-1">
                                    <CheckCircle className="w-4 h-4" />
                                    审批模式
                                  </Text>
                                </Col>
                                <Col>
                                  <Tag color="blue">
                                    {selectedNode.config.approvers?.mode ===
                                    "all"
                                      ? "全部审批"
                                      : selectedNode.config.approvers?.mode ===
                                        "any"
                                      ? "任一审批"
                                      : selectedNode.config.approvers?.mode ===
                                        "sequential"
                                      ? "顺序审批"
                                      : "任一审批"}
                                  </Tag>
                                </Col>
                              </Row>
                              <Row justify="space-between" align="middle">
                                <Col>
                                  <Text className="flex items-center gap-1">
                                    <Clock className="w-4 h-4" />
                                    超时时间
                                  </Text>
                                </Col>
                                <Col>
                                  <Tag color="orange">
                                    {selectedNode.config.timeout || 24}小时
                                  </Tag>
                                </Col>
                              </Row>
                              {selectedNode.config.escalation?.enabled && (
                                <Row justify="space-between" align="middle">
                                  <Col>
                                    <Text className="flex items-center gap-1">
                                      <AlertCircle className="w-4 h-4" />
                                      升级策略
                                    </Text>
                                  </Col>
                                  <Col>
                                    <Tag color="red">已启用</Tag>
                                  </Col>
                                </Row>
                              )}
                            </div>
                          </Card>
                        )}

                        {/* 条件节点详细配置 */}
                        {selectedNode.type === "condition" &&
                          selectedNode.config.conditions && (
                            <Card
                              size="small"
                              title="条件配置"
                              className="border border-yellow-200"
                            >
                              <div className="space-y-2">
                                <Row justify="space-between" align="middle">
                                  <Col>
                                    <Text>条件数量</Text>
                                  </Col>
                                  <Col>
                                    <Badge
                                      count={
                                        selectedNode.config.conditions.length
                                      }
                                      style={{ backgroundColor: "#faad14" }}
                                    />
                                  </Col>
                                </Row>
                              </div>
                            </Card>
                          )}

                        {/* 动作节点详细配置 */}
                        {selectedNode.type === "action" &&
                          selectedNode.config.actions && (
                            <Card
                              size="small"
                              title="动作配置"
                              className="border border-purple-200"
                            >
                              <div className="space-y-2">
                                <Row justify="space-between" align="middle">
                                  <Col>
                                    <Text>动作数量</Text>
                                  </Col>
                                  <Col>
                                    <Badge
                                      count={selectedNode.config.actions.length}
                                      style={{ backgroundColor: "#722ed1" }}
                                    />
                                  </Col>
                                </Row>
                              </div>
                            </Card>
                          )}
                      </div>
                    ) : (
                      <div className="text-center text-gray-500 py-8">
                        <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
                        <p>选择一个节点来查看其属性</p>
                      </div>
                    )}
                  </div>
                ),
              },
              {
                key: "overview",
                label: "概览",
                children: (
                  <div className="p-4">
                    <Title level={5}>工作流概览</Title>
                    <div className="space-y-4">
                      <div className="flex justify-between">
                        <Text>节点总数:</Text>
                        <Badge count={nodes.length} />
                      </div>
                      <div className="flex justify-between">
                        <Text>审批节点:</Text>
                        <Badge
                          count={
                            nodes.filter((n) => n.type === "approval").length
                          }
                        />
                      </div>
                      <div className="flex justify-between">
                        <Text>条件节点:</Text>
                        <Badge
                          count={
                            nodes.filter((n) => n.type === "condition").length
                          }
                        />
                      </div>
                      <div className="flex justify-between">
                        <Text>动作节点:</Text>
                        <Badge
                          count={
                            nodes.filter((n) => n.type === "action").length
                          }
                        />
                      </div>
                    </div>

                    <Divider />

                    <Title level={5}>验证状态</Title>
                    <div className="space-y-2">
                      <div className="flex items-center gap-2">
                        {nodes.filter((n) => n.type === "start").length ===
                        1 ? (
                          <CheckCircle className="w-4 h-4 text-green-500" />
                        ) : (
                          <AlertCircle className="w-4 h-4 text-red-500" />
                        )}
                        <Text>开始节点唯一</Text>
                      </div>
                      <div className="flex items-center gap-2">
                        {nodes.filter((n) => n.type === "end").length > 0 ? (
                          <CheckCircle className="w-4 h-4 text-green-500" />
                        ) : (
                          <AlertCircle className="w-4 h-4 text-red-500" />
                        )}
                        <Text>存在结束节点</Text>
                      </div>
                    </div>
                  </div>
                ),
              },
            ]}
          />
        </div>
      </div>

      {/* 模板选择模态框 */}
      <Modal
        title="选择工作流模板"
        open={templateModalVisible}
        onCancel={() => setTemplateModalVisible(false)}
        width={800}
        footer={null}
      >
        <div className="space-y-4">
          <Alert
            message="工单审批流程模板"
            description="选择一个预定义的模板来快速开始设计您的工单审批流程"
            type="info"
            showIcon
          />

          <Row gutter={[16, 16]}>
            {Object.entries(TICKET_APPROVAL_TEMPLATES).map(
              ([key, template]) => (
                <Col span={12} key={key}>
                  <Card
                    hoverable
                    onClick={() => loadTemplate(key)}
                    className="h-full"
                  >
                    <Card.Meta
                      title={template.name}
                      description={template.description}
                    />
                    <div className="mt-3">
                      <Text type="secondary">
                        节点数量: {template.nodes.length}
                      </Text>
                    </div>
                  </Card>
                </Col>
              )
            )}
          </Row>

          <div className="text-center pt-4">
            <Button
              onClick={() => {
                setNodes([]);
                setTemplateModalVisible(false);
              }}
            >
              从空白开始
            </Button>
          </div>
        </div>
      </Modal>

      {/* 节点配置模态框 */}
      <Modal
        title={`配置节点: ${selectedNode?.name || ""}`}
        open={nodeConfigModalVisible}
        onOk={handleNodeConfigSave}
        onCancel={() => {
          setNodeConfigModalVisible(false);
          form.resetFields();
        }}
        width={600}
        okText="保存配置"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="节点名称"
            name="name"
            rules={[{ required: true, message: "请输入节点名称" }]}
          >
            <Input placeholder="输入节点名称" />
          </Form.Item>

          <Form.Item label="节点描述" name="description">
            <TextArea rows={2} placeholder="输入节点描述" />
          </Form.Item>

          {selectedNode?.type === "approval" && (
            <>
              <Form.Item
                label="审批人类型"
                name={["config", "approvers", "type"]}
              >
                <Radio.Group>
                  <Radio value="user">指定用户</Radio>
                  <Radio value="role">角色</Radio>
                  <Radio value="group">用户组</Radio>
                </Radio.Group>
              </Form.Item>

              <Form.Item
                label="审批模式"
                name={["config", "approvers", "mode"]}
              >
                <Select>
                  <Option value="any">任一审批</Option>
                  <Option value="all">全部审批</Option>
                  <Option value="sequential">顺序审批</Option>
                </Select>
              </Form.Item>

              <Form.Item label="超时时间(小时)" name={["config", "timeout"]}>
                <InputNumber min={1} max={168} />
              </Form.Item>

              <Form.Item
                label="启用升级"
                name={["config", "escalation", "enabled"]}
              >
                <Switch />
              </Form.Item>
            </>
          )}

          {selectedNode?.type === "condition" && (
            <Form.Item label="条件设置">
              <Alert
                message="条件配置"
                description="条件节点用于根据工单字段值进行流程分支判断"
                type="info"
                className="mb-3"
              />
              {/* 这里可以添加更复杂的条件配置界面 */}
            </Form.Item>
          )}

          {selectedNode?.type === "action" && (
            <Form.Item label="动作设置">
              <Alert
                message="动作配置"
                description="动作节点用于执行自动化操作，如发送通知、更新字段等"
                type="info"
                className="mb-3"
              />
              {/* 这里可以添加动作配置界面 */}
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
};

export default TicketApprovalWorkflowDesigner;

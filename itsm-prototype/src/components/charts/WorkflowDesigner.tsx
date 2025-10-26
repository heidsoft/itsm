"use client";

import React, { useState, useEffect, useRef } from "react";
import {
  Card,
  Button,
  Space,
  Tooltip,
  Modal,
  Form,
  Input,
  Select,
  Tabs,
  Divider,
  Row,
  Col,
  Typography,
  Alert,
  Badge,
  App,
} from "antd";
import {
  Save,
  Play,
  Settings,
  Eye,
  Code,
  Download,
  Upload,
  Undo,
  Redo,
  ZoomIn,
  ZoomOut,
  Fullscreen,
  Plus,
  Trash2,
  Copy,
  Move,
  Link,
  Unlink,
  CheckCircle,
  AlertCircle,
  Clock,
  User,
  Users,
  Shield,
  FileText,
  Send,
  GitBranch,
  GitCommit,
  GitMerge,
} from "lucide-react";

const { Title, Text } = Typography;
const { Option } = Select;

interface WorkflowDesignerProps {
  workflow?: any;
  onSave: (data: any) => void;
  onCancel: () => void;
}

interface BPMNElement {
  id: string;
  type: string;
  name: string;
  x: number;
  y: number;
  width?: number;
  height?: number;
  properties?: Record<string, any>;
}

interface BPMNConnection {
  id: string;
  source: string;
  target: string;
  type: string;
  properties?: Record<string, any>;
}

export const WorkflowDesigner: React.FC<WorkflowDesignerProps> = ({
  workflow,
  onSave,
  onCancel,
}) => {
  const { message } = App.useApp();
  const [elements, setElements] = useState<BPMNElement[]>([]);
  const [connections, setConnections] = useState<BPMNConnection[]>([]);
  const [selectedElement, setSelectedElement] = useState<BPMNElement | null>(
    null
  );
  const [zoom, setZoom] = useState(1);
  const [history, setHistory] = useState<any[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [propertiesVisible, setPropertiesVisible] = useState(false);
  const [activeTab, setActiveTab] = useState("design");

  const canvasRef = useRef<HTMLDivElement>(null);

  // 初始化默认工作流
  useEffect(() => {
    if (!workflow) {
      // 创建默认的简单工作流
      const defaultElements: BPMNElement[] = [
        {
          id: "start_1",
          type: "startEvent",
          name: "开始",
          x: 100,
          y: 200,
          width: 60,
          height: 60,
          properties: {
            eventType: "start",
            name: "开始",
          },
        },
        {
          id: "user_task_1",
          type: "userTask",
          name: "用户任务",
          x: 300,
          y: 200,
          width: 120,
          height: 80,
          properties: {
            assignee: "",
            candidateUsers: "",
            candidateGroups: "",
            formKey: "",
            name: "用户任务",
          },
        },
        {
          id: "end_1",
          type: "endEvent",
          name: "结束",
          x: 500,
          y: 200,
          width: 60,
          height: 60,
          properties: {
            eventType: "end",
            name: "结束",
          },
        },
      ];

      const defaultConnections: BPMNConnection[] = [
        {
          id: "flow_1",
          source: "start_1",
          target: "user_task_1",
          type: "sequenceFlow",
          properties: {
            name: "流程",
          },
        },
        {
          id: "flow_2",
          source: "user_task_1",
          target: "end_1",
          type: "sequenceFlow",
          properties: {
            name: "流程",
          },
        },
      ];

      setElements(defaultElements);
      setConnections(defaultConnections);
      saveToHistory(defaultElements, defaultConnections);
    } else {
      // 加载现有工作流
      loadWorkflow(workflow);
    }
  }, [workflow]);

  const loadWorkflow = (workflowData: any) => {
    try {
      // 这里应该解析BPMN XML
      // 暂时使用模拟数据
      const parsedElements: BPMNElement[] = [
        {
          id: "start_1",
          type: "startEvent",
          name: "开始",
          x: 100,
          y: 200,
          width: 60,
          height: 60,
          properties: {
            eventType: "start",
            name: "开始",
          },
        },
      ];

      const parsedConnections: BPMNConnection[] = [];

      setElements(parsedElements);
      setConnections(parsedConnections);
      saveToHistory(parsedElements, parsedConnections);
    } catch (error) {
      message.error("加载工作流失败");
    }
  };

  const saveToHistory = (
    newElements: BPMNElement[],
    newConnections: BPMNConnection[]
  ) => {
    const newHistory = {
      elements: [...newElements],
      connections: [...newConnections],
      timestamp: Date.now(),
    };

    setHistory((prev) => {
      const updated = [...prev.slice(0, historyIndex + 1), newHistory];
      if (updated.length > 50) {
        return updated.slice(-50);
      }
      return updated;
    });
    setHistoryIndex((prev) => prev + 1);
  };

  const addElement = (type: string, x: number, y: number) => {
    const newElement: BPMNElement = {
      id: `${type}_${Date.now()}`,
      type,
      name: getDefaultName(type),
      x,
      y,
      width: getDefaultWidth(type),
      height: getDefaultHeight(type),
      properties: getDefaultProperties(type),
    };

    const newElements = [...elements, newElement];
    setElements(newElements);
    saveToHistory(newElements, connections);
  };

  const getDefaultName = (type: string) => {
    const names: Record<string, string> = {
      startEvent: "开始",
      userTask: "用户任务",
      serviceTask: "服务任务",
      scriptTask: "脚本任务",
      endEvent: "结束",
      exclusiveGateway: "排他网关",
      parallelGateway: "并行网关",
      inclusiveGateway: "包容网关",
    };
    return names[type] || "新元素";
  };

  const getDefaultWidth = (type: string) => {
    const widths: Record<string, number> = {
      startEvent: 60,
      endEvent: 60,
      userTask: 120,
      serviceTask: 120,
      scriptTask: 120,
      exclusiveGateway: 50,
      parallelGateway: 50,
      inclusiveGateway: 50,
    };
    return widths[type] || 100;
  };

  const getDefaultHeight = (type: string) => {
    const heights: Record<string, number> = {
      startEvent: 60,
      endEvent: 60,
      userTask: 80,
      serviceTask: 80,
      scriptTask: 80,
      exclusiveGateway: 50,
      parallelGateway: 50,
      inclusiveGateway: 50,
    };
    return heights[type] || 60;
  };

  const getDefaultProperties = (type: string) => {
    const properties: Record<string, any> = {
      startEvent: { eventType: "start", name: "开始" },
      userTask: {
        assignee: "",
        candidateUsers: "",
        candidateGroups: "",
        formKey: "",
        name: "用户任务",
      },
      serviceTask: {
        class: "",
        name: "服务任务",
      },
      scriptTask: {
        script: "",
        name: "脚本任务",
      },
      endEvent: { eventType: "end", name: "结束" },
      exclusiveGateway: { name: "排他网关" },
      parallelGateway: { name: "并行网关" },
      inclusiveGateway: { name: "包容网关" },
    };
    return properties[type] || { name: "新元素" };
  };

  const updateElement = (id: string, updates: Partial<BPMNElement>) => {
    const newElements = elements.map((el) =>
      el.id === id ? { ...el, ...updates } : el
    );
    setElements(newElements);
    saveToHistory(newElements, connections);
  };

  const deleteElement = (id: string) => {
    const newElements = elements.filter((el) => el.id !== id);
    const newConnections = connections.filter(
      (conn) => conn.source !== id && conn.target !== id
    );
    setElements(newElements);
    setConnections(newConnections);
    setSelectedElement(null);
    saveToHistory(newElements, newConnections);
  };

  const addConnection = (source: string, target: string) => {
    const newConnection: BPMNConnection = {
      id: `flow_${Date.now()}`,
      source,
      target,
      type: "sequenceFlow",
      properties: { name: "流程" },
    };

    const newConnections = [...connections, newConnection];
    setConnections(newConnections);
    saveToHistory(elements, newConnections);
  };

  const getElementIcon = (type: string) => {
    const icons: Record<string, React.ReactNode> = {
      startEvent: <CheckCircle className="w-4 h-4" />,
      userTask: <User className="w-4 h-4" />,
      serviceTask: <Settings className="w-4 h-4" />,
      scriptTask: <Code className="w-4 h-4" />,
      endEvent: <AlertCircle className="w-4 h-4" />,
      exclusiveGateway: <GitBranch className="w-4 h-4" />,
      parallelGateway: <GitMerge className="w-4 h-4" />,
      inclusiveGateway: <GitCommit className="w-4 h-4" />,
    };
    return icons[type] || <FileText className="w-4 h-4" />;
  };

  const getElementColor = (type: string) => {
    const colors: Record<string, string> = {
      startEvent: "#52c41a",
      userTask: "#1890ff",
      serviceTask: "#722ed1",
      scriptTask: "#fa8c16",
      endEvent: "#f5222d",
      exclusiveGateway: "#faad14",
      parallelGateway: "#13c2c2",
      inclusiveGateway: "#eb2f96",
    };
    return colors[type] || "#666";
  };

  const handleCanvasClick = (e: React.MouseEvent) => {
    if (e.target === canvasRef.current) {
      setSelectedElement(null);
    }
  };

  const handleElementClick = (element: BPMNElement) => {
    setSelectedElement(element);
  };

  const handleSave = () => {
    const workflowData = {
      elements,
      connections,
      bpmnXML: generateBPMNXML(),
      metadata: {
        version: "1.0.0",
        lastModified: new Date().toISOString(),
        totalElements: elements.length,
        totalConnections: connections.length,
      },
    };

    onSave(workflowData);
  };

  const generateBPMNXML = () => {
    // 生成BPMN 2.0 XML
    const xml = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="Definitions_1"
                  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    ${elements.map((element) => generateElementXML(element)).join("\n    ")}
    ${connections
      .map((connection) => generateConnectionXML(connection))
      .join("\n    ")}
  </bpmn:process>
</bpmn:definitions>`;

    return xml;
  };

  const generateElementXML = (element: BPMNElement) => {
    switch (element.type) {
      case "startEvent":
        return `<bpmn:startEvent id="${element.id}" name="${element.name}" />`;
      case "userTask":
        return `<bpmn:userTask id="${element.id}" name="${element.name}" />`;
      case "serviceTask":
        return `<bpmn:serviceTask id="${element.id}" name="${element.name}" />`;
      case "endEvent":
        return `<bpmn:endEvent id="${element.id}" name="${element.name}" />`;
      case "exclusiveGateway":
        return `<bpmn:exclusiveGateway id="${element.id}" name="${element.name}" />`;
      case "parallelGateway":
        return `<bpmn:parallelGateway id="${element.id}" name="${element.name}" />`;
      default:
        return `<bpmn:task id="${element.id}" name="${element.name}" />`;
    }
  };

  const generateConnectionXML = (connection: BPMNConnection) => {
    return `<bpmn:sequenceFlow id="${connection.id}" sourceRef="${connection.source}" targetRef="${connection.target}" />`;
  };

  const handleUndo = () => {
    if (historyIndex > 0) {
      const prevState = history[historyIndex - 1];
      setElements(prevState.elements);
      setConnections(prevState.connections);
      setHistoryIndex(historyIndex - 1);
    }
  };

  const handleRedo = () => {
    if (historyIndex < history.length - 1) {
      const nextState = history[historyIndex + 1];
      setElements(nextState.elements);
      setConnections(nextState.connections);
      setHistoryIndex(historyIndex + 1);
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* 工具栏 */}
      <div className="border-b border-gray-200 p-4 bg-white">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Space>
              <Tooltip title="保存">
                <Button
                  type="primary"
                  icon={<Save className="w-4 h-4" />}
                  onClick={handleSave}
                >
                  保存
                </Button>
              </Tooltip>
              <Tooltip title="预览">
                <Button icon={<Eye className="w-4 h-4" />}>预览</Button>
              </Tooltip>
              <Tooltip title="验证">
                <Button icon={<CheckCircle className="w-4 h-4" />}>验证</Button>
              </Tooltip>
            </Space>

            <Divider type="vertical" />

            <Space>
              <Tooltip title="撤销">
                <Button
                  icon={<Undo className="w-4 h-4" />}
                  disabled={historyIndex <= 0}
                  onClick={handleUndo}
                />
              </Tooltip>
              <Tooltip title="重做">
                <Button
                  icon={<Redo className="w-4 h-4" />}
                  disabled={historyIndex >= history.length - 1}
                  onClick={handleRedo}
                />
              </Tooltip>
            </Space>

            <Divider type="vertical" />

            <Space>
              <Tooltip title="放大">
                <Button
                  icon={<ZoomIn className="w-4 h-4" />}
                  onClick={() => setZoom((prev) => Math.min(prev + 0.1, 2))}
                />
              </Tooltip>
              <Tooltip title="缩小">
                <Button
                  icon={<ZoomOut className="w-4 h-4" />}
                  onClick={() => setZoom((prev) => Math.max(prev - 0.1, 0.5))}
                />
              </Tooltip>
              <Tooltip title="适应画布">
                <Button icon={<Fullscreen className="w-4 h-4" />} />
              </Tooltip>
            </Space>
          </div>

          <div className="flex items-center space-x-2">
            <Text className="text-sm text-gray-500">
              缩放: {Math.round(zoom * 100)}%
            </Text>
            <Badge count={elements.length} showZero>
              <Button size="small">元素</Button>
            </Badge>
            <Badge count={connections.length} showZero>
              <Button size="small">连接</Button>
            </Badge>
          </div>
        </div>
      </div>

      <div className="flex-1 flex">
        {/* 左侧工具栏 */}
        <div className="w-64 border-r border-gray-200 bg-gray-50 p-4">
          <Title level={5} className="mb-4">
            元素库
          </Title>

          <div className="space-y-2">
            <div className="text-sm font-medium text-gray-700 mb-2">事件</div>
            <div className="grid grid-cols-2 gap-2">
              <Tooltip title="开始事件">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("startEvent", 100, 100)}
                >
                  <CheckCircle className="w-5 h-5 text-green-500 mb-1" />
                  <span className="text-xs">开始</span>
                </Button>
              </Tooltip>
              <Tooltip title="结束事件">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("endEvent", 100, 100)}
                >
                  <AlertCircle className="w-5 h-5 text-red-500 mb-1" />
                  <span className="text-xs">结束</span>
                </Button>
              </Tooltip>
            </div>

            <div className="text-sm font-medium text-gray-700 mb-2 mt-4">
              任务
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Tooltip title="用户任务">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("userTask", 100, 100)}
                >
                  <User className="w-5 h-5 text-blue-500 mb-1" />
                  <span className="text-xs">用户任务</span>
                </Button>
              </Tooltip>
              <Tooltip title="服务任务">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("serviceTask", 100, 100)}
                >
                  <Settings className="w-5 h-5 text-purple-500 mb-1" />
                  <span className="text-xs">服务任务</span>
                </Button>
              </Tooltip>
              <Tooltip title="脚本任务">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("scriptTask", 100, 100)}
                >
                  <Code className="w-5 h-5 text-orange-500 mb-1" />
                  <span className="text-xs">脚本任务</span>
                </Button>
              </Tooltip>
            </div>

            <div className="text-sm font-medium text-gray-700 mb-2 mt-4">
              网关
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Tooltip title="排他网关">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("exclusiveGateway", 100, 100)}
                >
                  <GitBranch className="w-5 h-5 text-yellow-500 mb-1" />
                  <span className="text-xs">排他网关</span>
                </Button>
              </Tooltip>
              <Tooltip title="并行网关">
                <Button
                  size="small"
                  className="h-16 flex flex-col items-center justify-center"
                  onClick={() => addElement("parallelGateway", 100, 100)}
                >
                  <GitMerge className="w-5 h-5 text-cyan-500 mb-1" />
                  <span className="text-xs">并行网关</span>
                </Button>
              </Tooltip>
            </div>
          </div>
        </div>

        {/* 主画布区域 */}
        <div className="flex-1 flex flex-col">
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            className="flex-1 flex flex-col"
            tabBarStyle={{ margin: 0, padding: "0 16px" }}
            items={[
              {
                key: "design",
                label: "设计器",
                children: (
                  <div
                    ref={canvasRef}
                    className="flex-1 bg-gray-100 relative overflow-auto"
                    style={{
                      transform: `scale(${zoom})`,
                      transformOrigin: "top left",
                    }}
                    onClick={handleCanvasClick}
                  >
                    {/* 网格背景 */}
                    <div className="absolute inset-0 bg-grid-pattern opacity-20" />

                    {/* 元素 */}
                    {elements.map((element) => (
                      <div
                        key={element.id}
                        className={`absolute border-2 rounded cursor-move ${
                          selectedElement?.id === element.id
                            ? "border-blue-500 shadow-lg"
                            : "border-gray-300 hover:border-gray-400"
                        }`}
                        style={{
                          left: element.x,
                          top: element.y,
                          width: element.width,
                          height: element.height,
                          backgroundColor: getElementColor(element.type),
                        }}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleElementClick(element);
                        }}
                      >
                        <div className="flex items-center justify-center h-full text-white text-xs font-medium">
                          {getElementIcon(element.type)}
                          <span className="ml-1">{element.name}</span>
                        </div>
                      </div>
                    ))}

                    {/* 连接线 */}
                    {connections.map((connection) => {
                      const source = elements.find(
                        (el) => el.id === connection.source
                      );
                      const target = elements.find(
                        (el) => el.id === connection.target
                      );

                      if (!source || !target) return null;

                      const startX = source.x + (source.width || 0) / 2;
                      const startY = source.y + (source.height || 0) / 2;
                      const endX = target.x + (target.width || 0) / 2;
                      const endY = target.y + (target.height || 0) / 2;

                      return (
                        <svg
                          key={connection.id}
                          className="absolute pointer-events-none"
                          style={{
                            left: 0,
                            top: 0,
                            width: "100%",
                            height: "100%",
                          }}
                        >
                          <line
                            x1={startX}
                            y1={startY}
                            x2={endX}
                            y2={endY}
                            stroke="#666"
                            strokeWidth="2"
                            markerEnd="url(#arrowhead)"
                          />
                          <defs>
                            <marker
                              id="arrowhead"
                              markerWidth="10"
                              markerHeight="7"
                              refX="9"
                              refY="3.5"
                              orient="auto"
                            >
                              <polygon points="0 0, 10 3.5, 0 7" fill="#666" />
                            </marker>
                          </defs>
                        </svg>
                      );
                    })}
                  </div>
                ),
              },
              {
                key: "properties",
                label: "属性",
                children: (
                  <div className="p-4">
                    {selectedElement ? (
                      <div>
                        <Title level={5}>元素属性</Title>
                        <Form layout="vertical">
                          <Form.Item label="名称">
                            <Input
                              value={selectedElement.name}
                              onChange={(e) =>
                                updateElement(selectedElement.id, {
                                  name: e.target.value,
                                })
                              }
                            />
                          </Form.Item>

                          {selectedElement.type === "userTask" && (
                            <>
                              <Form.Item label="处理人">
                                <Input
                                  value={
                                    selectedElement.properties?.assignee || ""
                                  }
                                  onChange={(e) =>
                                    updateElement(selectedElement.id, {
                                      properties: {
                                        ...selectedElement.properties,
                                        assignee: e.target.value,
                                      },
                                    })
                                  }
                                  placeholder="输入用户ID或用户名"
                                />
                              </Form.Item>
                              <Form.Item label="候选用户">
                                <Input
                                  value={
                                    selectedElement.properties
                                      ?.candidateUsers || ""
                                  }
                                  onChange={(e) =>
                                    updateElement(selectedElement.id, {
                                      properties: {
                                        ...selectedElement.properties,
                                        candidateUsers: e.target.value,
                                      },
                                    })
                                  }
                                  placeholder="多个用户用逗号分隔"
                                />
                              </Form.Item>
                              <Form.Item label="候选组">
                                <Input
                                  value={
                                    selectedElement.properties
                                      ?.candidateGroups || ""
                                  }
                                  onChange={(e) =>
                                    updateElement(selectedElement.id, {
                                      properties: {
                                        ...selectedElement.properties,
                                        candidateGroups: e.target.value,
                                      },
                                    })
                                  }
                                  placeholder="多个组用逗号分隔"
                                />
                              </Form.Item>
                            </>
                          )}

                          {selectedElement.type === "serviceTask" && (
                            <Form.Item label="服务类">
                              <Input
                                value={selectedElement.properties?.class || ""}
                                onChange={(e) =>
                                  updateElement(selectedElement.id, {
                                    properties: {
                                      ...selectedElement.properties,
                                      class: e.target.value,
                                    },
                                  })
                                }
                                placeholder="输入服务类名"
                              />
                            </Form.Item>
                          )}

                          <Form.Item>
                            <Space>
                              <Button
                                type="primary"
                                danger
                                icon={<Trash2 className="w-4 h-4" />}
                                onClick={() =>
                                  deleteElement(selectedElement.id)
                                }
                              >
                                删除元素
                              </Button>
                            </Space>
                          </Form.Item>
                        </Form>
                      </div>
                    ) : (
                      <div className="text-center text-gray-500 py-8">
                        <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
                        <p>选择一个元素来编辑其属性</p>
                      </div>
                    )}
                  </div>
                ),
              },
              {
                key: "code",
                label: "代码",
                children: (
                  <div className="p-4">
                    <div className="flex items-center justify-between mb-4">
                      <Title level={5}>BPMN XML</Title>
                      <Space>
                        <Button icon={<Copy className="w-4 h-4" />}>
                          复制
                        </Button>
                        <Button icon={<Download className="w-4 h-4" />}>
                          导出
                        </Button>
                      </Space>
                    </div>
                    <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto max-h-96">
                      {generateBPMNXML()}
                    </pre>
                  </div>
                ),
              },
            ]}
          />
        </div>
      </div>
    </div>
  );
};

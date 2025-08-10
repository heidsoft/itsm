"use client";

import React, { useEffect, useRef, useState, useCallback } from "react";
import {
  Card,
  Button,
  Space,
  message,
  Upload,
  Modal,
  Form,
  Input,
  Select,
  Divider,
  Row,
  Col,
} from "antd";
import {
  SaveOutlined,
  PlayCircleOutlined,
  UploadOutlined,
  DownloadOutlined,
  UndoOutlined,
  RedoOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  FullscreenOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import dynamic from "next/dynamic";

const { Option } = Select;
const { TextArea } = Input;

// 动态导入BPMN.js以避免SSR问题
const BPMNModeler = dynamic(
  () => import("bpmn-js").then((mod) => ({ default: mod.BPMNModeler })),
  {
    ssr: false,
    loading: () => <div>Loading BPMN Designer...</div>,
  }
);

interface BPMNDesignerProps {
  initialXML?: string;
  onSave?: (xml: string) => void;
  onDeploy?: (xml: string) => void;
  readOnly?: boolean;
  height?: number;
}

interface WorkflowMetadata {
  name: string;
  description: string;
  version: string;
  category: string;
  tags: string[];
}

const BPMNDesigner: React.FC<BPMNDesignerProps> = ({
  initialXML = "",
  onSave,
  onDeploy,
  readOnly = false,
  height = 600,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<any>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [metadata, setMetadata] = useState<WorkflowMetadata>({
    name: "新工作流",
    description: "",
    version: "1.0.0",
    category: "general",
    tags: [],
  });
  const [showMetadataModal, setShowMetadataModal] = useState(false);
  const [canUndo, setCanUndo] = useState(false);
  const [canRedo, setCanRedo] = useState(false);

  // 默认BPMN XML模板
  const defaultXML = `<?xml version="1.0" encoding="UTF-8"?>
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
    <bpmn:userTask id="UserTask_1" name="用户任务">
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
        <bpmndi:BPMNLabel>
          <dc:Bounds x="158" y="145" width="24" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_1_di" bpmnElement="UserTask_1">
        <dc:Bounds x="240" y="80" width="100" height="80" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="270" y="105" width="40" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="392" y="102" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="398" y="145" width="24" height="14" />
        </bpmndi:BPMNLabel>
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
</bpmn:definitions>`;

  // 初始化BPMN设计器
  useEffect(() => {
    if (!containerRef.current || modelerRef.current) return;

    try {
      // 创建BPMN模型器
      const modeler = new BPMNModeler({
        container: containerRef.current,
        keyboard: {
          bindTo: window,
        },
        additionalModules: [
          // 可以在这里添加自定义模块
        ],
      });

      modelerRef.current = modeler;

      // 导入XML
      const xmlToImport = initialXML || defaultXML;
      modeler
        .importXML(xmlToImport)
        .then(() => {
          // 自动调整视图
          const canvas = modeler.get("canvas");
          canvas.zoom("fit-viewport");

          // 设置事件监听器
          const eventBus = modeler.get("eventBus");
          const commandStack = modeler.get("commandStack");

          // 监听命令栈变化
          eventBus.on("commandStack.changed", () => {
            setCanUndo(commandStack.canUndo());
            setCanRedo(commandStack.canRedo());
          });

          // 监听元素选择
          eventBus.on("element.click", (event: any) => {
            console.log("Selected element:", event.element);
          });

          message.success("BPMN设计器初始化成功");
        })
        .catch((err: any) => {
          console.error("导入XML失败:", err);
          message.error("导入XML失败: " + err.message);
        });

      // 清理函数
      return () => {
        if (modelerRef.current) {
          modelerRef.current.destroy();
          modelerRef.current = null;
        }
      };
    } catch (error) {
      console.error("初始化BPMN设计器失败:", error);
      message.error("初始化BPMN设计器失败");
    }
  }, [initialXML]);

  // 保存工作流
  const handleSave = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      if (onSave) {
        onSave(xml);
      }
      message.success("工作流保存成功");
    } catch (error) {
      console.error("保存失败:", error);
      message.error("保存失败: " + error);
    }
  }, [onSave]);

  // 部署工作流
  const handleDeploy = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      if (onDeploy) {
        onDeploy(xml);
      }
      message.success("工作流部署成功");
    } catch (error) {
      console.error("部署失败:", error);
      message.error("部署失败: " + error);
    }
  }, [onDeploy]);

  // 导入XML文件
  const handleImport = useCallback(async (file: File) => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const text = await file.text();
      await modelerRef.current.importXML(text);
      message.success("XML文件导入成功");
    } catch (error) {
      console.error("导入失败:", error);
      message.error("导入失败: " + error);
    }
  }, []);

  // 导出XML文件
  const handleExport = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      const blob = new Blob([xml], { type: "application/xml" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${metadata.name || "workflow"}.bpmn`;
      a.click();
      URL.revokeObjectURL(url);
      message.success("XML文件导出成功");
    } catch (error) {
      console.error("导出失败:", error);
      message.error("导出失败: " + error);
    }
  }, [metadata.name]);

  // 撤销操作
  const handleUndo = useCallback(() => {
    if (modelerRef.current && canUndo) {
      const commandStack = modelerRef.current.get("commandStack");
      commandStack.undo();
    }
  }, [canUndo]);

  // 重做操作
  const handleRedo = useCallback(() => {
    if (modelerRef.current && canRedo) {
      const commandStack = modelerRef.current.get("commandStack");
      commandStack.redo();
    }
  }, [canRedo]);

  // 缩放操作
  const handleZoom = useCallback((direction: "in" | "out" | "fit") => {
    if (!modelerRef.current) return;

    const canvas = modelerRef.current.get("canvas");
    const currentZoom = canvas.getZoom();

    switch (direction) {
      case "in":
        canvas.zoom(Math.min(currentZoom * 1.2, 3));
        break;
      case "out":
        canvas.zoom(Math.max(currentZoom / 1.2, 0.2));
        break;
      case "fit":
        canvas.zoom("fit-viewport");
        break;
    }
  }, []);

  // 切换全屏
  const toggleFullscreen = useCallback(() => {
    setIsFullscreen(!isFullscreen);
  }, [isFullscreen]);

  // 更新元数据
  const handleMetadataUpdate = useCallback((values: WorkflowMetadata) => {
    setMetadata(values);
    setShowMetadataModal(false);
    message.success("元数据更新成功");
  }, []);

  return (
    <div className={`bpmn-designer ${isFullscreen ? "fullscreen" : ""}`}>
      <Card
        title={
          <Space>
            <span>BPMN工作流设计器</span>
            {metadata.name && (
              <span className="text-gray-500">- {metadata.name}</span>
            )}
          </Space>
        }
        extra={
          <Space>
            <Button
              icon={<SettingOutlined />}
              onClick={() => setShowMetadataModal(true)}
              size="small"
            >
              元数据
            </Button>
            <Button
              icon={<UndoOutlined />}
              disabled={!canUndo}
              onClick={handleUndo}
              size="small"
            >
              撤销
            </Button>
            <Button
              icon={<RedoOutlined />}
              disabled={!canRedo}
              onClick={handleRedo}
              size="small"
            >
              重做
            </Button>
            <Divider type="vertical" />
            <Button
              icon={<ZoomOutOutlined />}
              onClick={() => handleZoom("out")}
              size="small"
            />
            <Button
              icon={<ZoomInOutlined />}
              onClick={() => handleZoom("in")}
              size="small"
            />
            <Button
              icon={<FullscreenOutlined />}
              onClick={() => handleZoom("fit")}
              size="small"
            >
              适应
            </Button>
            <Button
              icon={<FullscreenOutlined />}
              onClick={toggleFullscreen}
              size="small"
            >
              {isFullscreen ? "退出全屏" : "全屏"}
            </Button>
          </Space>
        }
        bodyStyle={{ padding: 0 }}
      >
        {/* 工具栏 */}
        <div
          className="bpmn-toolbar"
          style={{ padding: "8px 16px", borderBottom: "1px solid #f0f0f0" }}
        >
          <Row gutter={16} align="middle">
            <Col>
              <Space>
                <Upload
                  accept=".bpmn,.xml"
                  showUploadList={false}
                  beforeUpload={(file) => {
                    handleImport(file);
                    return false;
                  }}
                >
                  <Button icon={<UploadOutlined />} size="small">
                    导入XML
                  </Button>
                </Upload>
                <Button
                  icon={<DownloadOutlined />}
                  onClick={handleExport}
                  size="small"
                >
                  导出XML
                </Button>
              </Space>
            </Col>
            <Col>
              <Space>
                <Button
                  icon={<SaveOutlined />}
                  onClick={handleSave}
                  type="primary"
                  size="small"
                >
                  保存
                </Button>
                <Button
                  icon={<PlayCircleOutlined />}
                  onClick={handleDeploy}
                  type="primary"
                  size="small"
                >
                  部署
                </Button>
              </Space>
            </Col>
          </Row>
        </div>

        {/* BPMN设计器容器 */}
        <div
          ref={containerRef}
          style={{
            height: isFullscreen ? "calc(100vh - 120px)" : height,
            width: "100%",
            position: "relative",
          }}
        />

        {/* 元数据编辑模态框 */}
        <Modal
          title="工作流元数据"
          open={showMetadataModal}
          onCancel={() => setShowMetadataModal(false)}
          footer={null}
          width={600}
        >
          <Form
            layout="vertical"
            initialValues={metadata}
            onFinish={handleMetadataUpdate}
          >
            <Form.Item
              label="工作流名称"
              name="name"
              rules={[{ required: true, message: "请输入工作流名称" }]}
            >
              <Input placeholder="请输入工作流名称" />
            </Form.Item>
            <Form.Item label="描述" name="description">
              <TextArea rows={3} placeholder="请输入工作流描述" />
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  label="版本"
                  name="version"
                  rules={[{ required: true, message: "请输入版本号" }]}
                >
                  <Input placeholder="1.0.0" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="分类" name="category">
                  <Select placeholder="选择分类">
                    <Option value="general">通用</Option>
                    <Option value="it">IT服务</Option>
                    <Option value="hr">人力资源</Option>
                    <Option value="finance">财务</Option>
                    <Option value="sales">销售</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit">
                  保存
                </Button>
                <Button onClick={() => setShowMetadataModal(false)}>
                  取消
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>
      </Card>

      <style jsx>{`
        .bpmn-designer {
          width: 100%;
        }

        .bpmn-designer.fullscreen {
          position: fixed;
          top: 0;
          left: 0;
          width: 100vw;
          height: 100vh;
          z-index: 1000;
          background: white;
        }

        .bpmn-toolbar {
          background: #fafafa;
        }

        .bpmn-designer :global(.djs-palette) {
          border: 1px solid #ccc;
        }

        .bpmn-designer :global(.djs-context-pad) {
          border: 1px solid #ccc;
          border-radius: 4px;
        }
      `}</style>
    </div>
  );
};

export default BPMNDesigner;

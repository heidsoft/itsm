"use client";

import React, { useEffect, useState } from "react";
import {
  Card,
  Form,
  Input,
  Select,
  InputNumber,
  Button,
  Space,
  Divider,
  message,
} from "antd";
import { EditOutlined, SaveOutlined, CloseOutlined } from "@ant-design/icons";

const { Option } = Select;
const { TextArea } = Input;

interface BPMNElement {
  id: string;
  type: string;
  businessObject?: any;
  [key: string]: any;
}

interface BPMNPropertiesPanelProps {
  selectedElement: BPMNElement | null;
  modeler: any;
  onPropertyChange?: (elementId: string, properties: any) => void;
}

interface ElementProperties {
  name: string;
  description?: string;
  assignee?: string;
  priority?: string;
  timeout?: number;
  retryCount?: number;
  conditions?: string[];
  actions?: string[];
  [key: string]: any;
}

const BPMNPropertiesPanel: React.FC<BPMNPropertiesPanelProps> = ({
  selectedElement,
  modeler,
  onPropertyChange,
}) => {
  const [form] = Form.useForm();
  const [properties, setProperties] = useState<ElementProperties>({});
  const [isEditing, setIsEditing] = useState(false);
  const [originalProperties, setOriginalProperties] =
    useState<ElementProperties>({});

  // 当选择的元素改变时，更新属性
  useEffect(() => {
    if (selectedElement && modeler) {
      const elementProps = extractElementProperties(selectedElement);
      setProperties(elementProps);
      setOriginalProperties(elementProps);
      form.setFieldsValue(elementProps);
      setIsEditing(false);
    }
  }, [selectedElement, modeler, form]);

  // 提取元素属性
  const extractElementProperties = (
    element: BPMNElement
  ): ElementProperties => {
    const props: ElementProperties = {
      name: element.businessObject?.name || element.name || "",
      description: element.businessObject?.description || "",
      assignee: element.businessObject?.assignee || "",
      priority: element.businessObject?.priority || "medium",
      timeout: element.businessObject?.timeout || 24,
      retryCount: element.businessObject?.retryCount || 3,
      conditions: element.businessObject?.conditions || [],
      actions: element.businessObject?.actions || [],
    };

    // 根据元素类型添加特定属性
    switch (element.type) {
      case "bpmn:UserTask":
        props.assignee = element.businessObject?.assignee || "";
        props.priority = element.businessObject?.priority || "medium";
        break;
      case "bpmn:ServiceTask":
        props.timeout = element.businessObject?.timeout || 60;
        props.retryCount = element.businessObject?.retryCount || 3;
        break;
      case "bpmn:ExclusiveGateway":
        props.conditions = element.businessObject?.conditions || [];
        break;
      case "bpmn:ParallelGateway":
        props.actions = element.businessObject?.actions || [];
        break;
    }

    return props;
  };

  // 开始编辑
  const handleEdit = () => {
    setIsEditing(true);
  };

  // 保存属性
  const handleSave = async () => {
    try {
      const values = await form.validateFields();

      if (selectedElement && modeler) {
        // 更新BPMN元素属性
        updateElementProperties(selectedElement, values);

        // 通知父组件
        if (onPropertyChange) {
          onPropertyChange(selectedElement.id, values);
        }

        setProperties(values);
        setOriginalProperties(values);
        setIsEditing(false);
        message.success("属性保存成功");
      }
    } catch (error) {
      console.error("保存属性失败:", error);
      message.error("保存失败，请检查输入");
    }
  };

  // 取消编辑
  const handleCancel = () => {
    form.setFieldsValue(originalProperties);
    setIsEditing(false);
  };

  // 更新BPMN元素属性
  const updateElementProperties = (
    element: BPMNElement,
    properties: ElementProperties
  ) => {
    if (!modeler) return;

    const modeling = modeler.get("modeling");
    const moddle = modeler.get("moddle");

    // 更新基本属性
    if (properties.name !== element.businessObject?.name) {
      modeling.updateProperties(element, {
        name: properties.name,
      });
    }

    // 更新扩展属性
    const extensionElements =
      element.businessObject?.extensionElements ||
      moddle.create("bpmn:ExtensionElements");

    // 创建或更新自定义属性
    const customProperties = {
      description: properties.description,
      assignee: properties.assignee,
      priority: properties.priority,
      timeout: properties.timeout,
      retryCount: properties.retryCount,
      conditions: properties.conditions,
      actions: properties.actions,
    };

    // 更新扩展元素
    modeling.updateProperties(element, {
      extensionElements: extensionElements,
    });

    // 将自定义属性存储到扩展元素中
    Object.entries(customProperties).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        element.businessObject[`itsm:${key}`] = value;
      }
    });
  };

  // 渲染属性表单
  const renderPropertyForm = () => {
    if (!selectedElement) {
      return (
        <div style={{ textAlign: "center", padding: "20px", color: "#999" }}>
          请选择一个BPMN元素来查看和编辑其属性
        </div>
      );
    }

    return (
      <Form form={form} layout="vertical" disabled={!isEditing}>
        {/* 基本信息 */}
        <Divider orientation="left">基本信息</Divider>

        <Form.Item label="元素ID" name="id">
          <Input disabled />
        </Form.Item>

        <Form.Item label="元素类型" name="type">
          <Input disabled />
        </Form.Item>

        <Form.Item
          label="名称"
          name="name"
          rules={[{ required: true, message: "请输入名称" }]}
        >
          <Input placeholder="请输入元素名称" />
        </Form.Item>

        <Form.Item label="描述" name="description">
          <TextArea rows={3} placeholder="请输入描述" />
        </Form.Item>

        {/* 任务相关属性 */}
        {selectedElement.type === "bpmn:UserTask" && (
          <>
            <Divider orientation="left">任务属性</Divider>

            <Form.Item label="负责人" name="assignee">
              <Input placeholder="请输入负责人" />
            </Form.Item>

            <Form.Item label="优先级" name="priority">
              <Select placeholder="选择优先级">
                <Option value="low">低</Option>
                <Option value="medium">中</Option>
                <Option value="high">高</Option>
                <Option value="urgent">紧急</Option>
              </Select>
            </Form.Item>

            <Form.Item label="超时时间(小时)" name="timeout">
              <InputNumber min={1} max={720} style={{ width: "100%" }} />
            </Form.Item>
          </>
        )}

        {/* 服务任务属性 */}
        {selectedElement.type === "bpmn:ServiceTask" && (
          <>
            <Divider orientation="left">服务任务属性</Divider>

            <Form.Item label="超时时间(分钟)" name="timeout">
              <InputNumber min={1} max={1440} style={{ width: "100%" }} />
            </Form.Item>

            <Form.Item label="重试次数" name="retryCount">
              <InputNumber min={0} max={10} style={{ width: "100%" }} />
            </Form.Item>
          </>
        )}

        {/* 网关属性 */}
        {(selectedElement.type === "bpmn:ExclusiveGateway" ||
          selectedElement.type === "bpmn:ParallelGateway") && (
          <>
            <Divider orientation="left">网关属性</Divider>

            <Form.Item label="条件表达式" name="conditions">
              <TextArea rows={3} placeholder="请输入条件表达式" />
            </Form.Item>
          </>
        )}

        {/* 事件属性 */}
        {selectedElement.type.includes("Event") && (
          <>
            <Divider orientation="left">事件属性</Divider>

            <Form.Item label="事件类型" name="eventType">
              <Select placeholder="选择事件类型">
                <Option value="message">消息事件</Option>
                <Option value="timer">定时器事件</Option>
                <Option value="signal">信号事件</Option>
                <Option value="error">错误事件</Option>
              </Select>
            </Form.Item>
          </>
        )}

        {/* 操作按钮 */}
        <Divider />
        <Form.Item>
          <Space>
            {!isEditing ? (
              <Button
                type="primary"
                icon={<EditOutlined />}
                onClick={handleEdit}
              >
                编辑
              </Button>
            ) : (
              <>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  onClick={handleSave}
                >
                  保存
                </Button>
                <Button icon={<CloseOutlined />} onClick={handleCancel}>
                  取消
                </Button>
              </>
            )}
          </Space>
        </Form.Item>
      </Form>
    );
  };

  return (
    <Card
      title="属性面板"
      size="small"
      style={{ height: "100%", overflow: "auto" }}
      bodyStyle={{ padding: "12px" }}
    >
      {renderPropertyForm()}
    </Card>
  );
};

export default BPMNPropertiesPanel;

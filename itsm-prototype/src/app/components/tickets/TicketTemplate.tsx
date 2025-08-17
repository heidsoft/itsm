"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Button,
  Table,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Space,
  Tag,
  message,
  Popconfirm,
  Tooltip,
  Drawer,
  Divider,
  Typography,
  Row,
  Col,
  Upload,
  Checkbox,
} from "antd";
import {
  Plus,
  Edit,
  Delete,
  Copy,
  Eye,
  FileText,
  Settings,
  Save,
  CheckCircle,
  AlertTriangle,
  Info,
  Upload as UploadIcon,
} from "lucide-react";
import { LoadingEmptyError } from "../ui/LoadingEmptyError";
import { LoadingSkeleton } from "../ui/LoadingSkeleton";

const { TextArea } = Input;
const { Title, Text } = Typography;
const { Option } = Select;

interface TicketTemplate {
  id: number;
  name: string;
  description: string;
  category: string;
  subcategory: string;
  priority: string;
  impact: string;
  urgency: string;
  isActive: boolean;
  isDefault: boolean;
  fields: TemplateField[];
  workflow: string;
  sla: string;
  tags: string[];
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

interface TemplateField {
  id: string;
  name: string;
  label: string;
  type: 'text' | 'textarea' | 'select' | 'number' | 'date' | 'checkbox' | 'radio' | 'file';
  required: boolean;
  defaultValue?: any;
  options?: Array<{ label: string; value: any }>;
  placeholder?: string;
  helpText?: string;
  order: number;
}

interface TicketTemplateProps {
  onSelectTemplate?: (template: TicketTemplate) => void;
  mode?: 'select' | 'manage';
}

export const TicketTemplate: React.FC<TicketTemplateProps> = ({
  onSelectTemplate,
  mode = 'manage'
}) => {
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(null);
  const [form] = Form.useForm();
  const [fieldForm] = Form.useForm();

  // 模拟加载模板数据
  useEffect(() => {
    loadTemplates();
  }, []);

  const loadTemplates = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // 模拟模板数据
      const mockTemplates: TicketTemplate[] = [
        {
          id: 1,
          name: "网络故障标准模板",
          description: "用于处理常见的网络连接问题",
          category: "网络故障",
          subcategory: "连接问题",
          priority: "medium",
          impact: "medium",
          urgency: "medium",
          isActive: true,
          isDefault: false,
          fields: [
            {
              id: "network_type",
              name: "network_type",
              label: "网络类型",
              type: "select",
              required: true,
              options: [
                { label: "有线网络", value: "wired" },
                { label: "无线网络", value: "wireless" },
                { label: "VPN", value: "vpn" }
              ],
              order: 1
            },
            {
              id: "affected_users",
              name: "affected_users",
              label: "受影响用户数",
              type: "number",
              required: true,
              placeholder: "请输入受影响用户数量",
              helpText: "用于评估问题影响范围",
              order: 2
            }
          ],
          workflow: "网络故障处理流程",
          sla: "4小时",
          tags: ["网络", "标准", "常用"],
          createdBy: "系统管理员",
          createdAt: "2024-01-01 00:00:00",
          updatedAt: "2024-01-15 10:00:00"
        },
        {
          id: 2,
          name: "软件安装问题模板",
          description: "处理软件安装、配置相关问题",
          category: "软件故障",
          subcategory: "安装配置",
          priority: "low",
          impact: "low",
          urgency: "low",
          isActive: true,
          isDefault: false,
          fields: [
            {
              id: "software_name",
              name: "software_name",
              label: "软件名称",
              type: "text",
              required: true,
              placeholder: "请输入软件名称",
              order: 1
            },
            {
              id: "os_version",
              name: "os_version",
              label: "操作系统版本",
              type: "text",
              required: true,
              placeholder: "如：Windows 10 21H2",
              order: 2
            }
          ],
          workflow: "软件问题处理流程",
          sla: "8小时",
          tags: ["软件", "安装", "配置"],
          createdBy: "系统管理员",
          createdAt: "2024-01-01 00:00:00",
          updatedAt: "2024-01-15 10:00:00"
        }
      ];

      setTemplates(mockTemplates);
    } catch (error) {
      message.error("加载模板失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTemplate = () => {
    setEditingTemplate(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditTemplate = (template: TicketTemplate) => {
    setEditingTemplate(template);
    form.setFieldsValue(template);
    setModalVisible(true);
  };

  const handleCopyTemplate = (template: TicketTemplate) => {
    const newTemplate = {
      ...template,
      id: Date.now(),
      name: `${template.name} - 副本`,
      isDefault: false,
      createdAt: new Date().toLocaleString(),
      updatedAt: new Date().toLocaleString()
    };
    
    setTemplates(prev => [newTemplate, ...prev]);
    message.success("模板复制成功");
  };

  const handleDeleteTemplate = async (id: number) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      
      setTemplates(prev => prev.filter(t => t.id !== id));
      message.success("模板删除成功");
    } catch (error) {
      message.error("删除失败");
    }
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await form.validateFields();
      
      if (editingTemplate) {
        // 更新模板
        const updatedTemplate = {
          ...editingTemplate,
          ...values,
          updatedAt: new Date().toLocaleString()
        };
        
        setTemplates(prev => 
          prev.map(t => t.id === editingTemplate.id ? updatedTemplate : t)
        );
        message.success("模板更新成功");
      } else {
        // 创建新模板
        const newTemplate: TicketTemplate = {
          id: Date.now(),
          ...values,
          isActive: true,
          isDefault: false,
          fields: [],
          tags: values.tags || [],
          createdBy: "当前用户",
          createdAt: new Date().toLocaleString(),
          updatedAt: new Date().toLocaleString()
        };
        
        setTemplates(prev => [newTemplate, ...prev]);
        message.success("模板创建成功");
      }
      
      setModalVisible(false);
      form.resetFields();
    } catch (error) {
      console.error("保存模板失败:", error);
    }
  };

  const handleSelectTemplate = (template: TicketTemplate) => {
    if (onSelectTemplate) {
      onSelectTemplate(template);
    }
  };

  const columns = [
    {
      title: "模板名称",
      dataIndex: "name",
      key: "name",
      render: (text: string, record: TicketTemplate) => (
        <div>
          <div className="font-medium">{text}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      )
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      render: (category: string, record: TicketTemplate) => (
        <div>
          <Tag color="blue">{category}</Tag>
          {record.subcategory && <Tag color="cyan">{record.subcategory}</Tag>}
        </div>
      )
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      render: (priority: string) => {
        const colors = {
          low: 'blue',
          medium: 'orange',
          high: 'red',
          critical: 'purple'
        };
        return <Tag color={colors[priority as keyof typeof colors]}>{priority}</Tag>;
      }
    },
    {
      title: "状态",
      key: "status",
      render: (record: TicketTemplate) => (
        <div className="flex items-center gap-2">
          <Switch
            checked={record.isActive}
            size="small"
            onChange={(checked) => {
              setTemplates(prev => 
                prev.map(t => t.id === record.id ? { ...t, isActive: checked } : t)
              );
            }}
          />
          {record.isDefault && <Tag color="green">默认</Tag>}
        </div>
      )
    },
    {
      title: "SLA",
      dataIndex: "sla",
      key: "sla",
      render: (sla: string) => <Tag color="orange">{sla}</Tag>
    },
    {
      title: "操作",
      key: "actions",
      render: (record: TicketTemplate) => (
        <Space>
          {mode === 'select' && (
            <Button 
              type="primary" 
              size="small"
              onClick={() => handleSelectTemplate(record)}
            >
              使用模板
            </Button>
          )}
          <Tooltip title="查看详情">
            <Button 
              size="small" 
              icon={<Eye size={14} />}
              onClick={() => {
                setEditingTemplate(record);
                setDrawerVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button 
              size="small" 
              icon={<Edit size={14} />}
              onClick={() => handleEditTemplate(record)}
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button 
              size="small" 
              icon={<Copy size={14} />}
              onClick={() => handleCopyTemplate(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个模板吗？"
            onConfirm={() => handleDeleteTemplate(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button 
              size="small" 
              danger 
              icon={<Delete size={14} />}
            />
          </Popconfirm>
        </Space>
      )
    }
  ];

  if (loading) {
    return <LoadingSkeleton type="table" rows={5} columns={6} />;
  }

  if (templates.length === 0) {
    return (
      <LoadingEmptyError
        state="empty"
        empty={{
          title: "暂无工单模板",
          description: "创建第一个工单模板来标准化工单创建流程",
          actionText: "创建模板",
          onAction: handleCreateTemplate
        }}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* 头部操作区 */}
      <Card>
        <div className="flex justify-between items-center">
          <div>
            <Title level={4} className="mb-1">工单模板管理</Title>
            <Text type="secondary">管理和配置工单创建模板，提高工单处理效率</Text>
          </div>
          {mode === 'manage' && (
            <Button 
              type="primary" 
              icon={<Plus size={16} />}
              onClick={handleCreateTemplate}
            >
              创建模板
            </Button>
          )}
        </div>
      </Card>

      {/* 模板列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={templates}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => 
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`
          }}
        />
      </Card>

      {/* 创建/编辑模板模态框 */}
      <Modal
        title={editingTemplate ? "编辑模板" : "创建模板"}
        open={modalVisible}
        onOk={handleSaveTemplate}
        onCancel={() => setModalVisible(false)}
        okText="保存"
        cancelText="取消"
        width={800}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="模板名称"
                rules={[{ required: true, message: '请输入模板名称' }]}
              >
                <Input placeholder="请输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="category"
                label="主分类"
                rules={[{ required: true, message: '请选择主分类' }]}
              >
                <Select placeholder="请选择主分类">
                  <Option value="网络故障">网络故障</Option>
                  <Option value="硬件故障">硬件故障</Option>
                  <Option value="软件故障">软件故障</Option>
                  <Option value="权限问题">权限问题</Option>
                  <Option value="其他">其他</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="subcategory"
                label="子分类"
              >
                <Input placeholder="请输入子分类" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="默认优先级"
                rules={[{ required: true, message: '请选择默认优先级' }]}
              >
                <Select placeholder="请选择默认优先级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="impact"
                label="影响程度"
                rules={[{ required: true, message: '请选择影响程度' }]}
              >
                <Select placeholder="请选择影响程度">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="urgency"
                label="紧急程度"
                rules={[{ required: true, message: '请选择紧急程度' }]}
              >
                <Select placeholder="请选择紧急程度">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="description"
            label="模板描述"
          >
            <TextArea rows={3} placeholder="请输入模板描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="workflow"
                label="关联工作流"
              >
                <Select placeholder="请选择工作流" allowClear>
                  <Option value="标准处理流程">标准处理流程</Option>
                  <Option value="紧急处理流程">紧急处理流程</Option>
                  <Option value="变更审批流程">变更审批流程</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="sla"
                label="SLA目标"
              >
                <Input placeholder="如：4小时" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="tags"
            label="标签"
          >
            <Select mode="tags" placeholder="请输入标签">
              <Option value="标准">标准</Option>
              <Option value="常用">常用</Option>
              <Option value="紧急">紧急</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="isActive"
            label="启用状态"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 模板详情抽屉 */}
      <Drawer
        title="模板详情"
        placement="right"
        width={600}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        {editingTemplate && (
          <div className="space-y-6">
            <div>
              <Title level={5}>基本信息</Title>
              <div className="bg-gray-50 p-4 rounded-lg space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-600">模板名称:</span>
                  <span className="font-medium">{editingTemplate.name}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">分类:</span>
                  <span>{editingTemplate.category} / {editingTemplate.subcategory}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">优先级:</span>
                  <Tag color="orange">{editingTemplate.priority}</Tag>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">状态:</span>
                  <Tag color={editingTemplate.isActive ? 'green' : 'red'}>
                    {editingTemplate.isActive ? '启用' : '禁用'}
                  </Tag>
                </div>
              </div>
            </div>

            <div>
              <Title level={5}>自定义字段</Title>
              {editingTemplate.fields.length > 0 ? (
                <div className="space-y-2">
                  {editingTemplate.fields.map((field) => (
                    <div key={field.id} className="border rounded p-3">
                      <div className="flex justify-between items-center mb-2">
                        <span className="font-medium">{field.label}</span>
                        <Tag color={field.required ? 'red' : 'blue'}>
                          {field.required ? '必填' : '选填'}
                        </Tag>
                      </div>
                      <div className="text-sm text-gray-500">
                        类型: {field.type} • 顺序: {field.order}
                      </div>
                      {field.helpText && (
                        <div className="text-sm text-gray-600 mt-1">
                          {field.helpText}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center text-gray-500 py-4">
                  暂无自定义字段
                </div>
              )}
            </div>

            <div>
              <Title level={5}>其他信息</Title>
              <div className="bg-gray-50 p-4 rounded-lg space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-600">工作流:</span>
                  <span>{editingTemplate.workflow || '未设置'}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">SLA:</span>
                  <span>{editingTemplate.sla || '未设置'}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">创建人:</span>
                  <span>{editingTemplate.createdBy}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">创建时间:</span>
                  <span>{editingTemplate.createdAt}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">更新时间:</span>
                  <span>{editingTemplate.updatedAt}</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default TicketTemplate;

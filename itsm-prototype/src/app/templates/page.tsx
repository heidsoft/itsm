"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Modal,
  Form,
  message,
  Tag,
  Space,
  Row,
  Col,
  Statistic,
  Select,
  Switch,
  Divider,
  Typography,
  Popconfirm,
  Tooltip,
  Badge,
} from "antd";
import {
  Plus,
  Edit,
  Delete,
  Copy,
  Eye,
  Search,
  Filter,
  FileText,
  Settings,
  Users,
  Calendar,
  AlertTriangle,
  CheckCircle,
  Clock,
  RefreshCw,
  Download,
  Upload,
} from "lucide-react";
import LoadingEmptyError from "../components/ui/LoadingEmptyError";
import {
  ticketTemplateService,
  type TicketTemplate,
  type CreateTemplateRequest,
  type UpdateTemplateRequest,
} from "../../lib/services/ticket-template-service";

const { TextArea } = Input;
const { Option } = Select;
const { Title, Text } = Typography;

// 工单模板接口
interface TicketTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  priority: string;
  form_fields: Record<string, any>;
  workflow_steps: WorkflowStep[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
  tenant_id: string;
}

interface FormField {
  id: string;
  name: string;
  label: string;
  type:
    | "text"
    | "textarea"
    | "select"
    | "number"
    | "date"
    | "checkbox"
    | "radio";
  required: boolean;
  default_value?: string;
  options?: string[];
  placeholder?: string;
  validation_rules?: string;
  order: number;
}

interface WorkflowStep {
  id: string;
  name: string;
  type: "approval" | "assignment" | "notification" | "automation";
  assignee_group?: string;
  assignee_user?: string;
  due_time?: number;
  conditions?: string;
  order: number;
}

interface TemplateFilters {
  category?: string;
  is_active?: boolean;
  keyword?: string;
}

const TicketTemplatePage: React.FC = () => {
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(
    null
  );
  const [filters, setFilters] = useState<TemplateFilters>({});
  const [form] = Form.useForm();

  // 模拟数据（作为备用）
  const mockTemplates: TicketTemplate[] = [];

  // 加载模板数据
  const loadTemplates = async () => {
    try {
      setLoading(true);
      setError(null);

      // 调用真实API
      const response = await ticketTemplateService.getTemplates({
        page: 1,
        size: 100,
        sort_by: "created_at",
        sort_order: "desc",
      });

      setTemplates(response.data);
    } catch (error) {
      console.error("加载模板失败:", error);
      // 如果API调用失败，使用模拟数据作为备用
      setTemplates(mockTemplates);
      setError("加载模板数据失败，使用本地数据");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTemplates();
  }, []);

  // 处理创建模板
  const handleCreateTemplate = () => {
    setEditingTemplate(null);
    form.resetFields();
    setModalVisible(true);
  };

  // 处理编辑模板
  const handleEditTemplate = (template: TicketTemplate) => {
    setEditingTemplate(template);
    form.setFieldsValue({
      name: template.name,
      description: template.description,
      category: template.category,
      priority: template.priority,
      assignee_group: template.form_fields?.assignee_group,
      is_active: template.is_active,
    });
    setModalVisible(true);
  };

  // 处理删除模板
  const handleDeleteTemplate = async (id: string) => {
    try {
      await ticketTemplateService.deleteTemplate(id);

      setTemplates(templates.filter((t) => t.id !== id));
      message.success("模板删除成功");
    } catch (error) {
      message.error("删除失败");
      console.error("删除失败:", error);
    }
  };

  // 处理复制模板
    const handleCopyTemplate = async (template: TicketTemplate) => {
    try {
      const newName = `${template.name} - 副本`;
      const copiedTemplate = await ticketTemplateService.copyTemplate(
        template.id,
        newName
      );
      
      setTemplates([copiedTemplate, ...templates]);
      message.success("模板复制成功");
    } catch (error) {
      message.error("复制失败");
      console.error("复制失败:", error);
    }
  };

  // 处理表单提交
  const handleFormSubmit = async (values: any) => {
    try {
            if (editingTemplate) {
        // 更新模板
        const updatedTemplate = await ticketTemplateService.updateTemplate(
          editingTemplate.id,
          {
            name: values.name,
            description: values.description,
            category: values.category,
            priority: values.priority,
            form_fields: {
              assignee_group: values.assignee_group,
            },
            is_active: values.is_active,
          }
        );
        
        setTemplates(
          templates.map((t) =>
            t.id === editingTemplate.id ? updatedTemplate : t
          )
        );
        message.success("模板更新成功");
      } else {
        // 创建新模板
        const newTemplate = await ticketTemplateService.createTemplate({
          name: values.name,
          description: values.description,
          category: values.category,
          priority: values.priority,
          form_fields: {
            assignee_group: values.assignee_group,
          },
          workflow_steps: [],
          is_active: values.is_active,
        });

        setTemplates([newTemplate, ...templates]);
        message.success("模板创建成功");
      }

      setModalVisible(false);
      setEditingTemplate(null);
      form.resetFields();
    } catch (error) {
      message.error("操作失败");
      console.error("操作失败:", error);
    }
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: "green",
      inactive: "red",
    };
    return colors[status] || "default";
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: "blue",
      medium: "orange",
      high: "red",
      critical: "red",
    };
    return colors[priority] || "default";
  };

  // 表格列定义
  const columns = [
    {
      title: "模板名称",
      dataIndex: "name",
      key: "name",
      render: (name: string, record: TicketTemplate) => (
        <div>
          <div className="font-medium text-gray-900">{name}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>
          {priority === "low" && "低"}
          {priority === "medium" && "中"}
          {priority === "high" && "高"}
          {priority === "critical" && "紧急"}
        </Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "is_active",
      key: "is_active",
      width: 100,
      render: (isActive: boolean) => (
        <Badge
          status={isActive ? "success" : "error"}
          text={isActive ? "启用" : "禁用"}
        />
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 120,
      render: (date: string) => new Date(date).toLocaleDateString("zh-CN"),
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      key: "updated_at",
      width: 120,
      render: (date: string) => new Date(date).toLocaleDateString("zh-CN"),
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      render: (record: TicketTemplate) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              size="small"
            />
          </Tooltip>
          <Tooltip title="编辑模板">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              size="small"
              onClick={() => handleEditTemplate(record)}
            />
          </Tooltip>
          <Tooltip title="复制模板">
            <Button
              type="text"
              icon={<Copy className="w-4 h-4" />}
              size="small"
              onClick={() => handleCopyTemplate(record)}
            />
          </Tooltip>
          <Tooltip title="删除模板">
            <Popconfirm
              title="确定要删除这个模板吗？"
              onConfirm={() => handleDeleteTemplate(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="text"
                icon={<Delete className="w-4 h-4" />}
                size="small"
                danger
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  // 过滤模板
  const filteredTemplates = templates.filter((template) => {
    if (filters.category && template.category !== filters.category)
      return false;
    if (
      filters.is_active !== undefined &&
      template.is_active !== filters.is_active
    )
      return false;
    if (
      filters.keyword &&
      !template.name.toLowerCase().includes(filters.keyword.toLowerCase())
    )
      return false;
    return true;
  });

  // 统计数据
  const stats = {
    total: templates.length,
    active: templates.filter((t) => t.is_active).length,
    categories: new Set(templates.map((t) => t.category)).size,
  };

  // 确定当前状态
  const getCurrentState = () => {
    if (loading) return "loading";
    if (error) return "error";
    if (filteredTemplates.length === 0) return "empty";
    return "success";
  };

  return (
    <div className="p-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <Title level={2} className="mb-2">
            工单模板管理
          </Title>
          <Text type="secondary">管理和配置工单模板，提高工单创建效率</Text>
        </div>
        <Button
          type="primary"
          icon={<Plus className="w-4 h-4" />}
          onClick={handleCreateTemplate}
        >
          创建模板
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="shadow-sm">
            <Statistic
              title="总模板数"
              value={stats.total}
              prefix={<FileText className="w-5 h-5" />}
              valueStyle={{ color: "#3b82f6" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="shadow-sm">
            <Statistic
              title="启用模板"
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#10b981" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="shadow-sm">
            <Statistic
              title="模板分类"
              value={stats.categories}
              prefix={<Settings className="w-5 h-5" />}
              valueStyle={{ color: "#f59e0b" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选和操作栏 */}
      <Card className="shadow-sm">
        <Row gutter={[16, 16]} align="middle">
          <Col flex="auto">
            <Space size="middle">
              <Input
                placeholder="搜索模板..."
                prefix={<Search className="w-4 h-4" />}
                style={{ width: 300 }}
                value={filters.keyword}
                onChange={(e) =>
                  setFilters({ ...filters, keyword: e.target.value })
                }
              />
              <Select
                placeholder="分类筛选"
                style={{ width: 150 }}
                allowClear
                value={filters.category}
                onChange={(value) =>
                  setFilters({ ...filters, category: value })
                }
              >
                <Option value="系统问题">系统问题</Option>
                <Option value="用户管理">用户管理</Option>
                <Option value="硬件问题">硬件问题</Option>
                <Option value="网络问题">网络问题</Option>
              </Select>
              <Select
                placeholder="状态筛选"
                style={{ width: 120 }}
                allowClear
                value={filters.is_active}
                onChange={(value) =>
                  setFilters({ ...filters, is_active: value })
                }
              >
                <Option value={true}>启用</Option>
                <Option value={false}>禁用</Option>
              </Select>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={loadTemplates}
              >
                刷新
              </Button>
              <Button icon={<Download className="w-4 h-4" />}>导出</Button>
              <Button icon={<Upload className="w-4 h-4" />}>导入</Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 模板列表 */}
      <LoadingEmptyError
        state={getCurrentState()}
        loadingText="正在加载模板数据..."
        empty={{
          type: "workflow",
          title: "暂无模板",
          description: "当前没有找到匹配的工单模板",
          actions: [
            {
              text: "创建模板",
              icon: <Plus className="w-4 h-4" />,
              onClick: handleCreateTemplate,
              type: "primary",
            },
            {
              text: "刷新数据",
              icon: <RefreshCw className="w-4 h-4" />,
              onClick: loadTemplates,
            },
          ],
        }}
        error={{
          title: "加载失败",
          description: error || "无法获取模板数据，请稍后重试",
          onRetry: loadTemplates,
        }}
      >
        <Card className="shadow-sm">
          <Table
            columns={columns}
            dataSource={filteredTemplates}
            rowKey="id"
            pagination={{
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) =>
                `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            }}
            scroll={{ x: 1200 }}
          />
        </Card>
      </LoadingEmptyError>

      {/* 创建/编辑模板模态框 */}
      <Modal
        title={editingTemplate ? "编辑模板" : "创建模板"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleFormSubmit}
          initialValues={{
            priority: "medium",
            impact: "medium",
            urgency: "medium",
            is_active: true,
            sla_response_time: 60,
            sla_resolution_time: 240,
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="模板名称"
                rules={[{ required: true, message: "请输入模板名称" }]}
              >
                <Input placeholder="请输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="category"
                label="分类"
                rules={[{ required: true, message: "请选择分类" }]}
              >
                <Select placeholder="选择分类">
                  <Option value="系统问题">系统问题</Option>
                  <Option value="用户管理">用户管理</Option>
                  <Option value="硬件问题">硬件问题</Option>
                  <Option value="网络问题">网络问题</Option>
                  <Option value="软件问题">软件问题</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ required: true, message: "请输入模板描述" }]}
          >
            <TextArea rows={3} placeholder="请输入模板描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="默认优先级"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="选择优先级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="assignee_group" label="默认处理组">
                <Select placeholder="选择处理组" allowClear>
                  <Option value="一线支持">一线支持</Option>
                  <Option value="系统维护组">系统维护组</Option>
                  <Option value="用户管理组">用户管理组</Option>
                  <Option value="网络维护组">网络维护组</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="is_active" label="启用状态" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingTemplate ? "更新" : "创建"}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TicketTemplatePage;

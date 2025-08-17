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
  message,
  Space,
  Tag,
  Typography,
  Row,
  Col,
  Divider,
  Tabs,
  Switch,
  InputNumber,
  Upload,
  Tooltip,
  Popconfirm,
  Badge,
  Alert,
  Steps,
  Descriptions,
  List,
  Avatar,
  Progress,
  Statistic,
  Collapse,
  Radio,
  Checkbox,
  DatePicker,
  TimePicker,
  Rate,
  Slider,
  Transfer,
  TreeSelect,
  Cascader,
  Mentions,
  AutoComplete,
  Divider as AntdDivider,
} from "antd";
import {
  Plus,
  Edit,
  Delete,
  Copy,
  Eye,
  Download,
  Upload as UploadIcon,
  Settings,
  Clock,
  AlertTriangle,
  CheckCircle,
  FileText,
  Workflow,
  Shield,
  Target,
  Zap,
  BookOpen,
  Users,
  Database,
  BarChart3,
  Filter,
  Search,
  RefreshCw,
  Star,
  Heart,
  ThumbsUp,
  ThumbsDown,
  MessageSquare,
  Share2,
  Lock,
  Unlock,
  EyeOff,
  EyeOn,
  MoreHorizontal,
  ArrowRight,
  ArrowLeft,
  RotateCcw,
  Save,
  Play,
  Pause,
  Stop,
  SkipBack,
  SkipForward,
  Repeat,
  Shuffle,
  Volume2,
  VolumeX,
  Mic,
  MicOff,
  Video,
  VideoOff,
  Phone,
  PhoneOff,
  Mail,
  Send,
  Trash2,
  Archive,
  Tag as TagIcon,
  Hash,
  AtSign,
  HashIcon,
  HashTag,
  HashTagIcon,
  HashTagIcon2,
  HashTagIcon3,
  HashTagIcon4,
  HashTagIcon5,
  HashTagIcon6,
  HashTagIcon7,
  HashTagIcon8,
  HashTagIcon9,
  HashTagIcon10,
} from "lucide-react";
// AppLayout is handled by layout.tsx

const { Option } = Select;
const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;
const { Step } = Steps;
const { Panel } = Collapse;
const { RangePicker } = DatePicker;
const { TextArea } = Input;

// 工单模板类型定义
interface TicketTemplate {
  id: number;
  name: string;
  description: string;
  type: string;
  category: string;
  subcategory?: string;
  priority: string;
  estimatedTime: string;
  sla: string;
  slaType: 'hours' | 'days' | 'business_hours';
  impact: string;
  urgency: string;
  businessValue: string;
  source: string;
  assigneeGroup?: string;
  autoAssign: boolean;
  requiresApproval: boolean;
  approvalLevel: string;
  customFields: CustomField[];
  tags: string[];
  isActive: boolean;
  isPublic: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  usageCount: number;
  rating: number;
  version: string;
  icon: React.ReactNode;
  color: string;
}

interface CustomField {
  id: string;
  name: string;
  type: 'text' | 'number' | 'select' | 'multiselect' | 'date' | 'boolean' | 'file' | 'textarea';
  label: string;
  placeholder?: string;
  required: boolean;
  defaultValue?: any;
  options?: string[];
  validation?: string;
  helpText?: string;
  order: number;
}

// 模板分类数据
const templateCategories = [
  {
    key: 'incident',
    label: '事件管理',
    icon: <AlertTriangle size={16} />,
    color: 'red',
    templates: [
      {
        id: 1,
        name: "系统登录问题",
        description: "用户无法登录系统，需要技术支持",
        type: "incident",
        category: "系统访问",
        priority: "medium",
        estimatedTime: "2小时",
        sla: "4小时",
        slaType: "hours",
        impact: "individual",
        urgency: "medium",
        businessValue: "medium",
        source: "web",
        autoAssign: true,
        requiresApproval: false,
        approvalLevel: "none",
        customFields: [
          {
            id: "cf1",
            name: "error_message",
            type: "textarea",
            label: "错误信息",
            required: true,
            placeholder: "请描述具体的错误信息",
            helpText: "请提供完整的错误信息，包括错误代码和截图",
            order: 1
          },
          {
            id: "cf2",
            name: "browser_info",
            type: "text",
            label: "浏览器信息",
            required: false,
            placeholder: "Chrome 120.0.0.0",
            helpText: "使用的浏览器版本和操作系统",
            order: 2
          }
        ],
        tags: ["登录", "认证", "系统访问"],
        isActive: true,
        isPublic: true,
        createdBy: "系统管理员",
        createdAt: "2024-01-01",
        updatedAt: "2024-01-15",
        usageCount: 156,
        rating: 4.5,
        version: "1.2",
        icon: <Shield size={20} />,
        color: "red"
      },
      {
        id: 2,
        name: "打印机故障",
        description: "办公室打印机无法正常工作",
        type: "incident",
        category: "硬件设备",
        priority: "high",
        estimatedTime: "1小时",
        sla: "2小时",
        slaType: "hours",
        impact: "department",
        urgency: "high",
        businessValue: "high",
        source: "phone",
        autoAssign: true,
        requiresApproval: false,
        approvalLevel: "none",
        customFields: [
          {
            id: "cf3",
            name: "printer_model",
            type: "text",
            label: "打印机型号",
            required: true,
            placeholder: "HP LaserJet Pro M404n",
            helpText: "打印机品牌和型号",
            order: 1
          },
          {
            id: "cf4",
            name: "error_code",
            type: "text",
            label: "错误代码",
            required: false,
            placeholder: "E-01",
            helpText: "打印机显示的错误代码",
            order: 2
          }
        ],
        tags: ["打印机", "硬件", "设备故障"],
        isActive: true,
        isPublic: true,
        createdBy: "系统管理员",
        createdAt: "2024-01-01",
        updatedAt: "2024-01-10",
        usageCount: 89,
        rating: 4.2,
        version: "1.1",
        icon: <Settings size={20} />,
        color: "orange"
      }
    ]
  },
  {
    key: 'service_request',
    label: '服务请求',
    icon: <FileText size={16} />,
    color: 'blue',
    templates: [
      {
        id: 3,
        name: "软件安装请求",
        description: "需要安装新的办公软件",
        type: "service_request",
        category: "软件服务",
        priority: "low",
        estimatedTime: "30分钟",
        sla: "24小时",
        slaType: "business_hours",
        impact: "individual",
        urgency: "low",
        businessValue: "medium",
        source: "web",
        autoAssign: false,
        requiresApproval: true,
        approvalLevel: "manager",
        customFields: [
          {
            id: "cf5",
            name: "software_name",
            type: "text",
            label: "软件名称",
            required: true,
            placeholder: "Adobe Photoshop",
            helpText: "需要安装的软件名称和版本",
            order: 1
          },
          {
            id: "cf6",
            name: "license_type",
            type: "select",
            label: "许可证类型",
            required: true,
            options: ["免费版", "标准版", "专业版", "企业版"],
            helpText: "软件许可证类型",
            order: 2
          }
        ],
        tags: ["软件安装", "许可证", "办公软件"],
        isActive: true,
        isPublic: true,
        createdBy: "系统管理员",
        createdAt: "2024-01-01",
        updatedAt: "2024-01-20",
        usageCount: 234,
        rating: 4.7,
        version: "1.3",
        icon: <Zap size={20} />,
        color: "blue"
      }
    ]
  },
  {
    key: 'problem',
    label: '问题管理',
    icon: <BookOpen size={16} />,
    color: 'orange',
    templates: []
  },
  {
    key: 'change',
    label: '变更管理',
    icon: <Workflow size={16} />,
    color: 'purple',
    templates: []
  }
];

const TicketTemplatesPage = () => {
  const [templates, setTemplates] = useState<TicketTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(null);
  const [selectedCategory, setSelectedCategory] = useState('incident');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  useEffect(() => {
    loadTemplates();
  }, []);

  const loadTemplates = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      const allTemplates = templateCategories.flatMap(cat => cat.templates);
      setTemplates(allTemplates);
    } catch (error) {
      message.error('加载模板失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTemplate = () => {
    setEditingTemplate(null);
    setModalVisible(true);
  };

  const handleEditTemplate = (template: TicketTemplate) => {
    setEditingTemplate(template);
    setModalVisible(true);
  };

  const handleDeleteTemplate = async (id: number) => {
    try {
      setTemplates(prev => prev.filter(t => t.id !== id));
      message.success('模板删除成功');
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleCopyTemplate = (template: TicketTemplate) => {
    const newTemplate = {
      ...template,
      id: Date.now(),
      name: `${template.name} - 副本`,
      version: "1.0"
    };
    setTemplates(prev => [...prev, newTemplate]);
    message.success('模板复制成功');
  };

  const filteredTemplates = templates.filter(template => {
    const matchesKeyword = template.name.toLowerCase().includes(searchKeyword.toLowerCase()) ||
                          template.description.toLowerCase().includes(searchKeyword.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || template.type === selectedCategory;
    const matchesStatus = filterStatus === 'all' || 
                         (filterStatus === 'active' && template.isActive) ||
                         (filterStatus === 'inactive' && !template.isActive);
    
    return matchesKeyword && matchesCategory && matchesStatus;
  });

  const renderTemplateCard = (template: TicketTemplate) => (
    <Card
      key={template.id}
      hoverable
      className="h-full"
      actions={[
        <Tooltip title="编辑模板" key="edit">
          <Button type="text" icon={<Edit size={16} />} onClick={() => handleEditTemplate(template)} />
        </Tooltip>,
        <Tooltip title="复制模板" key="copy">
          <Button type="text" icon={<Copy size={16} />} onClick={() => handleCopyTemplate(template)} />
        </Tooltip>,
        <Tooltip title="删除模板" key="delete">
          <Popconfirm
            title="确定要删除这个模板吗？"
            onConfirm={() => handleDeleteTemplate(template.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<Delete size={16} />} />
          </Popconfirm>
        </Tooltip>
      ]}
    >
      <div className="flex items-start mb-3">
        <div className={`inline-flex items-center justify-center w-12 h-12 bg-${template.color}-50 rounded-lg mr-3`}>
          <span className={`text-${template.color}-500`}>{template.icon}</span>
        </div>
        <div className="flex-1 min-w-0">
          <Title level={5} className="mb-1 truncate">{template.name}</Title>
          <Text type="secondary" className="text-sm line-clamp-2">{template.description}</Text>
        </div>
      </div>

      <div className="space-y-2 mb-4">
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">类型</Text>
          <Tag color={template.color}>{template.category}</Tag>
        </div>
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">优先级</Text>
          <Tag color={template.priority === 'high' ? 'red' : template.priority === 'medium' ? 'orange' : 'green'}>
            {template.priority}
          </Tag>
        </div>
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">SLA</Text>
          <Text className="text-xs">{template.sla}</Text>
        </div>
      </div>

      <div className="flex items-center justify-between text-xs text-gray-500 mb-3">
        <span>使用次数: {template.usageCount}</span>
        <span>评分: {template.rating}/5</span>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Switch checked={template.isActive} size="small" />
          <Text className="text-xs">{template.isActive ? '启用' : '禁用'}</Text>
        </div>
        <div className="flex items-center space-x-2">
          <Switch checked={template.isPublic} size="small" />
          <Text className="text-xs">{template.isPublic ? '公开' : '私有'}</Text>
        </div>
      </div>
    </Card>
  );

  const renderTemplateList = (template: TicketTemplate) => (
    <Card key={template.id} className="mb-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div className={`inline-flex items-center justify-center w-10 h-10 bg-${template.color}-50 rounded-lg`}>
            <span className={`text-${template.color}-500`}>{template.icon}</span>
          </div>
          <div>
            <Title level={5} className="mb-1">{template.name}</Title>
            <Text type="secondary" className="text-sm">{template.description}</Text>
          </div>
        </div>
        
        <div className="flex items-center space-x-4">
          <div className="text-center">
            <Text className="text-xs text-gray-500">使用次数</Text>
            <div className="font-semibold">{template.usageCount}</div>
          </div>
          <div className="text-center">
            <Text className="text-xs text-gray-500">评分</Text>
            <div className="flex items-center">
              <Rate disabled defaultValue={template.rating} size="small" />
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Tag color={template.color}>{template.category}</Tag>
            <Tag color={template.priority === 'high' ? 'red' : template.priority === 'medium' ? 'orange' : 'green'}>
              {template.priority}
            </Tag>
          </div>
          <Space>
            <Button type="text" icon={<Edit size={16} />} onClick={() => handleEditTemplate(template)} />
            <Button type="text" icon={<Copy size={16} />} onClick={() => handleCopyTemplate(template)} />
            <Popconfirm
              title="确定要删除这个模板吗？"
              onConfirm={() => handleDeleteTemplate(template.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button type="text" danger icon={<Delete size={16} />} />
            </Popconfirm>
          </Space>
        </div>
      </div>
    </Card>
  );

  return (
    <>
      {/* 页面头部操作 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">工单模板管理</h1>
          <p className="text-gray-600 mt-1">管理和配置工单模板，提高工单创建效率</p>
        </div>
        <Space>
          <Button icon={<RefreshCw size={16} />} onClick={loadTemplates}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={handleCreateTemplate}
          >
            新建模板
          </Button>
        </Space>
      </div>
      {/* 统计信息 */}
      <Row gutter={16} className="mb-6">
        <Col span={6}>
          <Card>
            <Statistic
              title="总模板数"
              value={templates.length}
              prefix={<FileText size={16} style={{ color: "#3b82f6" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="活跃模板"
              value={templates.filter(t => t.isActive).length}
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircle size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总使用次数"
              value={templates.reduce((sum, t) => sum + t.usageCount, 0)}
              prefix={<BarChart3 size={16} style={{ color: "#faad14" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="平均评分"
              value={templates.length > 0 ? 
                (templates.reduce((sum, t) => sum + t.rating, 0) / templates.length).toFixed(1) : 0}
              prefix={<Star size={16} style={{ color: "#ff4d4f" }} />}
              suffix="/5"
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选和搜索 */}
      <Card className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Title level={5} className="mb-0">筛选条件</Title>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)}>
              <Radio.Button value="grid">网格视图</Radio.Button>
              <Radio.Button value="list">列表视图</Radio.Button>
            </Radio.Group>
          </Space>
        </div>

        <Row gutter={16} align="middle">
          <Col span={8}>
            <Input.Search
              placeholder="搜索模板名称或描述"
              allowClear
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              prefix={<Search size={16} />}
            />
          </Col>
          <Col span={6}>
            <Select
              value={selectedCategory}
              onChange={setSelectedCategory}
              style={{ width: "100%" }}
              placeholder="选择分类"
            >
              <Option value="all">全部分类</Option>
              {templateCategories.map(cat => (
                <Option key={cat.key} value={cat.key}>
                  <div className="flex items-center">
                    <span className={`text-${cat.color}-500 mr-2`}>{cat.icon}</span>
                    {cat.label}
                  </div>
                </Option>
              ))}
            </Select>
          </Col>
          <Col span={6}>
            <Select
              value={filterStatus}
              onChange={setFilterStatus}
              style={{ width: "100%" }}
              placeholder="状态筛选"
            >
              <Option value="all">全部状态</Option>
              <Option value="active">启用</Option>
              <Option value="inactive">禁用</Option>
            </Select>
          </Col>
          <Col span={4}>
            <Button type="primary" onClick={loadTemplates} block>
              应用筛选
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 模板列表 */}
      {loading ? (
        <Card>
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-50 rounded-full mb-4">
              <RefreshCw size={32} className="text-blue-500 animate-spin" />
            </div>
            <Text className="text-gray-500">正在加载模板...</Text>
          </div>
        </Card>
      ) : filteredTemplates.length === 0 ? (
        <Card>
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-24 h-24 bg-gray-50 rounded-full mb-4">
              <FileText size={48} className="text-gray-400" />
            </div>
            <Title level={4} className="text-gray-600 mb-2">暂无模板</Title>
            <Text type="secondary" className="text-base mb-6">
              当前没有找到匹配的工单模板
            </Text>
            <Button type="primary" icon={<Plus size={16} />} onClick={handleCreateTemplate}>
              创建第一个模板
            </Button>
          </div>
        </Card>
      ) : (
        <div>
          {viewMode === 'grid' ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {filteredTemplates.map(renderTemplateCard)}
            </div>
          ) : (
            <div>
              {filteredTemplates.map(renderTemplateList)}
            </div>
          )}
        </div>
      )}

      {/* 创建/编辑模板模态框 */}
      <Modal
        title={editingTemplate ? "编辑工单模板" : "新建工单模板"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={1000}
      >
        <Form layout="vertical" initialValues={editingTemplate || {}}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="模板名称"
                name="name"
                rules={[{ required: true, message: "请输入模板名称" }]}
              >
                <Input placeholder="请输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="模板类型"
                name="type"
                rules={[{ required: true, message: "请选择模板类型" }]}
              >
                <Select placeholder="请选择模板类型">
                  <Option value="incident">事件</Option>
                  <Option value="service_request">服务请求</Option>
                  <Option value="problem">问题</Option>
                  <Option value="change">变更</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="分类"
                name="category"
                rules={[{ required: true, message: "请输入分类" }]}
              >
                <Input placeholder="请输入分类" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="子分类"
                name="subcategory"
              >
                <Input placeholder="请输入子分类（可选）" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="描述"
            name="description"
            rules={[{ required: true, message: "请输入模板描述" }]}
          >
            <TextArea rows={3} placeholder="请详细描述模板用途和适用场景" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="优先级"
                name="priority"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="请选择优先级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="urgent">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="预计处理时间"
                name="estimatedTime"
                rules={[{ required: true, message: "请输入预计处理时间" }]}
              >
                <Input placeholder="如：2小时" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="SLA"
                name="sla"
                rules={[{ required: true, message: "请输入SLA" }]}
              >
                <Input placeholder="如：4小时" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="影响范围"
                name="impact"
                rules={[{ required: true, message: "请选择影响范围" }]}
              >
                <Select placeholder="请选择影响范围">
                  <Option value="individual">个人</Option>
                  <Option value="department">部门</Option>
                  <Option value="organization">组织</Option>
                  <Option value="customer">客户</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="紧急程度"
                name="urgency"
                rules={[{ required: true, message: "请选择紧急程度" }]}
              >
                <Select placeholder="请选择紧急程度">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">严重</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="业务价值"
                name="businessValue"
                rules={[{ required: true, message: "请选择业务价值" }]}
              >
                <Select placeholder="请选择业务价值">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">关键</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="来源"
                name="source"
                rules={[{ required: true, message: "请选择来源" }]}
              >
                <Select placeholder="请选择来源">
                  <Option value="web">Web门户</Option>
                  <Option value="email">邮件</Option>
                  <Option value="phone">电话</Option>
                  <Option value="chat">在线聊天</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="标签"
                name="tags"
              >
                <Select
                  mode="tags"
                  placeholder="请输入标签，按回车确认"
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Divider>高级设置</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="自动分配"
                name="autoAssign"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="需要审批"
                name="requiresApproval"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="模板状态"
                name="isActive"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="公开模板"
                name="isPublic"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="SLA类型"
                name="slaType"
                rules={[{ required: true, message: "请选择SLA类型" }]}
              >
                <Select placeholder="请选择SLA类型">
                  <Option value="hours">小时</Option>
                  <Option value="days">天</Option>
                  <Option value="business_hours">工作时间</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="审批级别"
                name="approvalLevel"
              >
                <Select placeholder="请选择审批级别">
                  <Option value="none">无需审批</Option>
                  <Option value="manager">经理审批</Option>
                  <Option value="director">总监审批</Option>
                  <Option value="executive">高管审批</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingTemplate ? "更新模板" : "创建模板"}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default TicketTemplatesPage;

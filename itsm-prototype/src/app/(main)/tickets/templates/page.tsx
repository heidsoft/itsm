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
  MoreHorizontal,
  ArrowRight,
  ArrowLeft,
  RotateCcw,
  Save,
  Play,
  Pause,
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

// Ticket template type definition
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

// Template category data
const templateCategories = [
  {
    key: 'incident',
    label: 'Incident Management',
    icon: <AlertTriangle size={16} />,
    color: 'red',
    templates: [
      {
        id: 1,
        name: "System Login Issue",
        description: "User cannot login to system, technical support needed",
        type: "incident",
        category: "System Access",
        priority: "high",
        estimatedTime: "2 hours",
        sla: "4 hours",
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
            label: "Error Message",
            required: true,
            placeholder: "Please describe the specific error message",
            helpText: "Please provide complete error information, including error codes and screenshots",
            order: 1
          },
          {
            id: "cf2",
            name: "browser_info",
            type: "text",
            label: "Browser Information",
            required: false,
            placeholder: "Chrome 120.0.0.0",
            helpText: "Browser version and operating system used",
            order: 2
          }
        ],
        tags: ["login", "authentication", "system access"],
        isActive: true,
        isPublic: true,
        createdBy: "System Administrator",
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
        name: "Printer Malfunction",
        description: "Office printer not working properly",
        type: "incident",
        category: "Hardware Equipment",
        priority: "medium",
        estimatedTime: "1 hour",
        sla: "2 hours",
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
            label: "Printer Model",
            required: true,
            placeholder: "HP LaserJet Pro M404n",
            helpText: "Printer brand and model",
            order: 1
          },
          {
            id: "cf4",
            name: "error_code",
            type: "text",
            label: "Error Code",
            required: false,
            placeholder: "E-01",
            helpText: "Error code displayed by printer",
            order: 2
          }
        ],
        tags: ["printer", "hardware", "equipment failure"],
        isActive: true,
        isPublic: true,
        createdBy: "System Administrator",
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
    label: 'Service Request',
    icon: <FileText size={16} />,
    color: 'blue',
    templates: [
      {
        id: 3,
        name: "Software Installation Request",
        description: "Need to install new office software",
        type: "service_request",
        category: "Software Service",
        priority: "low",
        estimatedTime: "30 minutes",
        sla: "24 hours",
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
            label: "Software Name",
            required: true,
            placeholder: "Adobe Photoshop",
            helpText: "Name and version of software to be installed",
            order: 1
          },
          {
            id: "cf6",
            name: "license_type",
            type: "select",
            label: "License Type",
            required: true,
            options: ["Free Version", "Standard Version", "Professional Version", "Enterprise Version"],
            helpText: "Software license type",
            order: 2
          }
        ],
        tags: ["software installation", "license", "office software"],
        isActive: true,
        isPublic: true,
        createdBy: "System Administrator",
        createdAt: "2024-01-01",
        updatedAt: "2024-01-20",
        usageCount: 234,
        rating: 4.6,
        version: "1.3",
        icon: <Zap size={20} />,
        color: "blue"
      }
    ]
  },
  {
    key: 'problem',
    label: 'Problem Management',
    icon: <BookOpen size={16} />,
    color: 'orange',
    templates: []
  },
  {
    key: 'change',
    label: 'Change Management',
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
      // Mock API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      const allTemplates = templateCategories.flatMap(cat => cat.templates);
      setTemplates(allTemplates);
    } catch (error) {
      message.error('Failed to load templates');
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
      message.success('Template deleted successfully');
    } catch (error) {
      message.error('Delete failed');
    }
  };

  const handleCopyTemplate = (template: TicketTemplate) => {
    const newTemplate = {
      ...template,
      id: Date.now(),
      name: `${template.name} - Copy`,
      version: "1.0"
    };
    setTemplates(prev => [...prev, newTemplate]);
    message.success('Template copied successfully');
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
        <Tooltip title="Edit template" key="edit">
          <Button type="text" icon={<Edit size={16} />} onClick={() => handleEditTemplate(template)} />
        </Tooltip>,
        <Tooltip title="Copy template" key="copy">
          <Button type="text" icon={<Copy size={16} />} onClick={() => handleCopyTemplate(template)} />
        </Tooltip>,
        <Tooltip title="Delete template" key="delete">
          <Popconfirm
            title="Are you sure you want to delete this template?"
            onConfirm={() => handleDeleteTemplate(template.id)}
            okText="Confirm"
            cancelText="Cancel"
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
          <Text type="secondary" className="text-xs">Type</Text>
          <Tag color={template.color}>{template.category}</Tag>
        </div>
        <div className="flex items-center justify-between">
          <Text type="secondary" className="text-xs">Priority</Text>
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
        <span>Usage Count: {template.usageCount}</span>
        <span>Rating: {template.rating}/5</span>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Switch checked={template.isActive} size="small" />
          <Text className="text-xs">{template.isActive ? 'Active' : 'Inactive'}</Text>
        </div>
        <div className="flex items-center space-x-2">
          <Switch checked={template.isPublic} size="small" />
          <Text className="text-xs">{template.isPublic ? 'Public' : 'Private'}</Text>
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
            <Text className="text-xs text-gray-500">Usage Count</Text>
            <div className="font-semibold">{template.usageCount}</div>
          </div>
          <div className="text-center">
            <Text className="text-xs text-gray-500">Rating</Text>
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
              title="Are you sure you want to delete this template?"
              onConfirm={() => handleDeleteTemplate(template.id)}
              okText="Confirm"
              cancelText="Cancel"
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
      {/* Page header actions */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Ticket Template Management</h1>
          <p className="text-gray-600 mt-1">Manage and configure ticket templates to improve ticket creation efficiency</p>
        </div>
        <Space>
          <Button icon={<RefreshCw size={16} />} onClick={loadTemplates}>
            Refresh
          </Button>
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={handleCreateTemplate}
          >
            New Template
          </Button>
        </Space>
      </div>
      {/* Statistics */}
      <Row gutter={16} className="mb-6">
        <Col span={6}>
          <Card>
            <Statistic
              title="Total Templates"
              value={templates.length}
              prefix={<FileText size={16} style={{ color: "#3b82f6" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Active Templates"
              value={templates.filter(t => t.isActive).length}
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircle size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Total Usage"
              value={templates.reduce((sum, t) => sum + t.usageCount, 0)}
              prefix={<BarChart3 size={16} style={{ color: "#faad14" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Average Rating"
              value={templates.length > 0 ? 
                (templates.reduce((sum, t) => sum + t.rating, 0) / templates.length).toFixed(1) : 0}
              prefix={<Star size={16} style={{ color: "#ff4d4f" }} />}
              suffix="/5"
            />
          </Card>
        </Col>
      </Row>

      {/* Filter and search */}
      <Card title="Template Management" className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Title level={5} className="mb-0">Filter Conditions</Title>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)}>
              <Radio.Button value="grid">Grid View</Radio.Button>
              <Radio.Button value="list">List View</Radio.Button>
            </Radio.Group>
          </Space>
        </div>

        <Row gutter={16} align="middle">
          <Col span={8}>
            <Input.Search
              placeholder="Search templates..."
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
              placeholder="Select category"
            >
              <Option value="all">All Categories</Option>
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
              placeholder="Status filter"
            >
              <Option value="all">All Status</Option>
              <Option value="active">Active</Option>
              <Option value="inactive">Inactive</Option>
            </Select>
          </Col>
          <Col span={4}>
            <Button type="primary" onClick={loadTemplates} block>
              Apply Filter
            </Button>
          </Col>
        </Row>
      </Card>

      {/* Template list */}
      {loading ? (
        <Card>
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-50 rounded-full mb-4">
              <RefreshCw size={32} className="text-blue-500 animate-spin" />
            </div>
            <Text className="text-gray-500">Loading templates...</Text>
          </div>
        </Card>
      ) : filteredTemplates.length === 0 ? (
        <Card>
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-24 h-24 bg-gray-50 rounded-full mb-4">
              <FileText size={48} className="text-gray-400" />
            </div>
            <Title level={4} className="text-gray-600 mb-2">No Templates</Title>
            <p className="text-gray-500 mb-4">
              No matching ticket templates found
            </p>
            <Button type="primary" onClick={() => setIsModalVisible(true)}>
              Create First Template
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

      {/* Create/Edit template modal */}
      <Modal
        title={editingTemplate ? "Edit Ticket Template" : "New Ticket Template"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={1000}
      >
        <Form layout="vertical" initialValues={editingTemplate || {}}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Template Name"
                name="name"
                rules={[{ required: true, message: "Please enter template name" }]}
              >
                <Input placeholder="Please enter template name" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Template Type"
                name="type"
                rules={[{ required: true, message: "Please select template type" }]}
              >
                <Select placeholder="Please select template type">
                  <Option value="incident">Incident</Option>
                  <Option value="service_request">Service Request</Option>
                  <Option value="problem">Problem</Option>
                  <Option value="change">Change</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Category"
                name="category"
                rules={[{ required: true, message: "Please select category" }]}
              >
                <Input placeholder="Please enter category" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Subcategory"
                name="subcategory"
              >
                <Input placeholder="Please enter subcategory (optional)" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="Description"
            name="description"
            rules={[{ required: true, message: "Please enter template description" }]}
          >
            <TextArea rows={3} placeholder="Please describe template purpose and applicable scenarios in detail" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="Priority"
                name="priority"
                rules={[{ required: true, message: "Please select priority" }]}
              >
                <Select placeholder="Please select priority">
                  <Option value="low">Low</Option>
                  <Option value="medium">Medium</Option>
                  <Option value="high">High</Option>
                  <Option value="urgent">Urgent</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Estimated Processing Time"
                name="estimatedTime"
                rules={[{ required: true, message: "Please enter estimated processing time" }]}
              >
                <Input placeholder="e.g.: 2 hours" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="SLA"
                name="sla"
                rules={[{ required: true, message: "Please enter SLA" }]}
              >
                <Input placeholder="e.g.: 4 hours" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="Impact Scope"
                name="impact"
                rules={[{ required: true, message: "Please select impact scope" }]}
              >
                <Select placeholder="Please select impact scope">
                  <Option value="individual">Individual</Option>
                  <Option value="department">Department</Option>
                  <Option value="organization">Organization</Option>
                  <Option value="customer">Customer</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Urgency Level"
                name="urgency"
                rules={[{ required: true, message: "Please select urgency level" }]}
              >
                <Select placeholder="Please select urgency level">
                  <Option value="low">Low</Option>
                  <Option value="medium">Medium</Option>
                  <Option value="high">High</Option>
                  <Option value="critical">Critical</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Business Value"
                name="businessValue"
                rules={[{ required: true, message: "Please select business value" }]}
              >
                <Select placeholder="Please select business value">
                  <Option value="low">Low</Option>
                  <Option value="medium">Medium</Option>
                  <Option value="high">High</Option>
                  <Option value="critical">Critical</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Source"
                name="source"
                rules={[{ required: true, message: "Please select source" }]}
              >
                <Select placeholder="Please select source">
                  <Option value="web">Web Portal</Option>
                  <Option value="email">Email</Option>
                  <Option value="phone">Phone</Option>
                  <Option value="chat">Online Chat</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Tags"
                name="tags"
              >
                <Select
                  mode="tags"
                  placeholder="Add tags..."
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Divider>Advanced Settings</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="Auto Assignment"
                name="autoAssign"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Requires Approval"
                name="requiresApproval"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Template Status"
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
                label="Public Template"
                name="isPublic"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="SLA Type"
                name="slaType"
                rules={[{ required: true, message: "Please select SLA type" }]}
              >
                <Select placeholder="Please select SLA type">
                  <Option value="hours">Hours</Option>
                  <Option value="days">Days</Option>
                  <Option value="business_hours">Business Hours</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Approval Level"
                name="approvalLevel"
              >
                <Select placeholder="Please select approval level">
                  <Option value="none">No Approval Required</Option>
                  <Option value="manager">Manager Approval</Option>
                  <Option value="director">Director Approval</Option>
                  <Option value="executive">Executive Approval</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingTemplate ? "Update Template" : "Create Template"}
              </Button>
              <Button onClick={() => setIsModalVisible(false)}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default TicketTemplatesPage;

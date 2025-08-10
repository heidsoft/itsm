"use client";

import { Plus, CheckCircle, Users, Search, Settings, Trash2, XCircle, Edit, Eye, Lock, Unlock, Key, Tag, Shield } from 'lucide-react';

import React, { useState } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Space,
  Typography,
  Modal,
  Form,
  Switch,
  Checkbox,
  Row,
  Col,
  Statistic,
  Badge,
  Tooltip,
  Popconfirm,
  message,
  Divider,
  Alert,
  Tabs,
} from "antd";
const { Title, Text } = Typography;
const { Option } = Select;

// 权限模块定义
const PERMISSION_MODULES = {
  DASHBOARD: "dashboard",
  TICKETS: "tickets",
  INCIDENTS: "incidents",
  PROBLEMS: "problems",
  CHANGES: "changes",
  SERVICE_CATALOG: "service_catalog",
  KNOWLEDGE_BASE: "knowledge_base",
  REPORTS: "reports",
  ADMIN: "admin",
  USERS: "users",
  ROLES: "roles",
  WORKFLOWS: "workflows",
  SYSTEM_CONFIG: "system_config",
} as const;

// 权限操作类型
const PERMISSION_ACTIONS = {
  VIEW: "view",
  CREATE: "create",
  EDIT: "edit",
  DELETE: "delete",
  APPROVE: "approve",
  ASSIGN: "assign",
  EXPORT: "export",
} as const;

// 模拟角色数据
const mockRoles = [
  {
    id: 1,
    name: "系统管理员",
    description: "拥有系统所有权限的超级管理员",
    userCount: 2,
    isSystem: true,
    isActive: true,
    createdAt: "2024-01-01 00:00",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.INCIDENTS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.PROBLEMS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.CHANGES]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.SERVICE_CATALOG]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.REPORTS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.ADMIN]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.USERS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.ROLES]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.WORKFLOWS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.SYSTEM_CONFIG]: Object.values(PERMISSION_ACTIONS),
    },
  },
  {
    id: 2,
    name: "IT支持工程师",
    description: "负责日常IT支持和工单处理",
    userCount: 15,
    isSystem: false,
    isActive: true,
    createdAt: "2024-01-15 10:30",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
        PERMISSION_ACTIONS.ASSIGN,
      ],
      [PERMISSION_MODULES.INCIDENTS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
      ],
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
      ],
      [PERMISSION_MODULES.REPORTS]: [PERMISSION_ACTIONS.VIEW],
    },
  },
  {
    id: 3,
    name: "服务台经理",
    description: "管理服务台团队和审批流程",
    userCount: 5,
    isSystem: false,
    isActive: true,
    createdAt: "2024-01-20 14:20",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.INCIDENTS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.PROBLEMS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.APPROVE,
      ],
      [PERMISSION_MODULES.CHANGES]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.APPROVE,
      ],
      [PERMISSION_MODULES.SERVICE_CATALOG]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.EDIT,
      ],
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.REPORTS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.EXPORT,
      ],
      [PERMISSION_MODULES.USERS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.EDIT,
      ],
    },
  },
  {
    id: 4,
    name: "普通用户",
    description: "只能提交和查看自己的请求",
    userCount: 150,
    isSystem: true,
    isActive: true,
    createdAt: "2024-01-01 00:00",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
      ],
      [PERMISSION_MODULES.SERVICE_CATALOG]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: [PERMISSION_ACTIONS.VIEW],
    },
  },
];

// 权限模块配置
const MODULE_CONFIG = {
  [PERMISSION_MODULES.DASHBOARD]: {
    label: "仪表盘",
    icon: "📊",
    category: "核心功能",
  },
  [PERMISSION_MODULES.TICKETS]: {
    label: "工单管理",
    icon: "🎫",
    category: "核心功能",
  },
  [PERMISSION_MODULES.INCIDENTS]: {
    label: "事件管理",
    icon: "🚨",
    category: "核心功能",
  },
  [PERMISSION_MODULES.PROBLEMS]: {
    label: "问题管理",
    icon: "🔧",
    category: "核心功能",
  },
  [PERMISSION_MODULES.CHANGES]: {
    label: "变更管理",
    icon: "🔄",
    category: "核心功能",
  },
  [PERMISSION_MODULES.SERVICE_CATALOG]: {
    label: "服务目录",
    icon: "📋",
    category: "服务管理",
  },
  [PERMISSION_MODULES.KNOWLEDGE_BASE]: {
    label: "知识库",
    icon: "📚",
    category: "服务管理",
  },
  [PERMISSION_MODULES.REPORTS]: {
    label: "报告分析",
    icon: "📈",
    category: "分析工具",
  },
  [PERMISSION_MODULES.ADMIN]: {
    label: "系统管理",
    icon: "⚙️",
    category: "系统管理",
  },
  [PERMISSION_MODULES.USERS]: {
    label: "用户管理",
    icon: "👥",
    category: "系统管理",
  },
  [PERMISSION_MODULES.ROLES]: {
    label: "角色管理",
    icon: "🛡️",
    category: "系统管理",
  },
  [PERMISSION_MODULES.WORKFLOWS]: {
    label: "工作流",
    icon: "🔀",
    category: "系统管理",
  },
  [PERMISSION_MODULES.SYSTEM_CONFIG]: {
    label: "系统配置",
    icon: "🔧",
    category: "系统管理",
  },
};

// 权限操作配置
const ACTION_CONFIG = {
  [PERMISSION_ACTIONS.VIEW]: { label: "查看", color: "blue" },
  [PERMISSION_ACTIONS.CREATE]: { label: "创建", color: "green" },
  [PERMISSION_ACTIONS.EDIT]: { label: "编辑", color: "orange" },
  [PERMISSION_ACTIONS.DELETE]: { label: "删除", color: "red" },
  [PERMISSION_ACTIONS.APPROVE]: { label: "审批", color: "purple" },
  [PERMISSION_ACTIONS.ASSIGN]: { label: "分配", color: "cyan" },
  [PERMISSION_ACTIONS.EXPORT]: { label: "导出", color: "geekblue" },
};

const RoleManagement = () => {
  // 状态管理
  const [roles, setRoles] = useState(mockRoles);
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [showModal, setShowModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState<any>(null);
  const [showPermissionModal, setShowPermissionModal] = useState(false);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 统计数据
  const stats = {
    total: roles.length,
    active: roles.filter((r) => r.isActive).length,
    system: roles.filter((r) => r.isSystem).length,
    totalUsers: roles.reduce((sum, role) => sum + role.userCount, 0),
  };

  // 过滤角色
  const filteredRoles = roles.filter((role) => {
    const matchesSearch = role.name
      .toLowerCase()
      .includes(searchTerm.toLowerCase());
    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "active" && role.isActive) ||
      (statusFilter === "inactive" && !role.isActive);
    return matchesSearch && matchesStatus;
  });

  // 处理角色状态切换
  const handleToggleStatus = (roleId: number) => {
    setRoles(
      roles.map((role) =>
        role.id === roleId ? { ...role, isActive: !role.isActive } : role
      )
    );
    message.success("角色状态已更新");
  };

  // 处理删除角色
  const handleDeleteRole = (roleId: number) => {
    setRoles(roles.filter((role) => role.id !== roleId));
    message.success("角色已删除");
  };

  // 处理编辑角色
  const handleEditRole = (role: any) => {
    setSelectedRole(role);
    form.setFieldsValue(role);
    setShowModal(true);
  };

  // 查看权限详情
  const handleViewPermissions = (role: any) => {
    setSelectedRole(role);
    setShowPermissionModal(true);
  };

  // 保存角色
  const handleSaveRole = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      await new Promise((resolve) => setTimeout(resolve, 1000));

      if (selectedRole) {
        // 编辑角色
        setRoles(
          roles.map((role) =>
            role.id === selectedRole.id
              ? { ...role, ...values, permissions: selectedRole.permissions }
              : role
          )
        );
        message.success("角色更新成功");
      } else {
        // 新建角色
        const newRole = {
          id: Math.max(...roles.map((r) => r.id)) + 1,
          ...values,
          userCount: 0,
          isSystem: false,
          createdAt: new Date().toLocaleString("zh-CN"),
          permissions: {},
        };
        setRoles([...roles, newRole]);
        message.success("角色创建成功");
      }

      setShowModal(false);
      setSelectedRole(null);
      form.resetFields();
    } catch (error) {
      message.error("保存失败，请检查必填项");
    } finally {
      setLoading(false);
    }
  };

  // 权限组织化显示
  const renderPermissionsByCategory = (permissions: any) => {
    const categories = {} as any;

    Object.entries(permissions).forEach(([moduleKey, actions]) => {
      const moduleConfig =
        MODULE_CONFIG[moduleKey as keyof typeof MODULE_CONFIG];
      if (moduleConfig && Array.isArray(actions) && actions.length > 0) {
        const category = moduleConfig.category;
        if (!categories[category]) {
          categories[category] = [];
        }
        categories[category].push({
          moduleKey,
          moduleConfig,
          actions,
        });
      }
    });

    return Object.entries(categories).map(
      ([category, modules]: [string, any]) => ({
        key: category,
        label: (
          <span>
            <Settings className="w-4 h-4 mr-1" />
            {category}
          </span>
        ),
        children: (
          <Row gutter={[16, 16]}>
            {modules.map((module: any) => (
              <Col xs={24} md={12} lg={8} key={module.moduleKey}>
                <Card size="small" className="h-full">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-lg">{module.moduleConfig.icon}</span>
                    <Text strong>{module.moduleConfig.label}</Text>
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {module.actions.map((action: string) => (
                      <Tag
                        key={action}
                        color={
                          ACTION_CONFIG[action as keyof typeof ACTION_CONFIG]
                            ?.color
                        }
                        size="small"
                      >
                        {ACTION_CONFIG[action as keyof typeof ACTION_CONFIG]
                          ?.label || action}
                      </Tag>
                    ))}
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        ),
      })
    );
  };

  // 表格列定义
  const columns = [
    {
      title: "角色信息",
      dataIndex: "name",
      key: "name",
      render: (_: any, record: any) => (
        <div>
          <div className="flex items-center gap-2">
            <Text strong>{record.name}</Text>
            {record.isSystem && (
              <Tooltip title="系统角色">
                <Crown className="w-4 h-4 text-amber-500" />
              </Tooltip>
            )}
          </div>
          <Text type="secondary" className="text-sm">
            {record.description}
          </Text>
        </div>
      ),
    },
    {
      title: "用户数量",
      dataIndex: "userCount",
      key: "userCount",
      align: "center" as const,
      render: (count: number) => (
        <div className="flex items-center justify-center gap-1">
          <Users className="w-4 h-4 text-gray-400" />
          <Badge
            count={count}
            showZero
            color={count > 0 ? "#1890ff" : "#d9d9d9"}
          />
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "isActive",
      key: "isActive",
      align: "center" as const,
      render: (isActive: boolean) => (
        <Tag
          icon={
            isActive ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <XCircle className="w-3 h-3" />
            )
          }
          color={isActive ? "success" : "error"}
        >
          {isActive ? "启用" : "禁用"}
        </Tag>
      ),
    },
    {
      title: "类型",
      dataIndex: "isSystem",
      key: "isSystem",
      align: "center" as const,
      render: (isSystem: boolean) => (
        <Tag
          icon={
            isSystem ? (
              <Lock className="w-3 h-3" />
            ) : (
              <Unlock className="w-3 h-3" />
            )
          }
          color={isSystem ? "purple" : "blue"}
        >
          {isSystem ? "系统角色" : "自定义角色"}
        </Tag>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      align: "center" as const,
    },
    {
      title: "操作",
      key: "actions",
      align: "center" as const,
      render: (_: any, record: any) => (
        <Space>
          <Tooltip title="查看权限">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => handleViewPermissions(record)}
            />
          </Tooltip>
          {!record.isSystem && (
            <>
              <Tooltip title="编辑角色">
                <Button
                  type="text"
                  icon={<Edit className="w-4 h-4" />}
                  onClick={() => handleEditRole(record)}
                />
              </Tooltip>
              <Tooltip title={record.isActive ? "禁用角色" : "启用角色"}>
                <Button
                  type="text"
                  icon={
                    record.isActive ? (
                      <XCircle className="w-4 h-4" />
                    ) : (
                      <CheckCircle className="w-4 h-4" />
                    )
                  }
                  onClick={() => handleToggleStatus(record.id)}
                />
              </Tooltip>
              <Popconfirm
                title="确定要删除这个角色吗？"
                description="删除后无法恢复，关联的用户将失去此角色权限。"
                onConfirm={() => handleDeleteRole(record.id)}
                okText="确定删除"
                cancelText="取消"
                okType="danger"
              >
                <Button
                  type="text"
                  danger
                  icon={<Trash2 className="w-4 h-4" />}
                />
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      {/* 页面标题 */}
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Shield className="inline-block w-6 h-6 mr-2" />
          角色权限管理
        </Title>
        <Text type="secondary">管理系统角色和权限分配，控制用户访问范围</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总角色数"
              value={stats.total}
              prefix={<Shield className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="启用角色"
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="系统角色"
              value={stats.system}
              prefix={<Key className="w-5 h-5" />}
              valueStyle={{ color: "#722ed1" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="关联用户"
              value={stats.totalUsers}
              prefix={<Users className="w-5 h-5" />}
              valueStyle={{ color: "#fa8c16" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card className="mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={12} lg={8}>
            <Input
              placeholder="搜索角色名称..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={6}>
            <Select
              placeholder="筛选状态"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: "100%" }}
            >
              <Option value="all">全部状态</Option>
              <Option value="active">启用</Option>
              <Option value="inactive">禁用</Option>
            </Select>
          </Col>
          <Col xs={24} md={4} lg={10} className="text-right">
            <Button
              type="primary"
              icon={<Plus className="w-4 h-4" />}
              onClick={() => {
                setSelectedRole(null);
                form.resetFields();
                setShowModal(true);
              }}
            >
              新建角色
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 角色列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={filteredRoles}
          rowKey="id"
          pagination={{
            total: filteredRoles.length,
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 角色编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedRole ? "编辑角色" : "新建角色"}
          </span>
        }
        open={showModal}
        onOk={handleSaveRole}
        onCancel={() => {
          setShowModal(false);
          setSelectedRole(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label="角色名称"
            name="name"
            rules={[{ required: true, message: "请输入角色名称" }]}
          >
            <Input placeholder="请输入角色名称" />
          </Form.Item>
          <Form.Item
            label="角色描述"
            name="description"
            rules={[{ required: true, message: "请输入角色描述" }]}
          >
            <Input.TextArea rows={3} placeholder="请输入角色描述" />
          </Form.Item>
          <Form.Item
            label="角色状态"
            name="isActive"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>

        {selectedRole && (
          <Alert
            message="权限配置"
            description="角色权限需要在专门的权限配置页面进行设置。"
            type="info"
            showIcon
            className="mt-4"
          />
        )}
      </Modal>

      {/* 权限详情模态框 */}
      <Modal
        title={
          <div className="flex items-center gap-2">
            <Eye className="w-5 h-5" />
            <span>{selectedRole?.name} - 权限详情</span>
            <Tag color={selectedRole?.isSystem ? "purple" : "blue"}>
              {selectedRole?.isSystem ? "系统角色" : "自定义角色"}
            </Tag>
          </div>
        }
        open={showPermissionModal}
        onCancel={() => setShowPermissionModal(false)}
        width={1000}
        footer={[
          <Button key="close" onClick={() => setShowPermissionModal(false)}>
            关闭
          </Button>,
        ]}
      >
        {selectedRole && (
          <div className="mt-4">
            <div className="mb-4">
              <Text type="secondary">{selectedRole.description}</Text>
            </div>

            {Object.keys(selectedRole.permissions).length > 0 ? (
              <Tabs
                items={renderPermissionsByCategory(selectedRole.permissions)}
                className="custom-tabs"
              />
            ) : (
              <Alert
                message="暂无权限"
                description="此角色还未配置任何权限。"
                type="info"
                showIcon
              />
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default RoleManagement;

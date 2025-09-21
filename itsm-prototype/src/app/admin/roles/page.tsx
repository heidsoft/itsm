"use client";

import { Plus, CheckCircle, Users, Search, Settings, Trash2, XCircle, Edit, Eye, Lock, Unlock, Key, Tag, Shield } from 'lucide-react';

import React, { useState, useEffect } from "react";
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
  UPDATE: "update",
  DELETE: "delete",
  EXPORT: "export",
  IMPORT: "import",
} as const;

// 引入角色API
import { RoleAPI } from "../../lib/role-api";

// 权限定义
const PERMISSIONS = [
  { module: PERMISSION_MODULES.DASHBOARD, name: "仪表盘", permissions: ["view"] },
  { module: PERMISSION_MODULES.TICKETS, name: "工单管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.INCIDENTS, name: "事件管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.PROBLEMS, name: "问题管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.CHANGES, name: "变更管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.SERVICE_CATALOG, name: "服务目录", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.KNOWLEDGE_BASE, name: "知识库", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.REPORTS, name: "报表分析", permissions: ["view"] },
  { module: PERMISSION_MODULES.ADMIN, name: "系统管理", permissions: ["view"] },
  { module: PERMISSION_MODULES.USERS, name: "用户管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.ROLES, name: "角色管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.WORKFLOWS, name: "工作流管理", permissions: ["view", "create", "update", "delete"] },
  { module: PERMISSION_MODULES.SYSTEM_CONFIG, name: "系统配置", permissions: ["view", "update"] },
];

export default function RoleManagement() {
  const [roles, setRoles] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState<any>(null);
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [stats, setStats] = useState({
    totalRoles: 0,
    activeRoles: 0,
    inactiveRoles: 0,
    totalUsers: 0,
  });
  const [availablePermissions, setAvailablePermissions] = useState<string[]>([]);

  // 加载角色数据
  const loadRoles = async () => {
    setLoading(true);
    try {
      const response = await RoleAPI.getRoles({
        search: searchTerm || undefined,
        status: statusFilter !== "all" ? statusFilter : undefined,
      });
      
      setRoles(response.roles);
      
      // 计算统计数据
      const totalRoles = response.roles.length;
      const activeRoles = response.roles.filter(r => r.status === "active").length;
      const inactiveRoles = response.roles.filter(r => r.status === "inactive").length;
      
      setStats({
        totalRoles,
        activeRoles,
        inactiveRoles,
        totalUsers: 128, // 模拟用户数
      });
    } catch (error) {
      console.error("加载角色数据失败:", error);
      message.error("加载角色数据失败");
    } finally {
      setLoading(false);
    }
  };

  // 加载权限列表
  const loadPermissions = async () => {
    try {
      const permissions = await RoleAPI.getPermissions();
      setAvailablePermissions(permissions);
    } catch (error) {
      console.error("加载权限列表失败:", error);
      // 使用默认权限列表
      const defaultPermissions: string[] = [];
      Object.values(PERMISSION_MODULES).forEach(module => {
        Object.values(PERMISSION_ACTIONS).forEach(action => {
          defaultPermissions.push(`${module}:${action}`);
        });
      });
      setAvailablePermissions(defaultPermissions);
    }
  };

  // 初始化加载数据
  useEffect(() => {
    loadRoles();
    loadPermissions();
  }, [searchTerm, statusFilter]);

  // 处理保存角色
  const handleSaveRole = async () => {
    try {
      const values = await form.validateFields();
      
      // 构造权限列表
      const permissions: string[] = [];
      Object.values(PERMISSION_MODULES).forEach(module => {
        Object.values(PERMISSION_ACTIONS).forEach(action => {
          const fieldName = `${module}_${action}`;
          if (values[fieldName]) {
            permissions.push(`${module}:${action}`);
          }
        });
      });
      
      const roleData = {
        name: values.name,
        description: values.description,
        permissions,
        status: values.status ? "active" : "inactive",
      };
      
      if (selectedRole) {
        // 更新角色
        await RoleAPI.updateRole(selectedRole.id, roleData);
        message.success("角色更新成功");
      } else {
        // 创建角色
        await RoleAPI.createRole(roleData);
        message.success("角色创建成功");
      }
      
      setShowModal(false);
      form.resetFields();
      setSelectedRole(null);
      loadRoles(); // 重新加载数据
    } catch (error) {
      console.error("保存角色失败:", error);
      message.error("保存角色失败");
    }
  };

  // 处理删除角色
  const handleDeleteRole = async (id: number) => {
    try {
      await RoleAPI.deleteRole(id);
      message.success("角色删除成功");
      loadRoles(); // 重新加载数据
    } catch (error) {
      console.error("删除角色失败:", error);
      message.error("删除角色失败");
    }
  };

  // 处理权限全选
  const handleSelectAllModule = (module: string, checked: boolean) => {
    const modulePermissions: Record<string, boolean> = {};
    Object.values(PERMISSION_ACTIONS).forEach(action => {
      modulePermissions[`${module}_${action}`] = checked;
    });
    form.setFieldsValue(modulePermissions);
  };

  // 表格列定义
  const columns = [
    {
      title: "角色信息",
      key: "info",
      render: (_: any, record: any) => (
        <div>
          <div className="font-medium text-gray-900">{record.name}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      ),
    },
    {
      title: "状态",
      key: "status",
      render: (_: any, record: any) => (
        <Badge 
          status={record.status === "active" ? "success" : "default"} 
          text={record.status === "active" ? "启用" : "禁用"} 
        />
      ),
    },
    {
      title: "权限数量",
      key: "permissions",
      dataIndex: "permissions",
      render: (permissions: string[]) => (
        <span>{permissions.length}</span>
      ),
    },
    {
      title: "创建时间",
      key: "created_at",
      dataIndex: "created_at",
      render: (createdAt: string) => (
        <span>{new Date(createdAt).toLocaleDateString()}</span>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 150,
      render: (_: any, record: any) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => {
                setSelectedRole(record);
                // 设置表单值
                const formValues: Record<string, any> = {
                  name: record.name,
                  description: record.description,
                  status: record.status === "active",
                };
                
                // 设置权限值
                Object.values(PERMISSION_MODULES).forEach(module => {
                  Object.values(PERMISSION_ACTIONS).forEach(action => {
                    const permission = `${module}:${action}`;
                    formValues[`${module}_${action}`] = record.permissions.includes(permission);
                  });
                });
                
                form.setFieldsValue(formValues);
                setShowModal(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            description="确定要删除这个角色吗？此操作不可恢复。"
            onConfirm={() => handleDeleteRole(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button
                type="text"
                danger
                icon={<Trash2 className="w-4 h-4" />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 权限配置标签页
  const PermissionConfigTab = () => (
    <div className="space-y-6">
      <Alert
        message="权限说明"
        description="为角色分配相应的权限，控制用户可以访问的功能模块和操作。"
        type="info"
        showIcon
      />
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {Object.entries(PERMISSION_MODULES).map(([key, module]) => {
          const moduleConfig = PERMISSIONS.find(p => p.module === module);
          if (!moduleConfig) return null;
          
          return (
            <Card key={key} size="small" title={
              <div className="flex items-center">
                <Shield className="w-4 h-4 mr-2" />
                {moduleConfig.name}
              </div>
            }>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">全选</span>
                  <Checkbox 
                    onChange={(e) => handleSelectAllModule(module, e.target.checked)}
                  />
                </div>
                <Divider className="my-2" />
                <div className="grid grid-cols-2 gap-2">
                  {moduleConfig.permissions.map(action => {
                    const actionLabel = {
                      view: "查看",
                      create: "创建",
                      update: "编辑",
                      delete: "删除",
                      export: "导出",
                      import: "导入",
                    }[action] || action;
                    
                    return (
                      <div key={action} className="flex items-center">
                        <Form.Item 
                          name={`${module}_${action}`} 
                          valuePropName="checked"
                          className="mb-0"
                        >
                          <Checkbox>
                            <span className="text-sm">{actionLabel}</span>
                          </Checkbox>
                        </Form.Item>
                      </div>
                    );
                  })}
                </div>
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );

  // 标签页配置
  const tabItems = [
    {
      key: "basic",
      label: "基本信息",
      children: (
        <div className="space-y-4">
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
            label="状态"
            name="status"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </div>
      ),
    },
    {
      key: "permissions",
      label: "权限配置",
      children: <PermissionConfigTab />,
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Key className="inline-block w-6 h-6 mr-2" />
          角色权限管理
        </Title>
        <Text type="secondary">管理系统角色和权限分配</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总角色数"
              value={stats.totalRoles}
              prefix={<Key className="w-5 h-5" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="启用角色"
              value={stats.activeRoles}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="禁用角色"
              value={stats.inactiveRoles}
              prefix={<XCircle className="w-5 h-5" />}
              valueStyle={{ color: "#ff4d4f" }}
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
          dataSource={roles}
          rowKey="id"
          loading={loading}
          pagination={{
            total: roles.length,
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
        width={800}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Tabs items={tabItems} type="card" />
        </Form>
      </Modal>
    </div>
  );
}

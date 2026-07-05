'use client';

import {
  Plus,
  Edit,
  Trash2,
  Shield,
  Users,
  Settings,
  Key,
  CheckCircle,
  XCircle,
  Search,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
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
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { RoleAPI } from '@/lib/api/role-api';
import { UserApi } from '@/lib/api/user-api';

const { Title, Text } = Typography;
const { Option } = Select;

// 权限模块定义
const PERMISSION_MODULES = {
  DASHBOARD: 'dashboard',
  TICKETS: 'ticket',
  INCIDENTS: 'incident',
  PROBLEMS: 'problem',
  CHANGES: 'change',
  SERVICE_CATALOG: 'service_catalog',
  SERVICE_REQUEST: 'service_request',
  KNOWLEDGE_BASE: 'knowledge',
  CMDB: 'cmdb',
  ASSETS: 'asset',
  RELEASES: 'release',
  REPORTS: 'report',
  ADMIN: 'admin',
  USERS: 'user',
  ROLES: 'role',
  GROUPS: 'groups',
  ORG: 'org',
  WORKFLOWS: 'bpmn',
  SYSTEM_CONFIG: 'system_config',
  AI: 'ai',
} as const;

// 权限操作类型
const PERMISSION_ACTIONS = {
  VIEW: 'view',
  READ: 'read',
  CREATE: 'create',
  WRITE: 'write',
  UPDATE: 'update',
  DELETE: 'delete',
  MANAGE: 'manage',
  APPROVE: 'approve',
  ASSIGN: 'assign',
  EXPORT: 'export',
  IMPORT: 'import',
  ALL: 'all',
} as const;

// 权限定义
const PERMISSIONS = [
  { module: PERMISSION_MODULES.DASHBOARD, name: '仪表盘', permissions: ['view'] },
  {
    module: PERMISSION_MODULES.TICKETS,
    name: '工单管理',
    permissions: ['view', 'create', 'update', 'delete'],
  },
  {
    module: PERMISSION_MODULES.INCIDENTS,
    name: '事件管理',
    permissions: ['view', 'create', 'update', 'delete'],
  },
  {
    module: PERMISSION_MODULES.PROBLEMS,
    name: '问题管理',
    permissions: ['view', 'create', 'update', 'delete'],
  },
  {
    module: PERMISSION_MODULES.CHANGES,
    name: '变更管理',
    permissions: ['view', 'create', 'update', 'delete'],
  },
  {
    module: PERMISSION_MODULES.SERVICE_CATALOG,
    name: '服务目录',
    permissions: ['read', 'write', 'delete'],
  },
  {
    module: PERMISSION_MODULES.SERVICE_REQUEST,
    name: '服务请求',
    permissions: ['read', 'write', 'delete', 'approve'],
  },
  {
    module: PERMISSION_MODULES.KNOWLEDGE_BASE,
    name: '知识库',
    permissions: ['read', 'write', 'delete'],
  },
  { module: PERMISSION_MODULES.CMDB, name: 'CMDB', permissions: ['read', 'write', 'delete'] },
  { module: PERMISSION_MODULES.ASSETS, name: '资产管理', permissions: ['read', 'write', 'delete'] },
  { module: PERMISSION_MODULES.RELEASES, name: '发布管理', permissions: ['read', 'write', 'delete'] },
  { module: PERMISSION_MODULES.REPORTS, name: '报表分析', permissions: ['read', 'view', 'export'] },
  { module: PERMISSION_MODULES.ADMIN, name: '系统管理', permissions: ['read', 'write', 'all'] },
  {
    module: PERMISSION_MODULES.USERS,
    name: '用户管理',
    permissions: ['read', 'write', 'delete', 'manage'],
  },
  {
    module: PERMISSION_MODULES.ROLES,
    name: '角色管理',
    permissions: ['read', 'write', 'delete', 'manage'],
  },
  {
    module: PERMISSION_MODULES.GROUPS,
    name: '用户组管理',
    permissions: ['read', 'write', 'delete'],
  },
  {
    module: PERMISSION_MODULES.ORG,
    name: '组织架构',
    permissions: ['read', 'write'],
  },
  {
    module: PERMISSION_MODULES.WORKFLOWS,
    name: '工作流管理',
    permissions: ['read', 'write', 'delete'],
  },
  { module: PERMISSION_MODULES.SYSTEM_CONFIG, name: '系统配置', permissions: ['read', 'write', 'update'] },
  { module: PERMISSION_MODULES.AI, name: 'AI 能力', permissions: ['read', 'write'] },
];

export default function RoleManagement() {
  interface RoleItem {
    id: number;
    name: string;
    code?: string;
    description?: string;
    status?: string;
    permissions: string[];
    createdAt?: string;
  }
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState<any>(null);
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
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
      const [rolesResponse, userStats] = await Promise.all([
        RoleAPI.getRoles({
          search: searchTerm || undefined,
          status: statusFilter !== 'all' ? statusFilter : undefined,
        }),
        UserApi.getUserStats().catch(() => ({ total: 0, active: 0, inactive: 0 })),
      ]);

      setRoles(rolesResponse.roles);

      // 计算统计数据
      // 注意: 后端角色实体没有 status 字段，所以 activeRoles/inactiveRoles 无法准确计算
      // 这里我们使用 totalRoles 作为总角色数，并说明状态统计的限制
      const totalRoles = rolesResponse.roles.length;
      // 由于后端 Role 实体没有 status 字段，所有角色默认为启用状态
      // 实际应用中可能需要通过 is_system 或其他字段来判断
      const activeRoles = totalRoles; // 所有非系统角色视为启用
      const inactiveRoles = 0; // 无法从后端获取此信息

      setStats({
        totalRoles,
        activeRoles,
        inactiveRoles,
        totalUsers: userStats.total,
      });
    } catch (error) {
      console.error('Failed to load roles:', error);
      message.error('加载角色数据失败');
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
    setLoading(true);
    try {
      const values = await form.validateFields();

      // 构建权限编码列表
      const permissionCodes: string[] = [];
      Object.values(PERMISSION_MODULES).forEach(module => {
        Object.values(PERMISSION_ACTIONS).forEach(action => {
          const fieldName = `${module}_${action}`;
          if (values[fieldName]) {
            permissionCodes.push(`${module}:${action}`);
          }
        });
      });

      // 更新角色基本信息（不含权限）
      const roleData = {
        name: values.name,
        code: values.code,
        description: values.description,
        status: (values.status ? 'active' : 'inactive') as 'active' | 'inactive',
      };

      let roleId: number;
      if (selectedRole) {
        const updated = await RoleAPI.updateRole(selectedRole.id, roleData);
        roleId = updated.id;
        message.success('角色更新成功');
      } else {
        const created = await RoleAPI.createRole({ ...roleData, permissions: permissionCodes });
        roleId = created.id;
        message.success('角色创建成功');
      }

      // 分配权限（使用专用接口）
      if (permissionCodes.length > 0) {
        try {
          const catalog = await RoleAPI.getPermissionCatalog();
          const codeToId = new Map(catalog.map(p => [p.code, p.id]));
          const permissionIds = permissionCodes
            .map(code => codeToId.get(code))
            .filter((id): id is number => id !== undefined && id !== 0);

          if (permissionIds.length > 0) {
            await RoleAPI.assignPermissions(roleId, permissionIds);
          }
        } catch (permError) {
          console.error('Failed to assign permissions:', permError);
          message.warning('角色基本信息已保存，但权限分配失败，请重试');
        }
      }

      setShowModal(false);
      form.resetFields();
      setSelectedRole(null);
      loadRoles(); // 重新加载数据
    } catch (error) {
      message.error('保存角色失败');
    } finally {
      setLoading(false);
    }
  };

  const handleInitPermissions = async () => {
    setLoading(true);
    try {
      await RoleAPI.initDefaultPermissions();
      await loadPermissions();
      message.success('默认权限字典初始化完成');
    } catch (error) {
      message.error('初始化权限字典失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理删除角色
  const handleDeleteRole = async (id: number) => {
    try {
      await RoleAPI.deleteRole(id);
      message.success('角色删除成功');
      loadRoles(); // 重新加载数据
    } catch (error) {
      message.error('删除角色失败');
    }
  };

  // 处理权限全选
  const handleSelectAllModule = (module: string, checked: boolean) => {
    const modulePermissions: Record<string, boolean> = {};
    const moduleConfig = PERMISSIONS.find(p => p.module === module);
    (moduleConfig?.permissions || []).forEach(action => {
      modulePermissions[`${module}_${action}`] = checked;
    });
    form.setFieldsValue(modulePermissions);
  };

  // 表格列定义
  const columns: ColumnsType<RoleItem> = [
    {
      title: '角色信息',
      key: 'info',
      render: (_: unknown, record: RoleItem) => (
        <div>
          <div className="font-medium text-gray-900">{record.name}</div>
          <div className="text-sm text-gray-500">
            {record.code ? `${record.code} · ` : ''}
            {record.description}
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (_: unknown, record: RoleItem) => {
        // 后端 Role 实体没有 status 字段时，按启用处理，避免空值显示为灰色但文本仍是启用。
        const isActive = record.status !== 'inactive';
        return (
          <Badge status={isActive ? 'success' : 'default'} text={isActive ? '启用' : '禁用'} />
        );
      },
    },
    {
      title: '权限数量',
      key: 'permissions',
      dataIndex: 'permissions',
      render: (permissions: string[]) => <span>{permissions?.length || 0}</span>,
    },
    {
      title: '创建时间',
      key: 'createdAt',
      dataIndex: 'createdAt',
      render: (createdAt: string) => (
        <span>{createdAt ? new Date(createdAt).toLocaleDateString() : '-'}</span>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: RoleItem) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => {
                setSelectedRole(record);
                // 设置表单值
                const formValues: Record<string, unknown> = {
                  name: record.name,
                  code: record.code,
                  description: record.description,
                  status: record.status !== 'inactive',
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
              <Button type="text" danger icon={<Trash2 className="w-4 h-4" />} />
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
            <Card
              key={key}
              size="small"
              title={
                <div className="flex items-center">
                  <Shield className="w-4 h-4 mr-2" />
                  {moduleConfig.name}
                </div>
              }
            >
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">全选</span>
                  <Checkbox onChange={e => handleSelectAllModule(module, e.target.checked)} />
                </div>
                <Divider className="my-2" />
                <div className="grid grid-cols-2 gap-2">
                  {moduleConfig.permissions.map(action => {
                    const actionLabel =
                      {
                        view: '查看',
                        create: '创建',
                        update: '编辑',
                        write: '写入',
                        read: '读取',
                        delete: '删除',
                        manage: '管理',
                        approve: '审批',
                        assign: '分配',
                        all: '全部',
                        export: '导出',
                        import: '导入',
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
      key: 'basic',
      label: '基本信息',
      children: (
        <div className="space-y-4">
          <Form.Item
            label="角色名称"
            name="name"
            rules={[{ required: true, message: '请输入角色名称' }]}
          >
            <Input placeholder="请输入角色名称" />
          </Form.Item>
          <Form.Item
            label="角色编码"
            name="code"
            tooltip="编码用于权限缓存和系统集成，建议使用英文、数字、下划线"
          >
            <Input placeholder="例如：it_manager、change_approver" disabled={selectedRole?.isSystem} />
          </Form.Item>
          <Form.Item
            label="角色描述"
            name="description"
            rules={[{ required: true, message: '请输入角色描述' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入角色描述" />
          </Form.Item>
          <Form.Item label="状态" name="status" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'permissions',
      label: '权限配置',
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
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="禁用角色"
              value={stats.inactiveRoles}
              prefix={<XCircle className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="关联用户"
              value={stats.totalUsers}
              prefix={<Users className="w-5 h-5" />}
              styles={{ content: { color: '#fa8c16' } }}
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
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={6}>
            <Select
              placeholder="筛选状态"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
            >
              <Option value="all">全部状态</Option>
              <Option value="active">启用</Option>
              <Option value="inactive">禁用</Option>
            </Select>
          </Col>
          <Col xs={24} md={4} lg={10} className="text-right">
            <Space>
              <Button onClick={handleInitPermissions} loading={loading}>
                初始化权限字典
              </Button>
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
            </Space>
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
            showTotal: total => `共 ${total} 条记录`,
          }}
          scroll={{ x: 760 }}
          className="enterprise-table"
        />
      </Card>

      {/* 角色编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedRole ? '编辑角色' : '新建角色'}
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

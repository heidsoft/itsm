'use client';

import {
  Plus,
  Edit,
  Trash2,
  Shield,
  Users,
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
  App,
  Divider,
  Alert,
  Tabs,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { RoleAPI } from '@/lib/api/role-api';
import { UserApi } from '@/lib/api/user-api';
import type { PermissionCatalogItem } from '@/lib/api/api-config';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;

// 按 resource 分组的权限模块，从后端 permission catalog 动态派生
interface PermissionModule {
  resource: string;
  label: string;
  actions: string[]; // 该资源下可用的操作代码，按 catalog 顺序去重
}

/**
 * 从后端权限目录（PermissionCatalogItem[]）派生前端权限矩阵。
 * - 按 resource 分组
 * - 每组内按 catalog 出现顺序保留 action，去重
 * - resource 展示名优先取 catalog 中 name 字段，缺失时兜底到 i18n 的 resourceLabels，再兜底到 resource 代码
 */
function derivePermissionModules(
  catalog: PermissionCatalogItem[],
  t: (key: string) => string
): PermissionModule[] {
  const grouped = new Map<string, { label: string; actions: string[] }>();
  for (const item of catalog) {
    const resource = item.resource || '';
    const action = item.action || '';
    if (!resource || !action) continue;
    let bucket = grouped.get(resource);
    if (!bucket) {
      // name 若与 code 相同，视为无中文，走 i18n 兜底
      const backendName = item.name && item.name !== item.code ? item.name : undefined;
      const i18nKey = `roles.resourceLabels.${resource}`;
      const i18nLabel = t(i18nKey);
      const fallback = i18nLabel === i18nKey ? undefined : i18nLabel;
      bucket = {
        label: backendName || fallback || resource,
        actions: [],
      };
      grouped.set(resource, bucket);
    }
    if (!bucket.actions.includes(action)) {
      bucket.actions.push(action);
    }
  }
  return Array.from(grouped.entries()).map(([resource, { label, actions }]) => ({
    resource,
    label,
    actions,
  }));
}

export default function RoleManagement() {
  interface RoleItem {
    id: number;
    name: string;
    code?: string;
    description?: string;
    status?: string;
    permissions: string[];
    createdAt?: string;
    isSystem?: boolean;
  }
  const { t } = useI18n();
  const { message } = App.useApp();
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState<RoleItem | null>(null);
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [stats, setStats] = useState({
    totalRoles: 0,
    activeRoles: 0,
    inactiveRoles: 0,
    totalUsers: 0,
  });
  const [, setAvailablePermissions] = useState<string[]>([]);
  // 从后端 permission catalog 动态派生的权限模块矩阵
  const [permissionModules, setPermissionModules] = useState<PermissionModule[]>([]);
  const [permissionCatalog, setPermissionCatalog] = useState<PermissionCatalogItem[]>([]);
  const [permissionsLoading, setPermissionsLoading] = useState(false);
  const [permissionsError, setPermissionsError] = useState<string | null>(null);

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

      const totalRoles = rolesResponse.roles.length;
      const activeRoles = totalRoles;
      const inactiveRoles = 0;

      setStats({
        totalRoles,
        activeRoles,
        inactiveRoles,
        totalUsers: userStats.total,
      });
    } catch (error) {
      console.error('Failed to load roles:', error);
      message.error(t('roles.loadRolesFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 加载权限目录（动态派生模块矩阵）
  const loadPermissions = async () => {
    setPermissionsLoading(true);
    setPermissionsError(null);
    try {
      const catalog = await RoleAPI.getPermissionCatalog();
      setPermissionCatalog(catalog);
      setAvailablePermissions(catalog.map(p => p.code));
      const modules = derivePermissionModules(catalog, t);
      setPermissionModules(modules);
      if (modules.length === 0) {
        setPermissionsError(t('roles.permissionsEmpty'));
      }
    } catch (error) {
      console.error('Failed to load permission catalog:', error);
      setPermissionCatalog([]);
      setAvailablePermissions([]);
      setPermissionModules([]);
      setPermissionsError(t('roles.loadPermissionsFailed'));
    } finally {
      setPermissionsLoading(false);
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

      // 构建权限编码列表（遍历后端返回的权限矩阵，而非硬编码）
      const permissionCodes: string[] = [];
      permissionModules.forEach(({ resource, actions }) => {
        actions.forEach(action => {
          const fieldName = `${resource}_${action}`;
          if (values[fieldName]) {
            permissionCodes.push(`${resource}:${action}`);
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
        message.success(t('roles.updateSuccess'));
      } else {
        const created = await RoleAPI.createRole({ ...roleData, permissions: permissionCodes });
        roleId = created.id;
        message.success(t('roles.createSuccess'));
      }

      // 分配权限（使用专用接口）
      if (permissionCodes.length > 0) {
        try {
          // 优先使用已加载的 catalog，避免重复请求
          const catalog =
            permissionCatalog.length > 0
              ? permissionCatalog
              : await RoleAPI.getPermissionCatalog();
          const codeToId = new Map(catalog.map(p => [p.code, p.id]));
          const permissionIds = permissionCodes
            .map(code => codeToId.get(code))
            .filter((id): id is number => id !== undefined && id !== 0);

          if (permissionIds.length > 0) {
            await RoleAPI.assignPermissions(roleId, permissionIds);
          }
        } catch (permError) {
          console.error('Failed to assign permissions:', permError);
          message.warning(t('roles.permissionAssignFailed'));
        }
      }

      setShowModal(false);
      form.resetFields();
      setSelectedRole(null);
      loadRoles(); // 重新加载数据
    } catch (error) {
      message.error(t('roles.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleInitPermissions = async () => {
    setLoading(true);
    try {
      await RoleAPI.initDefaultPermissions();
      await loadPermissions();
      message.success(t('roles.initPermissionsSuccess'));
    } catch (error) {
      message.error(t('roles.initPermissionsFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 处理删除角色
  const handleDeleteRole = async (id: number) => {
    try {
      await RoleAPI.deleteRole(id);
      message.success(t('roles.deleteSuccess'));
      loadRoles(); // 重新加载数据
    } catch (error) {
      message.error(t('roles.deleteFailed'));
    }
  };

  // 处理权限全选
  const handleSelectAllModule = (resource: string, checked: boolean) => {
    const modulePermissions: Record<string, boolean> = {};
    const moduleConfig = permissionModules.find(p => p.resource === resource);
    (moduleConfig?.actions || []).forEach(action => {
      modulePermissions[`${resource}_${action}`] = checked;
    });
    form.setFieldsValue(modulePermissions);
  };

  // 表格列定义
  const columns: ColumnsType<RoleItem> = [
    {
      title: t('roles.title'),
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
      title: t('roles.status'),
      key: 'status',
      render: (_: unknown, record: RoleItem) => {
        const isActive = record.status !== 'inactive';
        return (
          <Badge
            status={isActive ? 'success' : 'default'}
            text={isActive ? t('roles.active') : t('roles.inactive')}
          />
        );
      },
    },
    {
      title: t('roles.permissionsCount'),
      key: 'permissions',
      dataIndex: 'permissions',
      render: (permissions: string[]) => <span>{permissions?.length || 0}</span>,
    },
    {
      title: t('roles.createdAt'),
      key: 'createdAt',
      dataIndex: 'createdAt',
      render: (createdAt: string) => (
        <span>{createdAt ? new Date(createdAt).toLocaleDateString() : '-'}</span>
      ),
    },
    {
      title: t('roles.actions'),
      key: 'actions',
      width: 150,
      render: (_: unknown, record: RoleItem) => (
        <Space size="small">
          <Tooltip title={t('roles.editTooltip')}>
            <Button
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => {
                setSelectedRole(record);
                const formValues: Record<string, unknown> = {
                  name: record.name,
                  code: record.code,
                  description: record.description,
                  status: record.status !== 'inactive',
                };

                permissionModules.forEach(({ resource, actions }) => {
                  actions.forEach(action => {
                    const permission = `${resource}:${action}`;
                    formValues[`${resource}_${action}`] = record.permissions.includes(permission);
                  });
                });

                form.setFieldsValue(formValues);
                setShowModal(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title={t('roles.deleteConfirmTitle')}
            description={t('roles.deleteConfirmDescription')}
            onConfirm={() => handleDeleteRole(record.id)}
            okText={t('roles.confirm')}
            cancelText={t('roles.cancel')}
          >
            <Tooltip title={t('roles.deleteTooltip')}>
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
        message={t('roles.permissionInfo')}
        description={t('roles.permissionInfoDesc')}
        type="info"
        showIcon
      />

      {permissionsError && (
        <Alert
          message={permissionsError}
          type="warning"
          showIcon
          action={
            <Button size="small" type="link" onClick={handleInitPermissions} loading={loading}>
              {t('roles.initDefaultPermissions')}
            </Button>
          }
        />
      )}

      {permissionsLoading ? (
        <div className="text-center py-8 text-gray-500">{t('roles.loadPermissions')}</div>
      ) : permissionModules.length === 0 && !permissionsError ? (
        <div className="text-center py-8 text-gray-500">{t('roles.noPermissions')}</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {permissionModules.map(({ resource, label, actions }) => (
            <Card
              key={resource}
              size="small"
              title={
                <div className="flex items-center">
                  <Shield className="w-4 h-4 mr-2" />
                  {label}
                  <Text type="secondary" className="ml-2 text-xs">
                    {resource}
                  </Text>
                </div>
              }
            >
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">{t('roles.selectAll')}</span>
                  <Checkbox onChange={e => handleSelectAllModule(resource, e.target.checked)} />
                </div>
                <Divider className="my-2" />
                <div className="grid grid-cols-2 gap-2">
                  {actions.map(action => {
                    const actionI18nKey = `roles.actionLabels.${action}`;
                    const actionI18nLabel = t(actionI18nKey);
                    const actionLabel = actionI18nLabel === actionI18nKey ? action : actionI18nLabel;
                    return (
                      <div key={action} className="flex items-center">
                        <Form.Item
                          name={`${resource}_${action}`}
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
          ))}
        </div>
      )}
    </div>
  );

  // 标签页配置
  const tabItems = [
    {
      key: 'basic',
      label: t('roles.basicTab'),
      children: (
        <div className="space-y-4">
          <Form.Item
            label={t('roles.roleName')}
            name="name"
            rules={[{ required: true, message: t('roles.enterRoleName') }]}
          >
            <Input placeholder={t('roles.enterRoleName')} />
          </Form.Item>
          <Form.Item
            label={t('roles.roleCode')}
            name="code"
            tooltip={t('roles.roleCodeTooltip')}
          >
            <Input placeholder={t('roles.roleCodePlaceholder')} disabled={selectedRole?.isSystem} />
          </Form.Item>
          <Form.Item
            label={t('roles.roleDescription')}
            name="description"
            rules={[{ required: true, message: t('roles.enterRoleDescription') }]}
          >
            <Input.TextArea rows={3} placeholder={t('roles.enterRoleDescription')} />
          </Form.Item>
          <Form.Item label={t('roles.status')} name="status" valuePropName="checked">
            <Switch checkedChildren={t('roles.active')} unCheckedChildren={t('roles.inactive')} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'permissions',
      label: t('roles.permissionsTab'),
      children: <PermissionConfigTab />,
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <Key className="inline-block w-6 h-6 mr-2" />
          {t('roles.title')}
        </Title>
        <Text type="secondary">{t('roles.description')}</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('roles.total')}
              value={stats.totalRoles}
              prefix={<Key className="w-5 h-5" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('roles.enabled')}
              value={stats.activeRoles}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('roles.disabled')}
              value={stats.inactiveRoles}
              prefix={<XCircle className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('roles.userCount')}
              value={stats.totalUsers}
              prefix={<Users className="w-5 h-5" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={12} lg={8}>
            <Input
              placeholder={t('roles.searchPlaceholder')}
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={6}>
            <Select
              placeholder={t('roles.selectStatus')}
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: t('roles.allStatus') },
                { value: 'active', label: t('roles.statusActive') },
                { value: 'inactive', label: t('roles.statusInactive') },
              ]}
            />
          </Col>
          <Col xs={24} md={4} lg={10} className="text-right">
            <Space>
              <Button onClick={handleInitPermissions} loading={loading}>
                {t('roles.initPermissions')}
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
                {t('roles.create')}
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
            showTotal: total => t('roles.totalLabel', { total }),
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
            {selectedRole ? t('roles.edit') : t('roles.create')}
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
        okText={t('roles.save')}
        cancelText={t('roles.cancel')}
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Tabs items={tabItems} type="card" />
        </Form>
      </Modal>
    </div>
  );
}

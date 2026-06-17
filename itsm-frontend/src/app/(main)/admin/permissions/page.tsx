'use client';

import {
  RefreshCw,
  Save,
  Globe,
  BarChart3,
  Activity,
  Settings,
  Key,
  Layers,
  CheckCircle,
  Shield,
  Search,
} from 'lucide-react';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Space,
  Typography,
  Switch,
  Row,
  Col,
  Statistic,
  Badge,
  Tag,
  Tooltip,
  Alert,
  App,
  Tree,
  Spin,
} from 'antd';
import { RoleAPI } from '@/lib/api/role-api';
import type { Role } from '@/lib/api/api-config';
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
  READ: 'read',
  CREATE: 'create',
  WRITE: 'write',
  UPDATE: 'update',
  DELETE: 'delete',
  MANAGE: 'manage',
  APPROVE: 'approve',
  ASSIGN: 'assign',
  VIEW: 'view',
  EXPORT: 'export',
  ALL: 'all',
} as const;

// 权限模块配置
const MODULE_CONFIG: Record<string, { label: string; icon: string; description: string; category: string }> = {
  [PERMISSION_MODULES.DASHBOARD]: { label: '仪表盘', icon: '📊', description: '系统仪表盘和概览信息', category: '核心功能' },
  [PERMISSION_MODULES.TICKETS]: { label: '工单管理', icon: '🎫', description: '工单的创建、处理和管理', category: '核心功能' },
  [PERMISSION_MODULES.INCIDENTS]: { label: '事件管理', icon: '🚨', description: 'IT事件的记录和处理', category: '核心功能' },
  [PERMISSION_MODULES.PROBLEMS]: { label: '问题管理', icon: '🔧', description: '根本原因分析和问题解决', category: '核心功能' },
  [PERMISSION_MODULES.CHANGES]: { label: '变更管理', icon: '🔄', description: 'IT变更的规划和实施', category: '核心功能' },
  [PERMISSION_MODULES.SERVICE_CATALOG]: { label: '服务目录', icon: '📋', description: 'IT服务目录管理', category: '服务管理' },
  [PERMISSION_MODULES.SERVICE_REQUEST]: { label: '服务请求', icon: '🧾', description: '服务请求提交、审批和履约', category: '服务管理' },
  [PERMISSION_MODULES.KNOWLEDGE_BASE]: { label: '知识库', icon: '📚', description: '知识文档和解决方案', category: '服务管理' },
  [PERMISSION_MODULES.CMDB]: { label: 'CMDB', icon: '🧩', description: '配置项、关系和拓扑管理', category: '核心功能' },
  [PERMISSION_MODULES.ASSETS]: { label: '资产管理', icon: '💻', description: 'IT资产和许可证管理', category: '核心功能' },
  [PERMISSION_MODULES.RELEASES]: { label: '发布管理', icon: '🚀', description: '发布计划和发布执行', category: '核心功能' },
  [PERMISSION_MODULES.REPORTS]: { label: '报告分析', icon: '📈', description: '数据报告和分析功能', category: '分析工具' },
  [PERMISSION_MODULES.ADMIN]: { label: '系统管理', icon: '⚙️', description: '系统管理和配置', category: '系统管理' },
  [PERMISSION_MODULES.USERS]: { label: '用户管理', icon: '👥', description: '用户账户管理', category: '系统管理' },
  [PERMISSION_MODULES.ROLES]: { label: '角色管理', icon: '🛡️', description: '角色和权限管理', category: '系统管理' },
  [PERMISSION_MODULES.GROUPS]: { label: '用户组管理', icon: '👪', description: '用户组和候选组管理', category: '系统管理' },
  [PERMISSION_MODULES.ORG]: { label: '组织架构', icon: '🏢', description: '部门、团队和组织架构管理', category: '系统管理' },
  [PERMISSION_MODULES.WORKFLOWS]: { label: '工作流', icon: '🔀', description: '业务流程配置', category: '系统管理' },
  [PERMISSION_MODULES.SYSTEM_CONFIG]: { label: '系统配置', icon: '🔧', description: '系统参数和设置', category: '系统管理' },
  [PERMISSION_MODULES.AI]: { label: 'AI能力', icon: '🤖', description: 'AI 辅助与智能自动化能力', category: '系统管理' },
};

// 权限操作配置
const ACTION_CONFIG: Record<string, { label: string; color: string; description: string }> = {
  [PERMISSION_ACTIONS.READ]: { label: '读取', color: 'blue', description: '读取和浏览权限' },
  [PERMISSION_ACTIONS.CREATE]: { label: '创建', color: 'green', description: '创建新记录权限' },
  [PERMISSION_ACTIONS.WRITE]: { label: '写入', color: 'orange', description: '创建或修改业务数据权限' },
  [PERMISSION_ACTIONS.UPDATE]: { label: '更新', color: 'orange', description: '更新现有记录权限' },
  [PERMISSION_ACTIONS.DELETE]: { label: '删除', color: 'red', description: '删除记录权限' },
  [PERMISSION_ACTIONS.MANAGE]: { label: '管理', color: 'magenta', description: '管理配置和成员关系权限' },
  [PERMISSION_ACTIONS.APPROVE]: { label: '审批', color: 'purple', description: '审批和批准权限' },
  [PERMISSION_ACTIONS.ASSIGN]: { label: '分配', color: 'cyan', description: '分配和指派权限' },
  [PERMISSION_ACTIONS.VIEW]: { label: '查看', color: 'geekblue', description: '兼容旧版查看权限' },
  [PERMISSION_ACTIONS.EXPORT]: { label: '导出', color: 'geekblue', description: '数据导出权限' },
  [PERMISSION_ACTIONS.ALL]: { label: '全部', color: 'volcano', description: '平台级全部权限' },
};

// 分类配置
const CATEGORIES = [
  { id: '核心功能', name: '核心功能', icon: <Activity className="w-4 h-4" /> },
  { id: '服务管理', name: '服务管理', icon: <Globe className="w-4 h-4" /> },
  { id: '分析工具', name: '分析工具', icon: <BarChart3 className="w-4 h-4" /> },
  { id: '系统管理', name: '系统管理', icon: <Settings className="w-4 h-4" /> },
];

// 所有可能的权限字符串
function generateAllPermissionStrings(): string[] {
  const permissions: string[] = [];
  for (const moduleKey of Object.values(PERMISSION_MODULES)) {
    for (const actionKey of Object.values(PERMISSION_ACTIONS)) {
      permissions.push(`${moduleKey}:${actionKey}`);
    }
  }
  return permissions;
}

// 类型：每个模块的动作启用状态
interface ActionState {
  [actionId: string]: boolean;
}

// 类型：每个模块的启用状态
interface ModuleState {
  isEnabled: boolean;
  actions: ActionState;
}

// 类型：整个权限配置
interface PermissionState {
  [moduleId: string]: ModuleState;
}

// 创建默认（全部启用）权限状态
function createDefaultPermissionState(): PermissionState {
  const state: PermissionState = {};
  for (const moduleKey of Object.values(PERMISSION_MODULES)) {
    const actions: ActionState = {};
    for (const actionKey of Object.values(PERMISSION_ACTIONS)) {
      actions[actionKey] = true;
    }
    state[moduleKey] = { isEnabled: true, actions };
  }
  return state;
}

// 从角色 permissions string[] 构建 PermissionState
function buildPermissionStateFromStrings(permissionStrings: string[]): PermissionState {
  const state = createDefaultPermissionState();
  // 先全部设为 false
  for (const moduleKey of Object.values(PERMISSION_MODULES)) {
    state[moduleKey].isEnabled = false;
    for (const actionKey of Object.values(PERMISSION_ACTIONS)) {
      state[moduleKey].actions[actionKey] = false;
    }
  }
  // 根据权限字符串启用
  for (const perm of permissionStrings) {
    const [moduleKey, actionKey] = perm.split(':');
    if (moduleKey && state[moduleKey]) {
      state[moduleKey].isEnabled = true;
      if (actionKey && state[moduleKey].actions.hasOwnProperty(actionKey)) {
        state[moduleKey].actions[actionKey] = true;
      }
    }
  }
  // 如果模块有任何动作启用，则模块本身标记为启用
  for (const moduleKey of Object.values(PERMISSION_MODULES)) {
    if (!state[moduleKey].isEnabled) {
      const hasEnabledAction = Object.values(state[moduleKey].actions).some(v => v);
      state[moduleKey].isEnabled = hasEnabledAction;
    }
  }
  return state;
}

// 从 PermissionState 构建权限字符串数组
function buildPermissionStringsFromState(state: PermissionState): string[] {
  const permissions: string[] = [];
  for (const [moduleKey, moduleState] of Object.entries(state)) {
    for (const [actionKey, isEnabled] of Object.entries(moduleState.actions)) {
      if (isEnabled) {
        permissions.push(`${moduleKey}:${actionKey}`);
      }
    }
  }
  return permissions;
}

const PermissionConfiguration = () => {
  const { message } = App.useApp();
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null);
  const [permissionState, setPermissionState] = useState<PermissionState>(createDefaultPermissionState());
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [hasChanges, setHasChanges] = useState(false);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'card' | 'tree'>('card');
  const [loading, setLoading] = useState(false);
  const [rolesLoading, setRolesLoading] = useState(false);
  const [permissionCatalogCount, setPermissionCatalogCount] = useState(0);

  // 加载角色列表
  const loadRoles = useCallback(async () => {
    setRolesLoading(true);
    try {
      const response = await RoleAPI.getRoles({ page: 1, size: 100 });
      setRoles(response.roles || []);
      const catalog = await RoleAPI.getPermissionCatalog();
      setPermissionCatalogCount(catalog.filter(item => item.id > 0).length);
    } catch (error) {
      console.error('Failed to load roles:', error);
      message.error('加载角色列表失败');
    } finally {
      setRolesLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadRoles();
  }, [loadRoles]);

  // 选中角色后加载其权限
  const handleSelectRole = async (roleId: number) => {
    setSelectedRoleId(roleId);
    setLoading(true);
    setHasChanges(false);
    try {
      const role = await RoleAPI.getRole(roleId);
      const permissions = role.permissions || [];
      setPermissionState(buildPermissionStateFromStrings(permissions));
    } catch (error) {
      console.error('Failed to load role permissions:', error);
      message.error('加载角色权限失败');
    } finally {
      setLoading(false);
    }
  };

  const handleInitPermissions = async () => {
    setSaving(true);
    try {
      await RoleAPI.initDefaultPermissions();
      const catalog = await RoleAPI.getPermissionCatalog();
      setPermissionCatalogCount(catalog.filter(item => item.id > 0).length);
      message.success('默认权限字典初始化完成');
    } catch (error) {
      message.error('初始化权限字典失败');
    } finally {
      setSaving(false);
    }
  };

  // 过滤模块
  const filteredModuleKeys = Object.entries(MODULE_CONFIG)
    .filter(([key, config]) => {
      const matchesSearch =
        config.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
        config.description.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesCategory = categoryFilter === 'all' || config.category === categoryFilter;
      return matchesSearch && matchesCategory;
    })
    .map(([key]) => key);

  // 按分类分组
  const modulesByCategory: Record<string, string[]> = {};
  for (const key of filteredModuleKeys) {
    const cat = MODULE_CONFIG[key].category;
    if (!modulesByCategory[cat]) modulesByCategory[cat] = [];
    modulesByCategory[cat].push(key);
  }

  // 切换模块
  const handleToggleModule = (moduleId: string) => {
    setPermissionState(prev => {
      const current = prev[moduleId];
      const newEnabled = !current.isEnabled;
      return {
        ...prev,
        [moduleId]: {
          isEnabled: newEnabled,
          actions: Object.fromEntries(
            Object.keys(current.actions).map(k => [k, newEnabled])
          ),
        },
      };
    });
    setHasChanges(true);
  };

  // 切换操作
  const handleToggleAction = (moduleId: string, actionId: string) => {
    setPermissionState(prev => ({
      ...prev,
      [moduleId]: {
        ...prev[moduleId],
        isEnabled: true,
        actions: {
          ...prev[moduleId].actions,
          [actionId]: !prev[moduleId].actions[actionId],
        },
      },
    }));
    setHasChanges(true);
  };

  // 批量操作
  const handleBatchToggle = (category: string, enabled: boolean) => {
    setPermissionState(prev => {
      const newState = { ...prev };
      for (const moduleKey of Object.keys(MODULE_CONFIG)) {
        if (MODULE_CONFIG[moduleKey].category === category) {
          newState[moduleKey] = {
            isEnabled: enabled,
            actions: Object.fromEntries(
              Object.keys(prev[moduleKey].actions).map(k => [k, enabled])
            ),
          };
        }
      }
      return newState;
    });
    setHasChanges(true);
  };

  // 保存配置到后端
  const handleSave = async () => {
    if (!selectedRoleId) {
      message.warning('请先选择一个角色');
      return;
    }
    setSaving(true);
    try {
      const permissionStrings = buildPermissionStringsFromState(permissionState);
      await RoleAPI.updateRole(selectedRoleId, { permissions: permissionStrings });
      setHasChanges(false);
      message.success('权限配置保存成功');
    } catch (error) {
      console.error('Failed to save permissions:', error);
      message.error('保存失败，请重试');
    } finally {
      setSaving(false);
    }
  };

  // 重置为角色原始权限
  const handleReset = async () => {
    if (selectedRoleId) {
      await handleSelectRole(selectedRoleId);
    } else {
      setPermissionState(createDefaultPermissionState());
      setHasChanges(false);
    }
    message.info('配置已重置');
  };

  // 统计信息
  const stats = {
    totalModules: Object.keys(MODULE_CONFIG).length,
    enabledModules: Object.values(permissionState).filter(m => m.isEnabled).length,
    totalActions: Object.values(permissionState).reduce((sum, m) => sum + Object.keys(m.actions).length, 0),
    enabledActions: Object.values(permissionState).reduce((sum, m) => sum + Object.values(m.actions).filter(v => v).length, 0),
    catalogPermissions: permissionCatalogCount,
  };

  // 生成权限树数据
  const generateTreeData = () => {
    return CATEGORIES.map(category => ({
      title: (
        <div className="flex items-center justify-between">
          <span className="flex items-center gap-2">
            {category.icon}
            <Text strong>{category.name}</Text>
          </span>
          <div className="flex gap-2">
            <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, true)}>
              全部启用
            </Button>
            <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, false)}>
              全部禁用
            </Button>
          </div>
        </div>
      ),
      key: category.id,
      children:
        modulesByCategory[category.id]?.map((moduleKey: string) => {
          const moduleState = permissionState[moduleKey];
          const moduleConfig = MODULE_CONFIG[moduleKey];
          return {
            title: (
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-2">
                  <span>{moduleConfig.icon}</span>
                  <span>{moduleConfig.label}</span>
                  <Switch
                    size="small"
                    checked={moduleState?.isEnabled || false}
                    onChange={() => handleToggleModule(moduleKey)}
                  />
                </div>
              </div>
            ),
            key: moduleKey,
            children: Object.entries(ACTION_CONFIG).map(([actionKey, actionConf]) => ({
              title: (
                <div className="flex items-center justify-between w-full">
                  <Tag color={moduleState?.actions[actionKey] ? actionConf.color : 'default'}>
                    {actionConf.label}
                  </Tag>
                  <Switch
                    size="small"
                    checked={moduleState?.actions[actionKey] || false}
                    onChange={() => handleToggleAction(moduleKey, actionKey)}
                  />
                </div>
              ),
              key: `${moduleKey}-${actionKey}`,
            })),
          };
        }) || [],
    }));
  };

  // 渲染权限卡片视图
  const renderCardView = () => (
    <div className="space-y-6">
      {CATEGORIES.map(category => {
        const categoryModuleKeys = modulesByCategory[category.id] || [];
        if (categoryModuleKeys.length === 0) return null;

        return (
          <Card
            key={category.id}
            title={
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {category.icon}
                  <span>{category.name}</span>
                  <Badge count={categoryModuleKeys.length} color="blue" />
                </div>
                <Space>
                  <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, true)}>
                    全部启用
                  </Button>
                  <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, false)}>
                    全部禁用
                  </Button>
                </Space>
              </div>
            }
            className="enterprise-card"
          >
            <Row gutter={[16, 16]}>
              {categoryModuleKeys.map((moduleKey: string) => {
                const moduleConfig = MODULE_CONFIG[moduleKey];
                const moduleState = permissionState[moduleKey];
                return (
                  <Col xs={24} md={12} lg={8} key={moduleKey}>
                    <Card
                      size="small"
                      className="h-full"
                      title={
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <span>{moduleConfig.icon}</span>
                            <span className="text-sm">{moduleConfig.label}</span>
                          </div>
                          <Switch
                            size="small"
                            checked={moduleState?.isEnabled || false}
                            onChange={() => handleToggleModule(moduleKey)}
                          />
                        </div>
                      }
                    >
                      <Text type="secondary" className="text-xs mb-3 block">
                        {moduleConfig.description}
                      </Text>
                      <div className="space-y-2">
                        {Object.entries(ACTION_CONFIG).map(([actionKey, actionConf]) => (
                          <div key={actionKey} className="flex items-center justify-between">
                            <Tooltip title={actionConf.description}>
                              <Tag
                                color={moduleState?.actions[actionKey] ? actionConf.color : 'default'}
                                className="text-xs cursor-help"
                              >
                                {actionConf.label}
                              </Tag>
                            </Tooltip>
                            <Switch
                              size="small"
                              checked={moduleState?.actions[actionKey] || false}
                              onChange={() => handleToggleAction(moduleKey, actionKey)}
                              disabled={!moduleState?.isEnabled}
                            />
                          </div>
                        ))}
                      </div>
                    </Card>
                  </Col>
                );
              })}
            </Row>
          </Card>
        );
      })}
    </div>
  );

  return (
    <div className="p-6">
      {/* 页面标题 */}
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Key className="inline-block w-6 h-6 mr-2" />
          权限配置管理
        </Title>
        <Text type="secondary">配置系统功能模块和操作权限，定义访问控制策略</Text>
      </div>

      {/* 角色选择器 */}
      <Card className="mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={6}>
            <div className="flex items-center gap-2">
              <Text strong>选择角色：</Text>
              <Select
                placeholder="请选择角色"
                value={selectedRoleId || undefined}
                onChange={handleSelectRole}
                loading={rolesLoading}
                style={{ minWidth: 200 }}
                showSearch
                optionFilterProp="children"
              >
                {roles.map(role => (
                  <Option key={role.id} value={role.id}>
                    {role.name}
                  </Option>
                ))}
              </Select>
            </div>
          </Col>
          <Col xs={24} md={6}>
            <Input
              placeholder="搜索模块或权限..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder="筛选分类"
              value={categoryFilter}
              onChange={setCategoryFilter}
              style={{ width: '100%' }}
            >
              <Option value="all">全部分类</Option>
              {CATEGORIES.map(category => (
                <Option key={category.id} value={category.id}>
                  {category.name}
                </Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder="视图模式"
              value={viewMode}
              onChange={setViewMode}
              style={{ width: '100%' }}
            >
              <Option value="card">卡片视图</Option>
              <Option value="tree">树形视图</Option>
            </Select>
          </Col>
          <Col xs={24} md={4} className="text-right">
            <Space>
              <Button onClick={handleInitPermissions} loading={saving}>
                初始化权限字典
              </Button>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>
                重置
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                loading={saving}
                onClick={handleSave}
                disabled={!hasChanges || !selectedRoleId}
              >
                {saving ? '保存中...' : '保存配置'}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 未选择角色提示 */}
      {!selectedRoleId && (
        <Alert
          message="请先选择角色"
          description="从上方下拉框中选择一个角色，以查看和编辑该角色的权限配置。"
          type="info"
          showIcon
          className="mb-6"
        />
      )}

      {permissionCatalogCount === 0 && (
        <Alert
          message="权限字典尚未初始化"
          description="当前租户没有可用于真实授权的权限字典。请点击“初始化权限字典”，否则角色权限保存可能无法落到后端关联表。"
          type="warning"
          showIcon
          className="mb-6"
        />
      )}

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="功能模块"
              value={stats.enabledModules}
              suffix={`/ ${stats.totalModules}`}
              prefix={<Layers className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="操作权限"
              value={stats.enabledActions}
              suffix={`/ ${stats.totalActions}`}
              prefix={<Activity className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="启用模块"
              value={((stats.enabledModules / stats.totalModules) * 100).toFixed(1)}
              suffix="%"
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="权限覆盖率"
              value={((stats.enabledActions / stats.totalActions) * 100).toFixed(1)}
              suffix="%"
              prefix={<Shield className="w-5 h-5" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 配置变更提醒 */}
      {hasChanges && selectedRoleId && (
        <Alert
          message="配置已修改"
          description="您有未保存的权限配置更改，请及时保存。"
          type="warning"
          showIcon
          closable
          className="mb-6"
        />
      )}

      {/* 权限配置内容 */}
      <Card className="enterprise-card">
        {loading ? (
          <div className="flex justify-center items-center py-20">
            <Spin size="large" tip="加载权限数据..." />
          </div>
        ) : viewMode === 'card' ? (
          renderCardView()
        ) : (
          <Tree
            treeData={generateTreeData()}
            defaultExpandAll
            showLine
            className="enterprise-tree"
          />
        )}
      </Card>
    </div>
  );
};

export default PermissionConfiguration;

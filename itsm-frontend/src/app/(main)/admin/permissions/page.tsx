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
  Ticket,
  AlertCircle,
  Wrench,
  BookOpen,
  FileText,
  Book,
  Database,
  Monitor,
  Rocket,
  LineChart,
  Users,
  Building,
  GitMerge,
  SlidersHorizontal,
  Bot,
  Tag as TagIcon,
  KeyRound,
} from 'lucide-react';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
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
import { useI18n } from '@/lib/i18n/useI18n';
const { Title, Text } = Typography;

// 权限模块定义
const PERMISSION_MODULES = {
  DASHBOARD: 'dashboard',
  TICKETS: 'ticket',
  TICKET_CATEGORY: 'ticket_category',
  INCIDENTS: 'incident',
  PROBLEMS: 'problem',
  CHANGES: 'change',
  SERVICE_CATALOG: 'service_catalog',
  SERVICE_REQUEST: 'service_request',
  KNOWLEDGE_BASE: 'knowledge',
  CMDB: 'cmdb',
  ASSETS: 'asset',
  LICENSE: 'license',
  TICKET_TYPE: 'ticket_type',
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
  for (const moduleKey of Object.values(PERMISSION_MODULES)) {
    state[moduleKey].isEnabled = false;
    for (const actionKey of Object.values(PERMISSION_ACTIONS)) {
      state[moduleKey].actions[actionKey] = false;
    }
  }
  for (const perm of permissionStrings) {
    const [moduleKey, actionKey] = perm.split(':');
    if (moduleKey && state[moduleKey]) {
      state[moduleKey].isEnabled = true;
      if (actionKey && state[moduleKey].actions.hasOwnProperty(actionKey)) {
        state[moduleKey].actions[actionKey] = true;
      }
    }
  }
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

// 模块图标与分类映射（静态部分）
// 使用 lucide 图标组件，保证在所有环境下稳定渲染（避免 emoji 在部分字体下显示为占位符/文本）
const MODULE_META: Record<string, { icon: React.ComponentType<{ className?: string; size?: number }>; categoryKey: 'core' | 'service' | 'analysis' | 'system' }> = {
  [PERMISSION_MODULES.DASHBOARD]: { icon: BarChart3, categoryKey: 'core' },
  [PERMISSION_MODULES.TICKETS]: { icon: Ticket, categoryKey: 'core' },
  [PERMISSION_MODULES.INCIDENTS]: { icon: AlertCircle, categoryKey: 'core' },
  [PERMISSION_MODULES.PROBLEMS]: { icon: Wrench, categoryKey: 'core' },
  [PERMISSION_MODULES.CHANGES]: { icon: RefreshCw, categoryKey: 'core' },
  [PERMISSION_MODULES.SERVICE_CATALOG]: { icon: BookOpen, categoryKey: 'service' },
  [PERMISSION_MODULES.SERVICE_REQUEST]: { icon: FileText, categoryKey: 'service' },
  [PERMISSION_MODULES.KNOWLEDGE_BASE]: { icon: Book, categoryKey: 'service' },
  [PERMISSION_MODULES.CMDB]: { icon: Database, categoryKey: 'core' },
  [PERMISSION_MODULES.ASSETS]: { icon: Monitor, categoryKey: 'core' },
  [PERMISSION_MODULES.LICENSE]: { icon: KeyRound, categoryKey: 'core' },
  [PERMISSION_MODULES.TICKET_CATEGORY]: { icon: TagIcon, categoryKey: 'core' },
  [PERMISSION_MODULES.TICKET_TYPE]: { icon: Layers, categoryKey: 'core' },
  [PERMISSION_MODULES.RELEASES]: { icon: Rocket, categoryKey: 'core' },
  [PERMISSION_MODULES.REPORTS]: { icon: LineChart, categoryKey: 'analysis' },
  [PERMISSION_MODULES.ADMIN]: { icon: Settings, categoryKey: 'system' },
  [PERMISSION_MODULES.USERS]: { icon: Users, categoryKey: 'system' },
  [PERMISSION_MODULES.ROLES]: { icon: Shield, categoryKey: 'system' },
  [PERMISSION_MODULES.GROUPS]: { icon: Users, categoryKey: 'system' },
  [PERMISSION_MODULES.ORG]: { icon: Building, categoryKey: 'system' },
  [PERMISSION_MODULES.WORKFLOWS]: { icon: GitMerge, categoryKey: 'system' },
  [PERMISSION_MODULES.SYSTEM_CONFIG]: { icon: SlidersHorizontal, categoryKey: 'system' },
  [PERMISSION_MODULES.AI]: { icon: Bot, categoryKey: 'system' },
};

const ACTION_COLOR: Record<string, string> = {
  [PERMISSION_ACTIONS.READ]: 'blue',
  [PERMISSION_ACTIONS.CREATE]: 'green',
  [PERMISSION_ACTIONS.WRITE]: 'orange',
  [PERMISSION_ACTIONS.UPDATE]: 'orange',
  [PERMISSION_ACTIONS.DELETE]: 'red',
  [PERMISSION_ACTIONS.MANAGE]: 'magenta',
  [PERMISSION_ACTIONS.APPROVE]: 'purple',
  [PERMISSION_ACTIONS.ASSIGN]: 'cyan',
  [PERMISSION_ACTIONS.VIEW]: 'geekblue',
  [PERMISSION_ACTIONS.EXPORT]: 'geekblue',
  [PERMISSION_ACTIONS.ALL]: 'volcano',
};

const PermissionConfiguration = () => {
  const { t } = useI18n();
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

  // 分类配置（运行时国际化）
  const categories = useMemo(
    () => [
      { id: t('permissions.categories.core'), key: 'core', icon: <Activity className="w-4 h-4" /> },
      { id: t('permissions.categories.service'), key: 'service', icon: <Globe className="w-4 h-4" /> },
      { id: t('permissions.categories.analysis'), key: 'analysis', icon: <BarChart3 className="w-4 h-4" /> },
      { id: t('permissions.categories.system'), key: 'system', icon: <Settings className="w-4 h-4" /> },
    ],
    [t]
  );

  // 模块配置（运行时国际化）
  const moduleConfig = useMemo(() => {
    const result: Record<string, { label: string; icon: React.ComponentType<{ className?: string; size?: number }>; description: string; category: string; categoryKey: string }> = {};
    for (const moduleKey of Object.values(PERMISSION_MODULES)) {
      const meta = MODULE_META[moduleKey];
      const categoryName = t(`permissions.categories.${meta.categoryKey}`);
      result[moduleKey] = {
        label: t(`permissions.modules.${moduleKey.toUpperCase()}.label`),
        icon: meta.icon,
        description: t(`permissions.modules.${moduleKey.toUpperCase()}.description`),
        category: categoryName,
        categoryKey: meta.categoryKey,
      };
    }
    return result;
  }, [t]);

  // 操作配置（运行时国际化）
  const actionConfig = useMemo(() => {
    const result: Record<string, { label: string; color: string; description: string }> = {};
    for (const actionKey of Object.values(PERMISSION_ACTIONS)) {
      result[actionKey] = {
        label: t(`permissions.actions.${actionKey.toUpperCase()}`),
        color: ACTION_COLOR[actionKey],
        description: t(`permissions.actionDescriptions.${actionKey.toUpperCase()}`),
      };
    }
    return result;
  }, [t]);

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
      message.error(t('permissions.messages.loadRolesFailed'));
    } finally {
      setRolesLoading(false);
    }
  }, [message, t]);

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
      message.error(t('permissions.messages.loadPermissionsFailed'));
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
      message.success(t('permissions.messages.initSuccess'));
    } catch (error) {
      message.error(t('permissions.messages.initFailed'));
    } finally {
      setSaving(false);
    }
  };

  // 过滤模块
  const filteredModuleKeys = Object.entries(moduleConfig)
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
    const cat = moduleConfig[key].category;
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
      for (const moduleKey of Object.keys(moduleConfig)) {
        if (moduleConfig[moduleKey].category === category) {
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
      message.warning(t('permissions.messages.selectRoleFirst'));
      return;
    }
    setSaving(true);
    try {
      const permissionStrings = buildPermissionStringsFromState(permissionState);
      await RoleAPI.updateRole(selectedRoleId, { permissions: permissionStrings });
      setHasChanges(false);
      message.success(t('permissions.messages.saveSuccess'));
    } catch (error) {
      console.error('Failed to save permissions:', error);
      message.error(t('permissions.messages.saveFailed'));
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
    message.info(t('permissions.messages.reset'));
  };

  // 统计信息
  const stats = {
    totalModules: Object.keys(moduleConfig).length,
    enabledModules: Object.values(permissionState).filter(m => m.isEnabled).length,
    totalActions: Object.values(permissionState).reduce((sum, m) => sum + Object.keys(m.actions).length, 0),
    enabledActions: Object.values(permissionState).reduce((sum, m) => sum + Object.values(m.actions).filter(v => v).length, 0),
    catalogPermissions: permissionCatalogCount,
  };

  // 生成权限树数据
  const generateTreeData = () => {
    return categories.map(category => ({
      title: (
        <div className="flex items-center justify-between">
          <span className="flex items-center gap-2">
            {category.icon}
            <Text strong>{category.id}</Text>
          </span>
          <div className="flex gap-2">
            <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, true)}>
              {t('permissions.buttons.enableAll')}
            </Button>
            <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, false)}>
              {t('permissions.buttons.disableAll')}
            </Button>
          </div>
        </div>
      ),
      key: category.id,
      children:
        modulesByCategory[category.id]?.map((moduleKey: string) => {
          const moduleState = permissionState[moduleKey];
          const modConf = moduleConfig[moduleKey];
          return {
            title: (
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-2">
                  <modConf.icon className="w-4 h-4" />
                  <span>{modConf.label}</span>
                  <Switch
                    size="small"
                    checked={moduleState?.isEnabled || false}
                    onChange={() => handleToggleModule(moduleKey)}
                  />
                </div>
              </div>
            ),
            key: moduleKey,
            children: Object.entries(actionConfig).map(([actionKey, actionConf]) => ({
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
      {categories.map(category => {
        const categoryModuleKeys = modulesByCategory[category.id] || [];
        if (categoryModuleKeys.length === 0) return null;

        return (
          <Card
            key={category.id}
            title={
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {category.icon}
                  <span>{category.id}</span>
                  <Badge count={categoryModuleKeys.length} color="blue" />
                </div>
                <Space>
                  <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, true)}>
                    {t('permissions.buttons.enableAll')}
                  </Button>
                  <Button size="small" type="link" onClick={() => handleBatchToggle(category.id, false)}>
                    {t('permissions.buttons.disableAll')}
                  </Button>
                </Space>
              </div>
            }
            className="enterprise-card"
          >
            <Row gutter={[16, 16]}>
              {categoryModuleKeys.map((moduleKey: string) => {
                const modConf = moduleConfig[moduleKey];
                const moduleState = permissionState[moduleKey];
                return (
                  <Col xs={24} md={12} lg={8} key={moduleKey}>
                    <Card
                      size="small"
                      className="h-full"
                      title={
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <modConf.icon className="w-4 h-4" />
                            <span className="text-sm">{modConf.label}</span>
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
                        {modConf.description}
                      </Text>
                      <div className="space-y-2">
                        {Object.entries(actionConfig).map(([actionKey, actionConf]) => (
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
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <Title level={2} className="!mb-2">
          <Key className="inline-block w-6 h-6 mr-2" />
          {t('permissions.title')}
        </Title>
        <Text type="secondary">{t('permissions.description')}</Text>
      </div>

      {/* 角色选择器 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={6}>
            <div className="flex items-center gap-2">
              <Text strong>{t('permissions.selectRole')}</Text>
              <Select
                placeholder={t('permissions.placeholders.selectRole')}
                value={selectedRoleId || undefined}
                onChange={handleSelectRole}
                loading={rolesLoading}
                style={{ minWidth: 200 }}
                showSearch
                optionFilterProp="children"
                options={roles.map(role => ({ value: role.id, label: role.name }))}
              />
            </div>
          </Col>
          <Col xs={24} md={6}>
            <Input
              placeholder={t('permissions.placeholders.searchModules')}
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder={t('permissions.placeholders.filterCategory')}
              value={categoryFilter}
              onChange={setCategoryFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: t('permissions.allCategories') },
                ...categories.map(category => ({ value: category.id, label: category.id })),
              ]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder={t('permissions.placeholders.viewMode')}
              value={viewMode}
              onChange={setViewMode}
              style={{ width: '100%' }}
              options={[
                { value: 'card', label: t('permissions.viewModes.card') },
                { value: 'tree', label: t('permissions.viewModes.tree') },
              ]}
            />
          </Col>
          <Col xs={24} md={4} className="text-right">
            <Space>
              <Button onClick={handleInitPermissions} loading={saving}>
                {t('permissions.buttons.init')}
              </Button>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>
                {t('permissions.buttons.reset')}
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                loading={saving}
                onClick={handleSave}
                disabled={!hasChanges || !selectedRoleId}
              >
                {saving ? t('permissions.buttons.saving') : t('permissions.buttons.save')}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 未选择角色提示 */}
      {!selectedRoleId && (
        <Alert
          message={t('permissions.alerts.selectRoleMessage')}
          description={t('permissions.alerts.selectRoleDescription')}
          type="info"
          showIcon
        />
      )}

      {permissionCatalogCount === 0 && (
        <Alert
          message={t('permissions.alerts.catalogNotInitMessage')}
          description={t('permissions.alerts.catalogNotInitDescription')}
          type="warning"
          showIcon
        />
      )}

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('permissions.stats.modules')}
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
              title={t('permissions.stats.actions')}
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
              title={t('permissions.stats.enabledModules')}
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
              title={t('permissions.stats.coverage')}
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
          message={t('permissions.alerts.unsavedMessage')}
          description={t('permissions.alerts.unsavedDescription')}
          type="warning"
          showIcon
          closable
        />
      )}

      {/* 权限配置内容 */}
      <Card className="enterprise-card">
        {loading ? (
          <div className="flex justify-center items-center py-20">
            <Spin size="large" tip={t('permissions.loading')} />
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

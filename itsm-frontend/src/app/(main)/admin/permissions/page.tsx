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

import React, { useState, useEffect } from 'react';
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
  Collapse,
  Checkbox,
  Badge,
  Tag,
  Tooltip,
  Alert,
  App,
  Tabs,
  Tree,
} from 'antd';
import { RoleAPI } from '@/lib/api/role-api';
const { Title, Text } = Typography;
const { Option } = Select;
const { Panel } = Collapse;

// 类型定义
interface PermissionAction {
  id: string;
  name: string;
  description: string;
  color: string;
  isEnabled: boolean;
}

interface PermissionModule {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: React.ReactNode;
  isEnabled: boolean;
  actions: PermissionAction[];
}

interface PermissionCategory {
  id: string;
  name: string;
  description: string;
  icon: React.ReactNode;
}

interface PermissionConfig {
  modules: PermissionModule[];
  categories: PermissionCategory[];
}

// 权限模块定义
const PERMISSION_MODULES = {
  DASHBOARD: 'dashboard',
  TICKETS: 'tickets',
  INCIDENTS: 'incidents',
  PROBLEMS: 'problems',
  CHANGES: 'changes',
  SERVICE_CATALOG: 'service_catalog',
  KNOWLEDGE_BASE: 'knowledge_base',
  REPORTS: 'reports',
  ADMIN: 'admin',
  USERS: 'users',
  ROLES: 'roles',
  WORKFLOWS: 'workflows',
  SYSTEM_CONFIG: 'system_config',
} as const;

// 权限操作类型
const PERMISSION_ACTIONS = {
  VIEW: 'view',
  CREATE: 'create',
  EDIT: 'edit',
  DELETE: 'delete',
  APPROVE: 'approve',
  ASSIGN: 'assign',
  EXPORT: 'export',
} as const;

// 权限模块配置
const MODULE_CONFIG = {
  [PERMISSION_MODULES.DASHBOARD]: {
    label: '仪表盘',
    icon: '📊',
    description: '系统仪表盘和概览信息',
    category: '核心功能',
  },
  [PERMISSION_MODULES.TICKETS]: {
    label: '工单管理',
    icon: '🎫',
    description: '工单的创建、处理和管理',
    category: '核心功能',
  },
  [PERMISSION_MODULES.INCIDENTS]: {
    label: '事件管理',
    icon: '🚨',
    description: 'IT事件的记录和处理',
    category: '核心功能',
  },
  [PERMISSION_MODULES.PROBLEMS]: {
    label: '问题管理',
    icon: '🔧',
    description: '根本原因分析和问题解决',
    category: '核心功能',
  },
  [PERMISSION_MODULES.CHANGES]: {
    label: '变更管理',
    icon: '🔄',
    description: 'IT变更的规划和实施',
    category: '核心功能',
  },
  [PERMISSION_MODULES.SERVICE_CATALOG]: {
    label: '服务目录',
    icon: '📋',
    description: 'IT服务目录管理',
    category: '服务管理',
  },
  [PERMISSION_MODULES.KNOWLEDGE_BASE]: {
    label: '知识库',
    icon: '📚',
    description: '知识文档和解决方案',
    category: '服务管理',
  },
  [PERMISSION_MODULES.REPORTS]: {
    label: '报告分析',
    icon: '📈',
    description: '数据报告和分析功能',
    category: '分析工具',
  },
  [PERMISSION_MODULES.ADMIN]: {
    label: '系统管理',
    icon: '⚙️',
    description: '系统管理和配置',
    category: '系统管理',
  },
  [PERMISSION_MODULES.USERS]: {
    label: '用户管理',
    icon: '👥',
    description: '用户账户管理',
    category: '系统管理',
  },
  [PERMISSION_MODULES.ROLES]: {
    label: '角色管理',
    icon: '🛡️',
    description: '角色和权限管理',
    category: '系统管理',
  },
  [PERMISSION_MODULES.WORKFLOWS]: {
    label: '工作流',
    icon: '🔀',
    description: '业务流程配置',
    category: '系统管理',
  },
  [PERMISSION_MODULES.SYSTEM_CONFIG]: {
    label: '系统配置',
    icon: '🔧',
    description: '系统参数和设置',
    category: '系统管理',
  },
};

// 权限操作配置
const ACTION_CONFIG = {
  [PERMISSION_ACTIONS.VIEW]: {
    label: '查看',
    color: 'blue',
    description: '查看和浏览权限',
  },
  [PERMISSION_ACTIONS.CREATE]: {
    label: '创建',
    color: 'green',
    description: '创建新记录权限',
  },
  [PERMISSION_ACTIONS.EDIT]: {
    label: '编辑',
    color: 'orange',
    description: '修改现有记录权限',
  },
  [PERMISSION_ACTIONS.DELETE]: {
    label: '删除',
    color: 'red',
    description: '删除记录权限',
  },
  [PERMISSION_ACTIONS.APPROVE]: {
    label: '审批',
    color: 'purple',
    description: '审批和批准权限',
  },
  [PERMISSION_ACTIONS.ASSIGN]: {
    label: '分配',
    color: 'cyan',
    description: '分配和指派权限',
  },
  [PERMISSION_ACTIONS.EXPORT]: {
    label: '导出',
    color: 'geekblue',
    description: '数据导出权限',
  },
};

// 模拟权限配置数据
const mockPermissionConfig: PermissionConfig = {
  modules: Object.values(PERMISSION_MODULES).map(moduleKey => ({
    id: moduleKey,
    name: MODULE_CONFIG[moduleKey].label,
    description: MODULE_CONFIG[moduleKey].description,
    category: MODULE_CONFIG[moduleKey].category,
    icon: MODULE_CONFIG[moduleKey].icon,
    isEnabled: true,
    actions: Object.values(PERMISSION_ACTIONS).map(actionKey => ({
      id: actionKey,
      name: ACTION_CONFIG[actionKey].label,
      description: ACTION_CONFIG[actionKey].description,
      color: ACTION_CONFIG[actionKey].color,
      isEnabled: true,
    })),
  })),
  categories: [
    {
      id: '核心功能',
      name: '核心功能',
      description: '系统核心业务功能',
      icon: <Activity className="w-4 h-4" />,
    },
    {
      id: '服务管理',
      name: '服务管理',
      description: 'IT服务相关功能',
      icon: <Globe className="w-4 h-4" />,
    },
    {
      id: '分析工具',
      name: '分析工具',
      description: '数据分析和报告',
      icon: <BarChart3 className="w-4 h-4" />,
    },
    {
      id: '系统管理',
      name: '系统管理',
      description: '系统配置和管理',
      icon: <Settings className="w-4 h-4" />,
    },
  ],
};

const PermissionConfiguration = () => {
  const { message } = App.useApp();
  const [permissionConfig, setPermissionConfig] = useState<PermissionConfig>(mockPermissionConfig);
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [hasChanges, setHasChanges] = useState(false);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'card' | 'tree'>('card');
  const [loading, setLoading] = useState(false);

  // 加载权限数据
  const loadPermissions = async () => {
    setLoading(true);
    try {
      const permissions = await RoleAPI.getPermissions();
      // 如果API返回权限列表，可以用来更新配置
      if (permissions && permissions.length > 0) {
      }
    } catch (error) {
      message.error('加载权限失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPermissions();
  }, []);

  // 过滤模块
  const filteredModules = permissionConfig.modules.filter(module => {
    const matchesSearch =
      module.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      module.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = categoryFilter === 'all' || module.category === categoryFilter;
    return matchesSearch && matchesCategory;
  });

  // 按分类分组模块
  const modulesByCategory = filteredModules.reduce((acc: Record<string, PermissionModule[]>, module) => {
    if (!acc[module.category]) {
      acc[module.category] = [];
    }
    acc[module.category].push(module);
    return acc;
  }, {} as Record<string, PermissionModule[]>);

  // 处理模块状态切换
  const handleToggleModule = (moduleId: string) => {
    setPermissionConfig(prev => ({
      ...prev,
      modules: prev.modules.map(module =>
        module.id === moduleId ? { ...module, isEnabled: !module.isEnabled } : module
      ),
    }));
    setHasChanges(true);
  };

  // 处理操作状态切换
  const handleToggleAction = (moduleId: string, actionId: string) => {
    setPermissionConfig(prev => ({
      ...prev,
      modules: prev.modules.map(module =>
        module.id === moduleId
          ? {
              ...module,
              actions: module.actions.map(action =>
                action.id === actionId ? { ...action, isEnabled: !action.isEnabled } : action
              ),
            }
          : module
      ),
    }));
    setHasChanges(true);
  };

  // 批量操作
  const handleBatchToggle = (category: string, enabled: boolean) => {
    setPermissionConfig(prev => ({
      ...prev,
      modules: prev.modules.map(module =>
        module.category === category ? { ...module, isEnabled: enabled } : module
      ),
    }));
    setHasChanges(true);
  };

  // 保存配置
  const handleSave = async () => {
    setSaving(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1500));
      setHasChanges(false);
      message.success('权限配置保存成功！');
    } catch (error) {
      message.error('保存失败，请重试！');
    } finally {
      setSaving(false);
    }
  };

  // 重置配置
  const handleReset = () => {
    setPermissionConfig(mockPermissionConfig);
    setHasChanges(false);
    message.info('配置已重置');
  };

  // 统计信息
  const stats = {
    totalModules: permissionConfig.modules.length,
    enabledModules: permissionConfig.modules.filter(m => m.isEnabled).length,
    totalActions: permissionConfig.modules.reduce((sum, module) => sum + module.actions.length, 0),
    enabledActions: permissionConfig.modules.reduce(
      (sum, module) => sum + module.actions.filter(a => a.isEnabled).length,
      0
    ),
  };

  // 生成权限树数据
  const generateTreeData = () => {
    return permissionConfig.categories.map(category => ({
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
        modulesByCategory[category.id]?.map((module: PermissionModule) => ({
          title: (
            <div className="flex items-center justify-between w-full">
              <div className="flex items-center gap-2">
                <span>{module.icon}</span>
                <span>{module.name}</span>
                <Switch
                  size="small"
                  checked={module.isEnabled}
                  onChange={() => handleToggleModule(module.id)}
                />
              </div>
            </div>
          ),
          key: module.id,
          children: module.actions.map((action: PermissionAction) => ({
            title: (
              <div className="flex items-center justify-between w-full">
                <Tag color={action.color}>{action.name}</Tag>
                <Switch
                  size="small"
                  checked={action.isEnabled}
                  onChange={() => handleToggleAction(module.id, action.id)}
                />
              </div>
            ),
            key: `${module.id}-${action.id}`,
          })),
        })) || [],
    }));
  };

  // 渲染权限卡片视图
  const renderCardView = () => (
    <div className="space-y-6">
      {permissionConfig.categories.map(category => {
        const categoryModules = modulesByCategory[category.id] || [];
        if (categoryModules.length === 0) return null;

        return (
          <Card
            key={category.id}
            title={
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {category.icon}
                  <span>{category.name}</span>
                  <Badge count={categoryModules.length} color="blue" />
                </div>
                <Space>
                  <Button
                    size="small"
                    type="link"
                    onClick={() => handleBatchToggle(category.id, true)}
                  >
                    全部启用
                  </Button>
                  <Button
                    size="small"
                    type="link"
                    onClick={() => handleBatchToggle(category.id, false)}
                  >
                    全部禁用
                  </Button>
                </Space>
              </div>
            }
            className="enterprise-card"
          >
            <Row gutter={[16, 16]}>
              {categoryModules.map((module: PermissionModule) => (
                <Col xs={24} md={12} lg={8} key={module.id}>
                  <Card
                    size="small"
                    className="h-full"
                    title={
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span>{module.icon}</span>
                          <span className="text-sm">{module.name}</span>
                        </div>
                        <Switch
                          size="small"
                          checked={module.isEnabled}
                          onChange={() => handleToggleModule(module.id)}
                        />
                      </div>
                    }
                  >
                    <Text type="secondary" className="text-xs mb-3 block">
                      {module.description}
                    </Text>
                    <div className="space-y-2">
                      {module.actions.map((action: PermissionAction) => (
                        <div key={action.id} className="flex items-center justify-between">
                          <Tooltip title={action.description}>
                            <Tag
                              color={action.isEnabled ? action.color : 'default'}
                              className="text-xs cursor-help"
                            >
                              {action.name}
                            </Tag>
                          </Tooltip>
                          <Switch
                            size="small"
                            checked={action.isEnabled}
                            onChange={() => handleToggleAction(module.id, action.id)}
                            disabled={!module.isEnabled}
                          />
                        </div>
                      ))}
                    </div>
                  </Card>
                </Col>
              ))}
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
      {hasChanges && (
        <Alert
          message="配置已修改"
          description="您有未保存的权限配置更改，请及时保存。"
          type="warning"
          showIcon
          closable
          className="mb-6"
        />
      )}

      {/* 搜索和操作栏 */}
      <Card className="mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="搜索模块或权限..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={6}>
            <Select
              placeholder="筛选分类"
              value={categoryFilter}
              onChange={setCategoryFilter}
              style={{ width: '100%' }}
            >
              <Option value="all">全部分类</Option>
              {permissionConfig.categories.map(category => (
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
          <Col xs={24} md={6} className="text-right">
            <Space>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>
                重置
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                loading={saving}
                onClick={handleSave}
                disabled={!hasChanges}
              >
                {saving ? '保存中...' : '保存配置'}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 权限配置内容 */}
      <Card className="enterprise-card">
        {viewMode === 'card' ? (
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

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
  Modal,
} from 'antd';
import { RoleAPI } from '@/lib/api/role-api';
import type { PermissionCatalogItem, Role, PermissionCatalogMeta, PermissionPolicyPayload } from '@/lib/api/api-config';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { PERM } from '@/lib/permissions';
import { useI18n } from '@/lib/i18n/useI18n';
const { Title, Text } = Typography;

// 权限模块定义（导出供契约测试断言 i18n key 完整性）
export const PERMISSION_MODULES = {
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

// 翻译字典以常量名（TICKETS/KNOWLEDGE_BASE/WORKFLOWS 等）为 key，而 moduleKey
// 是权限码（ticket/knowledge/bpmn 等），直接 toUpperCase() 会导致 11 个模块
// 查不到翻译、渲染出原始 key。此处建立权限码 → i18n key 的显式映射。
export const MODULE_I18N_KEYS: Record<string, string> = Object.fromEntries(
  Object.entries(PERMISSION_MODULES).map(([name, value]) => [value, name]),
);

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

// 可授权的码空间完全由后端 GET /permissions 目录派生。硬编码的
// PERMISSION_MODULES / PERMISSION_ACTIONS 只承担 i18n key 与配色元数据，
// 不再用于生成权限码：历史上前端把 23 个模块 × 11 个动作做笛卡尔积提交，
// 其中大量组合（dashboard:create、groups:* 等）在 catalog 中根本不存在，
// 后端 permissionIDsByCodes 会整体拒绝；即便侥幸通过，也会覆盖掉角色上
// 未被矩阵展示的真实权限（ticket:assign、sla:*、msp:* …），造成权限丢失。
interface CatalogModule {
  resource: string;
  actions: string[];
}

function deriveCatalogModules(catalog: PermissionCatalogItem[]): CatalogModule[] {
  const grouped = new Map<string, string[]>();
  for (const item of catalog) {
    const resource = item.resource?.trim();
    const action = item.action?.trim();
    if (!resource || !action) continue;
    const actions = grouped.get(resource) ?? [];
    if (!actions.includes(action)) actions.push(action);
    grouped.set(resource, actions);
  }
  return Array.from(grouped, ([resource, actions]) => ({ resource, actions }));
}

// 全局动作并集只用于查标签/配色；每个模块实际可勾选的动作仍以
// module.actions 为准，避免渲染出该资源并不存在的组合。
function deriveCatalogActions(modules: CatalogModule[]): string[] {
  const actions: string[] = [];
  const seen = new Set<string>();
  for (const action of [...Object.values(PERMISSION_ACTIONS), ...modules.flatMap(m => m.actions)]) {
    if (seen.has(action)) continue;
    seen.add(action);
    actions.push(action);
  }
  return actions;
}

// 未选择角色/目录未就绪时用空白状态，不能展示“全部启用”误导当前快照。
function createPermissionState(modules: CatalogModule[], enabled: boolean): PermissionState {
  const state: PermissionState = {};
  for (const mod of modules) {
    const actions: ActionState = {};
    for (const action of mod.actions) actions[action] = enabled;
    state[mod.resource] = { isEnabled: enabled, actions };
  }
  return state;
}

function buildPermissionStateFromStrings(
  permissionStrings: string[],
  modules: CatalogModule[],
): PermissionState {
  const state = createPermissionState(modules, false);
  for (const perm of permissionStrings) {
    const [resource, action] = perm.split(':');
    const moduleState = state[resource];
    if (!moduleState) continue;
    moduleState.isEnabled = true;
    if (action && Object.prototype.hasOwnProperty.call(moduleState.actions, action)) {
      moduleState.actions[action] = true;
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
  // 策略编辑与后端 PUT /permissions/:id 的 role:write 同码；无权限只隐藏入口。
  const { hasPermission } = usePermissions();
  const canEditPolicy = hasPermission(PERM.role.write);
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null);
  const [permissionState, setPermissionState] = useState<PermissionState>({});
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [hasChanges, setHasChanges] = useState(false);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'card' | 'tree'>('card');
  const [loading, setLoading] = useState(false);
  const [rolesLoading, setRolesLoading] = useState(false);
  const [permissionCatalog, setPermissionCatalog] = useState<PermissionCatalogItem[]>([]);
  const [permissionCatalogCount, setPermissionCatalogCount] = useState(0);

  // 运行期策略：AI 主体授权汇总 + 策略编辑弹层
  const [catalogMeta, setCatalogMeta] = useState<PermissionCatalogMeta | null>(null);
  const [policyEditing, setPolicyEditing] = useState<PermissionCatalogItem | null>(null);
  const [policyDraft, setPolicyDraft] = useState<PermissionPolicyPayload | null>(null);
  const [policySaving, setPolicySaving] = useState(false);

  // 码 → 目录条目索引（策略徽标渲染用）
  const catalogByCode = useMemo(() => {
    const map = new Map<string, PermissionCatalogItem>();
    for (const item of permissionCatalog) map.set(item.code, item);
    return map;
  }, [permissionCatalog]);

  // 已开放给非人类主体（agent/service）的权限码。AI 默认零权限，
  // 出现在这里即管理员显式授权过——页面必须显眼展示，配合审批闭环审计。
  const aiAuthorizedCodes = useMemo(
    () =>
      permissionCatalog.filter(
        item =>
          item.allowedPrincipals?.includes('agent') === true ||
          item.allowedPrincipals?.includes('service') === true,
      ),
    [permissionCatalog],
  );

  const openPolicyModal = (item: PermissionCatalogItem) => {
    if (!canEditPolicy) return;
    setPolicyEditing(item);
    setPolicyDraft({
      riskLevel: item.riskLevel ?? 'medium',
      allowedPrincipals: item.allowedPrincipals?.length ? [...item.allowedPrincipals] : ['human'],
      requiresApproval: item.requiresApproval ?? false,
      minApprovals: item.minApprovals ?? 0,
      category: item.category ?? 'other',
      displayName: item.displayName ?? '',
      icon: item.icon ?? '',
      color: item.color ?? '',
      i18nKey: item.i18nKey ?? '',
      sortOrder: item.sortOrder ?? 0,
      isActive: item.isActive ?? true,
    });
  };

  const handleSavePolicy = async () => {
    if (!policyEditing || !policyDraft) return;
    setPolicySaving(true);
    try {
      await RoleAPI.updatePermissionPolicy(policyEditing.id, { policy: policyDraft });
      message.success(`权限 ${policyEditing.code} 策略已更新，立即生效`);
      setPolicyEditing(null);
      setPolicyDraft(null);
      // 策略改完立即回源目录，让风险/AI/审批徽标反映最新值
      const catalog = await RoleAPI.getPermissionCatalog();
      setPermissionCatalog(catalog);
    } catch (error) {
      // 后端 ValidatePolicy 会拒绝非法组合（如 AI 执行高风险却关审批）
      message.error(`策略保存失败：${(error as Error).message}`);
    } finally {
      setPolicySaving(false);
    }
  };

  const RISK_OPTIONS = [
    { value: 'low', label: '低风险' },
    { value: 'medium', label: '中风险' },
    { value: 'high', label: '高风险' },
    { value: 'critical', label: '危急' },
  ];
  const PRINCIPAL_OPTIONS = [
    { value: 'human', label: '人' },
    { value: 'agent', label: 'AI 代理' },
    { value: 'service', label: '服务账号' },
  ];

  /** 风险/AI/审批徽标：展示权限码的运行期策略摘要，点击进入策略编辑 */
  const renderPolicyBadges = (resource: string, action: string) => {
    const item = catalogByCode.get(`${resource}:${action}`);
    if (!item) return null;
    const risk = item.riskLevel ?? 'unknown';
    const riskColorByLevel: Record<string, string> = {
      low: 'green', medium: 'gold', high: 'volcano', critical: 'red',
    };
    const aiAllowed =
      item.allowedPrincipals?.includes('agent') === true ||
      item.allowedPrincipals?.includes('service') === true;
    return (
      <Space size={4} wrap>
        <Tag
          color={riskColorByLevel[risk] ?? 'default'}
          className="text-xs cursor-pointer"
          onClick={() => openPolicyModal(item)}
        >
          {RISK_OPTIONS.find(o => o.value === risk)?.label ?? risk}
        </Tag>
        {aiAllowed && (
          <Tag color="purple" className="text-xs cursor-pointer" onClick={() => openPolicyModal(item)}>
            AI 可执行
          </Tag>
        )}
        {item.requiresApproval && (
          <Tag color="orange" className="text-xs cursor-pointer" onClick={() => openPolicyModal(item)}>
            需 {item.minApprovals ?? 1} 人审批
          </Tag>
        )}
        {item.configured === false && (
          <Tooltip title="该码尚未被管理员显式确认，当前为派生默认策略（AI 零权限）">
            <Tag color="default" className="text-xs">未确认</Tag>
          </Tooltip>
        )}
      </Space>
    );
  };

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

  const catalogModules = useMemo(() => deriveCatalogModules(permissionCatalog), [permissionCatalog]);
  const catalogActions = useMemo(() => deriveCatalogActions(catalogModules), [catalogModules]);

  // 模块配置（运行时国际化）。资源来自后端目录；未知资源用代码本身兜底展示，
  // 不再因为 MODULE_META 缺项而崩溃或在 UI 上消失。
  const moduleConfig = useMemo(() => {
    type ModuleConfig = {
      label: string;
      icon: React.ComponentType<{ className?: string; size?: number }>;
      description: string;
      category: string;
      categoryKey: string;
      actions: string[];
    };
    const result: Record<string, ModuleConfig> = {};
    for (const mod of catalogModules) {
      const meta = MODULE_META[mod.resource];
      const categoryKey = meta?.categoryKey ?? 'system';
      const i18nKey = MODULE_I18N_KEYS[mod.resource];
      const labelKey = `permissions.modules.${i18nKey}.label`;
      const descriptionKey = `permissions.modules.${i18nKey}.description`;
      const localizedLabel = i18nKey ? t(labelKey) : '';
      const localizedDescription = i18nKey ? t(descriptionKey) : '';
      // 模块名兜底口径与角色权限页一致：先 permissions.modules，再 roles.resourceLabels
      // （后者已覆盖后端全部资源），最后才退回英文 resource code。
      const sharedLabelKey = `roles.resourceLabels.${mod.resource}`;
      const sharedLabel = t(sharedLabelKey);
      const resolvedLabel =
        i18nKey && localizedLabel !== labelKey
          ? localizedLabel
          : sharedLabel && sharedLabel !== sharedLabelKey
            ? sharedLabel
            : mod.resource;
      result[mod.resource] = {
        label: resolvedLabel,
        icon: meta?.icon ?? Shield,
        description: i18nKey && localizedDescription !== descriptionKey ? localizedDescription : '',
        category: t(`permissions.categories.${categoryKey}`),
        categoryKey,
        actions: mod.actions,
      };
    }
    return result;
  }, [catalogModules, t]);

  // 操作配置（运行时国际化）；目录里的新动作没有翻译时显示动作代码本身。
  const actionConfig = useMemo(() => {
    const result: Record<string, { label: string; color: string; description: string }> = {};
    for (const actionKey of catalogActions) {
      const labelKey = `permissions.actions.${actionKey.toUpperCase()}`;
      const descKey = `permissions.actionDescriptions.${actionKey.toUpperCase()}`;
      const localizedLabel = t(labelKey);
      const localizedDescription = t(descKey);
      result[actionKey] = {
        label: localizedLabel === labelKey ? actionKey : localizedLabel,
        color: ACTION_COLOR[actionKey] ?? 'default',
        description: localizedDescription === descKey ? '' : localizedDescription,
      };
    }
    return result;
  }, [catalogActions, t]);

  // 加载角色列表与权限目录。目录是权限矩阵的唯一数据源。
  const loadRoles = useCallback(async () => {
    setRolesLoading(true);
    try {
      const [response, catalog] = await Promise.all([
        RoleAPI.getRoles({ page: 1, pageSize: 100 }),
        RoleAPI.getPermissionCatalog(),
      ]);
      setRoles(response.items);
      setPermissionCatalog(catalog);
      setPermissionCatalogCount(catalog.filter(item => item.id > 0).length);
      // 元数据（minApprovalsMax 等）失败不阻断主流程：编辑弹层里再兜底取一次
      RoleAPI.getPermissionCatalogMeta().then(setCatalogMeta).catch(() => undefined);
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

  const loadRolePermissions = useCallback(async (roleId: number, modules: CatalogModule[]) => {
    setLoading(true);
    try {
      const role = await RoleAPI.getRole(roleId);
      setPermissionState(buildPermissionStateFromStrings(role.permissions || [], modules));
      setHasChanges(false);
    } catch (error) {
      console.error('Failed to load role permissions:', error);
      message.error(t('permissions.messages.loadPermissionsFailed'));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  // 选中角色或目录刷新时统一重建矩阵快照，避免「先渲染旧矩阵再被覆盖」的竞态。
  useEffect(() => {
    if (!selectedRoleId) {
      setPermissionState(createPermissionState(catalogModules, false));
      setHasChanges(false);
      return;
    }
    void loadRolePermissions(selectedRoleId, catalogModules);
  }, [selectedRoleId, catalogModules, loadRolePermissions]);

  const handleSelectRole = (roleId: number) => {
    setSelectedRoleId(roleId);
  };

  const handleInitPermissions = async () => {
    setSaving(true);
    try {
      await RoleAPI.initDefaultPermissions();
      await loadRoles();
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
      // 筛选值存稳定的 categoryKey，而不是翻译后的名称：切换语言后
      // 旧字符串会失配，导致分类筛选瞬间返回空列表。
      const matchesCategory = categoryFilter === 'all' || config.categoryKey === categoryFilter;
      return matchesSearch && matchesCategory;
    })
    .map(([key]) => key);

  // 按分类分组（key 稳定，显示名取当前语言）
  const modulesByCategory: Record<string, string[]> = {};
  for (const key of filteredModuleKeys) {
    const cat = moduleConfig[key].categoryKey;
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
        if (moduleConfig[moduleKey].categoryKey === category) {
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

  // 保存走专用授权接口：接收 permissionIds，空数组表示清空该角色全部权限。
  // 不再用 PUT /roles/:id 提交权限码字符串——后端会逐个校验码是否存在于
  // 本租户目录，任一不存在即整体失败，且角色基本信息已经先写库造成部分更新。
  const handleSave = async () => {
    if (!selectedRoleId) {
      message.warning(t('permissions.messages.selectRoleFirst'));
      return;
    }
    if (permissionCatalog.length === 0) {
      message.warning(t('permissions.alerts.catalogNotInitMessage'));
      return;
    }
    setSaving(true);
    try {
      const codes = buildPermissionStringsFromState(permissionState);
      const codeToId = new Map(permissionCatalog.map(item => [item.code, item.id]));
      const permissionIds = codes
        .map(code => codeToId.get(code))
        .filter((id): id is number => typeof id === 'number' && id > 0);
      if (permissionIds.length !== codes.length) {
        message.error(t('permissions.messages.saveFailed'));
        return;
      }
      await RoleAPI.assignPermissions(selectedRoleId, permissionIds);
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
      await loadRolePermissions(selectedRoleId, catalogModules);
    } else {
      setPermissionState(createPermissionState(catalogModules, false));
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
            <Button size="small" type="link" onClick={() => handleBatchToggle(category.key, true)}>
              {t('permissions.buttons.enableAll')}
            </Button>
            <Button size="small" type="link" onClick={() => handleBatchToggle(category.key, false)}>
              {t('permissions.buttons.disableAll')}
            </Button>
          </div>
        </div>
      ),
      key: category.key,
      children:
        modulesByCategory[category.key]?.map((moduleKey: string) => {
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
                    aria-label={`启用模块 ${modConf.label}`}
                    checked={moduleState?.isEnabled || false}
                    onChange={() => handleToggleModule(moduleKey)}
                  />
                </div>
              </div>
            ),
            key: moduleKey,
            children: modConf.actions.map(actionKey => {
              const actionConf = actionConfig[actionKey];
              return {
                title: (
                  <div className="flex items-center justify-between w-full gap-2">
                    <div className="flex items-center gap-1 flex-wrap">
                      <Tag color={moduleState?.actions[actionKey] ? actionConf.color : 'default'}>
                        {actionConf.label}
                      </Tag>
                      {renderPolicyBadges(moduleKey, actionKey)}
                    </div>
                    <Switch
                      size="small"
                      aria-label={`${modConf.label} ${actionConf.label} 权限`}
                      checked={moduleState?.actions[actionKey] || false}
                      onChange={() => handleToggleAction(moduleKey, actionKey)}
                    />
                  </div>
                ),
                key: `${moduleKey}-${actionKey}`,
              };
            }),
          };
        }) || [],
    }));
  };

  // 渲染权限卡片视图
  const renderCardView = () => (
    <div className="space-y-6">
      {categories.map(category => {
        const categoryModuleKeys = modulesByCategory[category.key] || [];
        if (categoryModuleKeys.length === 0) return null;

        return (
          <Card
            key={category.key}
            title={
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {category.icon}
                  <span>{category.id}</span>
                  <Badge count={categoryModuleKeys.length} color="blue" />
                </div>
                <Space>
                  <Button size="small" type="link" onClick={() => handleBatchToggle(category.key, true)}>
                    {t('permissions.buttons.enableAll')}
                  </Button>
                  <Button size="small" type="link" onClick={() => handleBatchToggle(category.key, false)}>
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
                            aria-label={`启用模块 ${modConf.label}`}
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
                        {modConf.actions.map(actionKey => {
                          const actionConf = actionConfig[actionKey];
                          return (
                            <div key={actionKey} className="flex items-center justify-between gap-2">
                              <div className="flex items-center gap-1 flex-wrap">
                                <Tooltip title={actionConf.description ? actionConf.description : undefined}>
                                  <Tag
                                    color={moduleState?.actions[actionKey] ? actionConf.color : 'default'}
                                    className="text-xs cursor-help"
                                  >
                                    {actionConf.label}
                                  </Tag>
                                </Tooltip>
                                {renderPolicyBadges(moduleKey, actionKey)}
                              </div>
                              <Switch
                                size="small"
                                aria-label={`${modConf.label} ${actionConf.label} 权限`}
                                checked={moduleState?.actions[actionKey] || false}
                                onChange={() => handleToggleAction(moduleKey, actionKey)}
                                disabled={!moduleState?.isEnabled}
                              />
                            </div>
                          );
                        })}
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

      {/* AI 主体授权汇总：AI 默认零权限，这里列出管理员显式开放给 agent/service 的码。
          AI 安全攻击门槛低，开放面必须持续可见、可审计。 */}
      <Alert
        type={aiAuthorizedCodes.length > 0 ? 'warning' : 'success'}
        showIcon
        icon={<Bot className="w-4 h-4" />}
        title={
          aiAuthorizedCodes.length > 0
            ? `${aiAuthorizedCodes.length} 个权限码已开放给 AI/服务主体执行`
            : '当前没有权限码开放给 AI 执行（默认安全态）'
        }
        description={
          aiAuthorizedCodes.length > 0 ? (
            <Space size={4} wrap>
              {aiAuthorizedCodes.map(item => (
                <Tooltip
                  key={item.code}
                  title={`风险 ${item.riskLevel ?? 'unknown'} · 需 ${item.minApprovals ?? 1} 人审批 · 点击查看`}
                >
                  <Tag color="purple" className="text-xs cursor-pointer" onClick={() => openPolicyModal(item)}>
                    {item.code}
                  </Tag>
                </Tooltip>
              ))}
            </Space>
          ) : (
            <Text type="secondary" className="text-xs">
              AI 代理执行任何操作都需要管理员在此处显式把对应权限码授权给 agent 主体；中高风险操作还会强制人工审批。
            </Text>
          )
        }
      />

      {/* 角色选择器 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={24} md={12} xl={8}>
            <div className="flex items-center gap-2">
              <Text strong className="field-label">
                {t('permissions.selectRole')}
              </Text>
              <Select
                placeholder={t('permissions.placeholders.selectRole')}
                value={selectedRoleId || undefined}
                onChange={handleSelectRole}
                loading={rolesLoading}
                style={{ flex: 1, minWidth: 0 }}
                showSearch
                optionFilterProp="children"
                options={roles.map(role => ({ value: role.id, label: role.name }))}
              />
            </div>
          </Col>
          <Col xs={24} sm={24} md={12} xl={8}>
            <Input
              placeholder={t('permissions.placeholders.searchModules')}
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={12} xl={4}>
            <Select
              placeholder={t('permissions.placeholders.filterCategory')}
              value={categoryFilter}
              onChange={setCategoryFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: t('permissions.allCategories') },
                ...categories.map(category => ({ value: category.key, label: category.id })),
              ]}
            />
          </Col>
          <Col xs={24} sm={12} md={12} xl={4}>
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
          <Col xs={24} className="text-right">
            <Space wrap>
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
                disabled={!hasChanges || !selectedRoleId || permissionCatalog.length === 0}
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
          title={t('permissions.alerts.selectRoleMessage')}
          description={t('permissions.alerts.selectRoleDescription')}
          type="info"
          showIcon
        />
      )}

      {permissionCatalogCount === 0 && (
        <Alert
          title={t('permissions.alerts.catalogNotInitMessage')}
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
              value={stats.totalModules > 0 ? ((stats.enabledModules / stats.totalModules) * 100).toFixed(1) : '0.0'}
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
              value={stats.totalActions > 0 ? ((stats.enabledActions / stats.totalActions) * 100).toFixed(1) : '0.0'}
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
          title={t('permissions.alerts.unsavedMessage')}
          description={t('permissions.alerts.unsavedDescription')}
          type="warning"
          showIcon
          closable
        />
      )}

      {/* 权限配置内容 */}
      <Card className="enterprise-card">
        {loading ? (
          <div className="flex flex-col gap-3 justify-center items-center py-20">
            {/* antd v6：Spin 的 tip 仅在嵌套模式下生效，改为独立文本提示 */}
            <Spin size="large" />
            <Text type="secondary">{t('permissions.loading')}</Text>
          </div>
        ) : viewMode === 'card' ? (
          renderCardView()
        ) : (
          <div className="permission-tree-scroll">
            <Tree
              treeData={generateTreeData()}
              showLine
              className="enterprise-tree"
            />
          </div>
        )}
      </Card>

      {/* 运行期策略编辑弹层：风险分级 / AI 主体授权 / 审批强度。
          安全不变量由后端 ValidatePolicy 强制（AI 执行中高风险必须开审批、
          高/危急必须双人），前端只做即时提示。 */}
      <Modal
        title={`权限策略：${policyEditing?.code ?? ''}`}
        open={policyEditing != null && policyDraft != null}
        onOk={handleSavePolicy}
        onCancel={() => {
          setPolicyEditing(null);
          setPolicyDraft(null);
        }}
        confirmLoading={policySaving}
        okText="保存（立即生效）"
        cancelText="取消"
        width={560}
      >
        {policyDraft && (
          <div className="space-y-4">
            <div>
              <Text strong className="block mb-1">风险分级</Text>
              <Text type="secondary" className="text-xs block mb-2">
                决定 AI 执行该操作时需要的人类确认强度；未在列表中的动作按中风险处理。
              </Text>
              <Select
                style={{ width: 200 }}
                value={policyDraft.riskLevel}
                onChange={v => setPolicyDraft({ ...policyDraft, riskLevel: v })}
                options={RISK_OPTIONS}
              />
            </div>
            <div>
              <Text strong className="block mb-1">允许执行主体</Text>
              <Text type="secondary" className="text-xs block mb-2">
                默认只有「人」。勾选「AI 代理」即显式授权 AI 执行该操作（AI 安全边界）。
              </Text>
              <Select
                mode="multiple"
                style={{ width: '100%' }}
                value={policyDraft.allowedPrincipals}
                onChange={v => setPolicyDraft({ ...policyDraft, allowedPrincipals: v })}
                options={PRINCIPAL_OPTIONS}
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                checked={policyDraft.requiresApproval}
                onChange={v => setPolicyDraft({ ...policyDraft, requiresApproval: v })}
                aria-label="需要人工审批"
              />
              <Text>执行前需人工审批</Text>
            </div>
            <div>
              <Text strong className="block mb-1">最少审批人数</Text>
              <Text type="secondary" className="text-xs block mb-2">
                需要互不相同的人类审批人数（上限 {catalogMeta?.minApprovalsMax ?? 2}）；高/危急风险必须 ≥2。
              </Text>
              <Select
                style={{ width: 200 }}
                value={policyDraft.minApprovals}
                onChange={v => setPolicyDraft({ ...policyDraft, minApprovals: v })}
                options={Array.from({ length: (catalogMeta?.minApprovalsMax ?? 2) + 1 }, (_, i) => ({
                  value: i,
                  label: `${i} 人`,
                }))}
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                checked={policyDraft.isActive}
                onChange={v => setPolicyDraft({ ...policyDraft, isActive: v })}
                aria-label="权限码启用"
              />
              <Text>启用该权限码（停用后对所有主体不可见）</Text>
            </div>
            {(policyDraft.allowedPrincipals.includes('agent') ||
              policyDraft.allowedPrincipals.includes('service')) &&
              policyDraft.riskLevel !== 'low' &&
              !policyDraft.requiresApproval && (
                <Alert
                  type="error"
                  showIcon
                  title="不允许的组合"
                  description="允许 AI 执行非低风险操作时必须开启人工审批（后端将拒绝保存）。"
                />
              )}
            {(policyDraft.allowedPrincipals.includes('agent') ||
              policyDraft.allowedPrincipals.includes('service')) &&
              (policyDraft.riskLevel === 'high' || policyDraft.riskLevel === 'critical') &&
              policyDraft.minApprovals < 2 && (
                <Alert
                  type="error"
                  showIcon
                  title="不允许的组合"
                  description="允许 AI 执行高/危急风险操作时必须配置双人审批（后端将拒绝保存）。"
                />
              )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default PermissionConfiguration;

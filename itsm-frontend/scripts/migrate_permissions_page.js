#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const FILE = path.join(__dirname, '..', 'src', 'app', '(main)', 'admin', 'permissions', 'page.tsx');

let content = fs.readFileSync(FILE, 'utf8');

const migrations = [
  // imports: add useI18n
  {
    pattern: "import { RoleAPI } from '@/lib/api/role-api';",
    replacement: "import { RoleAPI } from '@/lib/api/role-api';\nimport { useI18n } from '@/lib/i18n/useI18n';",
  },
  // component body
  {
    pattern: "const PermissionConfiguration = () => {\n  const { message } = App.useApp();",
    replacement: "const PermissionConfiguration = () => {\n  const { t } = useI18n();\n  const { message } = App.useApp();",
  },
  // module config labels
  { pattern: "[PERMISSION_MODULES.DASHBOARD]: { label: '仪表盘', icon: '📊', description: '系统仪表盘和概览信息', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.DASHBOARD]: { label: t('permissions.modules.DASHBOARD.label'), icon: '📊', description: t('permissions.modules.DASHBOARD.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.TICKETS]: { label: '工单管理', icon: '🎫', description: '工单的创建、处理和管理', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.TICKETS]: { label: t('permissions.modules.TICKETS.label'), icon: '🎫', description: t('permissions.modules.TICKETS.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.INCIDENTS]: { label: '事件管理', icon: '🚨', description: 'IT事件的记录和处理', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.INCIDENTS]: { label: t('permissions.modules.INCIDENTS.label'), icon: '🚨', description: t('permissions.modules.INCIDENTS.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.PROBLEMS]: { label: '问题管理', icon: '🔧', description: '根本原因分析和问题解决', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.PROBLEMS]: { label: t('permissions.modules.PROBLEMS.label'), icon: '🔧', description: t('permissions.modules.PROBLEMS.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.CHANGES]: { label: '变更管理', icon: '🔄', description: 'IT变更的规划和实施', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.CHANGES]: { label: t('permissions.modules.CHANGES.label'), icon: '🔄', description: t('permissions.modules.CHANGES.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.SERVICE_CATALOG]: { label: '服务目录', icon: '📋', description: 'IT服务目录管理', category: '服务管理' },",
    replacement: "[PERMISSION_MODULES.SERVICE_CATALOG]: { label: t('permissions.modules.SERVICE_CATALOG.label'), icon: '📋', description: t('permissions.modules.SERVICE_CATALOG.description'), category: t('permissions.categories.service') }," },
  { pattern: "[PERMISSION_MODULES.SERVICE_REQUEST]: { label: '服务请求', icon: '🧾', description: '服务请求提交、审批和履约', category: '服务管理' },",
    replacement: "[PERMISSION_MODULES.SERVICE_REQUEST]: { label: t('permissions.modules.SERVICE_REQUEST.label'), icon: '🧾', description: t('permissions.modules.SERVICE_REQUEST.description'), category: t('permissions.categories.service') }," },
  { pattern: "[PERMISSION_MODULES.KNOWLEDGE_BASE]: { label: '知识库', icon: '📚', description: '知识文档和解决方案', category: '服务管理' },",
    replacement: "[PERMISSION_MODULES.KNOWLEDGE_BASE]: { label: t('permissions.modules.KNOWLEDGE_BASE.label'), icon: '📚', description: t('permissions.modules.KNOWLEDGE_BASE.description'), category: t('permissions.categories.service') }," },
  { pattern: "[PERMISSION_MODULES.CMDB]: { label: 'CMDB', icon: '🧩', description: '配置项、关系和拓扑管理', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.CMDB]: { label: t('permissions.modules.CMDB.label'), icon: '🧩', description: t('permissions.modules.CMDB.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.ASSETS]: { label: '资产管理', icon: '💻', description: 'IT资产和许可证管理', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.ASSETS]: { label: t('permissions.modules.ASSETS.label'), icon: '💻', description: t('permissions.modules.ASSETS.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.RELEASES]: { label: '发布管理', icon: '🚀', description: '发布计划和发布执行', category: '核心功能' },",
    replacement: "[PERMISSION_MODULES.RELEASES]: { label: t('permissions.modules.RELEASES.label'), icon: '🚀', description: t('permissions.modules.RELEASES.description'), category: t('permissions.categories.core') }," },
  { pattern: "[PERMISSION_MODULES.REPORTS]: { label: '报告分析', icon: '📈', description: '数据报告和分析功能', category: '分析工具' },",
    replacement: "[PERMISSION_MODULES.REPORTS]: { label: t('permissions.modules.REPORTS.label'), icon: '📈', description: t('permissions.modules.REPORTS.description'), category: t('permissions.categories.analysis') }," },
  { pattern: "[PERMISSION_MODULES.ADMIN]: { label: '系统管理', icon: '⚙️', description: '系统管理和配置', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.ADMIN]: { label: t('permissions.modules.ADMIN.label'), icon: '⚙️', description: t('permissions.modules.ADMIN.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.USERS]: { label: '用户管理', icon: '👥', description: '用户账户管理', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.USERS]: { label: t('permissions.modules.USERS.label'), icon: '👥', description: t('permissions.modules.USERS.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.ROLES]: { label: '角色管理', icon: '🛡️', description: '角色和权限管理', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.ROLES]: { label: t('permissions.modules.ROLES.label'), icon: '🛡️', description: t('permissions.modules.ROLES.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.GROUPS]: { label: '用户组管理', icon: '👪', description: '用户组和候选组管理', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.GROUPS]: { label: t('permissions.modules.GROUPS.label'), icon: '👪', description: t('permissions.modules.GROUPS.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.ORG]: { label: '组织架构', icon: '🏢', description: '部门、团队和组织架构管理', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.ORG]: { label: t('permissions.modules.ORG.label'), icon: '🏢', description: t('permissions.modules.ORG.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.WORKFLOWS]: { label: '工作流', icon: '🔀', description: '业务流程配置', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.WORKFLOWS]: { label: t('permissions.modules.WORKFLOWS.label'), icon: '🔀', description: t('permissions.modules.WORKFLOWS.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.SYSTEM_CONFIG]: { label: '系统配置', icon: '🔧', description: '系统参数和设置', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.SYSTEM_CONFIG]: { label: t('permissions.modules.SYSTEM_CONFIG.label'), icon: '🔧', description: t('permissions.modules.SYSTEM_CONFIG.description'), category: t('permissions.categories.system') }," },
  { pattern: "[PERMISSION_MODULES.AI]: { label: 'AI能力', icon: '🤖', description: 'AI 辅助与自动化能力', category: '系统管理' },",
    replacement: "[PERMISSION_MODULES.AI]: { label: t('permissions.modules.AI.label'), icon: '🤖', description: t('permissions.modules.AI.description'), category: t('permissions.categories.system') }," },

  // action config
  { pattern: "[PERMISSION_ACTIONS.READ]: { label: '读取', color: 'blue', description: '读取和浏览权限' },",
    replacement: "[PERMISSION_ACTIONS.READ]: { label: t('permissions.actions.READ'), color: 'blue', description: t('permissions.actionDescriptions.READ') }," },
  { pattern: "[PERMISSION_ACTIONS.CREATE]: { label: '创建', color: 'green', description: '创建新记录权限' },",
    replacement: "[PERMISSION_ACTIONS.CREATE]: { label: t('permissions.actions.CREATE'), color: 'green', description: t('permissions.actionDescriptions.CREATE') }," },
  { pattern: "[PERMISSION_ACTIONS.WRITE]: { label: '写入', color: 'orange', description: '创建或修改业务数据权限' },",
    replacement: "[PERMISSION_ACTIONS.WRITE]: { label: t('permissions.actions.WRITE'), color: 'orange', description: t('permissions.actionDescriptions.WRITE') }," },
  { pattern: "[PERMISSION_ACTIONS.UPDATE]: { label: '更新', color: 'orange', description: '更新现有记录权限' },",
    replacement: "[PERMISSION_ACTIONS.UPDATE]: { label: t('permissions.actions.UPDATE'), color: 'orange', description: t('permissions.actionDescriptions.UPDATE') }," },
  { pattern: "[PERMISSION_ACTIONS.DELETE]: { label: '删除', color: 'red', description: '删除记录权限' },",
    replacement: "[PERMISSION_ACTIONS.DELETE]: { label: t('permissions.actions.DELETE'), color: 'red', description: t('permissions.actionDescriptions.DELETE') }," },
  { pattern: "[PERMISSION_ACTIONS.MANAGE]: { label: '管理', color: 'magenta', description: '管理配置和成员关系权限' },",
    replacement: "[PERMISSION_ACTIONS.MANAGE]: { label: t('permissions.actions.MANAGE'), color: 'magenta', description: t('permissions.actionDescriptions.MANAGE') }," },
  { pattern: "[PERMISSION_ACTIONS.APPROVE]: { label: '审批', color: 'purple', description: '审批和批准权限' },",
    replacement: "[PERMISSION_ACTIONS.APPROVE]: { label: t('permissions.actions.APPROVE'), color: 'purple', description: t('permissions.actionDescriptions.APPROVE') }," },
  { pattern: "[PERMISSION_ACTIONS.ASSIGN]: { label: '分配', color: 'cyan', description: '分配和指派权限' },",
    replacement: "[PERMISSION_ACTIONS.ASSIGN]: { label: t('permissions.actions.ASSIGN'), color: 'cyan', description: t('permissions.actionDescriptions.ASSIGN') }," },
  { pattern: "[PERMISSION_ACTIONS.VIEW]: { label: '查看', color: 'geekblue', description: '兼容旧版查看权限' },",
    replacement: "[PERMISSION_ACTIONS.VIEW]: { label: t('permissions.actions.VIEW'), color: 'geekblue', description: t('permissions.actionDescriptions.VIEW') }," },
  { pattern: "[PERMISSION_ACTIONS.EXPORT]: { label: '导出', color: 'geekblue', description: '数据导出权限' },",
    replacement: "[PERMISSION_ACTIONS.EXPORT]: { label: t('permissions.actions.EXPORT'), color: 'geekblue', description: t('permissions.actionDescriptions.EXPORT') }," },
  { pattern: "[PERMISSION_ACTIONS.ALL]: { label: '全部', color: 'volcano', description: '平台级全部权限' },",
    replacement: "[PERMISSION_ACTIONS.ALL]: { label: t('permissions.actions.ALL'), color: 'volcano', description: t('permissions.actionDescriptions.ALL') }," },

  // categories - dynamic use
  { pattern: "{ id: '核心功能', name: '核心功能', icon: <Activity className=\"w-4 h-4\" /> },", replacement: "{ id: t('permissions.categories.core'), name: t('permissions.categories.core'), icon: <Activity className=\"w-4 h-4\" /> }," },
  { pattern: "{ id: '服务管理', name: '服务管理', icon: <Globe className=\"w-4 h-4\" /> },", replacement: "{ id: t('permissions.categories.service'), name: t('permissions.categories.service'), icon: <Globe className=\"w-4 h-4\" /> }," },
  { pattern: "{ id: '分析工具', name: '分析工具', icon: <BarChart3 className=\"w-4 h-4\" /> },", replacement: "{ id: t('permissions.categories.analysis'), name: t('permissions.categories.analysis'), icon: <BarChart3 className=\"w-4 h-4\" /> }," },
  { pattern: "{ id: '系统管理', name: '系统管理', icon: <Settings className=\"w-4 h-4\" /> },", replacement: "{ id: t('permissions.categories.system'), name: t('permissions.categories.system'), icon: <Settings className=\"w-4 h-4\" /> }," },

  // messages
  { pattern: "message.error('加载角色列表失败');", replacement: "message.error(t('permissions.messages.loadRolesFailed'));" },
  { pattern: "message.error('加载角色权限失败');", replacement: "message.error(t('permissions.messages.loadPermissionsFailed'));" },
  { pattern: "message.success('默认权限字典初始化完成');", replacement: "message.success(t('permissions.messages.initSuccess'));" },
  { pattern: "message.error('初始化权限字典失败');", replacement: "message.error(t('permissions.messages.initFailed'));" },
  { pattern: "message.warning('请先选择一个角色');", replacement: "message.warning(t('permissions.messages.selectRoleFirst'));" },
  { pattern: "message.success('权限配置保存成功');", replacement: "message.success(t('permissions.messages.saveSuccess'));" },
  { pattern: "message.error('保存失败，请重试');", replacement: "message.error(t('permissions.messages.saveFailed'));" },
  { pattern: "message.info('配置已重置');", replacement: "message.info(t('permissions.messages.reset'));" },

  // tree view buttons
  { pattern: "全部启用\n            </Button>\n            <Button size=\"small\" type=\"link\" onClick={() => handleBatchToggle(category.id, false)}>\n              全部禁用", replacement: "{t('permissions.buttons.enableAll')}\n            </Button>\n            <Button size=\"small\" type=\"link\" onClick={() => handleBatchToggle(category.id, false)}>\n              {t('permissions.buttons.disableAll')}" },

  // page title
  { pattern: "          权限配置管理\n        </Title>\n        <Text type=\"secondary\">配置系统功能模块和操作权限，定义访问控制策略</Text>", replacement: "          {t('permissions.title')}\n        </Title>\n        <Text type=\"secondary\">{t('permissions.description')}</Text>" },

  // role selector
  { pattern: "<Text strong>选择角色：</Text>", replacement: "<Text strong>{t('permissions.selectRole')}</Text>" },
  { pattern: "placeholder=\"请选择角色\"", replacement: "placeholder={t('permissions.placeholders.selectRole')}" },
  { pattern: "placeholder=\"搜索模块或权限...\"", replacement: "placeholder={t('permissions.placeholders.searchModules')}" },
  { pattern: "placeholder=\"筛选分类\"", replacement: "placeholder={t('permissions.placeholders.filterCategory')}" },
  { pattern: "{ value: 'all', label: '全部分类' },", replacement: "{ value: 'all', label: t('permissions.allCategories') }," },
  { pattern: "placeholder=\"视图模式\"", replacement: "placeholder={t('permissions.placeholders.viewMode')}" },
  { pattern: "{ value: 'card', label: '卡片视图' },", replacement: "{ value: 'card', label: t('permissions.viewModes.card') }," },
  { pattern: "{ value: 'tree', label: '树形视图' },", replacement: "{ value: 'tree', label: t('permissions.viewModes.tree') }," },
  { pattern: "              初始化权限字典\n", replacement: "              {t('permissions.buttons.init')}\n" },
  { pattern: "                重置\n", replacement: "                {t('common.reset')}\n" },
  { pattern: "                {saving ? '保存中...' : '保存配置'}\n              </Button>", replacement: "                {saving ? t('permissions.buttons.saving') : t('permissions.buttons.save')}\n              </Button>" },

  // card view buttons
  { pattern: "                  全部启用\n", replacement: "                  {t('permissions.buttons.enableAll')}\n" },
  { pattern: "                  全部禁用\n", replacement: "                  {t('permissions.buttons.disableAll')}\n" },

  // alerts
  { pattern: "message=\"请先选择角色\"\n          description=\"从上方下拉框中选择一个角色，以查看和编辑该角色的权限配置。\"",
    replacement: "message={t('permissions.alerts.selectRoleMessage')}\n          description={t('permissions.alerts.selectRoleDescription')}" },
  { pattern: "message=\"权限字典尚未初始化\"\n          description=\"当前租户没有可用于真实授权的权限字典。请点击“初始化权限字典”，否则角色权限保存可能无法落到后端关联表。\"",
    replacement: "message={t('permissions.alerts.catalogNotInitMessage')}\n          description={t('permissions.alerts.catalogNotInitDescription')}" },
  { pattern: "message=\"配置已修改\"\n          description=\"您有未保存的权限配置更改，请及时保存。\"",
    replacement: "message={t('permissions.alerts.unsavedMessage')}\n          description={t('permissions.alerts.unsavedDescription')}" },

  // statistics
  { pattern: "title=\"功能模块\"", replacement: "title={t('permissions.stats.modules')}" },
  { pattern: "title=\"操作权限\"", replacement: "title={t('permissions.stats.actions')}" },
  { pattern: "title=\"启用模块\"", replacement: "title={t('permissions.stats.enabledModules')}" },
  { pattern: "title=\"权限覆盖率\"", replacement: "title={t('permissions.stats.coverage')}" },

  // loading
  { pattern: "tip=\"加载权限数据...\"", replacement: "tip={t('permissions.loading')}" },
];

let totalChanges = 0;
let failed = [];

for (const m of migrations) {
  const count = (content.match(new RegExp(m.pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g')) || []).length;
  if (count === 0) {
    failed.push(m.pattern.substring(0, 80));
    continue;
  }
  content = content.replace(m.pattern, m.replacement);
  totalChanges += count;
}

if (failed.length > 0) {
  console.error('Failed patterns:');
  failed.forEach(f => console.error('  -', f));
}

fs.writeFileSync(FILE, content);
console.log(`Migrated ${totalChanges} patterns to permissions/page.tsx`);
#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const FILE = path.join(__dirname, '..', 'src', 'app', '(main)', 'admin', 'groups', 'page.tsx');

let content = fs.readFileSync(FILE, 'utf8');

const migrations = [
  // imports: add useI18n
  {
    pattern: "import type { PermissionCatalogItem } from '@/lib/api/api-config';",
    replacement: "import type { PermissionCatalogItem } from '@/lib/api/api-config';\nimport { useI18n } from '@/lib/i18n/useI18n';",
  },

  // component body
  {
    pattern: "export default function GroupManagementPage() {\n  const { message } = App.useApp();",
    replacement: "export default function GroupManagementPage() {\n  const { t } = useI18n();\n  const { message } = App.useApp();",
  },

  // messages
  { pattern: "message.error('加载用户组失败');", replacement: "message.error(t('groups.messages.loadFailed'));" },
  { pattern: "message.success('用户组更新成功');", replacement: "message.success(t('groups.updateSuccess'));" },
  { pattern: "message.success('用户组创建成功');", replacement: "message.success(t('groups.createSuccess'));" },
  { pattern: "message.error('保存用户组失败');", replacement: "message.error(t('groups.messages.saveFailed'));" },
  { pattern: "message.success('用户组删除成功');", replacement: "message.success(t('groups.deleteSuccess'));" },
  { pattern: "message.error('删除用户组失败');", replacement: "message.error(t('groups.messages.deleteFailed'));" },
  { pattern: "message.error('加载组成员失败');", replacement: "message.error(t('groups.messages.loadMembersFailed'));" },
  { pattern: "message.success('组成员更新成功');", replacement: "message.success(t('groups.messages.membersUpdated'));" },
  { pattern: "message.error('保存组成员失败');", replacement: "message.error(t('groups.messages.saveMembersFailed'));" },

  // confirm modal
  { pattern: "title: '确认删除用户组',\n      content: `删除「${group.name}」后，关联成员关系也会被移除。确定继续吗？`,", replacement: "title: t('groups.confirmDelete'),\n      content: t('groups.confirmDeleteContent', { name: group.name })," },
  { pattern: "okText: '删除',", replacement: "okText: t('common.delete')," },
  { pattern: "cancelText: '取消',", replacement: "cancelText: t('common.cancel')," },

  // statistics
  { pattern: "label: '总用户组数',", replacement: "label: t('groups.stats.total')," },
  { pattern: "label: '当前页用户组',", replacement: "label: t('groups.stats.current')," },
  { pattern: "label: '搜索结果',", replacement: "label: t('common.search')," },
  { pattern: "label: '业务类型',", replacement: "label: t('groups.stats.type')," },
  { pattern: "value: '用户组',", replacement: "value: t('groups.title')," },

  // table columns
  { pattern: "title: '用户组名称',", replacement: "title: t('groups.groupName')," },
  { pattern: "<Text type=\"secondary\">{record.description || '暂无描述'}</Text>", replacement: "<Text type=\"secondary\">{record.description || t('common.noData')}</Text>" },
  { pattern: "title: '成员',", replacement: "title: t('groups.memberCount')," },
  { pattern: "title: '租户ID',", replacement: "title: t('groups.tenantId')," },
  { pattern: "title: '创建时间',", replacement: "title: t('groups.createdAt')," },
  { pattern: "title: '更新时间',", replacement: "title: t('groups.updatedAt')," },
  { pattern: "title: '操作',", replacement: "title: t('common.action')," },

  // row actions
  { pattern: "            成员", replacement: "{t('groups.actions.members')}" },
  { pattern: "            编辑", replacement: "{t('common.edit')}" },
  { pattern: "            删除", replacement: "{t('common.delete')}" },

  // header
  { pattern: "            用户组管理", replacement: "{t('groups.title')}" },
  { pattern: "<Text type=\"secondary\">管理用户组基础信息，为后续成员关系和审批候选组提供组织基础。</Text>", replacement: "<Text type=\"secondary\">{t('groups.description')}</Text>" },

  // buttons
  { pattern: "            新建用户组", replacement: "{t('groups.create')}" },
  { pattern: 'placeholder="搜索用户组名称或描述"', replacement: 'placeholder={t("groups.searchPlaceholder")}' },
  { pattern: "<Button onClick={loadGroups}>刷新</Button>", replacement: "<Button onClick={loadGroups}>{t('common.refresh')}</Button>" },
  { pattern: "<Empty description={search ? '没有匹配的用户组' : '暂无用户组'}>", replacement: "<Empty description={search ? t('groups.empty.searchEmpty') : t('groups.empty.noData')}>" },
  { pattern: "                  创建用户组", replacement: "{t('groups.create')}" },
  { pattern: "showTotal: total => `共 ${total} 条记录`,", replacement: "showTotal: total => t('common.totalLabel', { total })," },

  // modal title
  { pattern: "title={selectedGroup ? '编辑用户组' : '新建用户组'}", replacement: "title={selectedGroup ? t('groups.edit') : t('groups.create')}" },

  // form labels
  { pattern: 'label="用户组名称"', replacement: 'label={t("groups.groupName")}' },
  { pattern: "{ required: true, message: '请输入用户组名称' },", replacement: "{ required: true, message: t('groups.requiredName') }," },
  { pattern: "{ max: 100, message: '用户组名称不能超过100个字符' },", replacement: "{ max: 100, message: t('groups.nameMaxLength') }," },
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
console.log(`Migrated ${totalChanges} patterns to groups/page.tsx`);
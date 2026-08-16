#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const FILE = path.join(__dirname, '..', 'src', 'app', '(main)', 'admin', 'teams', 'page.tsx');

let content = fs.readFileSync(FILE, 'utf8');

const migrations = [
  // imports: add useI18n
  {
    pattern: "import { UserApi } from '@/lib/api/user-api';",
    replacement: "import { UserApi } from '@/lib/api/user-api';\nimport { useI18n } from '@/lib/i18n/useI18n';",
  },
  // component body
  {
    pattern: "export default function TeamManagement() {\n  const { message } = App.useApp();",
    replacement: "export default function TeamManagement() {\n  const { t } = useI18n();\n  const { message } = App.useApp();",
  },
  // messages
  { pattern: "message.error('加载团队数据失败');", replacement: "message.error(t('teamsPage.messages.loadFailed'));" },
  { pattern: "message.success('团队更新成功');", replacement: "message.success(t('teamsPage.messages.updateSuccess'));" },
  { pattern: "message.success('团队创建成功');", replacement: "message.success(t('teamsPage.messages.createSuccess'));" },
  { pattern: "message.error('保存团队失败');", replacement: "message.error(t('teamsPage.messages.saveFailed'));" },
  { pattern: "message.success('团队删除成功');", replacement: "message.success(t('teamsPage.messages.deleteSuccess'));" },
  { pattern: "message.error('删除团队失败');", replacement: "message.error(t('teamsPage.messages.deleteFailed'));" },

  // table columns
  { pattern: "title: '团队名称',", replacement: "title: t('teamsPage.teamName')," },
  { pattern: "title: '团队编码',", replacement: "title: t('teamsPage.teamCode')," },
  { pattern: "title: '团队经理',", replacement: "title: t('teamsPage.manager')," },
  { pattern: "title: '成员',", replacement: "title: t('teamsPage.members')," },
  { pattern: "title: '描述',", replacement: "title: t('common.description')," },
  { pattern: "title: '操作',", replacement: "title: t('common.action')," },

  // popconfirm
  { pattern: 'title="确认删除"', replacement: 'title={t("common.confirmDelete")}' },
  { pattern: 'description={`确定要删除团队"${record.name}"吗？`}', replacement: 'description={t("teamsPage.confirmDeleteContent", { name: record.name })}' },
  { pattern: 'okText="确认"', replacement: 'okText={t("common.confirm")}' },
  { pattern: 'cancelText="取消"', replacement: 'cancelText={t("common.cancel")}' },

  // header
  { pattern: "          团队管理\n", replacement: "          {t('teamsPage.title')}\n" },
  { pattern: "<Text type=\"secondary\">管理团队和团队成员</Text>", replacement: "<Text type=\"secondary\">{t('teamsPage.description')}</Text>" },

  // statistics
  { pattern: 'title="团队总数"', replacement: 'title={t("teamsPage.stats.total")}' },
  { pattern: 'title="团队成员总数"', replacement: 'title={t("teamsPage.stats.totalMembers")}' },

  // search bar
  { pattern: 'placeholder="搜索团队名称、编码或描述"', replacement: 'placeholder={t("teamsPage.searchPlaceholder")}' },
  { pattern: "          新建团队\n", replacement: "          {t('teamsPage.create')}\n" },
  { pattern: "            刷新\n", replacement: "            {t('common.refresh')}\n" },

  // empty state
  { pattern: "<Empty description={searchTerm ? '没有匹配的团队' : '暂无团队'}>", replacement: "<Empty description={searchTerm ? t('teamsPage.empty.searchEmpty') : t('teamsPage.empty.noData')}>" },
  { pattern: "                  新建团队\n", replacement: "                  {t('teamsPage.create')}\n" },
  { pattern: "showTotal: total => `共 ${total} 条记录`,", replacement: "showTotal: total => t('common.totalLabel', { total })," },

  // modal title
  { pattern: "{selectedTeam ? '编辑团队' : '新建团队'}", replacement: "{selectedTeam ? t('teamsPage.edit') : t('teamsPage.create')}" },
  { pattern: 'okText="保存"', replacement: 'okText={t("common.save")}' },
  { pattern: 'cancelText="取消"', replacement: 'cancelText={t("common.cancel")}' },

  // form
  { pattern: 'label="团队名称"\n            name="name"\n            rules={[{ required: true, message: \'请输入团队名称\' }]}\n          >\n            <Input placeholder="请输入团队名称" />',
    replacement: 'label={t("teamsPage.teamName")}\n            name="name"\n            rules={[{ required: true, message: t(\'teamsPage.form.requiredName\') }]}\n          >\n            <Input placeholder={t("teamsPage.form.namePlaceholder")} />' },
  { pattern: 'label="团队编码"\n            name="code"\n            rules={[{ required: true, message: \'请输入团队编码\' }]}\n          >\n            <Input placeholder="请输入团队编码（如：TEAM001）" />',
    replacement: 'label={t("teamsPage.teamCode")}\n            name="code"\n            rules={[{ required: true, message: t(\'teamsPage.form.requiredCode\') }]}\n          >\n            <Input placeholder={t("teamsPage.form.codePlaceholder")} />' },
  { pattern: 'label="团队经理"', replacement: 'label={t("teamsPage.manager")}' },
  { pattern: 'placeholder="选择团队经理"', replacement: 'placeholder={t("teamsPage.form.managerPlaceholder")}' },
  { pattern: 'label="描述"', replacement: 'label={t("common.description")}' },
  { pattern: '<TextArea rows={3} placeholder="请输入团队描述" />', replacement: '<TextArea rows={3} placeholder={t("teamsPage.form.descriptionPlaceholder")} />' },
];

let totalChanges = 0;
let failed = [];

for (const m of migrations) {
  const count = (content.match(new RegExp(m.pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g')) || []).length;
  if (count === 0) {
    failed.push(m.pattern.substring(0, 100));
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
console.log(`Migrated ${totalChanges} patterns to teams/page.tsx`);
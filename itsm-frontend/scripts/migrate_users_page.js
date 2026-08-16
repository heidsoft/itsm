#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const FILE = path.join(__dirname, '..', 'src', 'app', '(main)', 'admin', 'users', 'page.tsx');

let content = fs.readFileSync(FILE, 'utf8');

const t = 't';

// Mapping table - ZH text -> translation key + English fallback
const migrations = [
  // imports: add useI18n
  {
    pattern: "import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';",
    replacement: "import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';\nimport { useI18n } from '@/lib/i18n/useI18n';",
  },

  // component body: add t()
  {
    pattern: "  const { currentTenant } = useAuthStore();\n  useAuthStoreHydration();",
    replacement: "  const { t } = useI18n();\n  const { currentTenant } = useAuthStore();\n  useAuthStoreHydration();",
  },

  // message strings
  { pattern: "message.error('加载用户列表失败');", replacement: "message.error(t('users.messages.loadFailed'));" },
  { pattern: "message.error('无法获取租户信息，请重新登录');", replacement: "message.error(t('users.messages.noTenant'));" },
  { pattern: "message.success('用户创建成功');", replacement: "message.success(t('users.messages.createSuccess'));" },
  { pattern: "message.error('创建用户失败');", replacement: "message.error(t('users.messages.createFailed'));" },
  { pattern: "message.success('用户更新成功');", replacement: "message.success(t('users.messages.updateSuccess'));" },
  { pattern: "message.error('更新用户失败');", replacement: "message.error(t('users.messages.updateFailed'));" },
  { pattern: "message.success('用户删除成功');", replacement: "message.success(t('users.messages.deleteSuccess'));" },
  { pattern: "message.error('删除用户失败');", replacement: "message.error(t('users.messages.deleteFailed'));" },
  { pattern: "message.success(newStatus ? '用户已激活' : '用户已禁用');", replacement: "message.success(newStatus ? t('users.messages.activated') : t('users.messages.deactivated'));" },
  { pattern: "message.error('状态更新失败');", replacement: "message.error(t('users.messages.statusUpdateFailed'));" },
  { pattern: "message.success('密码重置成功');", replacement: "message.success(t('users.messages.passwordResetSuccess'));" },
  { pattern: "message.error('密码重置失败');", replacement: "message.error(t('users.messages.passwordResetFailed'));" },
  { pattern: "message.success('导出成功');", replacement: "message.success(t('users.messages.exportSuccess'));" },

  // table columns
  { pattern: "title: '用户名',\n      dataIndex: 'username',\n      key: 'username',\n      render: (text: string, record: User) => (\n        <Space>\n          <Text strong>{text}</Text>\n          {!record.active && <Tag color=\"red\">已禁用</Tag>}\n        </Space>\n      ),", replacement: "title: t('users.columns.username'),\n      dataIndex: 'username',\n      key: 'username',\n      render: (text: string, record: User) => (\n        <Space>\n          <Text strong>{text}</Text>\n          {!record.active && <Tag color=\"red\">{t('users.statusTag.deactivated')}</Tag>}\n        </Space>\n      )," },
  { pattern: "title: '姓名',", replacement: "title: t('users.columns.name')," },
  { pattern: "title: '邮箱',", replacement: "title: t('users.columns.email')," },
  { pattern: "title: '部门',", replacement: "title: t('users.columns.department')," },
  { pattern: "title: '电话',", replacement: "title: t('users.columns.phone')," },
  { pattern: "title: '状态',", replacement: "title: t('users.columns.status')," },
  { pattern: "title: '创建时间',", replacement: "title: t('users.columns.createdAt')," },
  { pattern: "title: '操作',", replacement: "title: t('common.action')," },

  // status switch labels
  { pattern: "checkedChildren=\"激活\"", replacement: "checkedChildren={t('users.statusTag.active')}" },
  { pattern: "unCheckedChildren=\"禁用\"", replacement: "unCheckedChildren={t('users.statusTag.deactivated')}" },

  // dropdown actions
  { pattern: "label: '编辑',", replacement: "label: t('common.edit')," },
  { pattern: "label: '重置密码',", replacement: "label: t('users.actions.resetPassword')," },
  { pattern: "label: '删除',", replacement: "label: t('common.delete')," },
  { pattern: "title: '确认删除',", replacement: "title: t('common.confirmDelete')," },
  { pattern: "content: `确定要删除用户 ${record.name} 吗？`,", replacement: "content: t('users.confirmDelete', { name: record.name })," },
  { pattern: "aria-label=\"更多操作\"", replacement: "aria-label={t('common.actions')}" },

  // page title
  { pattern: "            用户管理", replacement: "{t('users.title')}" },
  { pattern: "<Text type=\"secondary\">管理系统用户账户、权限和状态</Text>", replacement: "<Text type=\"secondary\">{t('users.description')}</Text>" },

  // statistics
  { pattern: "title=\"总用户数\"", replacement: "title={t('users.stats.total')}" },
  { pattern: "title=\"活跃用户\"", replacement: "title={t('users.stats.active')}" },
  { pattern: "title=\"禁用用户\"", replacement: "title={t('users.stats.inactive')}" },

  // filters
  { pattern: "placeholder=\"搜索用户名、姓名、邮箱\"", replacement: "placeholder={t('users.searchPlaceholder')}" },
  { pattern: "placeholder=\"状态筛选\"", replacement: "placeholder={t('users.filter.status')}" },
  { pattern: "placeholder=\"部门筛选\"", replacement: "placeholder={t('users.filter.department')}" },

  // status filter options
  { pattern: "{ value: 'active', label: '激活' },", replacement: "{ value: 'active', label: t('users.statusTag.active') }," },
  { pattern: "{ value: 'inactive', label: '禁用' },", replacement: "{ value: 'inactive', label: t('users.statusTag.deactivated') }," },

  // buttons
  { pattern: "                新建用户", replacement: "{t('users.createUser')}" },
  { pattern: "                导出", replacement: "{t('common.export')}" },

  // empty state
  { pattern: "<Empty description=\"暂无用户数据\" image={Empty.PRESENTED_IMAGE_SIMPLE}>", replacement: "<Empty description={t('users.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE}>" },
  { pattern: "              创建第一个用户", replacement: "{t('users.createFirst')}" },

  // CSV export
  { pattern: "用户列表_${new Date().toISOString().split('T')[0]}.csv", replacement: "{t('users.exportFilename')}_${new Date().toISOString().split('T')[0]}.csv" },

  // pagination
  { pattern: "showTotal: total => `共 ${total} 条记录`,", replacement: "showTotal: total => t('common.totalLabel', { total: total })," },

  // modals
  { pattern: "title=\"新建用户\"", replacement: "title={t('users.createUser')}" },
  { pattern: "title=\"编辑用户\"", replacement: "title={t('users.editUser')}" },
  { pattern: "title=\"重置密码\"", replacement: "title={t('users.actions.resetPassword')}" },

  // form labels and placeholders
  { pattern: "label=\"用户名\"", replacement: "label={t('users.columns.username')}" },
  { pattern: "label=\"姓名\"", replacement: "label={t('users.columns.name')}" },
  { pattern: "label=\"邮箱\"", replacement: "label={t('users.columns.email')}" },
  { pattern: "label=\"电话\"", replacement: "label={t('users.columns.phone')}" },
  { pattern: "label=\"部门\"", replacement: "label={t('users.columns.department')}" },
  { pattern: "label=\"密码\"", replacement: "label={t('users.form.password')}" },
  { pattern: "label=\"新密码\"", replacement: "label={t('users.form.newPassword')}" },

  // form rules
  { pattern: "{ required: true, message: '请输入用户名' },", replacement: "{ required: true, message: t('users.form.requiredUsername') }," },
  { pattern: "{ min: 3, message: '用户名至少3个字符' },", replacement: "{ min: 3, message: t('users.form.minUsername') }," },
  { pattern: "{ required: true, message: '请输入姓名' }", replacement: "{ required: true, message: t('users.form.requiredName') }" },
  { pattern: "{ required: true, message: '请输入邮箱' },", replacement: "{ required: true, message: t('users.form.requiredEmail') }," },
  { pattern: "{ type: 'email', message: '请输入有效的邮箱地址' },", replacement: "{ type: 'email', message: t('users.form.invalidEmail') }," },
  { pattern: "{ required: true, message: '请输入密码' },", replacement: "{ required: true, message: t('users.form.requiredPassword') }," },
  { pattern: "{ min: 6, message: '密码至少6个字符' },", replacement: "{ min: 6, message: t('users.form.minPassword') }," },
  { pattern: "{ required: true, message: '请输入新密码' },", replacement: "{ required: true, message: t('users.form.requiredNewPassword') }," },

  // form placeholders
  { pattern: "<Input placeholder=\"请输入用户名\" />", replacement: "<Input placeholder={t('users.form.usernamePlaceholder')} />" },
  { pattern: "<Input placeholder=\"请输入姓名\" />", replacement: "<Input placeholder={t('users.form.namePlaceholder')} />" },
  { pattern: "<Input placeholder=\"请输入邮箱\" />", replacement: "<Input placeholder={t('users.form.emailPlaceholder')} />" },
  { pattern: "<Input placeholder=\"请输入电话号码\" />", replacement: "<Input placeholder={t('users.form.phonePlaceholder')} />" },
  { pattern: "<Input placeholder=\"请输入密码\" />", replacement: "<Input.Password placeholder={t('users.form.passwordPlaceholder')} />" },
  { pattern: "<Input.Password placeholder=\"请输入密码\" />", replacement: "<Input.Password placeholder={t('users.form.passwordPlaceholder')} />" },
  { pattern: "<Input.Password placeholder=\"请输入新密码\" />", replacement: "<Input.Password placeholder={t('users.form.newPasswordPlaceholder')} />" },
  { pattern: "placeholder=\"请选择部门\"", replacement: "placeholder={t('users.form.departmentPlaceholder')}" },

  // department options
  { pattern: "{ value: 'IT部门', label: 'IT部门' }", replacement: "{ value: 'IT部门', label: t('users.departments.IT') }" },
  { pattern: "{ value: '财务部门', label: '财务部门' }", replacement: "{ value: '财务部门', label: t('users.departments.Finance') }" },
  { pattern: "{ value: '人事部门', label: '人事部门' }", replacement: "{ value: '人事部门', label: t('users.departments.HR') }" },
  { pattern: "{ value: '市场部门', label: '市场部门' }", replacement: "{ value: '市场部门', label: t('users.departments.Marketing') }" },

  // form buttons
  { pattern: "              创建用户", replacement: "{t('users.createUser')}" },
  { pattern: "                保存更改", replacement: "{t('users.form.save')}" },
  { pattern: "                重置密码", replacement: "{t('users.actions.resetPassword')}" },
  { pattern: "                取消", replacement: "{t('common.cancel')}" },
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
console.log(`Migrated ${totalChanges} patterns to ${path.basename(path.dirname(FILE))}/page.tsx`);
#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/app/(main)/admin/menus/page.tsx');
let content = fs.readFileSync(filePath, 'utf8');

if (!content.includes("import { useI18n }")) {
  content = content.replace(
    "import { iconMap, getIconByName } from '@/components/layout/sidebar/icons';",
    "import { iconMap, getIconByName } from '@/components/layout/sidebar/icons';\nimport { useI18n } from '@/lib/i18n';"
  );
}

content = content.replace(
  "export default function MenuManagementPage() {",
  "export default function MenuManagementPage() {\n  const { t } = useI18n();"
);

const map = {
  "'加载菜单列表失败'": "t('menus.loadFailed')",
  "'菜单更新成功'": "t('menus.updateSuccess')",
  "'菜单创建成功'": "t('menus.createSuccess')",
  "'保存菜单失败'": "t('common.saveFailed')",
  "'菜单已删除'": "t('menus.deleteSuccess')",
  "'删除失败'": "t('common.deleteFailed')",
  "'已禁用'": "t('menus.disabled')",
  "'已启用'": "t('menus.enabled')",
  "'操作失败'": "t('common.operationFailed')",
  "'已隐藏'": "t('menus.hidden')",
  "'已显示'": "t('menus.visible')",
  "title: '排序'": "title: t('menus.sortOrder')",
  "title: '名称'": "title: t('menus.menuName')",
  "title: '路径'": "title: t('menus.menuPath')",
  "title: '图标'": "title: t('menus.icon')",
  "title: '权限码'": "title: t('menus.permissionCode')",
  "title: '父菜单'": "title: t('menus.parentMenu')",
  "title: '操作'": "title: t('common.actions')",
  "r.isEnabled ? '已启用' : '已禁用'": "r.isEnabled ? t('menus.enabled') : t('menus.disabled')",
  "r.isVisible ? '可见' : '隐藏'": "r.isVisible ? t('menus.visible') : t('menus.hidden')",
  "        菜单管理\n      </Title>": "        {t('menus.title')}\n      </Title>",
  "管理侧边栏菜单：新增、编辑、删除，以及启用/禁用、显示/隐藏。": "t('menus.description')",
  "message=\"使用须知\"": "message={t('menus.usageTip')}",
  "title=\"总菜单数\"": "title={t('menus.totalMenus')}",
  "title=\"已启用\"": "title={t('menus.enabledMenus')}",
  "title=\"已禁用\"": "title={t('menus.disabledMenus')}",
  "title=\"已隐藏\"": "title={t('menus.hiddenMenus')}",
  "placeholder=\"搜索 名称 / 路径 / 权限码 / 图标\"": "placeholder={t('menus.searchPlaceholder')}",
  "          刷新\n        </Button>": "          {t('menus.refresh')}\n        </Button>",
  "          新建菜单\n        </Button>": "          {t('menus.createMenu')}\n        </Button>",
  "            {editing ? '编辑菜单' : '新建菜单'}": "            {editing ? t('menus.editMenu') : t('menus.createMenu')}",
  "okText=\"保存\"": "okText={t('common.save')}",
  "cancelText=\"取消\"": "cancelText={t('common.cancel')}",
  "label=\"菜单名称\"": "label={t('menus.menuName')}",
  "rules={[{ required: true, message: '请输入菜单名称' }]}": "rules={[{ required: true, message: t('common.required') }]}",
  "label=\"路径\"": "label={t('menus.menuPath')}",
  "{ required: true, message: '请输入路由路径' }": "{ required: true, message: t('menus.pathRequired', { name: '请输入路由路径' }) }",
  "message: '路径必须以 / 开头，仅支持字母、数字、- _ . /'": "message: t('menus.pathFormatError', { name: '路径必须以 / 开头，仅支持字母、数字、- _ . /' })",
  "tooltip=\"前端路由地址，例如 /admin/sla-templates\"": "tooltip={t('menus.pathTooltip', { name: '前端路由地址，例如 /admin/sla-templates' })}",
  "placeholder=\"/admin/sla-templates\"": "placeholder={t('menus.pathPlaceholder', { name: '/admin/sla-templates' })}",
  "label=\"图标\"": "label={t('menus.icon')}",
  "tooltip=\"Lucide React 图标名，可手输或从列表选择\"": "tooltip={t('menus.iconTooltip', { name: 'Lucide React 图标名，可手输或从列表选择' })}",
  "title=\"选择图标\"": "title={t('menus.selectIcon')}",
  "label=\"排序\"": "label={t('menus.sortOrder')}",
  "{ required: true, message: '请输入排序号' }": "{ required: true, message: t('menus.sortOrderRequired', { name: '请输入排序号' }) }",
  "label=\"权限码\"": "label={t('menus.permissionCode')}",
  "tooltip=\"菜单关联的权限码，留空则对所有登录用户可见\"": "tooltip={t('menus.permissionCodeTooltip', { name: '菜单关联的权限码，留空则对所有登录用户可见' })}",
  "label=\"父菜单\"": "label={t('menus.parentMenu')}",
  "tooltip=\"二级菜单需指定父菜单\"": "tooltip={t('menus.parentMenuTooltip', { name: '二级菜单需指定父菜单' })}",
  "label=\"描述\"": "label={t('common.description')}",
  "label=\"启用\"": "label={t('menus.isEnabled')}",
  "checkedChildren=\"启用\"": "checkedChildren={t('menus.enabled')}",
  "unCheckedChildren=\"禁用\"": "unCheckedChildren={t('menus.disabled')}",
  "label=\"可见\"": "label={t('menus.isVisible')}",
  "description=\"删除后不可恢复，关联的子菜单会变成根菜单。\"": "description={t('menus.deleteWarning', { name: '删除后不可恢复，关联的子菜单会变成根菜单。' })}",
  "title=\"确认删除\"": "title={t('common.confirmDelete')}",
  "okText=\"删除\"": "okText={t('common.delete')}",
  "title=\"编辑\"": "title={t('common.edit')}",
  "title=\"删除\"": "title={t('common.delete')}",
  "选项里没有找到'": "选项里没有找到'",
  "        全部状态\n      </Option>": "        全部状态\n      </Option>",
};

let count = 0;
for (const [from, to] of Object.entries(map)) {
  if (content.includes(from)) {
    content = content.split(from).join(to);
    count++;
  }
}

fs.writeFileSync(filePath, content);
console.log('Replacements made:', count);
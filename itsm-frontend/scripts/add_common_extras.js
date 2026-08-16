const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
let content = fs.readFileSync(filePath, 'utf8');

const lines = content.split('\n');

let commonEndIdx = -1;
for (let i = 0; i < lines.length; i++) {
  if (lines[i] === '    dashboard: {') {
    commonEndIdx = i - 1;
    break;
  }
}

const commonExtras = [
  "      status: '状态',",
  "      actions: '操作',",
  "      allStatus: '全部状态',",
  "      refreshAction: '刷新',",
  "      toggleEnableTooltip: '点击切换启用',",
  "      toggleVisibleTooltip: '点击切换可见',",
  "      deleteWarning: '删除后不可恢复，关联的子菜单会变成根菜单。',",
  "      pathRequired: '请输入路由路径',",
  "      pathFormatError: '路径必须以 / 开头，仅支持字母、数字、- _ . /',",
  "      pathTooltip: '前端路由地址，例如 /admin/sla-templates',",
  "      pathPlaceholder: '/admin/sla-templates',",
  "      iconPickerTitle: '选择图标',",
  "      iconTooltip: 'Lucide React 图标名，可手输或从列表选择',",
  "      sortOrderRequired: '请输入排序号',",
  "      permissionCodeTooltip: '菜单关联的权限码，留空则对所有登录用户可见',",
  "      parentMenuTooltip: '二级菜单需指定父菜单',",
  "      menuDescriptionTooltip: '可选：菜单用途说明',",
  "      topLevelMenu: '无（顶级菜单）',",
];

const checkKeys = ['status:', 'actions:', 'allStatus:'];
let conflict = false;
for (const key of checkKeys) {
  if (content.includes(`      ${key}`)) {
    conflict = true;
    break;
  }
}

if (conflict) {
  console.log('Conflict: keys already exist. Will skip if already added.');
} else {
  const newLines = [...lines.slice(0, commonEndIdx), ...commonExtras, ...lines.slice(commonEndIdx)];
  fs.writeFileSync(filePath, newLines.join('\n'));
  console.log('Added. New length:', newLines.length);
}
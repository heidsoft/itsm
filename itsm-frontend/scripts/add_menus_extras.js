const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
let content = fs.readFileSync(filePath, 'utf8');

const lines = content.split('\n');

let menusStart = -1;
for (let i = 0; i < lines.length; i++) {
  if (lines[i] === '    menus: {') {
    menusStart = i;
    break;
  }
}

let menusEnd = -1;
if (menusStart !== -1) {
  for (let j = menusStart + 1; j < lines.length; j++) {
    if (lines[j] === '    },') {
      let k = j + 1;
      while (k < lines.length && lines[k].startsWith('    // ')) k++;
      if (lines[k] && lines[k].startsWith('    workflowsPage:')) {
        menusEnd = j;
        break;
      }
    }
  }
}

if (menusStart === -1 || menusEnd === -1) {
  console.error('Could not find menus section. start:', menusStart, 'end:', menusEnd);
  process.exit(1);
}

console.log('Menus end at line', menusEnd + 1);

const missingKeys = [
  "      pathRequired: '请输入路由路径',",
  "      pathFormatError: '路径必须以 / 开头，仅支持字母、数字、- _ . /',",
  "      pathTooltip: '前端路由地址，例如 /admin/sla-templates',",
  "      pathPlaceholder: '/admin/sla-templates',",
  "      iconTooltip: 'Lucide React 图标名，可手输或从列表选择',",
  "      sortOrderRequired: '请输入排序号',",
  "      permissionCodeTooltip: '菜单关联的权限码，留空则对所有登录用户可见',",
  "      parentMenuTooltip: '二级菜单需指定父菜单',",
  "      deleteWarning: '删除后不可恢复，关联的子菜单会变成根菜单。',",
];

let added = 0;
for (const key of missingKeys) {
  const keyName = key.split(':')[0].trim();
  if (!content.includes(keyName + ':')) {
    lines.splice(menusEnd, 0, key);
    added++;
    menusEnd++;
  }
}

if (added > 0) {
  fs.writeFileSync(filePath, lines.join('\n'));
  console.log('Added keys:', added);
} else {
  console.log('No keys to add');
}
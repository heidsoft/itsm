const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
let content = fs.readFileSync(filePath, 'utf8');

const lines = content.split('\n');

let enUSStart = -1;
for (let i = 0; i < lines.length; i++) {
  if (lines[i] === "  'en-US': {") {
    enUSStart = i;
    break;
  }
}

let menusStart = -1;
for (let i = enUSStart; i < lines.length; i++) {
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
  console.error('Could not find en-US menus section. start:', menusStart, 'end:', menusEnd);
  process.exit(1);
}

const enUSMenusSection = lines.slice(menusStart, menusEnd + 1).join('\n');

const missingKeys = [
  "      pathRequired: 'Please enter the route path',",
  "      pathFormatError: 'Path must start with / and only contain letters, digits, - _ . /',",
  "      pathTooltip: 'Frontend route path, e.g. /admin/sla-templates',",
  "      pathPlaceholder: '/admin/sla-templates',",
  "      iconTooltip: 'Lucide React icon name; type it or pick from the list',",
  "      sortOrderRequired: 'Please enter sort order',",
  "      permissionCodeTooltip: 'Permission code linked to this menu; leave empty to allow all logged-in users',",
  "      parentMenuTooltip: 'Submenu items must specify a parent menu',",
  "      deleteWarning: 'This action cannot be undone. Submenus will become root menus.',",
];

let added = 0;
for (const key of missingKeys) {
  const keyName = key.split(':')[0].trim();
  if (!enUSMenusSection.includes(keyName + ':')) {
    lines.splice(menusEnd, 0, key);
    added++;
    menusEnd++;
  }
}

if (added > 0) {
  fs.writeFileSync(filePath, lines.join('\n'));
  console.log('Added en-US keys:', added);
} else {
  console.log('No keys to add');
}
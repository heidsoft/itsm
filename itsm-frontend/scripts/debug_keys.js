const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
let content = fs.readFileSync(filePath, 'utf8');

const keyNames = [
  'pathRequired',
  'pathFormatError',
  'pathTooltip',
  'pathPlaceholder',
  'iconTooltip',
  'sortOrderRequired',
  'permissionCodeTooltip',
  'parentMenuTooltip',
  'deleteWarning',
];

for (const keyName of keyNames) {
  const found = content.includes(keyName + ':');
  console.log(keyName + ':', found);
}
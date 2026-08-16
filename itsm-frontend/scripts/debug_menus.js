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
    if (lines[j] === '    },' && lines[j + 1] && lines[j + 1].startsWith('    workflowsPage:')) {
      menusEnd = j;
      break;
    }
  }
}

if (menusStart === -1 || menusEnd === -1) {
  console.error('Could not find menus section. start:', menusStart, 'end:', menusEnd);
  console.log('menus line:', JSON.stringify(lines[menusStart]));
  console.log('lines[1238]:', JSON.stringify(lines[1238]));
  console.log('lines[1239]:', JSON.stringify(lines[1239]));
  console.log('lines[1240]:', JSON.stringify(lines[1240]));
  console.log('startsWith check:', lines[1239] && lines[1239].startsWith('    workflowsPage:'));
  process.exit(1);
}
const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
const newSectionsPath = path.resolve(__dirname, '../zh_cn_new_sections.txt');

const original = fs.readFileSync(filePath, 'utf8');
const newSectionsRaw = fs.readFileSync(newSectionsPath, 'utf8');

const newSectionsLines = newSectionsRaw
  .split('\n')
  .filter((line) => {
    const trimmed = line.trim();
    if (trimmed.startsWith('# ===')) return false;
    if (trimmed.startsWith('// New:')) return false;
    return true;
  });

while (newSectionsLines.length && newSectionsLines[0].trim() === '') {
  newSectionsLines.shift();
}

const newSections = newSectionsLines.join('\n');

const lines = original.split('\n');

let zhCNCloseIdx = -1;
for (let i = 0; i < lines.length; i++) {
  if (lines[i] === '  },' && lines[i + 1] && lines[i + 1].startsWith("  'en-US':")) {
    zhCNCloseIdx = i;
    break;
  }
}

if (zhCNCloseIdx === -1) {
  console.error('Could not find zh-CN closing line');
  process.exit(1);
}

console.log('Inserting zh-CN at line', zhCNCloseIdx + 1, '(0-indexed)');
console.log('Lines before:', lines.length);
console.log('Lines to insert:', newSectionsLines.length);

const newLines = [
  ...lines.slice(0, zhCNCloseIdx),
  ...newSectionsLines,
  ...lines.slice(zhCNCloseIdx),
];

fs.writeFileSync(filePath, newLines.join('\n'));
console.log('Done. New file line count:', newLines.length);
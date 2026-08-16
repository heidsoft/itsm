const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
const newSectionsPath = path.resolve(__dirname, '../zh_cn_new_sections.txt');

const current = fs.readFileSync(filePath, 'utf8');
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

const lines = current.split('\n');

let badStart = -1;
for (let i = 0; i < lines.length; i++) {
  if (lines[i].trim().startsWith('# ===')) {
    badStart = i;
    break;
  }
}

let badEnd = -1;
for (let i = badStart; i < lines.length; i++) {
  if (lines[i] === '  },' && lines[i + 1] && lines[i + 1].startsWith("  'en-US':")) {
    badEnd = i;
    break;
  }
}

if (badStart === -1 || badEnd === -1) {
  console.error('Could not find bad block boundaries. badStart:', badStart, 'badEnd:', badEnd);
  process.exit(1);
}

console.log('Removing lines', badStart + 1, 'to', badEnd, 'inclusive');
console.log('Inserting', newSectionsLines.length, 'new lines');

const cleaned = [
  ...lines.slice(0, badStart),
  ...newSectionsLines,
  ...lines.slice(badEnd),
];

fs.writeFileSync(filePath, cleaned.join('\n'));
console.log('Done. New file line count:', cleaned.length);
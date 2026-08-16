const fs = require('fs');
const path = require('path');

const filePath = path.resolve(__dirname, '../src/lib/i18n/translations.ts');
const enUSSectionsPath = path.resolve(__dirname, '../en_us_new_sections.txt');

const current = fs.readFileSync(filePath, 'utf8');
const enUSLines = fs.readFileSync(enUSSectionsPath, 'utf8').split('\n');

const lines = current.split('\n');

let enUSCloseIdx = -1;
for (let i = lines.length - 1; i >= 0; i--) {
  if (lines[i] === '  },') {
    if (lines[i + 1] === '};') {
      enUSCloseIdx = i;
      break;
    }
  }
}

if (enUSCloseIdx === -1) {
  console.error('Could not find en-US closing line');
  process.exit(1);
}

console.log('Inserting en-US at line', enUSCloseIdx + 1, '(0-indexed)');
console.log('Lines before:', lines.length);
console.log('Lines to insert:', enUSLines.length);

const newLines = [
  ...lines.slice(0, enUSCloseIdx),
  ...enUSLines,
  ...lines.slice(enUSCloseIdx),
];

fs.writeFileSync(filePath, newLines.join('\n'));
console.log('Done. New file line count:', newLines.length);
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

if (enUSStart === -1) {
  console.error('Could not find en-US start');
  process.exit(1);
}

const replacements = [
  ['description: \'manage projects空间\'', 'description: \'manage project spaces\''],
  ['description: \'manage applications安装实例\'', 'description: \'manage application installations\''],
  ['description: \'SLA 监控仪表盘\'', 'description: \'SLA Monitoring Dashboard\''],
  ['description: \'analyze process bottlenecks\'', 'description: \'Analyze process bottlenecks\''],
  ['description: \'Process Instances管理\'', 'description: \'Process Instance Management\''],
  ['description: \'Ticket Approval流程\'', 'description: \'Ticket Approval Workflow\''],
  ['description: \'Approval中心\'', 'description: \'Approval Center\''],
  ['description: \'Service Request管理\'', 'description: \'Service Request Management\''],
  ['description: \'Assets管理\'', 'description: \'Asset Management\''],
  ['description: \'Licenses管理\'', 'description: \'License Management\''],
  ['description: \'Onboarding管理\'', 'description: \'Onboarding Management\''],
  ['description: \'Messages中心\'', 'description: \'Message Center\''],
  ['description: \'CommonPage\'', 'description: \'Common Page\''],
];

let count = 0;
for (let i = enUSStart; i < lines.length; i++) {
  for (const [from, to] of replacements) {
    if (lines[i].includes(from)) {
      lines[i] = lines[i].split(from).join(to);
      count++;
    }
  }
}

fs.writeFileSync(filePath, lines.join('\n'));
console.log('Replacements applied:', count);
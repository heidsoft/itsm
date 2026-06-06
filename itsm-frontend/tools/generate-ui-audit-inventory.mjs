import fs from 'node:fs';
import path from 'node:path';

const repoRoot = process.cwd();
const frontendRoot = path.basename(repoRoot) === 'itsm-frontend' ? repoRoot : path.join(repoRoot, 'itsm-frontend');
const mainPagesRoot = path.join(frontendRoot, 'src', 'app', '(main)');
const workflowComponentsRoot = path.join(frontendRoot, 'src', 'components', 'workflow');
const cmdbComponentsRoot = path.join(frontendRoot, 'src', 'components', 'cmdb');
const outputDir = path.join(frontendRoot, 'docs', 'ui-audit');

function walk(dir, predicate) {
  const result = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      result.push(...walk(fullPath, predicate));
    } else if (predicate(fullPath)) {
      result.push(fullPath);
    }
  }
  return result;
}

function classifyPageType(relativePath) {
  if (relativePath.includes('/create/') || relativePath.endsWith('/create/page.tsx')) return 'create';
  if (relativePath.includes('/new/') || relativePath.endsWith('/new/page.tsx')) return 'create';
  if (relativePath.includes('/edit/')) return 'edit';
  if (relativePath.includes('/[id]/')) return 'detail';
  if (relativePath.includes('/dashboard/')) return 'dashboard';
  if (relativePath.includes('/reports/') || relativePath.startsWith('reports/')) return 'report';
  if (relativePath.startsWith('admin/')) return 'admin';
  return 'list';
}

function classifyBatch(topLevelSegment) {
  if (['workflow', 'workflows', 'cmdb', 'tickets', 'changes', 'incidents', 'problems'].includes(topLevelSegment)) {
    return 'batch-1';
  }
  if (['reports', 'service-catalog', 'service-requests', 'knowledge', 'assets', 'licenses'].includes(topLevelSegment)) {
    return 'batch-2';
  }
  return 'batch-3';
}

function detectIssues(filePath, domainKind) {
  const source = fs.readFileSync(filePath, 'utf8');
  const lineCount = source.split('\n').length;
  const anyCount = (source.match(/\bany\b/g) || []).length;
  const consoleCount = (source.match(/console\./g) || []).length;
  const eslintDisableCount = (source.match(/eslint-disable/g) || []).length;

  const findings = [];

  if (lineCount >= 800) {
    findings.push({
      severity: 'P1',
      category: '实现质量',
      description: `超长单文件（${lineCount} 行），建议拆分页面编排、统计区、筛选区和详情/弹窗逻辑。`,
      recommendation: '拆分为页面壳层 + section 组件 + 独立 hooks / mapper',
      sharedLayer: false,
      blocksFollowup: true,
    });
  } else if (lineCount >= 500) {
    findings.push({
      severity: 'P2',
      category: '实现质量',
      description: `文件复杂度偏高（${lineCount} 行），建议优先拆分筛选栏、表格列配置和模态逻辑。`,
      recommendation: '抽出局部 section / toolbar / modal 组件',
      sharedLayer: false,
      blocksFollowup: false,
    });
  }

  if (anyCount >= 12) {
    findings.push({
      severity: 'P1',
      category: '实现质量',
      description: `存在较多 any 使用（${anyCount} 处），页面直接消费非结构化数据的风险较高。`,
      recommendation: '迁移数据适配到 API mapper 或 hook，并补内部类型定义',
      sharedLayer: true,
      blocksFollowup: domainKind === 'component',
    });
  } else if (anyCount >= 4) {
    findings.push({
      severity: 'P2',
      category: '实现质量',
      description: `存在多处 any 使用（${anyCount} 处），建议补齐局部类型与返回值约束。`,
      recommendation: '补足局部类型并减少页面内 any',
      sharedLayer: false,
      blocksFollowup: false,
    });
  }

  if (consoleCount > 0) {
    findings.push({
      severity: 'P2',
      category: '交互反馈',
      description: `检测到 console 输出（${consoleCount} 处），错误反馈仍偏开发态。`,
      recommendation: '统一通过 message / Alert / 错误态组件反馈，并保留结构化错误处理',
      sharedLayer: false,
      blocksFollowup: false,
    });
  }

  if (eslintDisableCount > 0) {
    findings.push({
      severity: 'P2',
      category: '实现质量',
      description: `存在 eslint-disable（${eslintDisableCount} 处），建议回到依赖和状态模型层面修正。`,
      recommendation: '消除临时关闭规则的依赖问题或拆分 effect / callback',
      sharedLayer: false,
      blocksFollowup: false,
    });
  }

  if (domainKind === 'page' && /<Button/.test(source) && /<Select/.test(source) && /<Table/.test(source)) {
    findings.push({
      severity: 'P3',
      category: '视觉一致性',
      description: '页面同时承载筛选、主操作和表格区，建议统一复用页面头部、筛选栏和统计卡基线。',
      recommendation: '收口到共享页面头部/筛选栏/统计卡组件',
      sharedLayer: true,
      blocksFollowup: false,
    });
  }

  return { lineCount, anyCount, consoleCount, eslintDisableCount, findings };
}

function toPageEntry(filePath) {
  const relativePath = path.relative(mainPagesRoot, filePath).replaceAll(path.sep, '/');
  const topLevelSegment = relativePath.split('/')[0];
  const analysis = detectIssues(filePath, 'page');

  return {
    path: relativePath,
    pageType: classifyPageType(relativePath),
    batch: classifyBatch(topLevelSegment),
    domain: topLevelSegment,
    lineCount: analysis.lineCount,
    issueCount: analysis.findings.length,
    issues: analysis.findings,
  };
}

function toComponentEntry(filePath, domain) {
  const relativePath = path.relative(frontendRoot, filePath).replaceAll(path.sep, '/');
  const analysis = detectIssues(filePath, 'component');

  return {
    path: relativePath,
    domain,
    lineCount: analysis.lineCount,
    issueCount: analysis.findings.length,
    issues: analysis.findings,
  };
}

const pageEntries = walk(mainPagesRoot, file => file.endsWith('page.tsx'))
  .sort()
  .map(toPageEntry);

const componentEntries = [
  ...walk(workflowComponentsRoot, file => file.endsWith('.tsx') || file.endsWith('.ts')).map(file =>
    toComponentEntry(file, 'workflow')
  ),
  ...walk(cmdbComponentsRoot, file => file.endsWith('.tsx') || file.endsWith('.ts')).map(file =>
    toComponentEntry(file, 'cmdb')
  ),
].sort((a, b) => b.lineCount - a.lineCount);

const topPages = [...pageEntries].sort((a, b) => b.lineCount - a.lineCount).slice(0, 15);
const topComponents = [...componentEntries].slice(0, 15);

const summary = {
  generatedAt: new Date().toISOString(),
  totalPages: pageEntries.length,
  pagesByBatch: pageEntries.reduce((acc, item) => {
    acc[item.batch] = (acc[item.batch] || 0) + 1;
    return acc;
  }, {}),
  pagesByDomain: pageEntries.reduce((acc, item) => {
    acc[item.domain] = (acc[item.domain] || 0) + 1;
    return acc;
  }, {}),
  topPages,
  topComponents,
};

const inventory = {
  summary,
  pages: pageEntries,
  complexComponents: componentEntries,
};

fs.mkdirSync(outputDir, { recursive: true });
fs.writeFileSync(path.join(outputDir, 'inventory.json'), `${JSON.stringify(inventory, null, 2)}\n`);

const markdown = [
  '# Frontend UI Audit Inventory',
  '',
  `Generated at: ${summary.generatedAt}`,
  '',
  '## Coverage',
  '',
  `- Total pages: ${summary.totalPages}`,
  `- Batch 1 pages: ${summary.pagesByBatch['batch-1'] || 0}`,
  `- Batch 2 pages: ${summary.pagesByBatch['batch-2'] || 0}`,
  `- Batch 3 pages: ${summary.pagesByBatch['batch-3'] || 0}`,
  '',
  '## Highest Complexity Pages',
  '',
  '| Path | Batch | Type | Lines | Issues |',
  '| --- | --- | --- | ---: | ---: |',
  ...topPages.map(item => `| \`${item.path}\` | ${item.batch} | ${item.pageType} | ${item.lineCount} | ${item.issueCount} |`),
  '',
  '## Highest Complexity workflow/cmdb Components',
  '',
  '| Path | Domain | Lines | Issues |',
  '| --- | --- | ---: | ---: |',
  ...topComponents.map(item => `| \`${item.path}\` | ${item.domain} | ${item.lineCount} | ${item.issueCount} |`),
  '',
  '## Notes',
  '',
  '- Full machine-readable audit ledger lives in `docs/ui-audit/inventory.json`.',
  '- Severity and issue categories are heuristic seeds for manual review, not final product decisions.',
].join('\n');

fs.writeFileSync(path.join(outputDir, 'README.md'), `${markdown}\n`);

console.log(`Generated UI audit inventory with ${summary.totalPages} pages.`);

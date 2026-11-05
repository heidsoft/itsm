#!/usr/bin/env node
/**
 * Doc-claim check — keep the numbers the docs promise in step with the code.
 *
 * README and docs/tools/itsm-mcp.md are read by operators *and* by agents
 * deciding what the surface can do. A stale "268 commands" or "20 curated
 * tools" is a product statement that is no longer true, and reviewers do not
 * reliably notice a number drifting in prose. The claims with a stable source
 * of truth are asserted here and wired into `npm run check`:
 *
 *   docs/tools/itsm-mcp.md : 共 N 条命令 / 精选工具（N 个）/ 其余 N 条
 *   itsm-cli/README.md     : N 个精选工具 / 覆盖全部 N 条命令
 *
 * Run after `npm run build` (reads the compiled registry, the same source the
 * MCP server serves from).
 */
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '..');

const { registry } = await import(path.join(root, 'dist/agent/registry.js'));
const { buildTools } = await import(path.join(root, 'dist/mcp/catalog.js'));

const commands = registry().length;
const tools = buildTools();
const curated = tools.filter((t) => t.command).length;
const meta = tools.length - curated;
/** Every command not reachable through a native tool is reachable via itsm_call. */
const remaining = commands - curated;

const CHECKS = [
  {
    file: path.resolve(root, '..', 'docs/tools/itsm-mcp.md'),
    label: '命令总数',
    actual: commands,
    pattern: /共 (\d+) 条命令/,
  },
  {
    file: path.resolve(root, '..', 'docs/tools/itsm-mcp.md'),
    label: '精选工具数',
    actual: curated,
    pattern: /精选工具\*\*（(\d+) 个）/,
  },
  {
    file: path.resolve(root, '..', 'docs/tools/itsm-mcp.md'),
    label: '其余命令数',
    actual: remaining,
    pattern: /其余 (\d+) 条/,
  },
  {
    file: path.resolve(root, 'README.md'),
    label: '精选工具数',
    actual: curated,
    pattern: /(\d+) 个精选工具/,
  },
  {
    file: path.resolve(root, 'README.md'),
    label: '命令总数',
    actual: commands,
    pattern: /覆盖全部 (\d+) 条命令/,
  },
];

let failed = 0;
for (const check of CHECKS) {
  const rel = path.relative(path.resolve(root, '..'), check.file);
  const text = readFileSync(check.file, 'utf8');
  const match = check.pattern.exec(text);
  if (!match) {
    failed++;
    console.error(`✗ ${rel}: 找不到「${check.label}」声明（正则 ${check.pattern}）`);
    continue;
  }
  const claimed = Number(match[1]);
  if (claimed !== check.actual) {
    failed++;
    console.error(`✗ ${rel}: ${check.label} 写作 ${claimed}，实测 ${check.actual}`);
  } else {
    console.log(`✓ ${rel}: ${check.label} = ${claimed}`);
  }
}

if (failed) {
  console.error(`\n文档计数与代码不一致（${failed} 处）。更新文档后重跑；数字必须来自注册表/工具面，不要手写猜测。`);
  process.exit(1);
}
console.log(`✓ doc claims in sync (${commands} commands, ${curated} curated, ${meta} meta tools)`);

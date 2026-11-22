#!/usr/bin/env node
'use strict';

// 版本一致性棘轮：防止配置文件里的版本号在新的方向上继续漂移。
//
// 当前仓库存在多处历史遗留的版本字面量（见 version-drift-baseline.json），
// 选哪一个是"正式版本"属于发布决策，本守卫不替产品做这个决定。它只保证：
//   1. 每个受监控表面的取值仍与已登记的基线一致（无人悄悄改动）；
//   2. 有新表面与新版本不一致时立即失败，而不是继续积累漂移；
//   3. --strict 下所有表面必须与 canonical 一致（完成对齐后启用）。

const fs = require('node:fs');
const path = require('node:path');

const ROOT = process.env.VERSION_GUARD_ROOT || process.cwd();

const SURFACES = [
  {
    name: '.env.prod.example VERSION',
    file: '.env.prod.example',
    read: (s) => matchOne(s, /^VERSION=(.+)$/m),
  },
  {
    name: 'backend defaultAppVersion',
    file: 'itsm-backend/router/router.go',
    read: (s) => matchOne(s, /const\s+defaultAppVersion\s*=\s*"([^"]+)"/),
  },
  {
    name: 'GA readiness version',
    file: 'itsm-backend/router/ga_readiness.go',
    read: (s) => matchOne(s, /\bVersion:\s*"([^"]+)"/),
  },
  {
    name: 'frontend package version',
    file: 'itsm-frontend/package.json',
    read: (s) => matchOne(s, /"version"\s*:\s*"([^"]+)"/),
  },
  {
    name: 'docs release baseline',
    file: 'docs/documentation-governance.md',
    read: (s) => matchOne(s, /最新发布标签为\s*`?v?([0-9]+\.[0-9]+\.[0-9]+)/),
  },
];

function matchOne(content, re) {
  const m = content.match(re);
  return m ? m[1] : null;
}

function readSurface(surface) {
  const file = path.join(ROOT, surface.file);
  if (!fs.existsSync(file)) return { name: surface.name, file: surface.file, value: null, error: '文件不存在' };
  try {
    const value = surface.read(fs.readFileSync(file, 'utf8'));
    return { name: surface.name, file: surface.file, value, error: value ? null : '无法解析版本字段' };
  } catch (error) {
    return { name: surface.name, file: surface.file, value: null, error: error.message };
  }
}

function main() {
  const strict = process.argv.includes('--strict');
  const canonicalArg = process.argv.find((a) => a.startsWith('--canonical='));
  let canonical = canonicalArg ? canonicalArg.slice('--canonical='.length) : null;
  if (!canonical) {
    canonical = process.env.VERSION_GUARD_CANONICAL || null;
  }
  if (!canonical) {
    console.error('用法: node scripts/check-version-consistency.js --canonical=<version> [--strict]');
    process.exit(2);
  }

  const baselinePath = path.join(ROOT, 'scripts', 'version-drift-baseline.json');
  const baselineFile = fs.existsSync(baselinePath)
    ? JSON.parse(fs.readFileSync(baselinePath, 'utf8'))
    : {};
  const baseline = baselineFile.surfaces || {};

  const problems = [];
  const known = [];
  for (const surface of SURFACES) {
    const { name, file, value, error } = readSurface(surface);
    if (error) {
      problems.push(`${name} (${file}): ${error}`);
      continue;
    }
    if (value === canonical) continue;

    const registered = baseline[name];
    if (registered === value) {
      if (strict) problems.push(`${name}: ${value} 仍与 canonical ${canonical} 不一致`);
      else known.push(`${name}: ${value}（已登记的历史漂移）`);
      continue;
    }
    problems.push(`${name}: ${value}（既不等于 canonical ${canonical}，也与登记的基线 ${registered ?? '无'} 不同）`);
  }

  for (const line of known) console.log(`WARN ${line}`);
  for (const line of problems) console.error(`FAIL ${line}`);
  if (problems.length > 0) process.exit(1);

  console.log(`版本一致性检查通过（canonical=${canonical}，已登记漂移 ${known.length} 项）`);
}

main();

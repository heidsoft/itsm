#!/usr/bin/env node
'use strict';

// 版本一致性棘轮：防止配置文件里的版本号在新的方向上继续漂移。
//
// 2026-11-24：历史漂移已对账到 v1.6.60（见 scripts/version-drift-baseline.json）。
// 守卫语义：
//   1. 每个受监控表面的取值必须与 canonical 一致；
//   2. 若不一致且已在基线登记，advisory 模式 WARN、--strict 模式 FAIL；
//   3. 出现未登记的新版本立即 FAIL（防止继续积累漂移）；
//   4. 额外校验 README / README.en 的 Go 徽章与 itsm-backend/go.mod 的 go 指令一致。

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
  {
    name: 'MCP fallback version',
    file: 'itsm-backend/handlers/mcp/adapters.go',
    read: (s) => matchOne(s, /const\s+defaultVersion\s*=\s*"([^"]+)"/),
  },
  {
    name: 'swagger @version',
    file: 'itsm-backend/main.go',
    read: (s) => matchOne(s, /@version\s+([0-9]+\.[0-9]+\.[0-9]+)/),
  },
  {
    name: 'UPGRADE stable version',
    file: 'UPGRADE.md',
    read: (s) => matchOne(s, /当前稳定版本：`([0-9]+\.[0-9]+\.[0-9]+)`/),
  },
  {
    name: 'K8s backend image tag',
    file: 'deploy/k8s/20-backend.yaml',
    read: (s) => matchOne(s, /ghcr\.io\/heidsoft\/itsm-backend:([0-9]+\.[0-9]+\.[0-9]+)/),
  },
  {
    name: 'K8s frontend image tag',
    file: 'deploy/k8s/30-frontend.yaml',
    read: (s) => matchOne(s, /ghcr\.io\/heidsoft\/itsm-frontend:([0-9]+\.[0-9]+\.[0-9]+)/),
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

// Go 工具链版本与产品版本无关，单独按 go.mod 校验，避免把两者混为一谈。
function checkGoToolchain() {
  const problems = [];
  const goModPath = path.join(ROOT, 'itsm-backend', 'go.mod');
  if (!fs.existsSync(goModPath)) return ['itsm-backend/go.mod: 文件不存在'];
  const goMod = fs.readFileSync(goModPath, 'utf8');
  const expected = matchOne(goMod, /^go\s+([0-9]+\.[0-9]+\.[0-9]+)/m);
  if (!expected) return ['itsm-backend/go.mod: 无法解析 go 指令'];

  for (const file of ['README.md', 'README.en.md']) {
    const full = path.join(ROOT, file);
    if (!fs.existsSync(full)) {
      problems.push(`${file}: 文件不存在`);
      continue;
    }
    const value = matchOne(fs.readFileSync(full, 'utf8'), /badge\/Go-([0-9]+\.[0-9]+\.[0-9]+)-/);
    if (!value) {
      problems.push(`${file}: 无法解析 Go 徽章版本`);
      continue;
    }
    if (value !== expected) {
      problems.push(`${file}: Go 徽章 ${value} 与 itsm-backend/go.mod 的 ${expected} 不一致`);
    }
  }
  return problems;
}

function main() {
  const strict = process.argv.includes('--strict');
  const canonicalArg = process.argv.find((a) => a.startsWith('--canonical='));
  let canonical = canonicalArg ? canonicalArg.slice('--canonical='.length) : null;
  if (!canonical) canonical = process.env.VERSION_GUARD_CANONICAL || null;
  if (!canonical) {
    console.error('用法: node scripts/check-version-consistency.js --canonical=<version> [--strict]');
    process.exit(2);
  }
  canonical = String(canonical).replace(/^v/, '');

  const baselinePath = path.join(ROOT, 'scripts', 'version-drift-baseline.json');
  const baselineFile = fs.existsSync(baselinePath)
    ? JSON.parse(fs.readFileSync(baselinePath, 'utf8'))
    : {};
  const baseline = baselineFile.surfaces || {};

  const problems = checkGoToolchain();
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

  console.log(`版本一致性检查通过（canonical=${canonical}，受控表面 ${SURFACES.length} 个，Go 徽章 2 个，已登记漂移 ${known.length} 项）`);
}

main();

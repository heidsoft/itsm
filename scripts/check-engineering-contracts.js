#!/usr/bin/env node

'use strict';

const fs = require('node:fs');
const path = require('node:path');

const REPO_ROOT = path.resolve(__dirname, '..');

const CONTRACTS = [
  {
    id: 'browser-api-same-origin',
    file: 'itsm-frontend/src/lib/api/api-config.ts',
    require: [/API_BASE_URL\s*=\s*process\.env\.NEXT_PUBLIC_API_URL\s*\|\|\s*''/],
    forbid: [/localhost:8090/, /process\.env\.ITSM_BACKEND_URL/],
  },
  {
    id: 'production-env-api-base',
    file: '.env.prod.example',
    require: [/^NEXT_PUBLIC_API_URL=\s*$/m, /^ITSM_BACKEND_URL=http:\/\/itsm-backend:8090\s*$/m],
    forbid: [/^NEXT_PUBLIC_API_URL=\/api\s*$/m, /^NEXT_PUBLIC_API_URL=https?:\/\//m],
  },
  {
    id: 'frontend-docker-env-isolation',
    file: 'itsm-frontend/.dockerignore',
    require: [/^\.env\.\*\s*$/m],
    forbid: [],
  },
  {
    id: 'production-compose-runtime-contract',
    file: 'docker-compose.prod.yml',
    require: [
      /ITSM_CORS_ALLOWED_ORIGINS=/,
      /http:\/\/127\.0\.0\.1:3000/,
      /http:\/\/127\.0\.0\.1\/health/,
    ],
    forbid: [/^\s*- CORS_ALLOWED_ORIGINS=/m],
  },
  {
    id: 'frontend-ci-same-origin',
    file: '.github/workflows/frontend-ci.yml',
    require: [/NEXT_PUBLIC_API_URL:\s*''/],
    forbid: [/NEXT_PUBLIC_API_URL:\s*https?:\/\//],
  },
  {
    id: 'release-same-origin',
    file: '.github/workflows/release.yml',
    require: [/NEXT_PUBLIC_API_URL:\s*''/],
    forbid: [/NEXT_PUBLIC_API_URL:\s*https?:\/\//],
  },
  {
    id: 'reproducible-docs',
    file: '.github/workflows/docs.yml',
    require: [/pip install --requirement requirements-docs\.txt/],
    forbid: [/pip install mkdocs==/],
  },
];

function evaluateContracts(root = REPO_ROOT, contracts = CONTRACTS) {
  const failures = [];
  const passed = [];

  for (const contract of contracts) {
    const absolutePath = path.join(root, contract.file);
    if (!fs.existsSync(absolutePath)) {
      failures.push({ id: contract.id, file: contract.file, reason: 'file is missing' });
      continue;
    }

    const content = fs.readFileSync(absolutePath, 'utf8');
    const missing = contract.require.filter(pattern => !pattern.test(content));
    const forbidden = contract.forbid.filter(pattern => pattern.test(content));

    if (missing.length || forbidden.length) {
      failures.push({
        id: contract.id,
        file: contract.file,
        reason: [
          ...missing.map(pattern => `missing ${pattern}`),
          ...forbidden.map(pattern => `forbidden ${pattern}`),
        ].join('; '),
      });
    } else {
      passed.push({ id: contract.id, file: contract.file });
    }
  }

  return { passed, failures };
}

function main() {
  const result = evaluateContracts();
  const json = process.argv.includes('--json');

  if (json) {
    process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  } else {
    for (const item of result.passed) {
      console.log(`PASS ${item.id} (${item.file})`);
    }
    for (const item of result.failures) {
      console.error(`FAIL ${item.id} (${item.file}): ${item.reason}`);
    }
    console.log(`Engineering contracts: ${result.passed.length} passed, ${result.failures.length} failed`);
  }

  process.exitCode = result.failures.length ? 1 : 0;
}

if (require.main === module) {
  main();
}

module.exports = { CONTRACTS, evaluateContracts };

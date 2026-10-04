#!/usr/bin/env node
/**
 * P1-5: 列表信封契约测试
 * 检查后端列表 DTO 是否返回标准五键信封 { items, total, page, pageSize, totalPages }
 * 用法: node scripts/check-list-envelope-contract.js
 *
 * 规则: 所有列表接口必须返回 { items, total, page, pageSize, totalPages }
 * 禁止: records, list, size, limit, offset, totalCount 等别名
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const BACKEND_DIR = path.join(__dirname, '..', 'itsm-backend');

function findGoFiles(dir, pattern = /\.go$/) {
  const results = [];
  if (!fs.existsSync(dir)) return results;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (!entry.name.includes('ent/generated') && !entry.name.includes('vendor')) {
        results.push(...findGoFiles(fullPath, pattern));
      }
    } else if (pattern.test(entry.name)) {
      results.push(fullPath);
    }
  }
  return results;
}

function checkDtoFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const violations = [];

  // 查找 ListResponse 结构体
  let inListResponse = false;
  let braceDepth = 0;
  let structStart = 0;
  const fields = new Set();

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // 检测 ListResponse 结构体
    if (/type\s+\w*ListResponse\s+struct\s*\{/.test(line)) {
      inListResponse = true;
      structStart = i + 1;
      braceDepth = 0;
      fields.clear();
    }

    if (inListResponse) {
      // 提取字段名
      const fieldMatch = line.match(/^\s*(\w+)\s+/);
      if (fieldMatch) {
        fields.add(fieldMatch[1]);
      }

      if (/{/.test(line)) braceDepth++;
      if (/}/.test(line)) {
        braceDepth--;
        if (braceDepth === 0) {
          // 检查是否包含标准五键
          // 但允许纯 {items, total} 的不分页形状（docs/api-reference.md 承认的合法形状）
          // 只有声明了部分分页键（page/pageSize/totalPages 任一）时才要求凑满五元组
          const pagingKeys = ['Page', 'PageSize', 'TotalPages'];
          const declaresPaging = pagingKeys.some(k => fields.has(k));

          if (declaresPaging) {
            const required = ['Items', 'Total', 'Page', 'PageSize', 'TotalPages'];
            const missing = required.filter(f => !fields.has(f));

            if (missing.length > 0 && fields.size > 0) {
              violations.push({
                line: structStart,
                missing,
                found: Array.from(fields),
              });
            }
          } else {
            // 不分页形状至少要 items + total
            const required = ['Items', 'Total'];
            const missing = required.filter(f => !fields.has(f));

            if (missing.length > 0 && fields.size > 0) {
              violations.push({
                line: structStart,
                missing,
                found: Array.from(fields),
              });
            }
          }

          inListResponse = false;
        }
      }
    }
  }

  return violations;
}

function checkJsonTags(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const violations = [];

  // 查找禁止的 JSON tag
  const banned = [
    { pattern: /json:"records"/g, name: 'records' },
    { pattern: /json:"list"/g, name: 'list' },
    { pattern: /json:"size"/g, name: 'size' },
    { pattern: /json:"limit"/g, name: 'limit' },
    { pattern: /json:"offset"/g, name: 'offset' },
    { pattern: /json:"totalCount"/g, name: 'totalCount' },
  ];

  const lines = content.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    for (const ban of banned) {
      if (ban.pattern.test(line)) {
        violations.push({
          line: i + 1,
          banned: ban.name,
          snippet: line.trim(),
        });
      }
    }
  }

  return violations;
}

// Main
console.log('🔍 检查列表信封契约...');

const dtoFiles = findGoFiles(path.join(BACKEND_DIR, 'dto'), /dto\.go$/);
const handlerFiles = findGoFiles(path.join(BACKEND_DIR, 'handlers'));
const allFiles = [...dtoFiles, ...handlerFiles];

let totalViolations = 0;

// 检查 1: ListResponse 结构体是否包含标准五键
for (const file of dtoFiles) {
  const violations = checkDtoFile(file);
  if (violations.length > 0) {
    const relPath = path.relative(path.join(__dirname, '..'), file);
    for (const v of violations) {
      console.log(`\n❌ ${relPath}:${v.line} ListResponse 缺少标准字段`);
      console.log(`   缺少: ${v.missing.join(', ')}`);
      console.log(`   现有: ${v.found.join(', ')}`);
      totalViolations++;
    }
  }
}

// 检查 2: 禁止的 JSON tag
for (const file of allFiles) {
  const violations = checkJsonTags(file);
  if (violations.length > 0) {
    const relPath = path.relative(path.join(__dirname, '..'), file);
    for (const v of violations) {
      console.log(`\n❌ ${relPath}:${v.line} 使用禁止的 JSON tag`);
      console.log(`   禁止: "${v.banned}"`);
      console.log(`   代码: ${v.snippet}`);
      totalViolations++;
    }
  }
}

if (totalViolations > 0) {
  console.log(`\n共发现 ${totalViolations} 处列表信封违规。`);
  console.log('标准信封: { items, total, page, pageSize, totalPages }');
  console.log('禁止别名: records, list, size, limit, offset, totalCount');
  process.exit(1);
}

console.log('✅ 列表信封契约检查通过');
process.exit(0);

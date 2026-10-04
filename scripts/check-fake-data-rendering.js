#!/usr/bin/env node
/**
 * P1-8: 假数据渲染检测
 * 防止 Card/Table 里硬编码假数据（Avatar 假人、Progress 假进度等）
 * 用法: node scripts/check-fake-data-rendering.js
 *
 * 检测模式:
 * 1. Avatar 组件在 Table columns 中渲染（可能是假人）
 * 2. Progress percent={0} 或 percent={100} 且无真实数据源
 * 3. render: () => '硬编码字符串'（非 '-' 或 '暂无'）
 */

const fs = require('fs');
const path = require('path');

const SRC_DIR = path.join(__dirname, '..', 'itsm-frontend', 'src');
const EXCLUDE_DIRS = ['__tests__', 'node_modules', '.next'];

function shouldExclude(filePath) {
  return EXCLUDE_DIRS.some(dir => filePath.includes(dir));
}

function findTsxFiles(dir) {
  const results = [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (!shouldExclude(fullPath)) {
        results.push(...findTsxFiles(fullPath));
      }
    } else if (entry.name.endsWith('.tsx') || entry.name.endsWith('.jsx')) {
      results.push(fullPath);
    }
  }
  return results;
}

function checkFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const violations = [];

  // 检测 1: Table columns 中的 Avatar（可能是假人）
  let inColumns = false;
  let columnBraceDepth = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (/columns\s*=\s*\{?\[/.test(line)) {
      inColumns = true;
      columnBraceDepth = 0;
    }

    if (inColumns) {
      // 检测 Avatar 在 columns 中
      if (/<Avatar\b/.test(line) && !/avatarUrl|avatar.*record/i.test(line)) {
        violations.push({
          line: i + 1,
          type: 'Avatar in columns',
          snippet: line.trim(),
          suggestion: '如果无真实数据，改为 render: () => \'-\'',
        });
      }

      // 检测 Progress percent={0} 或 percent={100}
      if (/<Progress\b/.test(line) && /percent=\{[0100]\}/.test(line)) {
        violations.push({
          line: i + 1,
          type: 'Progress with hardcoded percent',
          snippet: line.trim(),
          suggestion: '如果无真实数据，改为空态展示',
        });
      }

      // 检测 render: () => '硬编码字符串'（排除合法的 '-' 和 '暂无'）
      if (/render:\s*\(\)\s*=>\s*['"]/.test(line)) {
        const match = line.match(/render:\s*\(\)\s*=>\s*['"]([^'"]+)['"]/);
        if (match) {
          const value = match[1];
          // 合法的空态展示
          const allowed = ['-', '暂无数据', '暂无', '-', 'N/A'];
          if (!allowed.includes(value) && value.length > 2) {
            violations.push({
              line: i + 1,
              type: 'Hardcoded render string',
              snippet: line.trim(),
              suggestion: `考虑使用 '-' 或 '暂无数据'`,
            });
          }
        }
      }

      if (/\]\}?/.test(line)) {
        inColumns = false;
      }
    }
  }

  return violations;
}

// Main
const files = findTsxFiles(SRC_DIR);
let totalViolations = 0;

console.log('🔍 检查假数据渲染...');

for (const file of files) {
  const violations = checkFile(file);
  if (violations.length > 0) {
    const relPath = path.relative(path.join(__dirname, '..'), file);
    for (const v of violations) {
      console.log(`\n❌ ${relPath}:${v.line}`);
      console.log(`   类型: ${v.type}`);
      console.log(`   代码: ${v.snippet}`);
      console.log(`   建议: ${v.suggestion}`);
      totalViolations++;
    }
  }
}

if (totalViolations > 0) {
  console.log(`\n共发现 ${totalViolations} 处可疑的假数据渲染。`);
  console.log('说明: 这些模式可能是假数据，也可能是真实数据，请人工确认。');
  console.log('如果确认是真实数据，可以忽略此警告。');
  // 不 exit 1，因为可能是误报，只作为警告
  process.exit(0);
}

console.log('✅ 未发现可疑的假数据渲染');
process.exit(0);

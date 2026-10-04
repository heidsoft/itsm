#!/usr/bin/env node
/**
 * P0-3: Alert 冗余描述检测
 * 防止 Alert 同时写 message + 长 description 导致冗余引导
 * 用法: node scripts/check-alert-verbosity.js
 *
 * 规则: Alert 组件如果同时有 message 和 description，
 * description 文本长度不应超过 60 字符（约 30 个汉字）。
 * 超过则认为冗余，应合并到 message 或删除。
 */

const fs = require('fs');
const path = require('path');
const { globSync } = require('fs');

const SRC_DIR = path.join(__dirname, '..', 'itsm-frontend', 'src');
const MAX_DESC_LENGTH = 60;

function findTsxFiles(dir) {
  const results = [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...findTsxFiles(fullPath));
    } else if (entry.name.endsWith('.tsx') || entry.name.endsWith('.jsx')) {
      results.push(fullPath);
    }
  }
  return results;
}

function extractStringFromProp(line) {
  // Match string literal: prop="text" or prop={'text'} or prop={"text"}
  const m = line.match(/=\s*"([^"]*)"/) || line.match(/=\s*\{['"]([^'"]*)['"]\}/);
  return m ? m[1] : null;
}

function checkFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const violations = [];

  let inAlert = false;
  let alertStartLine = 0;
  let messageValue = null;
  let descriptionValue = null;
  let braceDepth = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();

    // Detect <Alert start
    if (/<Alert\b/.test(line) && !inAlert) {
      inAlert = true;
      alertStartLine = i + 1;
      messageValue = null;
      descriptionValue = null;
      braceDepth = 0;
    }

    if (inAlert) {
      // Track message prop
      if (/message\s*=/.test(line)) {
        const val = extractStringFromProp(line);
        if (val) messageValue = val;
      }

      // Track description prop
      if (/description\s*=/.test(line)) {
        const val = extractStringFromProp(line);
        if (val) descriptionValue = val;
      }

      // Detect self-closing or end of Alert
      if (/\/>/.test(line) || (/<\/Alert>/.test(line))) {
        if (messageValue && descriptionValue && descriptionValue.length > MAX_DESC_LENGTH) {
          violations.push({
            line: alertStartLine,
            message: messageValue,
            description: descriptionValue,
            descLength: descriptionValue.length,
          });
        }
        inAlert = false;
      }
    }
  }

  return violations;
}

// Main
const files = findTsxFiles(SRC_DIR);
let totalViolations = 0;

console.log('🔍 检查 Alert 组件冗余描述...');

for (const file of files) {
  const violations = checkFile(file);
  if (violations.length > 0) {
    const relPath = path.relative(path.join(__dirname, '..'), file);
    for (const v of violations) {
      console.log(`\n❌ ${relPath}:${v.line}`);
      console.log(`   message: "${v.message}"`);
      console.log(`   description (${v.descLength}字): "${v.description}"`);
      totalViolations++;
    }
  }
}

if (totalViolations > 0) {
  console.log(`\n共发现 ${totalViolations} 处 Alert 冗余描述。`);
  console.log(`规则: description 长度不应超过 ${MAX_DESC_LENGTH} 字符。`);
  console.log('建议: 合并 message + description 为单条 message，或删除冗余描述。');
  process.exit(1);
}

console.log('✅ 未发现 Alert 冗余描述');
process.exit(0);

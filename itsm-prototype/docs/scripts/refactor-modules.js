#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

/**
 * ITSM 前端模块重构脚本
 * 目标：清理重复目录，统一模块结构
 */

const SRC_DIR = path.join(__dirname, '..', 'src');

// 需要合并的目录映射
const MERGE_MAPPINGS = {
  // 合并 app/lib 到 lib
  'app/lib': 'lib',
  
  // 合并 app/components 到 components
  'app/components': 'components',
  
  // 合并 app/hooks 到 lib/hooks
  'app/hooks': 'lib/hooks',
};

// 需要删除的重复文件
const DUPLICATE_FILES = [
  // HTTP客户端重复
  'lib/http-client.ts', // 保留 app/lib/http-client.ts
  'lib/api-client.ts', // 保留 app/lib/http-client.ts
  
  // 认证服务重复
  'lib/auth.ts', // 保留 app/lib/auth-service.ts
  
  // 事件API重复
  'lib/incident-api.ts', // 保留 app/lib/incident-api.ts
];

// 需要重组的组件
const COMPONENT_REORGANIZATION = {
  // UI组件
  'components/ui': [
    'app/components/ui',
    'lib/components/ui',
  ],
  
  // 业务组件
  'components/business': [
    'app/components/tickets',
    'app/components/common',
  ],
  
  // 布局组件
  'components/layout': [
    'app/components/layout',
    'lib/components/layout',
  ],
};

function log(message) {
  console.log(`[REFACTOR] ${message}`);
}

function ensureDir(dirPath) {
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
    log(`Created directory: ${dirPath}`);
  }
}

function copyFile(src, dest) {
  ensureDir(path.dirname(dest));
  
  if (fs.existsSync(dest)) {
    log(`Warning: File already exists: ${dest}`);
    return false;
  }
  
  fs.copyFileSync(src, dest);
  log(`Copied: ${src} -> ${dest}`);
  return true;
}

function moveFile(src, dest) {
  if (!fs.existsSync(src)) {
    log(`Warning: Source file not found: ${src}`);
    return false;
  }
  
  ensureDir(path.dirname(dest));
  
  if (fs.existsSync(dest)) {
    log(`Warning: Destination file already exists: ${dest}`);
    return false;
  }
  
  fs.renameSync(src, dest);
  log(`Moved: ${src} -> ${dest}`);
  return true;
}

function deleteFile(filePath) {
  if (fs.existsSync(filePath)) {
    fs.unlinkSync(filePath);
    log(`Deleted: ${filePath}`);
    return true;
  }
  return false;
}

function processDirectory(srcDir, destDir) {
  if (!fs.existsSync(srcDir)) {
    log(`Warning: Source directory not found: ${srcDir}`);
    return;
  }
  
  const items = fs.readdirSync(srcDir);
  
  for (const item of items) {
    const srcPath = path.join(srcDir, item);
    const destPath = path.join(destDir, item);
    const stat = fs.statSync(srcPath);
    
    if (stat.isDirectory()) {
      processDirectory(srcPath, destPath);
    } else {
      copyFile(srcPath, destPath);
    }
  }
}

function main() {
  log('Starting ITSM module refactoring...');
  
  // 1. 合并目录
  log('Step 1: Merging directories...');
  for (const [src, dest] of Object.entries(MERGE_MAPPINGS)) {
    const srcPath = path.join(SRC_DIR, src);
    const destPath = path.join(SRC_DIR, dest);
    
    if (fs.existsSync(srcPath)) {
      log(`Merging ${src} -> ${dest}`);
      processDirectory(srcPath, destPath);
    }
  }
  
  // 2. 删除重复文件
  log('Step 2: Removing duplicate files...');
  for (const file of DUPLICATE_FILES) {
    const filePath = path.join(SRC_DIR, file);
    deleteFile(filePath);
  }
  
  // 3. 重组组件
  log('Step 3: Reorganizing components...');
  for (const [dest, sources] of Object.entries(COMPONENT_REORGANIZATION)) {
    const destPath = path.join(SRC_DIR, dest);
    ensureDir(destPath);
    
    for (const source of sources) {
      const srcPath = path.join(SRC_DIR, source);
      if (fs.existsSync(srcPath)) {
        log(`Moving ${source} -> ${dest}`);
        processDirectory(srcPath, destPath);
      }
    }
  }
  
  log('Module refactoring completed!');
  log('Next steps:');
  log('1. Update import paths in all files');
  log('2. Update tsconfig.json path mappings');
  log('3. Run type checking to verify changes');
}

if (require.main === module) {
  main();
}

module.exports = { main };

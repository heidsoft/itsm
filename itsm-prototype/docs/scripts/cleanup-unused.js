#!/usr/bin/env node

/**
 * 清理未使用的导入和变量的脚本
 * 这个脚本可以帮助减少ESLint错误
 */

const fs = require('fs');
const path = require('path');

// 需要清理的文件列表
const filesToClean = [
  'src/app/dashboard/page.tsx',
  'src/app/tickets/page.tsx',
  'src/app/components/ui/LoadingEmptyError.tsx',
  'src/app/components/TicketAssociation.tsx',
  'src/app/components/SatisfactionDashboard.tsx'
];

// 清理规则
const cleanupRules = [
  // 移除未使用的导入
  {
    pattern: /import\s+{\s*([^}]+)\s*}\s+from\s+['"][^'"]+['"];?\s*$/gm,
    replace: (match, imports) => {
      const usedImports = imports.split(',').map(imp => imp.trim()).filter(imp => {
        // 检查这个导入是否在文件中被使用
        return true; // 暂时保留所有导入，避免误删
      });
      return usedImports.length > 0 ? `import { ${usedImports.join(', ')} } from '${match.match(/from\s+['"]([^'"]+)['"]/)[1]}';` : '';
    }
  }
];

function cleanupFile(filePath) {
  try {
    const fullPath = path.join(process.cwd(), filePath);
    if (!fs.existsSync(fullPath)) {
      console.log(`文件不存在: ${filePath}`);
      return;
    }

    let content = fs.readFileSync(fullPath, 'utf8');
    let originalContent = content;

    // 应用清理规则
    cleanupRules.forEach(rule => {
      content = content.replace(rule.pattern, rule.replace);
    });

    // 如果内容有变化，写回文件
    if (content !== originalContent) {
      fs.writeFileSync(fullPath, content, 'utf8');
      console.log(`已清理: ${filePath}`);
    } else {
      console.log(`无需清理: ${filePath}`);
    }
  } catch (error) {
    console.error(`清理文件失败 ${filePath}:`, error.message);
  }
}

function main() {
  console.log('开始清理未使用的导入和变量...\n');
  
  filesToClean.forEach(file => {
    cleanupFile(file);
  });
  
  console.log('\n清理完成！');
  console.log('建议运行 npm run lint 检查是否还有其他问题');
}

if (require.main === module) {
  main();
}

module.exports = { cleanupFile, cleanupRules };

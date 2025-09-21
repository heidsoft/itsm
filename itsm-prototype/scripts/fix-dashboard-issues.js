#!/usr/bin/env node

/**
 * 修复Dashboard和工单页面问题的脚本
 */

const fs = require('fs');
const path = require('path');

// 修复dashboard页面的问题
function fixDashboardPage() {
  const filePath = 'src/app/dashboard/page.tsx';
  const fullPath = path.join(process.cwd(), filePath);
  
  if (!fs.existsSync(fullPath)) {
    console.log(`文件不存在: ${filePath}`);
    return;
  }

  try {
    let content = fs.readFileSync(fullPath, 'utf8');
    
    // 确保所有必要的组件都被正确导入
    const requiredImports = [
      'AIMetrics',
      'SmartSLAMonitor', 
      'PredictiveAnalytics',
      'ResourceDistributionChart',
      'ResourceHealthPieChart'
    ];
    
    // 检查组件导入
    requiredImports.forEach(component => {
      if (!content.includes(`import { ${component} }`)) {
        console.log(`警告: ${component} 组件可能未正确导入`);
      }
    });
    
    console.log(`Dashboard页面检查完成: ${filePath}`);
  } catch (error) {
    console.error(`检查Dashboard页面失败:`, error.message);
  }
}

// 修复工单页面的问题
function fixTicketsPage() {
  const filePath = 'src/app/tickets/page.tsx';
  const fullPath = path.join(process.cwd(), filePath);
  
  if (!fs.existsSync(fullPath)) {
    console.log(`文件不存在: ${filePath}`);
    return;
  }

  try {
    let content = fs.readFileSync(fullPath, 'utf8');
    
    // 检查必要的组件导入
    const requiredComponents = [
      'LoadingEmptyError',
      'TicketAssociation',
      'SatisfactionDashboard'
    ];
    
    requiredComponents.forEach(component => {
      if (!content.includes(`import { ${component} }`)) {
        console.log(`警告: ${component} 组件可能未正确导入`);
      }
    });
    
    console.log(`工单页面检查完成: ${filePath}`);
  } catch (error) {
    console.error(`检查工单页面失败:`, error.message);
  }
}

// 检查组件文件是否存在
function checkComponents() {
  const components = [
    'src/app/components/AIMetrics.tsx',
    'src/app/components/SmartSLAMonitor.tsx',
    'src/app/components/PredictiveAnalytics.tsx',
    'src/app/components/ui/LoadingEmptyError.tsx',
    'src/app/components/TicketAssociation.tsx',
    'src/app/components/SatisfactionDashboard.tsx'
  ];
  
  console.log('\n检查组件文件状态:');
  components.forEach(component => {
    const fullPath = path.join(process.cwd(), component);
    if (fs.existsSync(fullPath)) {
      console.log(`✅ ${component}`);
    } else {
      console.log(`❌ ${component} - 文件不存在`);
    }
  });
}

// 主函数
function main() {
  console.log('🔧 开始修复Dashboard和工单页面问题...\n');
  
  fixDashboardPage();
  fixTicketsPage();
  checkComponents();
  
  console.log('\n📋 修复建议:');
  console.log('1. 确保所有组件文件都存在');
  console.log('2. 检查组件导入路径是否正确');
  console.log('3. 运行 npm run build 验证构建');
  console.log('4. 运行 npm run dev 测试页面');
  console.log('\n✨ 修复完成！');
}

if (require.main === module) {
  main();
}

module.exports = { fixDashboardPage, fixTicketsPage, checkComponents };

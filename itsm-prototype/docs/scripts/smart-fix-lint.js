#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Common lucide-react components that are frequently used
const commonComponents = [
  'Plus', 'CheckCircle', 'Clock', 'Users', 'Search', 'Settings', 'Trash2', 'XCircle',
  'Edit', 'Filter', 'Eye', 'Lock', 'Unlock', 'Calendar', 'Key', 'GitMerge', 'Cpu',
  'HardDrive', 'BookOpen', 'TrendingUp', 'AlertTriangle', 'Target', 'User', 'MessageSquare',
  'Tag', 'Shield', 'Sector', 'Bell', 'ListFilter', 'MoreHorizontal', 'PlusCircle',
  'ListClock'
];

function findUsedComponents(content) {
  const usedComponents = [];
  
  commonComponents.forEach(component => {
    // Check if component is used in JSX (as a component)
    const jsxPattern = new RegExp(`<${component}[\\s/>]`, 'g');
    if (jsxPattern.test(content)) {
      usedComponents.push(component);
    }
  });
  
  return usedComponents;
}

function fixImportStatement(content, usedComponents) {
  // Remove all lucide-react imports first
  content = content.replace(/import\s*{[^}]*}\s*from\s*['"]lucide-react['"];?\s*/g, '');
  
  // Add back only the used components
  if (usedComponents.length > 0) {
    const importStatement = `import { ${usedComponents.join(', ')} } from 'lucide-react';\n`;
    // Find the first import statement and add before it
    const lines = content.split('\n');
    let insertIndex = 0;
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].trim().startsWith('import ')) {
        insertIndex = i;
        break;
      }
    }
    lines.splice(insertIndex, 0, importStatement);
    content = lines.join('\n');
  }
  
  return content;
}

function removeUnusedVariables(content) {
  // Remove unused useState and useEffect imports
  const unusedHooks = ['useState', 'useEffect'];
  unusedHooks.forEach(hook => {
    const hookPattern = new RegExp(`\\b${hook}\\b`, 'g');
    const matches = content.match(hookPattern);
    if (matches && matches.length === 1) {
      // Only remove if it's in the import statement and not used elsewhere
      const importPattern = new RegExp(`import\\s*{[^}]*\\b${hook}\\b[^}]*}\\s*from\\s*['"]react['"]`, 'g');
      if (importPattern.test(content)) {
        content = content.replace(importPattern, (match) => {
          const newImports = match
            .replace(/import\s*{/, 'import {')
            .replace(/}\s*from\s*['"]react['"]/, '}')
            .replace(new RegExp(`\\s*,\\s*${hook}\\s*`), '')
            .replace(new RegExp(`\\s*${hook}\\s*,?\\s*`), '')
            .replace(/,\s*,/, ',')
            .replace(/{\s*,/, '{')
            .replace(/,\s*}/, '}')
            .replace(/{\s*}/, '');
          
          if (newImports.trim() === 'import {') {
            return '';
          }
          return newImports + " from 'react'";
        });
      }
    }
  });
  
  return content;
}

function processFile(filePath) {
  try {
    const fullPath = path.join(process.cwd(), filePath);
    if (!fs.existsSync(fullPath)) {
      console.log(`File not found: ${filePath}`);
      return;
    }
    
    let content = fs.readFileSync(fullPath, 'utf8');
    let modified = false;
    
    // Find used components
    const usedComponents = findUsedComponents(content);
    
    // Fix import statements
    const newContent = fixImportStatement(content, usedComponents);
    if (newContent !== content) {
      content = newContent;
      modified = true;
    }
    
    // Remove unused variables
    const cleanedContent = removeUnusedVariables(content);
    if (cleanedContent !== content) {
      content = cleanedContent;
      modified = true;
    }
    
    if (modified) {
      fs.writeFileSync(fullPath, content, 'utf8');
      console.log(`Fixed: ${filePath}`);
    }
  } catch (error) {
    console.error(`Error processing ${filePath}:`, error.message);
  }
}

// Process specific files that have parsing errors
const filesToProcess = [
  'src/app/admin/approval-chains/page.tsx',
  'src/app/admin/escalation-rules/page.tsx',
  'src/app/admin/groups/page.tsx',
  'src/app/admin/permissions/page.tsx',
  'src/app/admin/roles/page.tsx',
  'src/app/admin/service-catalogs/page.tsx',
  'src/app/admin/sla-definitions/page.tsx',
  'src/app/admin/system-config/page.tsx',
  'src/app/admin/tenants/page.tsx',
  'src/app/admin/users/page.tsx',
  'src/app/admin/workflows/page.tsx',
  'src/app/changes/[changeId]/page.tsx',
  'src/app/changes/page.tsx',
  'src/app/cmdb/[ciId]/page.tsx',
  'src/app/components/GlobalSearch.tsx',
  'src/app/components/MobileLayout.tsx',
  'src/app/components/TicketDetail.tsx',
  'src/app/components/ui/DataTable.tsx',
  'src/app/incidents/[incidentId]/page.tsx',
  'src/app/knowledge-base/[articleId]/page.tsx',
  'src/app/login/page.tsx',
  'src/app/my-requests/page.tsx',
  'src/app/problems/[problemId]/page.tsx',
  'src/app/sla/[slaId]/page.tsx'
];

console.log('Starting smart lint fixes...');
filesToProcess.forEach(processFile);
console.log('Smart lint fixes completed!'); 
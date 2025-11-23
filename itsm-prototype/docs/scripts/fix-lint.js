#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Common unused imports to remove
const unusedImports = [
  'Edit', 'Filter', 'Eye', 'Plus', 'Trash2', 'Lock', 'Unlock', 'Settings',
  'Calendar', 'Users', 'Clock', 'Key', 'Search', 'CheckCircle', 'XCircle',
  'GitMerge', 'Cpu', 'HardDrive', 'BookOpen', 'TrendingUp', 'AlertTriangle',
  'Target', 'User', 'MessageSquare', 'Tag', 'Shield', 'Sector', 'Bell',
  'ListFilter', 'MoreHorizontal', 'PlusCircle', 'useState', 'useEffect'
];

// Files to process
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
  'src/app/changes/new/page.tsx',
  'src/app/changes/page.tsx',
  'src/app/cmdb/[ciId]/page.tsx',
  'src/app/components/GlobalSearch.tsx',
  'src/app/components/MobileLayout.tsx',
  'src/app/components/TicketDetail.tsx',
  'src/app/components/ui/DataTable.tsx',
  'src/app/components/ui/FormField.tsx',
  'src/app/components/ui/VirtualList.tsx',
  'src/app/dashboard/charts.tsx',
  'src/app/debug-auth/page.tsx',
  'src/app/hooks/useCache.ts',
  'src/app/improvements/page.tsx',
  'src/app/incidents/[incidentId]/page.tsx',
  'src/app/incidents/page.tsx',
  'src/app/knowledge-base/[articleId]/page.tsx',
  'src/app/knowledge-base/page.tsx',
  'src/app/lib/api-config.ts',
  'src/app/lib/auth-service.ts',
  'src/app/lib/cache-manager.ts',
  'src/app/lib/http-client.ts',
  'src/app/lib/incident-api.ts',
  'src/app/lib/mock-data.ts',
  'src/app/lib/store.ts',
  'src/app/lib/ticket-api.ts',
  'src/app/login/page.tsx',
  'src/app/my-requests/page.tsx',
  'src/app/problems/[problemId]/page.tsx',
  'src/app/problems/page.tsx',
  'src/app/reports/page.tsx',
  'src/app/service-catalog/request/forms/ApplyVmForm.tsx',
  'src/app/sla/[slaId]/page.tsx',
  'src/app/sla/new/page.tsx',
  'src/app/sla/page.tsx',
  'src/app/tickets/[ticketId]/page.tsx',
  'src/app/tickets/create/page.tsx',
  'src/app/tickets/page.tsx'
];

function removeUnusedImports(content, filePath) {
  let modified = false;
  
  // Remove unused imports from lucide-react
  unusedImports.forEach(importName => {
    const importRegex = new RegExp(`\\b${importName}\\b`, 'g');
    if (content.includes(importName)) {
      // Check if it's in an import statement
      const importStatementRegex = new RegExp(`import\\s*{[^}]*\\b${importName}\\b[^}]*}\\s*from\\s*['"]lucide-react['"]`, 'g');
      if (importStatementRegex.test(content)) {
        // Remove the import from the import statement
        content = content.replace(importStatementRegex, (match) => {
          const newImports = match
            .replace(/import\s*{/, 'import {')
            .replace(/}\s*from\s*['"]lucide-react['"]/, '}')
            .replace(new RegExp(`\\s*,\\s*${importName}\\s*`), '')
            .replace(new RegExp(`\\s*${importName}\\s*,?\\s*`), '')
            .replace(/,\s*,/, ',')
            .replace(/{\s*,/, '{')
            .replace(/,\s*}/, '}')
            .replace(/{\s*}/, '');
          
          if (newImports.trim() === 'import {') {
            return '';
          }
          return newImports + " from 'lucide-react'";
        });
        modified = true;
      }
    }
  });
  
  // Remove empty import statements
  content = content.replace(/import\s*{\s*}\s*from\s*['"]lucide-react['"];?\s*/g, '');
  
  return { content, modified };
}

function fixAnyTypes(content) {
  // Replace common any types with more specific types
  const replacements = [
    { from: ': any', to: ': unknown' },
    { from: 'any[]', to: 'unknown[]' },
    { from: 'Record<string, any>', to: 'Record<string, unknown>' },
    { from: 'Promise<any>', to: 'Promise<unknown>' }
  ];
  
  let modified = false;
  replacements.forEach(({ from, to }) => {
    if (content.includes(from)) {
      content = content.replace(new RegExp(from.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), to);
      modified = true;
    }
  });
  
  return { content, modified };
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
    
    // Remove unused imports
    const importResult = removeUnusedImports(content, filePath);
    if (importResult.modified) {
      content = importResult.content;
      modified = true;
    }
    
    // Fix any types
    const typeResult = fixAnyTypes(content);
    if (typeResult.modified) {
      content = typeResult.content;
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

// Process all files
console.log('Starting lint fixes...');
filesToProcess.forEach(processFile);
console.log('Lint fixes completed!'); 
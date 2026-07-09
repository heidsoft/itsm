#!/usr/bin/env node
/**
 * API Path Static Check Script
 *
 * Compares frontend API paths against backend registered routes to prevent route drift.
 *
 * Usage:
 *   node scripts/check-api-paths.js [--verbose] [--json]
 *
 * Exit codes:
 *   0 - All paths match or only warnings
 *   1 - Missing paths or mismatches
 */

const fs = require('fs');
const path = require('path');

// Configuration
const BACKEND_ROUTER_PATH = path.join(__dirname, '../itsm-backend/router/router.go');
const FRONTEND_API_DIR = path.join(__dirname, '../itsm-frontend/src/lib');
const FRONTEND_API_SUBDIRS = ['api', 'services'];

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

// Get all TypeScript files in frontend API directories
function getFilesRecursively(dir, extensions = ['.ts', '.tsx']) {
  const files = [];

  if (!fs.existsSync(dir)) {
    return files;
  }

  function traverse(currentDir) {
    const entries = fs.readdirSync(currentDir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(currentDir, entry.name);
      if (entry.isDirectory()) {
        traverse(fullPath);
      } else if (extensions.some(ext => entry.name.endsWith(ext))) {
        files.push(fullPath);
      }
    }
  }

  traverse(dir);
  return files;
}

// Parse backend routes from router.go
function parseBackendRoutes(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const routes = [];

  // Match route patterns like:
  // .GET("/path", handler)
  // .POST("/path", handler)
  // .PUT("/path", handler)
  // .DELETE("/path", handler)
  // .PATCH("/path", handler)
  const routeRegex = /\.(\w+)\(["']([^"']+)["']\s*,/g;
  let match;

  while ((match = routeRegex.exec(content)) !== null) {
    const method = match[1].toUpperCase();
    let routePath = match[2];

    // Skip paths with path parameters for now (they'll be normalized)
    // e.g., /api/v1/users/:id -> /api/v1/users/*
    if (routePath.includes(':')) {
      routePath = routePath.replace(/:[^/]+/g, '*');
    }

    routes.push({
      method,
      path: routePath,
      raw: match[0],
    });
  }

  return routes;
}

// Extract API paths from frontend files
function extractFrontendPaths() {
  const files = [];
  for (const subdir of FRONTEND_API_SUBDIRS) {
    const dir = path.join(FRONTEND_API_DIR, subdir);
    files.push(...getFilesRecursively(dir));
  }

  const paths = [];

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf-8');
    const relPath = path.relative(process.cwd(), file);

    // Match patterns like:
    // httpClient.get('/api/v1/...')
    // httpClient.post('/api/v1/...')
    // '/api/v1/...' as endpoint
    const patterns = [
      /httpClient\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]/g,
      /['"](\/api\/v1\/[^'"]+)['"]/g,
      /`(https?:\/\/[^`]+)?(\/api\/v1\/[^`]+)`/g,
      /endpoint\s*=\s*['"]([^"']+)['"]/g,
      /baseUrl\s*=\s*['"]([^"']+)['"]/g,
    ];

    for (const pattern of patterns) {
      let match;
      while ((match = pattern.exec(content)) !== null) {
        let apiPath = match[2] || match[1];

        // Skip absolute URLs and template literals with variables
        if (apiPath.startsWith('http') || apiPath.includes('${')) {
          continue;
        }

        // Skip query parameters
        apiPath = apiPath.split('?')[0];

        // Skip paths with variables like ${id}
        if (apiPath.includes('${')) {
          continue;
        }

        // Normalize paths with IDs
        apiPath = apiPath.replace(/\/\d+/g, '/*');
        apiPath = apiPath.replace(/\/[a-f0-9-]{36}/gi, '/*');

        if (apiPath && apiPath.startsWith('/api/')) {
          paths.push({
            path: apiPath,
            file: relPath,
            method: match[1] || 'GET',
          });
        }
      }
    }
  }

  return paths;
}

// Build a map of backend routes by path
function buildBackendPathMap(routes) {
  const map = new Map();

  for (const route of routes) {
    const key = route.path;
    if (!map.has(key)) {
      map.set(key, new Set());
    }
    map.get(key).add(route.method);
  }

  return map;
}

// Normalize path for comparison
function normalizePath(p) {
  return p
    .replace(/\/\*/g, '/{id}')
    .replace(/\/\{\w+\}/g, '/*');
}

// Check paths
function checkPaths(frontendPaths, backendPathMap, verbose = false) {
  const results = {
    matched: [],
    missing: [],
    warnings: [],
  };

  // Normalize backend paths for comparison
  const normalizedBackend = new Map();
  for (const [backendPath, methods] of backendPathMap) {
    const normalized = normalizePath(backendPath);
    if (!normalizedBackend.has(normalized)) {
      normalizedBackend.set(normalized, { original: backendPath, methods });
    } else {
      // Merge methods
      for (const m of methods) {
        normalizedBackend.get(normalized).methods.add(m);
      }
    }
  }

  // Check each frontend path
  for (const fp of frontendPaths) {
    const normalized = normalizePath(fp.path);

    // Direct match
    if (normalizedBackend.has(normalized)) {
      results.matched.push(fp);
      if (verbose) {
        log(`  ✓ ${fp.path}`, 'green');
      }
      continue;
    }

    // Check if it's under a valid prefix
    let found = false;
    for (const backendPath of normalizedBackend.keys()) {
      if (normalized.startsWith(backendPath.replace(/\/\*$/, '/'))) {
        found = true;
        break;
      }
    }

    if (found) {
      results.matched.push(fp);
    } else {
      // Check if it might be a new endpoint that needs backend implementation
      results.missing.push(fp);
      log(`  ✗ ${fp.path} (${fp.file}) - NOT FOUND IN BACKEND`, 'red');
    }
  }

  // Check for common path variations
  const commonIssues = [];
  for (const fp of frontendPaths) {
    // Check for snake_case vs camelCase issues
    if (fp.path.includes('_')) {
      commonIssues.push(`  ⚠ ${fp.path} uses snake_case (consider camelCase)`);
    }
  }

  if (commonIssues.length > 0 && verbose) {
    log('\nCommon Issues:', 'yellow');
    commonIssues.forEach(issue => log(issue, 'yellow'));
  }

  return results;
}

// Main
function main() {
  const args = process.argv.slice(2);
  const verbose = args.includes('--verbose') || args.includes('-v');
  const jsonOutput = args.includes('--json') || args.includes('-j');

  log('🔍 ITSM API Path Static Check\n', 'cyan');
  log('Parsing backend routes...', 'blue');

  if (!fs.existsSync(BACKEND_ROUTER_PATH)) {
    log(`Error: Backend router not found at ${BACKEND_ROUTER_PATH}`, 'red');
    process.exit(1);
  }

  const backendRoutes = parseBackendRoutes(BACKEND_ROUTER_PATH);
  log(`  Found ${backendRoutes.length} backend routes`, 'green');

  log('\nExtracting frontend API paths...', 'blue');

  const frontendPaths = extractFrontendPaths();

  // Deduplicate
  const seen = new Set();
  const uniquePaths = frontendPaths.filter(fp => {
    const key = fp.path + fp.method;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });

  log(`  Found ${uniquePaths.length} frontend API paths`, 'green');

  log('\nChecking paths...\n', 'blue');

  const backendPathMap = buildBackendPathMap(backendRoutes);
  const results = checkPaths(uniquePaths, backendPathMap, verbose);

  // Output results
  log('\n📊 Results:', 'cyan');
  log(`  Matched: ${results.matched.length}`, 'green');
  log(`  Missing: ${results.missing.length}`, results.missing.length > 0 ? 'red' : 'green');

  if (jsonOutput) {
    console.log(JSON.stringify({
      matched: results.matched.length,
      missing: results.missing.length,
      missingPaths: results.missing,
      backendRoutes: backendRoutes.length,
      frontendPaths: uniquePaths.length,
    }, null, 2));
  }

  if (results.missing.length > 0) {
    log('\n⚠️  Some frontend paths are not registered in the backend!', 'yellow');
    log('    Either add the backend route or fix the frontend path.\n', 'yellow');
    process.exit(1);
  }

  log('\n✅ All paths are valid!', 'green');
  process.exit(0);
}

main();

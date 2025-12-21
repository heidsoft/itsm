#!/usr/bin/env node

/**
 * 前后端类型同步工具
 * 确保前端TypeScript类型与后端Go结构体保持一致
 */

const fs = require('fs');
const path = require('path');
const https = require('https');

// 配置
const config = {
  backendUrl: process.env.BACKEND_URL || 'http://localhost:8080',
  swaggerEndpoint: '/swagger.json',
  typesOutputDir: './itsm-frontend/src/types/generated',
  sharedTypesFile: './shared-types/common-types.json',
};

// 类型映射：Go -> TypeScript
const typeMapping = {
  'string': 'string',
  'int': 'number',
  'int64': 'number',
  'float64': 'number',
  'bool': 'boolean',
  'time.Time': 'Date',
  'interface{}': 'any',
  '*string': 'string | null',
  '*int': 'number | null',
  '*int64': 'number | null',
  '*float64': 'number | null',
  '*bool': 'boolean | null',
};

// 字段名转换：snake_case -> camelCase
const toCamelCase = (str) => {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
};

// 字段名转换：camelCase -> snake_case
const toSnakeCase = (str) => {
  return str.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
};

// 获取Swagger规范
async function getSwaggerSpec() {
  return new Promise((resolve, reject) => {
    const url = `${config.backendUrl}${config.swaggerEndpoint}`;
    
    https.get(url, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          resolve(JSON.parse(data));
        } catch (error) {
          reject(error);
        }
      });
    }).on('error', reject);
  });
}

// 解析类型定义
function parseType(schema, name = 'AnonymousType') {
  if (schema.$ref) {
    const refName = schema.$ref.replace('#/definitions/', '');
    return refName;
  }

  if (schema.type === 'array') {
    const itemType = parseType(schema.items);
    return `${itemType}[]`;
  }

  if (schema.type === 'object' && schema.properties) {
    const properties = Object.entries(schema.properties).map(([key, prop]) => {
      const type = parseType(prop);
      const isRequired = schema.required && schema.required.includes(key);
      const optional = isRequired ? '' : '?';
      return `  ${toCamelCase(key)}${optional}: ${type};`;
    });

    return `interface ${name} {\n${properties.join('\n')}\n}`;
  }

  if (schema.enum) {
    const enumValues = schema.enum.map(v => `'${v}'`).join(' | ');
    return enumValues;
  }

  return typeMapping[schema.type] || 'any';
}

// 生成TypeScript类型文件
function generateTypeFile(typeName, schema) {
  const typeDefinition = parseType(schema, typeName);
  const fileContent = `/**
 * 自动生成的前端类型定义
 * 来源: ${config.backendUrl}/swagger.json
 * 生成时间: ${new Date().toISOString()}
 */

export ${typeDefinition}

export default ${typeName};
`;

  return fileContent;
}

// 生成API端点配置
function generateApiEndpoints(swaggerSpec) {
  const endpoints = {};
  
  Object.entries(swaggerSpec.paths).forEach(([path, methods]) => {
    Object.entries(methods).forEach(([method, spec]) => {
      if (method === 'parameters') return;
      
      const tag = spec.tags?.[0] || 'default';
      const operationId = spec.operationId;
      
      if (!endpoints[tag]) {
        endpoints[tag] = {};
      }
      
      endpoints[tag][operationId] = {
        method: method.toUpperCase(),
        path: path.replace('/api/v1', ''),
        summary: spec.summary,
        description: spec.description,
        parameters: spec.parameters || [],
        responses: spec.responses || {},
      };
    });
  });

  return endpoints;
}

// 生成共享类型文件
function generateSharedTypes(swaggerSpec) {
  const sharedTypes = {
    api: {
      version: 'v1',
      baseURL: '/api/v1',
      endpoints: {
        auth: '/auth',
        users: '/users',
        incidents: '/incidents',
        changes: '/changes',
        services: '/services',
        dashboard: '/dashboard',
        sla: '/sla',
        reports: '/reports',
        knowledge: '/knowledge',
      },
    },
    response: {
      format: {
        success: {
          code: 200,
          message: 'success',
          data: 'T',
        },
        error: {
          code: 'number',
          message: 'string',
          details: 'object?',
        },
      },
    },
    pagination: {
      request: {
        page: 'number',
        page_size: 'number',
        sort_by: 'string?',
        sort_order: "'asc' | 'desc'?",
      },
      response: {
        items: 'T[]',
        total: 'number',
        page: 'number',
        page_size: 'number',
        total_pages: 'number',
      },
    },
    common: {
      status: {
        incident: swaggerSpec.definitions?.IncidentStatus?.enum || [
          'new', 'assigned', 'in_progress', 'resolved', 'closed', 'reopened'
        ],
        change: swaggerSpec.definitions?.ChangeStatus?.enum || [
          'draft', 'pending', 'approved', 'rejected', 
          'in_progress', 'completed', 'rolled_back', 'cancelled'
        ],
        service: swaggerSpec.definitions?.ServiceStatus?.enum || [
          'active', 'inactive', 'maintenance', 'degraded'
        ],
        user: swaggerSpec.definitions?.UserStatus?.enum || [
          'active', 'inactive', 'suspended', 'pending'
        ],
      },
      priority: swaggerSpec.definitions?.Priority?.enum || [
        'low', 'medium', 'high', 'critical'
      ],
      impact: swaggerSpec.definitions?.Impact?.enum || [
        'low', 'medium', 'high', 'critical'
      ],
      urgency: swaggerSpec.definitions?.Urgency?.enum || [
        'low', 'medium', 'high', 'critical'
      ],
    },
  };

  return JSON.stringify(sharedTypes, null, 2);
}

// 确保目录存在
function ensureDirectoryExists(dirPath) {
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
  }
}

// 主函数
async function main() {
  try {
    console.log('🔄 开始同步前后端类型...');
    
    // 获取Swagger规范
    console.log('📡 获取Swagger规范...');
    const swaggerSpec = await getSwaggerSpec();
    
    // 确保输出目录存在
    ensureDirectoryExists(config.typesOutputDir);
    
    // 生成类型文件
    console.log('📝 生成TypeScript类型文件...');
    Object.entries(swaggerSpec.definitions || {}).forEach(([typeName, schema]) => {
      const fileName = `${toCamelCase(typeName)}.ts`;
      const filePath = path.join(config.typesOutputDir, fileName);
      
      const fileContent = generateTypeFile(typeName, schema);
      fs.writeFileSync(filePath, fileContent);
      
      console.log(`✅ 生成类型文件: ${fileName}`);
    });
    
    // 生成共享类型文件
    console.log('🔄 更新共享类型文件...');
    const sharedTypesContent = generateSharedTypes(swaggerSpec);
    fs.writeFileSync(config.sharedTypesFile, sharedTypesContent);
    
    // 生成API端点信息
    console.log('🔗 生成API端点信息...');
    const apiEndpoints = generateApiEndpoints(swaggerSpec);
    const endpointsFile = path.join(config.typesOutputDir, 'api-endpoints.ts');
    
    const endpointsContent = `/**
 * 自动生成的API端点信息
 * 来源: ${config.backendUrl}/swagger.json
 * 生成时间: ${new Date().toISOString()}
 */

export const API_ENDPOINTS = ${JSON.stringify(apiEndpoints, null, 2)} as const;

export type ApiEndpoint = typeof API_ENDPOINTS;

export default API_ENDPOINTS;
`;
    
    fs.writeFileSync(endpointsFile, endpointsContent);
    
    console.log('✅ API端点信息生成完成');
    
    // 生成索引文件
    const indexFile = path.join(config.typesOutputDir, 'index.ts');
    const typeExports = Object.keys(swaggerSpec.definitions || {})
      .map(typeName => `export type { ${toCamelCase(typeName)} } from './${toCamelCase(typeName)}';`)
      .join('\n');
    
    const indexContent = `/**
 * 自动生成的类型导出索引
 * 生成时间: ${new Date().toISOString()}
 */

${typeExports}

export { API_ENDPOINTS } from './api-endpoints';
export type { ApiEndpoint } from './api-endpoints';
`;
    
    fs.writeFileSync(indexFile, indexContent);
    
    console.log('🎉 前后端类型同步完成！');
    console.log(`📁 输出目录: ${config.typesOutputDir}`);
    console.log(`📄 共享类型文件: ${config.sharedTypesFile}`);
    
  } catch (error) {
    console.error('❌ 同步失败:', error.message);
    process.exit(1);
  }
}

// 如果直接运行此脚本
if (require.main === module) {
  main();
}

module.exports = {
  main,
  parseType,
  toCamelCase,
  toSnakeCase,
  generateTypeFile,
  generateSharedTypes,
};
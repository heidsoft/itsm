/**
 * 查询缓存配置
 * 不同类型的查询使用不同的缓存时间
 */
export const CACHE_TIMES = {
  // 静态数据：几乎不变化的数据
  STATIC: {
    staleTime: 60 * 60 * 1000, // 1小时
    gcTime: 2 * 60 * 60 * 1000, // 2小时
  },
  // 半静态数据：变化不频繁的数据
  SEMI_STATIC: {
    staleTime: 15 * 60 * 1000, // 15分钟
    gcTime: 30 * 60 * 1000, // 30分钟
  },
  // 动态数据：经常变化的数据
  DYNAMIC: {
    staleTime: 1 * 60 * 1000, // 1分钟
    gcTime: 5 * 60 * 1000, // 5分钟
  },
  // 实时数据：需要最新的数据
  REAL_TIME: {
    staleTime: 0,
    gcTime: 1 * 60 * 1000, // 1分钟
  },
};

// 查询键前缀，用于分类缓存
export const QUERY_KEYS = {
  // 用户相关
  USER: 'user',
  USER_LIST: 'user-list',
  USER_PROFILE: 'user-profile',
  
  // 工单相关
  TICKET: 'ticket',
  TICKET_LIST: 'ticket-list',
  TICKET_STATS: 'ticket-stats',
  
  // CMDB相关
  CMDB_CI: 'cmdb-ci',
  CMDB_CI_LIST: 'cmdb-ci-list',
  CMDB_CI_TYPE: 'cmdb-ci-type',
  
  // 知识库相关
  KNOWLEDGE_ARTICLE: 'knowledge-article',
  KNOWLEDGE_CATEGORY: 'knowledge-category',
  
  // 工作流相关
  WORKFLOW: 'workflow',
  WORKFLOW_DEFINITION: 'workflow-definition',
  
  // 字典数据
  DICTIONARY: 'dictionary',
  TICKET_STATUS: 'ticket-status',
  TICKET_PRIORITY: 'ticket-priority',
  TICKET_CATEGORY: 'ticket-category',
  
  // 租户相关
  TENANT: 'tenant',
  TENANT_LIST: 'tenant-list',
};

/**
 * 工单模板增强类型定义
 * 遵循设计文档规范，支持高级字段配置、条件显示、版本控制等
 */

// ==================== 字段类型枚举 ====================

export enum FieldType {
  // 基础类型
  TEXT = 'text',
  TEXTAREA = 'textarea',
  NUMBER = 'number',
  DATE = 'date',
  DATETIME = 'datetime',
  TIME = 'time',
  
  // 选择类型
  SELECT = 'select',
  MULTI_SELECT = 'multi_select',
  RADIO = 'radio',
  CHECKBOX = 'checkbox',
  
  // 高级类型
  USER_PICKER = 'user_picker',
  DEPARTMENT_PICKER = 'department_picker',
  FILE_UPLOAD = 'file_upload',
  RICH_TEXT = 'rich_text',
  CASCADER = 'cascader',
  RATING = 'rating',
  SLIDER = 'slider',
  COLOR_PICKER = 'color_picker',
  
  // 特殊类型
  DIVIDER = 'divider',
  SECTION_TITLE = 'section_title',
}

// ==================== 字段验证 ====================

export interface FieldValidation {
  min?: number;                      // 最小值/最小长度
  max?: number;                      // 最大值/最大长度
  pattern?: string;                  // 正则表达式
  customMessage?: string;            // 自定义错误消息
  minLength?: number;                // 最小字符长度
  maxLength?: number;                // 最大字符长度
  minValue?: number;                 // 最小数值
  maxValue?: number;                 // 最大数值
  email?: boolean;                   // 邮箱格式
  url?: boolean;                     // URL格式
  phone?: boolean;                   // 电话格式
  custom?: (value: any) => boolean | string; // 自定义验证函数
}

// ==================== 字段选项 ====================

export interface FieldOption {
  label: string;
  value: string | number;
  color?: string;
  icon?: string;
  disabled?: boolean;
  children?: FieldOption[];         // 级联选择子选项
}

// ==================== 条件显示 ====================

export interface FieldConditional {
  field: string;                     // 依赖字段ID
  operator: 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'greater_than' | 'less_than' | 'in' | 'not_in';
  value: any;                        // 比较值
  and?: FieldConditional[];          // AND 条件组
  or?: FieldConditional[];           // OR 条件组
}

// ==================== 模板字段 ====================

export interface TemplateField {
  id: string;                        // 字段唯一ID
  name: string;                      // 字段名称（数据库字段名）
  label: string;                     // 字段标签（显示名称）
  type: FieldType;                   // 字段类型
  required: boolean;                 // 是否必填
  defaultValue?: any;                // 默认值
  placeholder?: string;              // 占位符
  helpText?: string;                 // 帮助文本
  tooltip?: string;                  // 工具提示
  validation?: FieldValidation;      // 验证规则
  options?: FieldOption[];           // 选项（下拉、单选、多选）
  conditional?: FieldConditional;    // 条件显示
  order: number;                     // 排序
  width?: number | string;           // 字段宽度（栅格）
  disabled?: boolean;                // 是否禁用
  hidden?: boolean;                  // 是否隐藏
  
  // 高级配置
  maxFileSize?: number;              // 文件最大大小（字节）
  acceptedFileTypes?: string[];      // 接受的文件类型
  multiple?: boolean;                // 是否多选（文件上传、用户选择等）
  showSearch?: boolean;              // 是否显示搜索（选择器）
  allowClear?: boolean;              // 是否允许清空
  
  // 富文本配置
  richTextConfig?: {
    toolbar?: string[];              // 工具栏配置
    height?: number;                 // 编辑器高度
    plugins?: string[];              // 插件
  };
  
  // 级联选择配置
  cascaderConfig?: {
    changeOnSelect?: boolean;
    showSearch?: boolean;
    loadData?: (selectedOptions: FieldOption[]) => Promise<void>;
  };
}

// ==================== 模板类别 ====================

export interface TemplateCategory {
  id: string;
  name: string;
  description?: string;
  icon?: string;
  color?: string;
  parentId?: string;
  order: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// ==================== 模板权限 ====================

export enum TemplateVisibility {
  PUBLIC = 'public',                 // 公开（所有人可见）
  PRIVATE = 'private',               // 私有（仅创建人可见）
  DEPARTMENT = 'department',         // 部门（指定部门可见）
  ROLE = 'role',                     // 角色（指定角色可见）
  TEAM = 'team',                     // 团队（指定团队可见）
}

export interface TemplatePermission {
  visibility: TemplateVisibility;
  allowedDepartments?: string[];     // 允许的部门IDs
  allowedRoles?: string[];           // 允许的角色IDs
  allowedTeams?: string[];           // 允许的团队IDs
  allowedUsers?: string[];           // 允许的用户IDs
  denyDepartments?: string[];        // 拒绝的部门IDs
  denyRoles?: string[];              // 拒绝的角色IDs
  denyUsers?: string[];              // 拒绝的用户IDs
}

// ==================== 模板默认值 ====================

export interface TemplateDefaults {
  type?: string;                     // 工单类型
  priority?: string;                 // 优先级
  category?: string;                 // 分类
  assigneeId?: string;               // 默认处理人
  teamId?: string;                   // 默认团队
  tags?: string[];                   // 默认标签
  slaId?: string;                    // SLA配置
  [key: string]: any;                // 其他自定义默认值
}

// ==================== 模板自动化 ====================

export interface TemplateAutomation {
  autoAssign?: boolean;              // 自动分配
  assignmentRule?: {
    type: 'round_robin' | 'load_balance' | 'skill_based' | 'random';
    targetTeamId?: string;
    targetUserIds?: string[];
    skillTags?: string[];
  };
  
  requiresApproval?: boolean;        // 需要审批
  approvalWorkflow?: {
    workflowId?: string;
    approvalLevel: 'none' | 'manager' | 'director' | 'executive' | 'custom';
    approvers?: string[];            // 审批人IDs
    approvalType: 'sequential' | 'parallel' | 'any_one';
  };
  
  autoNotify?: boolean;              // 自动通知
  notificationRules?: {
    recipients?: string[];           // 收件人IDs
    channels?: ('email' | 'sms' | 'in_app' | 'webhook')[];
    template?: string;               // 通知模板
    triggerEvents?: string[];        // 触发事件
  }[];
  
  autoTag?: boolean;                 // 自动标签
  tagRules?: {
    condition?: string;              // 条件表达式
    tags?: string[];
  }[];
}

// ==================== 模板版本 ====================

export interface TemplateVersion {
  version: string;                   // 版本号（如 1.0.0）
  changelog?: string;                // 变更日志
  createdBy: string;                 // 创建人ID
  createdAt: string;                 // 创建时间
  isPublished: boolean;              // 是否已发布
  isDraft: boolean;                  // 是否草稿
}

// ==================== 工单模板 ====================

export interface TicketTemplate {
  // 基础信息
  id: string;                        // 模板ID (UUID)
  name: string;                      // 模板名称
  description: string;               // 模板描述
  categoryId: string;                // 分类ID
  icon?: string;                     // 图标URL或图标名称
  coverImage?: string;               // 封面图URL
  color?: string;                    // 主题颜色
  
  // 字段配置
  fields: TemplateField[];           // 字段列表
  
  // 默认值
  defaults: TemplateDefaults;        // 默认值配置
  
  // 权限配置
  permission: TemplatePermission;    // 权限配置
  
  // 自动化配置
  automation?: TemplateAutomation;   // 自动化配置
  
  // 版本控制
  version: string;                   // 当前版本
  versionHistory?: TemplateVersion[]; // 版本历史
  
  // 状态和元数据
  isActive: boolean;                 // 是否启用
  isDraft: boolean;                  // 是否草稿
  isArchived: boolean;               // 是否归档
  usageCount: number;                // 使用次数
  rating: number;                    // 平均评分
  ratingCount: number;               // 评分人数
  
  // 审计字段
  createdBy: string;                 // 创建人ID
  createdAt: string;                 // 创建时间
  updatedBy?: string;                // 更新人ID
  updatedAt: string;                 // 更新时间
  publishedAt?: string;              // 发布时间
  publishedBy?: string;              // 发布人ID
  
  // 扩展字段
  metadata?: Record<string, any>;    // 元数据
  tags?: string[];                   // 标签
}

// ==================== API请求/响应类型 ====================

export interface CreateTemplateRequest {
  name: string;
  description: string;
  categoryId: string;
  fields: TemplateField[];
  defaults?: TemplateDefaults;
  permission?: TemplatePermission;
  automation?: TemplateAutomation;
  icon?: string;
  coverImage?: string;
  color?: string;
  tags?: string[];
}

export interface UpdateTemplateRequest extends Partial<CreateTemplateRequest> {
  isActive?: boolean;
  isDraft?: boolean;
  version?: string;
}

export interface TemplateListQuery {
  // 分页
  page?: number;
  pageSize?: number;
  
  // 搜索
  search?: string;
  
  // 筛选
  categoryId?: string;
  visibility?: TemplateVisibility;
  isActive?: boolean;
  isDraft?: boolean;
  isArchived?: boolean;
  tags?: string[];
  
  // 排序
  sortBy?: 'name' | 'usageCount' | 'rating' | 'createdAt' | 'updatedAt';
  sortOrder?: 'asc' | 'desc';
  
  // 权限筛选
  userId?: string;                   // 当前用户ID，用于权限过滤
}

export interface TemplateListResponse {
  templates: TicketTemplate[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CreateTicketFromTemplateRequest {
  templateId: string;
  fieldValues: Record<string, any>; // 字段值
  overrides?: {                     // 覆盖默认值
    priority?: string;
    assigneeId?: string;
    teamId?: string;
    tags?: string[];
    [key: string]: any;
  };
}

export interface TemplateUsageStats {
  templateId: string;
  usageCount: number;
  lastUsedAt?: string;
  avgCreationTime: number;          // 平均创建时间（秒）
  successRate: number;               // 成功率（%）
  topUsers: Array<{
    userId: string;
    username: string;
    count: number;
  }>;
  usageByDate: Array<{
    date: string;
    count: number;
  }>;
}

export interface TemplateRating {
  templateId: string;
  userId: string;
  rating: number;                    // 1-5星
  comment?: string;
  createdAt: string;
}

export interface TemplateDuplicateRequest {
  templateId: string;
  newName: string;
  copyFields?: boolean;
  copyAutomation?: boolean;
  copyPermission?: boolean;
}

export interface TemplateExportFormat {
  format: 'json' | 'yaml' | 'excel';
  includeStats?: boolean;
  includeVersionHistory?: boolean;
}

export interface TemplateImportRequest {
  format: 'json' | 'yaml' | 'excel';
  data: string | File;
  overwriteExisting?: boolean;
  validateOnly?: boolean;
}

export interface TemplateValidationResult {
  isValid: boolean;
  errors: Array<{
    field: string;
    message: string;
    severity: 'error' | 'warning';
  }>;
  warnings: Array<{
    field: string;
    message: string;
  }>;
}

// ==================== 字段配置器使用的辅助类型 ====================

export interface FieldDesignerConfig {
  fieldTypes: Array<{
    type: FieldType;
    label: string;
    icon: string;
    description: string;
    category: 'basic' | 'advanced' | 'special';
    defaultConfig: Partial<TemplateField>;
  }>;
  
  validationRules: Array<{
    name: string;
    label: string;
    applicableTypes: FieldType[];
    configType: 'boolean' | 'number' | 'string' | 'regex';
  }>;
}

// ==================== 工具类型 ====================

export type TemplateFieldValue = string | number | boolean | string[] | number[] | Date | null;

export type TemplateFieldMap = Record<string, TemplateFieldValue>;

export interface TemplatePreview {
  template: TicketTemplate;
  sampleData?: TemplateFieldMap;
}

// ==================== 导出所有类型 ====================

export default TicketTemplate;


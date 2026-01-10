/**
 * 服务目录和自助门户类型定义
 */

// ==================== 服务目录 ====================

/**
 * 服务类别
 */
export enum ServiceCategory {
  IT_SERVICE = 'it_service',
  BUSINESS_SERVICE = 'business_service',
  TECHNICAL_SERVICE = 'technical_service',
  SUPPORT_SERVICE = 'support_service',
}

/**
 * 服务状态
 */
export enum ServiceStatus {
  DRAFT = 'draft',
  PUBLISHED = 'published',
  RETIRED = 'retired',
}

/**
 * 服务项
 */
export interface ServiceItem {
  id: string;
  name: string;
  category: ServiceCategory;
  status: ServiceStatus;
  shortDescription: string;
  fullDescription?: string;
  ciTypeId?: number;
  cloudServiceId?: number;
  
  // 图标和图片
  icon?: string;
  banner?: string;
  screenshots?: string[];
  
  // 服务详情
  provider?: string;              // 服务提供商
  owner?: number;                 // 负责人ID
  ownerName?: string;
  supportTeam?: string;           // 支持团队
  
  // 定价
  pricing?: {
    type: 'free' | 'fixed' | 'variable';
    amount?: number;
    currency?: string;
    billingCycle?: 'one_time' | 'monthly' | 'yearly';
  };
  
  // 可用性
  availability?: {
    businessHours?: string;
    supportLevel?: 'basic' | 'standard' | 'premium';
    responseTime?: number;        // 响应时间（小时）
    resolutionTime?: number;      // 解决时间（小时）
  };
  
  // 关联
  relatedCIs?: string[];          // 关联的CI
  requiredServices?: string[];    // 依赖的服务
  documentation?: string[];       // 文档链接
  
  // 表单配置
  requestForm?: ServiceRequestForm;
  
  // 统计
  viewCount?: number;
  requestCount?: number;
  rating?: number;
  
  // 标签和搜索
  tags: string[];
  searchKeywords?: string[];
  
  // 审批流程
  requiresApproval: boolean;
  approvalWorkflow?: string;      // 审批工作流ID
  
  // 元数据
  createdBy: number;
  createdByName: string;
  updatedBy?: number;
  updatedByName?: string;
  createdAt: Date;
  updatedAt: Date;
  publishedAt?: Date;
}

/**
 * 服务请求表单
 */
export interface ServiceRequestForm {
  fields: ServiceFormField[];
  layout?: 'single' | 'two_column';
  submitButtonText?: string;
  successMessage?: string;
}

/**
 * 表单字段
 */
export interface ServiceFormField {
  id: string;
  name: string;
  label: string;
  type: 'text' | 'textarea' | 'number' | 'date' | 'select' | 'multiselect' | 'checkbox' | 'radio' | 'file';
  required: boolean;
  placeholder?: string;
  defaultValue?: any;
  options?: Array<{
    label: string;
    value: string;
  }>;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
  helpText?: string;
  dependsOn?: {
    field: string;
    value: any;
  };
}

// ==================== 服务请求 ====================

/**
 * 服务请求状态
 */
export enum ServiceRequestStatus {
  DRAFT = 'draft',
  SUBMITTED = 'submitted',
  PENDING_APPROVAL = 'pending_approval',
  APPROVED = 'approved',
  REJECTED = 'rejected',
  IN_PROGRESS = 'in_progress',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled',
}

/**
 * 服务请求
 */
export interface ServiceRequest {
  id: number;
  requestNumber: string;
  serviceId: string;
  serviceName: string;
  
  status: ServiceRequestStatus;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  
  // 请求内容
  formData: Record<string, any>;
  additionalNotes?: string;
  attachments?: string[];
  
  // 关联工单
  ticketId?: number;
  ticketNumber?: string;
  
  // 审批信息
  approvals?: Array<{
    approverId: number;
    approverName: string;
    status: 'pending' | 'approved' | 'rejected';
    comment?: string;
    decidedAt?: Date;
  }>;
  
  // 履行信息
  fulfillmentNotes?: string;
  estimatedCompletionTime?: Date;
  actualCompletionTime?: Date;
  
  // 评价
  rating?: number;
  feedback?: string;
  
  // 请求人信息
  requestedBy: number;
  requestedByName: string;
  requestedByEmail?: string;
  requestedFor?: number;          // 为谁请求
  requestedForName?: string;
  
  // 元数据
  createdAt: Date;
  updatedAt: Date;
  submittedAt?: Date;
  completedAt?: Date;
}

// ==================== 自助门户 ====================

/**
 * 门户配置
 */
export interface PortalConfig {
  id: string;
  name: string;
  
  // 品牌
  branding: {
    logo?: string;
    primaryColor?: string;
    secondaryColor?: string;
    headerText?: string;
    footerText?: string;
  };
  
  // 首页配置
  homepage: {
    banner?: {
      title: string;
      subtitle?: string;
      image?: string;
    };
    featuredServices?: string[];   // 推荐服务ID
    announcement?: string;
    quickLinks?: Array<{
      title: string;
      url: string;
      icon?: string;
    }>;
  };
  
  // 功能配置
  features: {
    enableSearch: boolean;
    enableRating: boolean;
    enableFavorites: boolean;
    enableNotifications: boolean;
    showServicePrice: boolean;
    showServiceOwner: boolean;
  };
  
  // 自定义页面
  customPages?: Array<{
    id: string;
    title: string;
    slug: string;
    content: string;
    order: number;
  }>;
  
  updatedAt: Date;
}

/**
 * 服务收藏
 */
export interface ServiceFavorite {
  id: string;
  serviceId: string;
  userId: number;
  createdAt: Date;
}

/**
 * 服务评分
 */
export interface ServiceRating {
  id: string;
  serviceId: string;
  requestId?: number;
  userId: number;
  userName: string;
  rating: number;                 // 1-5
  comment?: string;
  helpful?: number;               // 有用计数
  createdAt: Date;
}

// ==================== 服务目录统计 ====================

/**
 * 服务目录统计
 */
export interface ServiceCatalogStats {
  totalServices: number;
  publishedServices: number;
  totalRequests: number;
  pendingRequests: number;
  completedRequests: number;
  
  servicesByCategory: Record<ServiceCategory, number>;
  requestsByStatus: Record<ServiceRequestStatus, number>;
  
  topServices: Array<{
    service: ServiceItem;
    requestCount: number;
    avgRating: number;
  }>;
  
  recentRequests: ServiceRequest[];
  
  trends: {
    date: string;
    requests: number;
    completions: number;
  }[];
}

/**
 * 服务分析
 */
export interface ServiceAnalytics {
  serviceId: string;
  period: {
    start: Date;
    end: Date;
  };
  
  metrics: {
    totalRequests: number;
    completedRequests: number;
    avgCompletionTime: number;    // 小时
    completionRate: number;        // 完成率（%）
    avgRating: number;
    totalViews: number;
  };
  
  requestTrend: {
    date: string;
    count: number;
  }[];
  
  userSatisfaction: {
    rating: number;
    count: number;
  }[];
  
  peakHours: {
    hour: number;
    count: number;
  }[];
}

// ==================== API请求/响应 ====================

/**
 * 创建服务请求
 */
export interface CreateServiceItemRequest {
  name: string;
  category: ServiceCategory;
  shortDescription: string;
  fullDescription?: string;
  ciTypeId?: number;
  cloudServiceId?: number;
  icon?: string;
  provider?: string;
  owner?: number;
  supportTeam?: string;
  pricing?: ServiceItem['pricing'];
  availability?: ServiceItem['availability'];
  requestForm?: ServiceRequestForm;
  tags?: string[];
  requiresApproval?: boolean;
  approvalWorkflow?: string;
  status?: ServiceStatus;
}

/**
 * 更新服务请求
 */
export type UpdateServiceItemRequest = Partial<CreateServiceItemRequest>;

/**
 * 创建服务请求
 */
export interface CreateServiceRequestRequest {
  serviceId: string;
  formData: Record<string, any>;
  priority?: 'low' | 'medium' | 'high' | 'urgent';
  additionalNotes?: string;
  requestedFor?: number;
}

/**
 * 服务查询参数
 */
export interface ServiceQuery {
  category?: ServiceCategory;
  status?: ServiceStatus;
  search?: string;
  tags?: string[];
  minRating?: number;
  page?: number;
  pageSize?: number;
  sortBy?: 'name' | 'rating' | 'popularity' | 'newest';
  sortOrder?: 'asc' | 'desc';
}

/**
 * 服务请求查询
 */
export interface ServiceRequestQuery {
  serviceId?: string;
  status?: ServiceRequestStatus;
  requestedBy?: number;
  startDate?: string;
  endDate?: string;
  page?: number;
  pageSize?: number;
}

export default ServiceItem;

export enum ServiceCategory {
  IT_SERVICE = 'it_service',
  BUSINESS_SERVICE = 'business_service',
  TECHNICAL_SERVICE = 'technical_service',
  SUPPORT_SERVICE = 'support_service',
}

export enum ServiceStatus {
  DRAFT = 'draft',
  PUBLISHED = 'published',
  RETIRED = 'retired',
  // Backend legacy alignment
  ENABLED = 'enabled',
  DISABLED = 'disabled',
}

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

/** Display labels only: category and pricing-unit values remain unchanged in API requests. */
export const SERVICE_CATEGORY_LABELS: Readonly<Record<string, string>> = {
  云计算: 'Cloud Resource Services',
  支持: 'Account & Permissions',
  安全: 'Security Services',
  数据库: 'Database',
  存储: 'Storage',
  网络: 'Network',
  通讯: 'Communications',
  协作: 'Collaboration',
  开发: 'Development',
  it_service: 'IT Service',
  business_service: 'Business Service',
  technical_service: 'Technical Service',
  support_service: 'Support Service',
};

const PRICING_UNIT_LABELS: Readonly<Record<string, string>> = {
  月: 'month',
  次: 'use',
  用户: 'user',
};

export const serviceCategoryLabel = (value?: string | null): string =>
  value ? (Object.hasOwn(SERVICE_CATEGORY_LABELS, value) ? SERVICE_CATEGORY_LABELS[value] : value) : '';

export const pricingUnitLabel = (value?: string | null): string =>
  value ? (Object.hasOwn(PRICING_UNIT_LABELS, value) ? PRICING_UNIT_LABELS[value] : value) : '';

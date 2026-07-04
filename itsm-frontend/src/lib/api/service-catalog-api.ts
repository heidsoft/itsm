/**
 * 服务目录 API 服务
 */

import { httpClient } from './http-client';
import type {
  ServiceItem,
  PortalConfig,
  ServiceFavorite,
  ServiceRating,
  ServiceCatalogStats,
  ServiceAnalytics,
  CreateServiceItemRequest,
  UpdateServiceItemRequest,
  CreateServiceRequestRequest,
  ServiceQuery,
  ServiceRequestQuery,
} from '@/types/service-catalog';

export class ServiceCatalogApi {
  // ==================== 内部适配（对齐后端 /api/v1/service-catalogs & /api/v1/service-requests） ====================

  private static unsupportedFeature(feature: string): never {
    throw new Error(`${feature}暂未接入后端，请在能力开放前关闭入口或补齐服务端接口。`);
  }

  private static toBackendStatus(status?: unknown): 'enabled' | 'disabled' | undefined {
    // V0：后端服务目录状态枚举为 enabled/disabled；前端为 draft/published/retired
    if (!status) return undefined;
    const s = String(status);
    if (s === 'published') return 'enabled';
    if (s === 'enabled') return 'enabled';
    if (s === 'disabled') return 'disabled';
    return 'disabled';
  }

  private static toFrontendStatus(status?: unknown) {
    const s = String(status || '');
    return (s === 'enabled' ? 'published' : 'retired') as any;
  }

  private static escapeCSV(value: unknown): string {
    const text =
      value instanceof Date
        ? value.toISOString()
        : value === undefined || value === null
          ? ''
          : String(value);
    return `"${text.replace(/"/g, '""')}"`;
  }

  private static toBackendRequestStatus(status?: unknown): string | undefined {
    if (!status) return undefined;
    const s = String(status);
    if (s === 'pending_approval' || s === 'pending') return 'submitted';
    if (s === 'approved') return 'security_approved';
    if (s === 'in_progress') return 'provisioning';
    if (s === 'completed') return 'delivered';
    return s;
  }

   
  private static toServiceItem(raw: any): ServiceItem {
    // 后端 dto.ServiceCatalogResponse: {id,name,category,description,deliveryTime,status,ciTypeId,cloudServiceId,createdAt,updatedAt}
    return {
      id: String(raw?.id),
      name: String(raw?.name || ''),
      // 这里保留后端 category 的原始字符串（前端页面目前以中文分类做统计/图标）
       
      category: (raw?.category as any) || ('it_service' as any),
      status: ServiceCatalogApi.toFrontendStatus(raw?.status),
      shortDescription: String(raw?.description || ''),
      fullDescription: String(raw?.description || ''),
      ciTypeId: typeof raw?.ciTypeId === 'number' ? raw.ciTypeId : undefined,
      cloudServiceId: typeof raw?.cloudServiceId === 'number' ? raw.cloudServiceId : undefined,
      tags: [],
      requiresApproval: true,
      createdBy: 0,
      createdByName: '',
      createdAt: raw?.createdAt ? new Date(raw.createdAt) : new Date(),
      updatedAt: raw?.updatedAt ? new Date(raw.updatedAt) : new Date(),
      availability: {
        // 后端 deliveryTime 为 string（天/小时口径未统一）；V0先用于展示，不做严格含义
        responseTime: raw?.deliveryTime ? Number(raw.deliveryTime) : undefined,
      },
    };
  }

   
  private static toServiceRequest(raw: any): any {
    const catalogId = raw?.catalogId ?? raw?.catalog_id ?? raw?.serviceId;
    const requesterId = raw?.requesterId ?? raw?.requester_id ?? raw?.requestedBy;
    const createdAt = raw?.createdAt ?? raw?.created_at;
    const updatedAt = raw?.updatedAt ?? raw?.updated_at;
    const catalog = raw?.catalog || {
      id: catalogId,
      name: raw?.serviceName || raw?.title || (catalogId ? `服务 #${catalogId}` : '未知服务'),
      category: raw?.category || '',
      description: raw?.reason || '',
    };
    const requester = raw?.requester || {
      id: requesterId,
      name: raw?.requesterName || raw?.requestedByName || (requesterId ? `用户 #${requesterId}` : '-'),
      email: raw?.requestedByEmail || '',
    };

    return {
      ...raw,
      requestNumber: raw?.requestNumber || `REQ-${String(raw?.id || 0).padStart(5, '0')}`,
      serviceId: String(catalogId || ''),
      serviceName: raw?.serviceName || catalog?.name || '-',
      requesterName: raw?.requesterName || requester?.name || '-',
      requestedBy: requesterId,
      requestedByName: raw?.requestedByName || requester?.name || '-',
      catalog,
      requester,
      catalogId,
      catalog_id: catalogId,
      requesterId,
      requester_id: requesterId,
      ciId: raw?.ciId ?? raw?.ci_id,
      ci_id: raw?.ci_id ?? raw?.ciId,
      formData: raw?.formData ?? raw?.form_data ?? {},
      form_data: raw?.form_data ?? raw?.formData ?? {},
      costCenter: raw?.costCenter ?? raw?.cost_center,
      cost_center: raw?.cost_center ?? raw?.costCenter,
      dataClassification: raw?.dataClassification ?? raw?.data_classification,
      data_classification: raw?.data_classification ?? raw?.dataClassification,
      needsPublicIP: raw?.needsPublicIP ?? raw?.needs_public_ip,
      needs_public_ip: raw?.needs_public_ip ?? raw?.needsPublicIP,
      sourceIPWhitelist: raw?.sourceIPWhitelist ?? raw?.source_ip_whitelist,
      source_ip_whitelist: raw?.source_ip_whitelist ?? raw?.sourceIPWhitelist,
      complianceAck: raw?.complianceAck ?? raw?.compliance_ack,
      compliance_ack: raw?.compliance_ack ?? raw?.complianceAck,
      currentLevel: raw?.currentLevel ?? raw?.current_level,
      current_level: raw?.current_level ?? raw?.currentLevel,
      totalLevels: raw?.totalLevels ?? raw?.total_levels,
      total_levels: raw?.total_levels ?? raw?.totalLevels,
      expireAt: raw?.expireAt ?? raw?.expire_at,
      expire_at: raw?.expire_at ?? raw?.expireAt,
      createdAt,
      created_at: createdAt,
      updatedAt,
      updated_at: updatedAt,
    };
  }

  // ==================== 服务项管理 ====================

  /**
   * 获取服务列表
   */
  static async getServices(query?: ServiceQuery): Promise<{
    services: ServiceItem[];
    total: number;
  }> {
    const page = query?.page ?? 1;
    const size = query?.pageSize ?? 10;
    const category = query?.category ? String(query.category) : undefined;
    const status = ServiceCatalogApi.toBackendStatus(query?.status);

    const resp = await httpClient.get<{
      catalogs: unknown[];
      services: unknown[];
      total: number;
      page: number;
      size: number;
    }>('/api/v1/service-catalogs', {
      page,
      size,
      ...(category ? { category } : {}),
      ...(status ? { status } : {}),
    });

    // 优先使用 services 字段，否则降级使用 catalogs
    const rawItems = resp.services || resp.catalogs || [];

    let services = rawItems.map(ServiceCatalogApi.toServiceItem);
    // 后端当前不支持 search；先在前端做兜底过滤
    if (query?.search) {
      const q = query.search.toLowerCase();
      services = services.filter(
        s =>
          (s.name || '').toLowerCase().includes(q) ||
          (s.shortDescription || '').toLowerCase().includes(q)
      );
    }

    return { services, total: resp.total || 0 };
  }

  /**
   * 获取单个服务
   */
  static async getService(id: string): Promise<ServiceItem> {
    const resp = await httpClient.get<any>(`/api/v1/service-catalogs/${id}`);
    return ServiceCatalogApi.toServiceItem(resp);
  }

  /**
   * 创建服务
   */
  static async createService(request: CreateServiceItemRequest): Promise<ServiceItem> {
    const payload = {
      name: request.name,
      category: String(request.category),
      description: request.shortDescription || request.fullDescription || '',
      ci_type_id: request.ciTypeId,
      cloud_service_id: request.cloudServiceId,
      delivery_time: String(
        request.availability?.responseTime ?? request.availability?.resolutionTime ?? 1
      ),
      status: ServiceCatalogApi.toBackendStatus((request as any).status) || 'enabled',
    };
    const resp = await httpClient.post<any>('/api/v1/service-catalogs', payload);
    return ServiceCatalogApi.toServiceItem(resp);
  }

  /**
   * 更新服务
   */
  static async updateService(id: string, request: UpdateServiceItemRequest): Promise<ServiceItem> {
    const payload: Record<string, unknown> = {};
    if (request.name !== undefined) payload.name = request.name;
    if (request.category !== undefined) payload.category = String(request.category);
    if (request.shortDescription !== undefined || request.fullDescription !== undefined) {
      payload.description = request.shortDescription || request.fullDescription || '';
    }
    if (request.availability?.responseTime !== undefined) {
      payload.delivery_time = String(request.availability.responseTime);
    }
    if (request.ciTypeId !== undefined) payload.ci_type_id = request.ciTypeId;
    if (request.cloudServiceId !== undefined) payload.cloud_service_id = request.cloudServiceId;
    const st = ServiceCatalogApi.toBackendStatus((request as any).status);
    if (st) payload.status = st;

    const resp = await httpClient.put<any>(`/api/v1/service-catalogs/${id}`, payload);
    return ServiceCatalogApi.toServiceItem(resp);
  }

  /**
   * 删除服务
   */
  static async deleteService(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/service-catalogs/${id}`);
  }

  /**
   * 发布服务
   */
  static async publishService(id: string): Promise<ServiceItem> {
    const resp = await httpClient.put<any>(`/api/v1/service-catalogs/${id}`, {
      status: 'enabled',
    });
    return ServiceCatalogApi.toServiceItem(resp);
  }

  /**
   * 停用服务
   */
  static async retireService(id: string): Promise<ServiceItem> {
    const resp = await httpClient.put<any>(`/api/v1/service-catalogs/${id}`, {
      status: 'disabled',
    });
    return ServiceCatalogApi.toServiceItem(resp);
  }

  /**
   * 复制服务
   */
  static async cloneService(id: string, name: string): Promise<ServiceItem> {
    const src = await ServiceCatalogApi.getService(id);
    return ServiceCatalogApi.createService({
      ...src,
      id: undefined as any,
      name,
    } as any);
  }

  // ==================== 服务请求管理 ====================

  /**
   * 获取服务请求列表
   */
  static async getServiceRequests(query?: ServiceRequestQuery): Promise<{
    requests: unknown[];
    total: number;
  }> {
    const page = query?.page ?? 1;
    const size = query?.pageSize ?? 10;
    const requestedStatus = query?.status ? String(query.status) : undefined;
    const isPendingApproval =
      requestedStatus === 'pending_approval' || requestedStatus === 'pending';
    const endpoint = isPendingApproval
      ? '/api/v1/service-requests/approvals/pending'
      : '/api/v1/service-requests/me';
    const status = isPendingApproval
      ? undefined
      : ServiceCatalogApi.toBackendRequestStatus(requestedStatus);

    const resp = await httpClient.get<any>(endpoint, {
      page,
      size,
      ...(status ? { status } : {}),
    });
    const rawRequests = resp.requests || resp.items || [];
    return {
      requests: rawRequests.map(ServiceCatalogApi.toServiceRequest),
      total: resp.total || 0,
    };
  }

  /**
   * 获取单个服务请求
   */
  static async getServiceRequest(id: number): Promise<any> {
    const resp = await httpClient.get<any>(`/api/v1/service-requests/${id}`);
    return ServiceCatalogApi.toServiceRequest(resp);
  }

  /**
   * 创建服务请求
   */
  static async createServiceRequest(request: CreateServiceRequestRequest): Promise<any> {
    // 前端 CreateServiceRequestRequest: { serviceId, formData, ... }
    // 后端 CreateServiceRequestRequest: { catalog_id, title, reason, form_data, ... , compliance_ack }
    const reason =
      (request.formData && (request.formData.reason || request.formData.notes)) ||
      request.additionalNotes ||
      '';

    const title = (request.formData && (request.formData.title || request.formData.name)) || '';

    // V0：最小字段集合。复杂字段（成本中心/分级/到期/公网白名单）可先从 formData 透传，后续再做强校验与表单化。
    const payload: unknown = {
      catalog_id: Number(request.serviceId),
      title: title ? String(title) : undefined,
      reason,
      form_data: request.formData || {},
      compliance_ack: Boolean(request.formData?.compliance_ack ?? true), // 以表单勾选为准，兜底为 true
      data_classification: String(request.formData?.data_classification || 'internal'),
      needs_public_ip: Boolean(request.formData?.needs_public_ip || false),
      source_ip_whitelist: Array.isArray(request.formData?.source_ip_whitelist)
        ? request.formData?.source_ip_whitelist
        : undefined,
      cost_center: request.formData?.cost_center
        ? String(request.formData?.cost_center)
        : undefined,
      expire_at: request.formData?.expire_at ? request.formData?.expire_at : undefined,
    };

    return httpClient.post('/api/v1/service-requests', payload);
  }

  /**
   * 取消服务请求
   */
  static async cancelServiceRequest(id: number, reason?: string): Promise<void> {
    await httpClient.put(`/api/v1/service-requests/${id}/status`, {
      status: 'cancelled',
      comment: reason,
    } as any);
  }

  /**
   * 审批服务请求
   */
  static async approveServiceRequest(id: number, comment?: string): Promise<void> {
    await httpClient.post(`/api/v1/service-requests/${id}/approval`, {
      action: 'approve',
      comment,
    });
  }

  /**
   * 拒绝服务请求
   */
  static async rejectServiceRequest(id: number, reason: string): Promise<void> {
    await httpClient.post(`/api/v1/service-requests/${id}/approval`, {
      action: 'reject',
      comment: reason,
    });
  }

  /**
   * 完成服务请求
   */
  static async completeServiceRequest(id: number, notes?: string): Promise<void> {
    await httpClient.put(`/api/v1/service-requests/${id}/status`, {
      status: 'completed',
      comment: notes,
    } as any);
  }

  /**
   * 获取服务请求详情（包含审批历史）
   */
  static async getServiceRequestDetail(id: number): Promise<any> {
    const response = await httpClient.get(`/api/v1/service-requests/${id}`);
    return response;
  }

  /**
   * 获取待审批数量
   */
  static async getPendingApprovalCount(): Promise<number> {
    const response = await httpClient.get<{ total: number }>(
      '/api/v1/service-requests/approvals/pending'
    );
    return response.total || 0;
  }

  // ==================== 收藏和评分 ====================

  /**
   * 添加收藏
   */
  static async addFavorite(serviceId: string): Promise<ServiceFavorite> {
     
    const _serviceId = serviceId;
    return ServiceCatalogApi.unsupportedFeature('服务收藏');
  }

  /**
   * 取消收藏
   */
  static async removeFavorite(serviceId: string): Promise<void> {
     
    const _serviceId = serviceId;
    ServiceCatalogApi.unsupportedFeature('服务收藏');
  }

  /**
   * 获取收藏列表
   */
  static async getFavorites(): Promise<ServiceFavorite[]> {
    // 后端未提供收藏列表接口。读取路径返回空列表，写入路径必须显式失败，避免用户误以为收藏已保存。
    return [];
  }

  /**
   * 评分服务
   */
  static async rateService(
    serviceId: string,
    rating: number,
    comment?: string
  ): Promise<ServiceRating> {
     
    const _args = { serviceId, rating, comment };
    return ServiceCatalogApi.unsupportedFeature('服务评分');
  }

  /**
   * 获取服务评分
   */
  static async getServiceRatings(
    serviceId: string,
    params?: {
      page?: number;
      pageSize?: number;
    }
  ): Promise<{
    ratings: ServiceRating[];
    total: number;
    avgRating: number;
  }> {
    return { ratings: [], total: 0, avgRating: 0 };
  }

  /**
   * 标记评分有用
   */
  static async markRatingHelpful(ratingId: string): Promise<void> {
     
    const _ratingId = ratingId;
    ServiceCatalogApi.unsupportedFeature('评分有用标记');
  }

  // ==================== 门户配置 ====================

  /**
   * 获取门户配置
   */
  static async getPortalConfig(): Promise<PortalConfig> {
    // 本地只提供只读默认配置；未接后端的能力默认关闭，避免页面展示不可持久化操作。
    return {
      id: 'default',
      name: '默认门户',
      branding: {
        primaryColor: '#1890ff',
      },
      homepage: {},
      features: {
        enableSearch: true,
        enableRating: false,
        enableFavorites: false,
        enableNotifications: false,
        showServicePrice: false,
        showServiceOwner: true,
      },
      updatedAt: new Date(),
    };
  }

  /**
   * 更新门户配置
   */
  static async updatePortalConfig(config: Partial<PortalConfig>): Promise<PortalConfig> {
     
    const _config = config;
    return ServiceCatalogApi.unsupportedFeature('门户配置更新');
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取服务目录统计
   */
  static async getCatalogStats(): Promise<ServiceCatalogStats> {
    // 调用后端实际统计接口
    const resp = await httpClient.get<{
      totalServices: number;
      publishedServices: number;
      categories: Record<string, number>;
    }>('/api/v1/service-catalogs/stats');

    return {
      totalServices: resp.totalServices || 0,
      publishedServices: resp.publishedServices || 0,
      totalRequests: 0,
      avgRating: 0,
      topCategories: Object.entries(resp.categories || {}).map(([name, count]) => ({
        name,
        count: count as number,
      })),
      requestsByStatus: {},
      servicesByStatus: {},
    } as any;
  }

  /**
   * 获取服务分析
   */
  static async getServiceAnalytics(
    serviceId: string,
    params?: {
      startDate?: string;
      endDate?: string;
    }
  ): Promise<ServiceAnalytics> {
    // 后端暂未实现服务分析，返回空数据
     
    const _unused = serviceId;
    return {
      serviceId,
      period: {
        start: params?.startDate
          ? new Date(params.startDate)
          : new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
        end: params?.endDate ? new Date(params.endDate) : new Date(),
      },
      metrics: {
        totalRequests: 0,
        completedRequests: 0,
        avgCompletionTime: 0,
        completionRate: 0,
        avgRating: 0,
        totalViews: 0,
      },
      requestTrend: [],
      userSatisfaction: [],
      peakHours: [],
    };
  }

  /**
   * 记录服务浏览
   */
  static async recordServiceView(serviceId: string): Promise<void> {
    // V0：不做
    return;
  }

  /**
   * 导出服务目录
   */
  static async exportCatalog(format: 'excel' | 'pdf'): Promise<Blob> {
    const response = await ServiceCatalogApi.getServices({ page: 1, pageSize: 1000 });
    const header = [
      'ID',
      '服务名称',
      '分类',
      '状态',
      '描述',
      '交付时间',
      '创建时间',
      '更新时间',
    ];
    const rows = response.services.map(service => [
      service.id,
      service.name,
      service.category,
      service.status,
      service.shortDescription,
      service.availability?.responseTime || '',
      service.createdAt,
      service.updatedAt,
    ]);
    const csv = [header, ...rows]
      .map(row => row.map(ServiceCatalogApi.escapeCSV).join(','))
      .join('\n');
    const type = format === 'pdf' ? 'text/csv;charset=utf-8' : 'text/csv;charset=utf-8';
    return new Blob([`\uFEFF${csv}`], { type });
  }
}

export default ServiceCatalogApi;
export const ServiceCatalogAPI = ServiceCatalogApi;

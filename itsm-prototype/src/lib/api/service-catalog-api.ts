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

  private static toServiceItem(raw: any): ServiceItem {
    // 后端 dto.ServiceCatalogResponse: {id,name,category,description,delivery_time,status,created_at,updated_at}
    return {
      id: String(raw?.id),
      name: String(raw?.name || ''),
      // 这里保留后端 category 的原始字符串（前端页面目前以中文分类做统计/图标）
      category: (raw?.category as any) || ('it_service' as any),
      status: ServiceCatalogApi.toFrontendStatus(raw?.status),
      shortDescription: String(raw?.description || ''),
      fullDescription: String(raw?.description || ''),
      tags: [],
      requiresApproval: true,
      createdBy: 0,
      createdByName: '',
      createdAt: raw?.created_at ? new Date(raw.created_at) : new Date(),
      updatedAt: raw?.updated_at ? new Date(raw.updated_at) : new Date(),
      availability: {
        // 后端 delivery_time 为 string（天/小时口径未统一）；V0先用于展示，不做严格含义
        responseTime: raw?.delivery_time ? Number(raw.delivery_time) : undefined,
      },
    };
  }

  // ==================== 服务项管理 ====================

  /**
   * 获取服务列表
   */
  static async getServices(
    query?: ServiceQuery
  ): Promise<{
    services: ServiceItem[];
    total: number;
  }> {
    const page = query?.page ?? 1;
    const size = query?.pageSize ?? 10;
    const category = query?.category ? String(query.category) : undefined;
    const status = ServiceCatalogApi.toBackendStatus(query?.status);

    const resp = await httpClient.get<{
      catalogs: any[];
      total: number;
      page: number;
      size: number;
    }>('/api/v1/service-catalogs', {
      page,
      size,
      ...(category ? { category } : {}),
      ...(status ? { status } : {}),
    });

    let services = (resp.catalogs || []).map(ServiceCatalogApi.toServiceItem);
    // 后端当前不支持 search；先在前端做兜底过滤
    if (query?.search) {
      const q = query.search.toLowerCase();
      services = services.filter(s => (s.name || '').toLowerCase().includes(q) || (s.shortDescription || '').toLowerCase().includes(q));
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
  static async createService(
    request: CreateServiceItemRequest
  ): Promise<ServiceItem> {
    const payload = {
      name: request.name,
      category: String(request.category),
      description: request.shortDescription || request.fullDescription || '',
      delivery_time: String(request.availability?.responseTime ?? request.availability?.resolutionTime ?? 1),
      status: ServiceCatalogApi.toBackendStatus((request as any).status) || 'enabled',
    };
    const resp = await httpClient.post<any>('/api/v1/service-catalogs', payload);
    return ServiceCatalogApi.toServiceItem(resp);
  }

  /**
   * 更新服务
   */
  static async updateService(
    id: string,
    request: UpdateServiceItemRequest
  ): Promise<ServiceItem> {
    const payload: Record<string, unknown> = {};
    if (request.name !== undefined) payload.name = request.name;
    if (request.category !== undefined) payload.category = String(request.category);
    if (request.shortDescription !== undefined || request.fullDescription !== undefined) {
      payload.description = request.shortDescription || request.fullDescription || '';
    }
    if (request.availability?.responseTime !== undefined) {
      payload.delivery_time = String(request.availability.responseTime);
    }
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
  static async cloneService(
    id: string,
    name: string
  ): Promise<ServiceItem> {
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
  static async getServiceRequests(
    query?: ServiceRequestQuery
  ): Promise<{
    requests: any[];
    total: number;
  }> {
    // V0：后端仅提供“我的请求”列表；先对齐可用接口
    const page = query?.page ?? 1;
    const size = query?.pageSize ?? 10;
    const status = query?.status ? String(query.status) : undefined;
    const resp = await httpClient.get<any>('/api/v1/service-requests/me', {
      page,
      size,
      ...(status ? { status } : {}),
    });
    return { requests: resp.requests || [], total: resp.total || 0 };
  }

  /**
   * 获取单个服务请求
   */
  static async getServiceRequest(id: number): Promise<any> {
    return httpClient.get(`/api/v1/service-requests/${id}`);
  }

  /**
   * 创建服务请求
   */
  static async createServiceRequest(
    request: CreateServiceRequestRequest
  ): Promise<any> {
    // 前端 CreateServiceRequestRequest: { serviceId, formData, ... }
    // 后端 CreateServiceRequestRequest: { catalog_id, reason }
    const reason =
      (request.formData && (request.formData.reason || request.formData.notes)) ||
      request.additionalNotes ||
      '';
    return httpClient.post('/api/v1/service-requests', {
      catalog_id: Number(request.serviceId),
      reason,
    });
  }

  /**
   * 取消服务请求
   */
  static async cancelServiceRequest(
    id: number,
    reason?: string
  ): Promise<void> {
    // 后端未提供 cancelled 状态（V0）；避免静默错误，直接提示上层处理
    throw new Error(`后端暂不支持取消服务请求（id=${id}），请先实现 cancelled 状态或取消接口。原因: ${reason || ''}`);
  }

  /**
   * 审批服务请求
   */
  static async approveServiceRequest(
    id: number,
    comment?: string
  ): Promise<void> {
    // V0：后端仅有状态更新接口；审批语义后续在V1补齐
    await httpClient.put(`/api/v1/service-requests/${id}/status`, {
      status: 'in_progress',
      comment,
    } as any);
  }

  /**
   * 拒绝服务请求
   */
  static async rejectServiceRequest(
    id: number,
    reason: string
  ): Promise<void> {
    await httpClient.put(`/api/v1/service-requests/${id}/status`, {
      status: 'rejected',
      comment: reason,
    } as any);
  }

  /**
   * 完成服务请求
   */
  static async completeServiceRequest(
    id: number,
    notes?: string
  ): Promise<void> {
    await httpClient.put(`/api/v1/service-requests/${id}/status`, {
      status: 'completed',
      comment: notes,
    } as any);
  }

  // ==================== 收藏和评分 ====================

  /**
   * 添加收藏
   */
  static async addFavorite(serviceId: string): Promise<ServiceFavorite> {
    throw new Error('后端暂未实现服务收藏功能（V0）。');
  }

  /**
   * 取消收藏
   */
  static async removeFavorite(serviceId: string): Promise<void> {
    throw new Error('后端暂未实现服务收藏功能（V0）。');
  }

  /**
   * 获取收藏列表
   */
  static async getFavorites(): Promise<ServiceFavorite[]> {
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
    throw new Error('后端暂未实现服务评分功能（V0）。');
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
    throw new Error('后端暂未实现评分有用功能（V0）。');
  }

  // ==================== 门户配置 ====================

  /**
   * 获取门户配置
   */
  static async getPortalConfig(): Promise<PortalConfig> {
    throw new Error('后端暂未实现自助门户配置接口（V0）。');
  }

  /**
   * 更新门户配置
   */
  static async updatePortalConfig(
    config: Partial<PortalConfig>
  ): Promise<PortalConfig> {
    throw new Error('后端暂未实现自助门户配置接口（V0）。');
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取服务目录统计
   */
  static async getCatalogStats(): Promise<ServiceCatalogStats> {
    // V0：后端未提供 stats；前端可用 getServices + 本地统计
    const { services, total } = await ServiceCatalogApi.getServices({ page: 1, pageSize: 1000 } as any);
    return {
      totalServices: total,
      publishedServices: services.filter(s => String((s as any).status) === 'published').length,
      totalRequests: 0,
      avgRating: 0,
      topCategories: [],
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
    throw new Error('后端暂未实现服务分析接口（V0）。');
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
  static async exportCatalog(
    format: 'excel' | 'pdf'
  ): Promise<Blob> {
    throw new Error(`后端暂未实现服务目录导出（V0），format=${format}`);
  }
}

export default ServiceCatalogApi;
export const ServiceCatalogAPI = ServiceCatalogApi;

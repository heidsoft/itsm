/**
 * 服务目录 API 服务
 */

import { httpClient } from './http-client';
import type {
  ServiceItem,
  ServiceRequest,
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
    return httpClient.get('/api/v1/service-catalog/services', query);
  }

  /**
   * 获取单个服务
   */
  static async getService(id: string): Promise<ServiceItem> {
    return httpClient.get(`/api/v1/service-catalog/services/${id}`);
  }

  /**
   * 创建服务
   */
  static async createService(
    request: CreateServiceItemRequest
  ): Promise<ServiceItem> {
    return httpClient.post('/api/v1/service-catalog/services', request);
  }

  /**
   * 更新服务
   */
  static async updateService(
    id: string,
    request: UpdateServiceItemRequest
  ): Promise<ServiceItem> {
    return httpClient.put(
      `/api/v1/service-catalog/services/${id}`,
      request
    );
  }

  /**
   * 删除服务
   */
  static async deleteService(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/service-catalog/services/${id}`);
  }

  /**
   * 发布服务
   */
  static async publishService(id: string): Promise<ServiceItem> {
    return httpClient.post(`/api/v1/service-catalog/services/${id}/publish`);
  }

  /**
   * 停用服务
   */
  static async retireService(id: string): Promise<ServiceItem> {
    return httpClient.post(`/api/v1/service-catalog/services/${id}/retire`);
  }

  /**
   * 复制服务
   */
  static async cloneService(
    id: string,
    name: string
  ): Promise<ServiceItem> {
    return httpClient.post(`/api/v1/service-catalog/services/${id}/clone`, {
      name,
    });
  }

  // ==================== 服务请求管理 ====================

  /**
   * 获取服务请求列表
   */
  static async getServiceRequests(
    query?: ServiceRequestQuery
  ): Promise<{
    requests: ServiceRequest[];
    total: number;
  }> {
    return httpClient.get('/api/v1/service-catalog/requests', query);
  }

  /**
   * 获取单个服务请求
   */
  static async getServiceRequest(id: number): Promise<ServiceRequest> {
    return httpClient.get(`/api/v1/service-catalog/requests/${id}`);
  }

  /**
   * 创建服务请求
   */
  static async createServiceRequest(
    request: CreateServiceRequestRequest
  ): Promise<ServiceRequest> {
    return httpClient.post('/api/v1/service-catalog/requests', request);
  }

  /**
   * 取消服务请求
   */
  static async cancelServiceRequest(
    id: number,
    reason?: string
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/service-catalog/requests/${id}/cancel`,
      { reason }
    );
  }

  /**
   * 审批服务请求
   */
  static async approveServiceRequest(
    id: number,
    comment?: string
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/service-catalog/requests/${id}/approve`,
      { comment }
    );
  }

  /**
   * 拒绝服务请求
   */
  static async rejectServiceRequest(
    id: number,
    reason: string
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/service-catalog/requests/${id}/reject`,
      { reason }
    );
  }

  /**
   * 完成服务请求
   */
  static async completeServiceRequest(
    id: number,
    notes?: string
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/service-catalog/requests/${id}/complete`,
      { notes }
    );
  }

  // ==================== 收藏和评分 ====================

  /**
   * 添加收藏
   */
  static async addFavorite(serviceId: string): Promise<ServiceFavorite> {
    return httpClient.post('/api/v1/service-catalog/favorites', {
      serviceId,
    });
  }

  /**
   * 取消收藏
   */
  static async removeFavorite(serviceId: string): Promise<void> {
    return httpClient.delete(
      `/api/v1/service-catalog/favorites/${serviceId}`
    );
  }

  /**
   * 获取收藏列表
   */
  static async getFavorites(): Promise<ServiceFavorite[]> {
    return httpClient.get('/api/v1/service-catalog/favorites');
  }

  /**
   * 评分服务
   */
  static async rateService(
    serviceId: string,
    rating: number,
    comment?: string
  ): Promise<ServiceRating> {
    return httpClient.post('/api/v1/service-catalog/ratings', {
      serviceId,
      rating,
      comment,
    });
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
    return httpClient.get(
      `/api/v1/service-catalog/services/${serviceId}/ratings`,
      params
    );
  }

  /**
   * 标记评分有用
   */
  static async markRatingHelpful(ratingId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/service-catalog/ratings/${ratingId}/helpful`
    );
  }

  // ==================== 门户配置 ====================

  /**
   * 获取门户配置
   */
  static async getPortalConfig(): Promise<PortalConfig> {
    return httpClient.get('/api/v1/service-catalog/portal/config');
  }

  /**
   * 更新门户配置
   */
  static async updatePortalConfig(
    config: Partial<PortalConfig>
  ): Promise<PortalConfig> {
    return httpClient.put('/api/v1/service-catalog/portal/config', config);
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取服务目录统计
   */
  static async getCatalogStats(): Promise<ServiceCatalogStats> {
    return httpClient.get('/api/v1/service-catalog/stats');
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
    return httpClient.get(
      `/api/v1/service-catalog/services/${serviceId}/analytics`,
      params
    );
  }

  /**
   * 记录服务浏览
   */
  static async recordServiceView(serviceId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/service-catalog/services/${serviceId}/view`
    );
  }

  /**
   * 导出服务目录
   */
  static async exportCatalog(
    format: 'excel' | 'pdf'
  ): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/service-catalog/export',
      data: { format },
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export default ServiceCatalogApi;
export const ServiceCatalogAPI = ServiceCatalogApi;

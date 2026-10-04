/**
 * 服务目录 API 服务
 */

import { httpClient } from './http-client';
import type {
  ServiceItem,
  ServiceStatus,
  ServiceCategory,
  PortalConfig,
  ServiceFavorite,
  ServiceRating,
  ServiceCatalogStats,
  CreateServiceItemRequest,
  UpdateServiceItemRequest,
  CreateServiceRequestRequest,
  ServiceQuery,
  ServiceRequestQuery,
} from '@/types/service-catalog';

// 必须是字符串字面量：派生模板常量无法被 api-contract 扫描器静态解析。
const SERVICE_CATALOGS_PATH = '/api/v1/service-catalogs';
// 后端 common.MaxPageSize=100。超过它的页长不报错，而是被单点回落成默认 20，
// 所以「取全」只能翻页表达。
const MAX_PAGE_SIZE = 100;
// 导出的硬上限。超过它必须显式报错并让调用方收窄过滤条件，禁止静默导出半截 CSV。
const MAX_EXPORT_RECORDS = 5000;

/** GET /api/v1/service-catalogs 的信封；集合键只有 items，页长只有 pageSize。 */
interface ServiceCatalogListEnvelope {
  items: unknown[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export class ServiceCatalogApi {
  // ==================== 内部适配（对齐后端 /api/v1/service-catalogs & /api/v1/service-requests） ====================

  private static unsupportedFeature(feature: string): never {
    throw new Error(`${feature}暂未开放`);
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
    // 后端 /api/v1/service-catalogs 契约枚举：enabled/disabled。
    // 旧 schema 默认曾为 'active'，迁移 017_normalize_service_catalog_status 后
    // 只会出现 enabled/disabled，但保留 'active' 别名兼容在途事务中的快照。
    const s = String(status || '');
    if (s === 'enabled' || s === 'active') return 'published' as ServiceStatus;
    return 'retired' as ServiceStatus;
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
       
      category: (raw?.category as ServiceCategory) || ('it_service' as ServiceCategory),
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
    // Bug 修复：清理重复的 `raw?.field ?? raw?.field` fallback 表达式。
    // 这是复制粘贴残留，不会引发运行时错误，但会误导阅读并掩盖潜在缺陷。
    const catalogId = raw?.catalogId ?? raw?.serviceId;
    const requesterId = raw?.requesterId ?? raw?.requestedBy;
    const createdAt = raw?.createdAt;
    const updatedAt = raw?.updatedAt;
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
      requesterId,
      ciId: raw?.ciId,
      formData: raw?.formData ?? {},
      costCenter: raw?.costCenter,
      dataClassification: raw?.dataClassification,
      needsPublicIp: raw?.needsPublicIp ?? raw?.needsPublicIP,
      sourceIpWhitelist: raw?.sourceIpWhitelist ?? raw?.sourceIPWhitelist,
      complianceAck: raw?.complianceAck,
      currentLevel: raw?.currentLevel,
      totalLevels: raw?.totalLevels,
      expireAt: raw?.expireAt,
      createdAt,
      updatedAt,
    };
  }

  // ==================== 服务项管理 ====================

  /**
   * 获取服务列表（单页）。
   *
   * 集合键固定为 items，页长固定为 pageSize；后端 common.GetPaginationFromQuery 是
   * 唯一的夹紧所有者（缺省 1/20，只采纳 (0,100]）。需要读全量的调用方走
   * ServiceCatalogApi.getAllServices，不要靠放大 pageSize 表达「全部」。
   */
  static async getServices(query?: ServiceQuery): Promise<{
    services: ServiceItem[];
    total: number;
  }> {
    const page = query?.page ?? 1;
    const category = query?.category ? String(query.category) : undefined;
    const status = ServiceCatalogApi.toBackendStatus(query?.status);

    const resp = await httpClient.get<ServiceCatalogListEnvelope>(SERVICE_CATALOGS_PATH, {
      page,
      pageSize: query?.pageSize ?? 10,
      ...(category ? { category } : {}),
      ...(status ? { status } : {}),
    });

    const services = resp.items.map(ServiceCatalogApi.toServiceItem);
    // 后端列表端点没有关键词参数，这里只过滤「当前页」，因此 total 会大于返回条数。
    // 补齐服务端 search 属于账本 E4-24，不得由调用方误当成全集。
    if (query?.search) {
      const q = query.search.toLowerCase();
      return {
        services: services.filter(
          s =>
            (s.name || '').toLowerCase().includes(q) ||
            (s.shortDescription || '').toLowerCase().includes(q),
        ),
        total: resp.total,
      };
    }

    return { services, total: resp.total };
  }

  /**
   * 翻页读满 maxRecords 条或读到末页。
   *
   * 后端 common.MaxPageSize=100：请求更大的页长不报错，而是被单点回落成默认 20，
   * 所以「取全」只能翻页。返回 complete=false 表示数据源被 maxRecords 截断，
   * 调用方（导出/报表）必须显式处理，不能把截断结果当成全集。
   */
  static async getAllServices(
    query?: Omit<ServiceQuery, 'page' | 'pageSize'>,
    maxRecords = 200,
  ): Promise<{ services: ServiceItem[]; total: number; complete: boolean }> {
    const category = query?.category ? String(query.category) : undefined;
    const status = ServiceCatalogApi.toBackendStatus(query?.status);
    const services: ServiceItem[] = [];
    let total = 0;

    for (let page = 1; ; page += 1) {
      const resp = await httpClient.get<ServiceCatalogListEnvelope>(SERVICE_CATALOGS_PATH, {
        page,
        pageSize: MAX_PAGE_SIZE,
        ...(category ? { category } : {}),
        ...(status ? { status } : {}),
      });
      total = resp.total;
      services.push(...resp.items.map(ServiceCatalogApi.toServiceItem));

      // 终止条件用响应里的生效页长，不用请求值：后端回落时两者可能不同。
      if (
        resp.items.length < resp.pageSize ||
        services.length >= Math.min(maxRecords, resp.total)
      ) {
        break;
      }
    }

    return { services, total, complete: services.length >= total };
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
      ciTypeId: request.ciTypeId,
      cloudServiceId: request.cloudServiceId,
      deliveryTime: String(
        request.availability?.responseTime ?? request.availability?.resolutionTime ?? 1
      ),
      status: ServiceCatalogApi.toBackendStatus(request.status) || 'enabled',
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
      payload.deliveryTime = String(request.availability.responseTime);
    }
    if (request.ciTypeId !== undefined) payload.ciTypeId = request.ciTypeId;
    if (request.cloudServiceId !== undefined) payload.cloudServiceId = request.cloudServiceId;
    const st = ServiceCatalogApi.toBackendStatus(request.status);
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
    const { id: _omit, ...rest } = src;
    return ServiceCatalogApi.createService({
      ...rest,
      name,
    });
  }

  // ==================== 服务请求管理 ====================

  /**
   * 获取服务请求列表
   *
   * 服务请求列表的信封只有一个形状：`{items,total,page,pageSize,totalPages}`。
   * 这里不再读 `requests` 别名，也不再发送 `size`（后端分页参数的唯一名字是 `pageSize`）。
   * 同一资源在本文件与 `service-request-api.ts` 各有一套客户端实体，已登记为待收敛债务。
   */
  static async getServiceRequests(query?: ServiceRequestQuery): Promise<{
    items: unknown[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  }> {
    const page = query?.page ?? 1;
    const pageSize = query?.pageSize ?? 10;
    const requestedStatus = query?.status ? String(query.status) : undefined;
    const isPendingApproval =
      requestedStatus === 'pending_approval' || requestedStatus === 'pending';
    const endpoint = isPendingApproval
      ? '/api/v1/service-requests/approvals/pending'
      : '/api/v1/service-requests/me';
    const status = isPendingApproval
      ? undefined
      : ServiceCatalogApi.toBackendRequestStatus(requestedStatus);

    const resp = await httpClient.get<{
      items: unknown[];
      total: number;
      page: number;
      pageSize: number;
      totalPages: number;
    }>(endpoint, {
      page,
      pageSize,
      ...(status ? { status } : {}),
    });
    return {
      items: resp.items.map(ServiceCatalogApi.toServiceRequest),
      total: resp.total,
      page: resp.page,
      pageSize: resp.pageSize,
      totalPages: resp.totalPages,
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
      catalogId: Number(request.serviceId),
      title: title ? String(title) : undefined,
      reason,
      formData: request.formData || {},
      complianceAck: Boolean(request.formData?.complianceAck ?? true), // 以表单勾选为准，兜底为 true
      dataClassification: String(request.formData?.dataClassification || 'internal'),
      needsPublicIp: Boolean(request.formData?.needsPublicIp || false),
      sourceIpWhitelist: Array.isArray(request.formData?.sourceIpWhitelist)
        ? request.formData?.sourceIpWhitelist
        : undefined,
      costCenter: request.formData?.costCenter
        ? String(request.formData?.costCenter)
        : undefined,
      expireAt: request.formData?.expireAt ? request.formData?.expireAt : undefined,
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
    });
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
    });
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
    // 后端没有收藏接口。读取路径同样显式失败，空列表无法区分「没收藏」和「能力未接」。
    return ServiceCatalogApi.unsupportedFeature('服务收藏');
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
    const _args = { serviceId, params };
    return ServiceCatalogApi.unsupportedFeature('服务评分查询');
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
    // 契约与 handlers/service_catalog.ServiceStats 逐字段一致：后端只给服务数量与分类计数。
    // 请求量/趋势/热门服务没有服务端实现，不得在这里补零，否则页面会把「没数据」显示成「0 个请求」。
    return httpClient.get<ServiceCatalogStats>('/api/v1/service-catalogs/stats');
  }

  /**
   * 记录服务浏览
   */
  static async recordServiceView(serviceId: string): Promise<void> {
    const _serviceId = serviceId;
    // 写入路径必须显式失败，静默丢弃会让调用方以为浏览量已记录。
    return ServiceCatalogApi.unsupportedFeature('服务浏览记录');
  }

  /**
   * 导出服务目录
   */
  static async exportCatalog(format: 'excel' | 'pdf'): Promise<Blob> {
    // 原先发 size=1000：后端 binding 允许到 1000，但 handler 又静默夹成 100，
    // 所以导出从来就没有拿到 1000 条；信封收敛后同样的请求会回落成 20 条。
    // 两种都说明「放大页长取全」不成立 —— 取全必须翻页，且截断要显式失败。
    const { services, total, complete } = await ServiceCatalogApi.getAllServices(
      undefined,
      MAX_EXPORT_RECORDS,
    );
    if (!complete) {
      throw new Error(
        `服务目录共 ${total} 条，超过单次导出上限 ${MAX_EXPORT_RECORDS} 条，请按分类或状态收窄条件后重试。`,
      );
    }
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
    const rows = services.map(service => [
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

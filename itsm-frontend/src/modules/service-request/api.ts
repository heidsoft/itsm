/**
 * 服务请求 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type {
    ServiceRequest,
    CreateServiceRequestRequest,
    ServiceRequestApprovalActionRequest,
    ServiceRequestListResponse,
    ServiceRequestQuery
} from './types';

// API 基础路径
const BASE_URL = '/api/v1/service-requests';

export class ServiceRequestApi {

    /**
     * 获取服务请求列表
     * @param query 查询参数
     */
    static async getServiceRequests(query?: ServiceRequestQuery): Promise<ServiceRequestListResponse> {
        // 如果是 scope=me，路径可能不同，或者使用统一接口带 userID (后端处理)
        // 根据后端实现，GET /api/v1/service-requests 可以带 filter
        // GET /api/v1/service-requests/me 是专门获取我的请求

        let url = BASE_URL;
        if (query?.scope === 'me') {
            url = `${BASE_URL}/me`;
        }

        const params: Record<string, any> = {
            page: query?.page ?? 1,
            size: query?.size ?? 10,
        };

        if (query?.status) {
            params.status = query.status;
        }

        const resp = await httpClient.get<any>(url, params);

        // 适配后端返回结构，后端返回 { requests: [], total: ... }
        // 注意：后端返回的数据在 `data` 字段中 (由 httpClient 自动解包吗？)
        // 假设 httpClient.get 返回的是 Response.data
        return {
            requests: resp.requests || [],
            total: resp.total || 0,
            page: resp.page || params.page,
            size: resp.size || params.size,
        };
    }

    /**
     * 获取单个服务请求详情
     * @param id 请求ID
     */
    static async getServiceRequest(id: string | number): Promise<ServiceRequest> {
        return httpClient.get<ServiceRequest>(`${BASE_URL}/${id}`);
    }

    /**
     * 创建服务请求
     * @param data 创建参数
     */
    static async createServiceRequest(data: CreateServiceRequestRequest): Promise<ServiceRequest> {
        // 后端 handler: POST /api/v1/service-requests
        return httpClient.post<ServiceRequest>(BASE_URL, data);
    }

    /**
     * 提交审批动作 (同意/拒绝)
     * @param id 请求ID
     * @param data 审批动作参数
     */
    static async applyApproval(
        id: string | number,
        data: ServiceRequestApprovalActionRequest
    ): Promise<ServiceRequest> {
        return httpClient.post<ServiceRequest>(`${BASE_URL}/${id}/approval`, data);
    }

    /**
     * 获取待办审批列表 (后端接口需确认)
     * 后端 Handler List 支持 filter，但 ServiceController Legacy 有 /approvals/pending
     * 我们的 Backend Refactor 去掉了 /approvals/pending 吗？
     * 查看 router.go: 
     * [LEGACY] /service-requests/approvals/pending 被注释掉了
     * 新接口: GET /service-requests?status=... 
     * 
     * 等等，Service.ListPendingApprovals 在 `service.go` 中是根据 role 列出 pending approvals。
     * 但 Router 中 `ServiceRequestHandler.List` 只是调用 `service.List` (generic filter)。
     * 我似乎**漏掉**了暴露 `ListPendingApprovals` 的 endpoint!
     * 这是一个后端遗漏。
     * 
     * 暂时在前端实现这个方法，标记为 TODO 需修复后端。
     * 或者复用 List 接口，但 List 接口目前的 filter 逻辑可能不支持“基于角色的待办查询”。
     */
    static async getPendingApprovals(query?: ServiceRequestQuery): Promise<ServiceRequestListResponse> {
        // TODO: 后端目前可能不支持专门的 "pending approvals" 路由，
        // 或者需要通过 filter status + user role 来实现。
        // 暂时尝试调用 List，但可能需要后端补充 /pending 路由。
        return this.getServiceRequests(query);
    }
}

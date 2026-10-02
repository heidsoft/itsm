import { httpClient } from './http-client';
import type {
  GetTenantsParams} from './api-config';
import type {
  MSPAllocation,
  CreateAllocationRequest,
  MSPAllocationListResponse,
  MSPCustomersResponse,
  MSPCustomerTicketsResponse,
  MSPCustomerReportListResponse,
  MSPPerformanceReportListResponse,
  MSPAllocationHistory,
  MSPContext,
} from '@/types/msp';

export class MSPAPI {
  // ==================== MSP 分配管理 ====================

  /**
   * 获取 MSP 分配列表（当前 MSP 用户）
   */
  static async getAllocations(params?: GetTenantsParams): Promise<MSPAllocationListResponse> {
    return httpClient.get<MSPAllocationListResponse>('/api/v1/msp/allocations', params);
  }

  /**
   * 创建新的 MSP 分配（仅 MSP Manager 可用）
   */
  static async createAllocation(data: CreateAllocationRequest): Promise<MSPAllocation> {
    return httpClient.post<MSPAllocation>('/api/v1/msp/allocations', data);
  }

  /**
   * 解除 MSP 分配
   */
  static async deallocate(mspUserId: number, customerTenantId: number, reason?: string): Promise<void> {
    return httpClient.post<void>('/api/v1/msp/allocations/deallocate', {
      mspUserId: mspUserId,
      customerTenantId: customerTenantId,
      reason,
    });
  }

  // ==================== MSP 客户管理 ====================

  /**
   * 获取当前 MSP 员工有权访问的所有客户列表
   */
  static async getCustomers(params?: GetTenantsParams): Promise<MSPCustomersResponse> {
    return httpClient.get<MSPCustomersResponse>('/api/v1/msp/customers', params);
  }

  /**
   * 获取指定客户的工单（MSP 视角）
   */
  static async getCustomerTickets(
    customerTenantId: number,
    params?: { status?: string; page?: number; pageSize?: number }
  ): Promise<MSPCustomerTicketsResponse> {
    return httpClient.get<MSPCustomerTicketsResponse>(
      `/api/v1/msp/customers/${customerTenantId}/tickets`,
      params
    );
  }

  /**
   * 为工单分配 MSP 技术员
   */
  static async assignTechnician(
    ticketId: number,
    customerTenantId: number,
    assignerUserId?: number
  ): Promise<{ id: number; status: string }> {
    return httpClient.post<{ id: number; status: string }>(
      `/api/v1/msp/tickets/${ticketId}/assign`,
      {
        customerTenantId: customerTenantId,
        assignerUserId: assignerUserId,
      }
    );
  }

  // ==================== MSP 报表 ====================

  /**
   * 获取客户服务报表（按调用者租户的区间聚合，不支持分页）。
   * 契约只接受 startDate/endDate：按客户租户或按员工分组从未实现。
   */
  static async getCustomerReports(
    params: { startDate: string; endDate: string }
  ): Promise<MSPCustomerReportListResponse> {
    return httpClient.get<MSPCustomerReportListResponse>('/api/v1/msp/reports/customers', params);
  }

  /**
   * 获取绩效报表（按调用者租户的区间聚合，不支持分页）。
   * 发送 mspUserId 会被后端拒绝（400），不是被忽略。
   */
  static async getMSPPerformanceReports(
    params: { startDate: string; endDate: string }
  ): Promise<MSPPerformanceReportListResponse> {
    return httpClient.get<MSPPerformanceReportListResponse>('/api/v1/msp/reports/performance', params);
  }

  // ==================== 辅助方法 ====================

  /**
   * 检查当前用户是否是 MSP 员工
   */
  static async isMSPUser(): Promise<{ isMSP: boolean; isAdmin: boolean }> {
    try {
      const res = await httpClient.get<{ isMsp: boolean; isAdmin?: boolean }>('/api/v1/msp/status');
      return {
        isMSP: res.isMsp || false,
        isAdmin: res.isAdmin || false,
      };
    } catch {
      return { isMSP: false, isAdmin: false };
    }
  }

  /**
   * 获取当前用户的 MSP 上下文
   */
  static async getMSPContext(): Promise<MSPContext> {
    return httpClient.get<MSPContext>('/api/v1/msp/context');
  }

  // ==================== 审计与历史 ====================

  /**
   * 获取分配历史记录
   */
  static async getAllocationHistory(
    params: { mspUserId?: number; customerTenantId?: number; startDate?: string; endDate?: string }
  ): Promise<MSPAllocationHistory[]> {
    return httpClient.get<MSPAllocationHistory[]>('/api/v1/msp/allocations/history', params);
  }
}

import { httpClient } from './http-client';
import type {
  ApiResponse,
  GetTenantsParams} from './api-config';
import {
  PaginationResponse
} from './api-config';
import type {
  MSPAllocation,
  CreateAllocationRequest,
  MSPAllocationListResponse,
  MSPCustomersResponse,
  MSPCustomerTicketsResponse,
  MSPCustomerReport,
  MSPAllocationHistory,
  MSPContext,
} from '@/types/msp';

export class MSPAPI {
  // ==================== MSP 分配管理 ====================

  /**
   * 获取 MSP 分配列表（当前 MSP 用户）
   */
  static async getAllocations(params?: GetTenantsParams): Promise<ApiResponse<MSPAllocationListResponse>> {
    return httpClient.get<ApiResponse<MSPAllocationListResponse>>('/api/v1/msp/allocations', params);
  }

  /**
   * 创建新的 MSP 分配（仅 MSP Manager 可用）
   */
  static async createAllocation(data: CreateAllocationRequest): Promise<ApiResponse<MSPAllocation>> {
    return httpClient.post<ApiResponse<MSPAllocation>>('/api/v1/msp/allocations', data);
  }

  /**
   * 解除 MSP 分配
   */
  static async deallocate(mspUserId: number, customerTenantId: number, reason?: string): Promise<ApiResponse<void>> {
    return httpClient.post<ApiResponse<void>>('/api/v1/msp/allocations/deallocate', {
      msp_user_id: mspUserId,
      customer_tenant_id: customerTenantId,
      reason,
    });
  }

  // ==================== MSP 客户管理 ====================

  /**
   * 获取当前 MSP 员工有权访问的所有客户列表
   */
  static async getCustomers(params?: GetTenantsParams): Promise<ApiResponse<MSPCustomersResponse>> {
    return httpClient.get<ApiResponse<MSPCustomersResponse>>('/api/v1/msp/customers', params);
  }

  /**
   * 获取指定客户的工单（MSP 视角）
   */
  static async getCustomerTickets(
    customerTenantId: number,
    params?: { status?: string; page?: number; pageSize?: number }
  ): Promise<ApiResponse<MSPCustomerTicketsResponse>> {
    return httpClient.get<ApiResponse<MSPCustomerTicketsResponse>>(
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
  ): Promise<ApiResponse<{ id: number; status: string }>> {
    return httpClient.post<ApiResponse<{ id: number; status: string }>>(
      `/api/v1/msp/tickets/${ticketId}/assign`,
      {
        customer_tenant_id: customerTenantId,
        assigner_user_id: assignerUserId,
      }
    );
  }

  // ==================== MSP 报表 ====================

  /**
   * 获取客户服务报表
   */
  static async getCustomerReports(
    params: { start_date: string; end_date: string; customer_tenant_id?: number }
  ): Promise<ApiResponse<MSPCustomerReport[]>> {
    return httpClient.get<ApiResponse<MSPCustomerReport[]>>('/api/v1/msp/reports/customers', params);
  }

  /**
   * 获取 MSP 员工绩效报表
   */
  static async getMSPPerformanceReports(
    params: { start_date: string; end_date: string; msp_user_id?: number }
  ): Promise<ApiResponse<MSPCustomerReport[]>> {
    return httpClient.get<ApiResponse<MSPCustomerReport[]>>('/api/v1/msp/reports/performance', params);
  }

  // ==================== 辅助方法 ====================

  /**
   * 检查当前用户是否是 MSP 员工
   */
  static async isMSPUser(): Promise<{ isMSP: boolean; isAdmin: boolean }> {
    try {
      const res = await httpClient.get<ApiResponse<{ is_msp: boolean; is_admin?: boolean }>>('/api/v1/msp/status');
      return {
        isMSP: res.data?.is_msp || false,
        isAdmin: res.data?.is_admin || false,
      };
    } catch {
      return { isMSP: false, isAdmin: false };
    }
  }

  /**
   * 获取当前用户的 MSP 上下文
   */
  static async getMSPContext(): Promise<ApiResponse<MSPContext>> {
    return httpClient.get<ApiResponse<MSPContext>>('/api/v1/msp/context');
  }

  // ==================== 审计与历史 ====================

  /**
   * 获取分配历史记录
   */
  static async getAllocationHistory(
    params: { msp_user_id?: number; customer_tenant_id?: number; start_date?: string; end_date?: string }
  ): Promise<ApiResponse<MSPAllocationHistory[]>> {
    return httpClient.get<ApiResponse<MSPAllocationHistory[]>>('/api/v1/msp/allocations/history', params);
  }
}

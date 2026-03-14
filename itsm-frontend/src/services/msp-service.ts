import { MSPAPI } from '@/lib/api/msp-api';
import type {
  MSPAllocation,
  CreateAllocationRequest,
  MSPCustomersResponse,
  MSPCustomerTicketsResponse,
  MSPCustomerReport,
  MSPAllocationHistory,
  MSPContext,
} from '@/types/msp';

export class MBPService {
  // 缓存当前用户 MSP 状态
  private static _isMSPUser: boolean | null = null;
  private static mspContext: MSPContext | null = null;

  /**
   * 获取所有分配（MSP 员工）
   */
  static async getAllocations(params?: { page?: number; pageSize?: number }): Promise<{
    allocations: MSPAllocation[];
    total: number;
  }> {
    const res = await MSPAPI.getAllocations(params);
    if (res.data) {
      return {
        allocations: res.data.allocations,
        total: res.data.total,
      };
    }
    return { allocations: [], total: 0 };
  }

  /**
   * 创建分配（MSP Manager）
   */
  static async createAllocation(data: CreateAllocationRequest): Promise<MSPAllocation> {
    const res = await MSPAPI.createAllocation(data);
    if (res.code !== 0) {
      throw new Error(res.message || '创建分配失败');
    }
    return res.data;
  }

  /**
   * 解除分配
   */
  static async deallocate(mspUserId: number, customerTenantId: number, reason?: string): Promise<void> {
    const res = await MSPAPI.deallocate(mspUserId, customerTenantId, reason);
    if (res.code !== 0) {
      throw new Error(res.message || '解除分配失败');
    }
  }

  /**
   * 获取客户列表（MSP 视角）
   */
  static async getCustomers(): Promise<{
    customers: { id: number; code: string; name: string }[];
    total: number;
  }> {
    const res = await MSPAPI.getCustomers();
    if (res.data) {
      return {
        customers: res.data.customers,
        total: res.data.total,
      };
    }
    return { customers: [], total: 0 };
  }

  /**
   * 获取指定客户的工单
   */
  static async getCustomerTickets(
    customerTenantId: number,
    params?: { status?: string; page?: number; pageSize?: number }
  ): Promise<{
    tickets: any[];
    total: number;
  }> {
    const res = await MSPAPI.getCustomerTickets(customerTenantId, params);
    if (res.data && res.data.tickets) {
      return {
        tickets: Array.isArray(res.data.tickets) ? res.data.tickets : [res.data.tickets],
        total: res.data.total || 0,
      };
    }
    return { tickets: [], total: 0 };
  }

  /**
   * 为工单分配 MSP 技术员
   */
  static async assignTechnician(
    ticketId: number,
    customerTenantId: number,
    assignerUserId?: number
  ): Promise<{ id: number; status: string }> {
    const res = await MSPAPI.assignTechnician(ticketId, customerTenantId, assignerUserId);
    if (res.code !== 0) {
      throw new Error(res.message || '分配技术员失败');
    }
    return res.data;
  }

  /**
   * 获取客户服务报表
   */
  static async getCustomerReports(params: {
    start_date: string;
    end_date: string;
    customer_tenant_id?: number;
  }): Promise<MSPCustomerReport[]> {
    const res = await MSPAPI.getCustomerReports(params);
    return res.data || [];
  }

  /**
   * 检查当前用户是否是 MSP 员工（带缓存）
   */
  static async isMSPUser(): Promise<boolean> {
    if (this._isMSPUser !== null) {
      return this._isMSPUser;
    }
    this._isMSPUser = await MSPAPI.isMSPUser();
    return this._isMSPUser;
  }

  /**
   * 获取当前 MSP 上下文（带缓存）
   */
  static async getMSPContext(): Promise<MSPContext | null> {
    if (this.mspContext !== null) {
      return this.mspContext;
    }
    const res = await MSPAPI.getMSPContext();
    if (res.code === 0 && res.data) {
      this.mspContext = res.data;
      return this.mspContext;
    }
    return null;
  }

  /**
   * 刷新 MSP 缓存
   */
  static refreshCache(): void {
    this._isMSPUser = null;
    this.mspContext = null;
  }

  /**
   * 获取分配历史
   */
  static async getAllocationHistory(params: {
    msp_user_id?: number;
    customer_tenant_id?: number;
    start_date?: string;
    end_date?: string;
  }): Promise<MSPAllocationHistory[]> {
    const res = await MSPAPI.getAllocationHistory(params);
    return res.data || [];
  }
}

export default MBPService;

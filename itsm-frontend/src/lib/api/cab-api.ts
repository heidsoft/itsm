import { httpClient } from '@/lib/api/http-client';
import type {
  CabMember,
  AddCabMemberRequest,
  UpdateCabMemberRequest,
  CabBoardType,
} from '@/types/cab';

/**
 * CAB 成员名册 API 客户端。
 * 对应后端 handlers/cab —— /api/v1/cab/members。
 * 注意：CAB 审批流转本身由审批链引擎处理，此处只管理成员名册。
 */
export class CabApi {
  /** 列出某类型 CAB 成员（含未激活，便于管理端启停）。 */
  static async getMembers(boardType: CabBoardType = 'CAB'): Promise<CabMember[]> {
    return httpClient.get<CabMember[]>('/api/v1/cab/members', { type: boardType });
  }

  /** 新增 CAB 成员（默认激活）。 */
  static async addMember(req: AddCabMemberRequest): Promise<CabMember> {
    return httpClient.post<CabMember>('/api/v1/cab/members', req);
  }

  /** 更新 CAB 成员（角色 / 激活状态）。 */
  static async updateMember(
    id: number,
    req: UpdateCabMemberRequest
  ): Promise<CabMember> {
    return httpClient.put<CabMember>(`/api/v1/cab/members/${id}`, req);
  }

  /** 移除 CAB 成员。 */
  static async removeMember(id: number): Promise<{ deleted: number }> {
    return httpClient.delete<{ deleted: number }>(`/api/v1/cab/members/${id}`);
  }
}

export default CabApi;

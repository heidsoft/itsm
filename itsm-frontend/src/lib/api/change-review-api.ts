import { httpClient } from '@/lib/api/http-client';
import type {
  ReviewMember,
  AddReviewMemberRequest,
  UpdateReviewMemberRequest,
  ReviewBoardType,
} from '@/types/change-review';

/**
 * 变更评审组名册 API 客户端。
 * 对应后端 handlers/change_review —— /api/v1/change-review/members。
 * 注意：评审审批流转本身由审批链引擎处理，此处只管理成员名册。
 */
export class ChangeReviewApi {
  /** 列出某类型评审组成员（含未激活，便于管理端启停）。 */
  static async getMembers(boardType: ReviewBoardType = 'REVIEW'): Promise<ReviewMember[]> {
    return httpClient.get<ReviewMember[]>('/api/v1/change-review/members', { type: boardType });
  }

  /** 新增评审组成员（默认激活）。 */
  static async addMember(req: AddReviewMemberRequest): Promise<ReviewMember> {
    return httpClient.post<ReviewMember>('/api/v1/change-review/members', req);
  }

  /** 更新评审组成员（角色 / 激活状态）。 */
  static async updateMember(
    id: number,
    req: UpdateReviewMemberRequest
  ): Promise<ReviewMember> {
    return httpClient.put<ReviewMember>(`/api/v1/change-review/members/${id}`, req);
  }

  /** 移除评审组成员。 */
  static async removeMember(id: number): Promise<{ deleted: number }> {
    return httpClient.delete<{ deleted: number }>(`/api/v1/change-review/members/${id}`);
  }
}

export default ChangeReviewApi;

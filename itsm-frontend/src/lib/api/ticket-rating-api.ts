import { httpClient } from './http-client';

// 工单评分类型定义
export interface TicketRating {
  rating: number; // 1-5星
  comment: string;
  ratedAt?: string;
  ratedBy: number;
  ratedByName?: string;
}

export interface SubmitTicketRatingRequest {
  rating: number; // 1-5星
  comment?: string;
}

export interface RatingStats {
  totalRatings: number;
  averageRating: number;
  ratingDistribution: {
    [key: number]: number; // {1: 10, 2: 5, 3: 8, 4: 20, 5: 30}
  };
  byAssignee?: {
    [key: number]: {
      assigneeId: number;
      assigneeName: string;
      totalRatings: number;
      averageRating: number;
    };
  };
  byCategory?: {
    [key: number]: {
      categoryId: number;
      categoryName: string;
      totalRatings: number;
      averageRating: number;
    };
  };
}

export interface GetRatingStatsParams {
  assigneeId?: number;
  categoryId?: number;
  startDate?: string;
  endDate?: string;
}

export class TicketRatingApi {
  // 提交工单评分
  static async submitRating(
    ticketId: number,
    data: SubmitTicketRatingRequest
  ): Promise<TicketRating> {
    return httpClient.post<TicketRating>(`/api/v1/tickets/${ticketId}/rating`, data);
  }

  // 获取工单评分
  static async getRating(ticketId: number): Promise<TicketRating | null> {
    return httpClient.get<TicketRating | null>(`/api/v1/tickets/${ticketId}/rating`);
  }

  // 获取评分统计
  static async getRatingStats(params?: GetRatingStatsParams): Promise<RatingStats> {
    return httpClient.get<RatingStats>('/api/v1/tickets/rating-stats', params);
  }
}

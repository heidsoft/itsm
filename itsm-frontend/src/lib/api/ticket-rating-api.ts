import { httpClient } from './http-client';

// 工单评分类型定义
export interface TicketRating {
  rating: number; // 1-5星
  comment: string;
  rated_at?: string;
  rated_by: number;
  rated_by_name?: string;
}

export interface SubmitTicketRatingRequest {
  rating: number; // 1-5星
  comment?: string;
}

export interface RatingStats {
  total_ratings: number;
  average_rating: number;
  rating_distribution: {
    [key: number]: number; // {1: 10, 2: 5, 3: 8, 4: 20, 5: 30}
  };
  by_assignee?: {
    [key: number]: {
      assignee_id: number;
      assignee_name: string;
      total_ratings: number;
      average_rating: number;
    };
  };
  by_category?: {
    [key: number]: {
      category_id: number;
      category_name: string;
      total_ratings: number;
      average_rating: number;
    };
  };
}

export interface GetRatingStatsParams {
  assignee_id?: number;
  category_id?: number;
  start_date?: string;
  end_date?: string;
}

export class TicketRatingApi {
  // 提交工单评分
  static async submitRating(ticketId: number, data: SubmitTicketRatingRequest): Promise<TicketRating> {
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


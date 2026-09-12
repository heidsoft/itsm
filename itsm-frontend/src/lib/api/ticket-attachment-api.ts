/**
 * 工单附件API
 * 提供工单附件的上传、查询、下载、预览、删除功能
 */

import { httpClient } from './http-client';

export interface TicketAttachment {
  id: number;
  ticketId: number;
  fileName: string;
  filePath: string;
  fileUrl: string;
  fileSize: number;
  fileType: string;
  mimeType: string;
  uploadedBy: number;
  uploader?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role?: string;
    department?: string;
    tenantId?: number;
  };
  createdAt: string;
}

export interface ListTicketAttachmentsResponse {
  attachments: TicketAttachment[];
  total: number;
}

export class TicketAttachmentApi {
  /**
   * 获取工单附件列表
   */
  static async listAttachments(ticketId: number): Promise<ListTicketAttachmentsResponse> {
    const response = await httpClient.get<ListTicketAttachmentsResponse>(
      `/api/v1/tickets/${ticketId}/attachments`
    );
    return response;
  }

  /**
   * 上传附件
   */
  static async uploadAttachment(
    ticketId: number,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<TicketAttachment> {
    const formData = new FormData();
    formData.append('file', file);

    // 必须走 httpClient：它统一负责 X-CSRF-Token、withCredentials（httpOnly cookie）、
    // 租户 header、CSRF 轮换后重试与 camelCase 转换。自建 XHR 会被后端以
    // 403 "CSRF token missing" 拒绝。
    return httpClient.post<TicketAttachment>(
      `/api/v1/tickets/${ticketId}/attachments`,
      formData,
      { onUploadProgress: onProgress }
    );
  }

  /**
   * 下载附件
   */
  static getDownloadUrl(ticketId: number, attachmentId: number): string {
    return `/api/v1/tickets/${ticketId}/attachments/${attachmentId}`;
  }

  /**
   * 预览附件
   */
  static getPreviewUrl(ticketId: number, attachmentId: number): string {
    return `/api/v1/tickets/${ticketId}/attachments/${attachmentId}/preview`;
  }

  /**
   * 删除附件
   */
  static async deleteAttachment(ticketId: number, attachmentId: number): Promise<void> {
    await httpClient.delete(`/api/v1/tickets/${ticketId}/attachments/${attachmentId}`);
  }

  /**
   * 格式化文件大小
   */
  static formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  }

  /**
   * 获取文件图标类型
   */
  static getFileIconType(mimeType: string): string {
    if (mimeType.startsWith('image/')) return 'image';
    if (mimeType.startsWith('video/')) return 'video';
    if (mimeType.startsWith('audio/')) return 'audio';
    if (mimeType.includes('pdf')) return 'pdf';
    if (mimeType.includes('word') || mimeType.includes('document')) return 'word';
    if (mimeType.includes('excel') || mimeType.includes('spreadsheet')) return 'excel';
    if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'powerpoint';
    if (mimeType.includes('zip') || mimeType.includes('rar')) return 'archive';
    return 'file';
  }
}

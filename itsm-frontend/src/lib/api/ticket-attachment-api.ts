/**
 * 工单附件API
 * 提供工单附件的上传、查询、下载、预览、删除功能
 */

import { httpClient } from './http-client';
import { API_BASE_URL } from '@/app/lib/api-config';

export interface TicketAttachment {
  id: number;
  ticket_id: number;
  file_name: string;
  file_path: string;
  file_url: string;
  file_size: number;
  file_type: string;
  mime_type: string;
  uploaded_by: number;
  uploader?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role?: string;
    department?: string;
    tenant_id?: number;
  };
  created_at: string;
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

    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();

      // 监听上传进度
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable && onProgress) {
          const progress = (event.loaded / event.total) * 100;
          onProgress(progress);
        }
      });

      // 监听完成
      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            const response = JSON.parse(xhr.responseText);
            if (response.code === 200 && response.data) {
              resolve(response.data);
            } else {
              reject(new Error(response.message || '上传失败'));
            }
          } catch (error) {
            reject(new Error('响应格式错误'));
          }
        } else {
          reject(new Error(`上传失败: ${xhr.statusText}`));
        }
      });

      // 监听错误
      xhr.addEventListener('error', () => {
        reject(new Error('上传失败'));
      });

      const baseURL = API_BASE_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';
      xhr.open('POST', `${baseURL}/api/v1/tickets/${ticketId}/attachments`);

      // 添加认证头
      const token = httpClient.getAuthToken();
      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      }

      xhr.send(formData);
    });
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
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
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


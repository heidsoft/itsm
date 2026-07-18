import { TicketAttachmentApi } from '@/lib/api/ticket-attachment-api';
import type { AttachmentAdapter, AttachmentItem } from '../types';

export const ticketAttachmentAdapter: AttachmentAdapter = {
  async list(targetId) {
    const res = await TicketAttachmentApi.listAttachments(Number(targetId));
    return (res.attachments || []) as unknown as AttachmentItem[];
  },
  async upload(targetId, file, onProgress) {
    const res = await TicketAttachmentApi.uploadAttachment(Number(targetId), file, onProgress);
    return res as unknown as AttachmentItem;
  },
  getDownloadUrl(targetId, attachmentId) {
    return TicketAttachmentApi.getDownloadUrl(Number(targetId), attachmentId);
  },
  getPreviewUrl(targetId, attachmentId) {
    return TicketAttachmentApi.getPreviewUrl(Number(targetId), attachmentId);
  },
  async remove(targetId, attachmentId) {
    await TicketAttachmentApi.deleteAttachment(Number(targetId), attachmentId);
  },
};

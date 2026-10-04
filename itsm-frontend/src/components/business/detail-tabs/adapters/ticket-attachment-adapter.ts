import { TicketAttachmentApi } from '@/lib/api/ticket-attachment-api';
import type { AttachmentAdapter } from '../types';

export const ticketAttachmentAdapter: AttachmentAdapter = {
  async list(targetId) {
    const res = await TicketAttachmentApi.listAttachments(Number(targetId));
    return res.items;
  },
  async upload(targetId, file, onProgress) {
    return TicketAttachmentApi.uploadAttachment(Number(targetId), file, onProgress);
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

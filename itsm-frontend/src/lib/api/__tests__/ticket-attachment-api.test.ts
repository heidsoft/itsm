import { TicketAttachmentApi } from '@/lib/api/ticket-attachment-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    getAuthToken: jest.fn().mockReturnValue('mock-token'),
  },
}));

jest.mock('@/lib/api/api-config', () => ({
  API_BASE_URL: 'http://localhost:8090',
}));

const mockGet = httpClient.get as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('TicketAttachmentApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('listAttachments', () => {
    it('should list attachments for a ticket', async () => {
      const mockData = { attachments: [{ id: 1, fileName: 'test.pdf' }], total: 1 };
      mockGet.mockResolvedValue(mockData);
      const result = await TicketAttachmentApi.listAttachments(10);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/10/attachments');
      expect(result.attachments).toHaveLength(1);
    });
  });

  describe('getDownloadUrl', () => {
    it('should return correct download URL', () => {
      const url = TicketAttachmentApi.getDownloadUrl(5, 3);
      expect(url).toBe('/api/v1/tickets/5/attachments/3');
    });
  });

  describe('getPreviewUrl', () => {
    it('should return correct preview URL', () => {
      const url = TicketAttachmentApi.getPreviewUrl(5, 3);
      expect(url).toBe('/api/v1/tickets/5/attachments/3/preview');
    });
  });

  describe('deleteAttachment', () => {
    it('should delete an attachment', async () => {
      mockDelete.mockResolvedValue(undefined);
      await TicketAttachmentApi.deleteAttachment(5, 3);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/5/attachments/3');
    });
  });

  describe('formatFileSize', () => {
    it('should format 0 bytes', () => {
      expect(TicketAttachmentApi.formatFileSize(0)).toBe('0 B');
    });
    it('should format KB', () => {
      expect(TicketAttachmentApi.formatFileSize(1024)).toBe('1 KB');
    });
    it('should format MB', () => {
      expect(TicketAttachmentApi.formatFileSize(1048576)).toBe('1 MB');
    });
  });

  describe('getFileIconType', () => {
    it('should return image for image mime types', () => {
      expect(TicketAttachmentApi.getFileIconType('image/png')).toBe('image');
    });
    it('should return pdf for pdf mime types', () => {
      expect(TicketAttachmentApi.getFileIconType('application/pdf')).toBe('pdf');
    });
    it('should return file for unknown mime types', () => {
      expect(TicketAttachmentApi.getFileIconType('application/octet-stream')).toBe('file');
    });
  });
});

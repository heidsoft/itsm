import { A2UIApi } from '../a2ui-api';
import { httpClient } from '../http-client';

// Mock httpClient.getBaseURL
jest.mock('../http-client', () => ({
  httpClient: {
    getBaseURL: jest.fn(() => 'http://localhost:3000'),
  },
}));

// Mock global fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('A2UIApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('generateTicketForm', () => {
    it('should post to a2ui/ticket/form', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ code: 0, message: 'success', messages: ['form generated'] }),
      });

      const result = await A2UIApi.generateTicketForm('create ticket', 'surface-1');
      expect(mockFetch).toHaveBeenCalledWith('http://localhost:3000/api/v1/a2ui/ticket/form', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ intent: 'create ticket', surfaceId: 'surface-1' }),
      });
      expect(result.code).toBe(0);
      expect(result.messages).toEqual(['form generated']);
    });

    it('should handle null surfaceId', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ code: 0, message: 'success', messages: [] }),
      });

      await A2UIApi.generateTicketForm('intent', null);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          body: JSON.stringify({ intent: 'intent', surfaceId: null }),
        })
      );
    });

    it('should throw on non-ok response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ code: 1, message: 'Internal error' }),
      });

      await expect(A2UIApi.generateTicketForm('x', null)).rejects.toThrow('Internal error');
    });

    it('should throw on non-zero code', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ code: 1, message: 'Validation error' }),
      });

      await expect(A2UIApi.generateTicketForm('x', null)).rejects.toThrow('Validation error');
    });
  });

  describe('handleTicketAction', () => {
    it('should post to a2ui/ticket/action', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ code: 0, message: 'ok', messages: ['action done'] }),
      });

      const result = await A2UIApi.handleTicketAction('submit', 'surface-2', { ticketId: 123 });
      expect(mockFetch).toHaveBeenCalledWith('http://localhost:3000/api/v1/a2ui/ticket/action', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'submit', surfaceId: 'surface-2', context: { ticketId: 123 } }),
      });
      expect(result.messages).toEqual(['action done']);
    });
  });

  describe('error propagation', () => {
    it('should propagate network errors', async () => {
      mockFetch.mockRejectedValue(new Error('Network failure'));
      await expect(A2UIApi.generateTicketForm('x', null)).rejects.toThrow('Network failure');
    });

    it('should use status as error when no message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 403,
        json: () => Promise.resolve({ code: 1 }),
      });

      await expect(A2UIApi.generateTicketForm('x', null)).rejects.toThrow('HTTP error! status: 403');
    });
  });
});

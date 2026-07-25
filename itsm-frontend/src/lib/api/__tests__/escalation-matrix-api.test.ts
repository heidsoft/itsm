import { EscalationMatrixApi, DEFAULT_ESCALATION_MATRIX } from '@/lib/api/escalation-matrix-api';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

describe('EscalationMatrixApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getMatrix', () => {
    it('should return default escalation matrix', async () => {
      const res = await EscalationMatrixApi.getMatrix();
      expect(res).toEqual(DEFAULT_ESCALATION_MATRIX);
    });

    it('should have P1, P2, P3 priorities', async () => {
      const res = await EscalationMatrixApi.getMatrix();
      expect(res.P1).toBeDefined();
      expect(res.P2).toBeDefined();
      expect(res.P3).toBeDefined();
    });

    it('should have correct P1 escalation levels', async () => {
      const res = await EscalationMatrixApi.getMatrix();
      expect(res.P1).toHaveLength(3);
      expect(res.P1[0].level).toBe(1);
      expect(res.P1[0].thresholdMinutes).toBe(5);
      expect(res.P1[1].level).toBe(2);
      expect(res.P1[2].level).toBe(3);
    });

    it('should have correct P2 escalation levels', async () => {
      const res = await EscalationMatrixApi.getMatrix();
      expect(res.P2).toHaveLength(2);
      expect(res.P2[0].thresholdMinutes).toBe(30);
    });

    it('should have correct P3 escalation levels', async () => {
      const res = await EscalationMatrixApi.getMatrix();
      expect(res.P3).toHaveLength(1);
      expect(res.P3[0].thresholdMinutes).toBe(240);
    });
  });
});

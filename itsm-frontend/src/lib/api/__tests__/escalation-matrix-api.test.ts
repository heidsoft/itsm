import {
  EscalationMatrixApi,
  DEFAULT_ESCALATION_MATRIX,
  type EscalationMatrix,
} from '@/lib/api/escalation-matrix-api';
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

const mockGet = jest.mocked(httpClient.get);

// 后端 GET /api/v1/escalation-matrices 的 data 就是矩阵本身：
// 键为优先级（critical/high/medium/low），级别字段为 afterMinutes/notifyRoles/notifyUserIDs/description。
const SERVER_MATRIX: EscalationMatrix = {
  critical: [
    {
      level: 1,
      afterMinutes: 10,
      notifyRoles: ['manager'],
      notifyUserIDs: [7],
      description: '自定义：10 分钟未响应升级至部门经理',
    },
  ],
  low: [
    {
      level: 1,
      afterMinutes: 960,
      notifyRoles: ['it_admin'],
      notifyUserIDs: [],
      description: '自定义：16 小时未响应升级至 IT 管理员',
    },
  ],
};

describe('EscalationMatrixApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getMatrix', () => {
    it('请求升级矩阵接口', async () => {
      mockGet.mockResolvedValue(SERVER_MATRIX);
      await EscalationMatrixApi.getMatrix();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/escalation-matrices');
    });

    it('原样返回服务端矩阵，不得用内置默认覆盖租户自定义', async () => {
      mockGet.mockResolvedValue(SERVER_MATRIX);
      const res = await EscalationMatrixApi.getMatrix();
      expect(res).toEqual(SERVER_MATRIX);
      expect(res.critical[0].afterMinutes).toBe(10);
      expect(res.low[0].notifyRoles).toEqual(['it_admin']);
    });

    it('空矩阵属不可用而非「未配置」，必须抛出而不是静默回落默认', async () => {
      mockGet.mockResolvedValue({});
      await expect(EscalationMatrixApi.getMatrix()).rejects.toThrow(
        '升级矩阵返回为空，无法确认当前生效配置',
      );
    });

    it('响应缺失同样抛出', async () => {
      mockGet.mockResolvedValue(null);
      await expect(EscalationMatrixApi.getMatrix()).rejects.toThrow(
        '升级矩阵返回为空，无法确认当前生效配置',
      );
    });

    it('请求失败向上抛出，交页面显式报错', async () => {
      mockGet.mockRejectedValue(new Error('HTTP error! status: 500'));
      await expect(EscalationMatrixApi.getMatrix()).rejects.toThrow('HTTP error! status: 500');
    });
  });

  describe('DEFAULT_ESCALATION_MATRIX', () => {
    it('与后端 DefaultEscalationMatrix 同键同字段，避免页面自造优先级', () => {
      expect(Object.keys(DEFAULT_ESCALATION_MATRIX)).toEqual(['critical', 'high', 'medium', 'low']);
      expect(DEFAULT_ESCALATION_MATRIX.critical.map(l => l.afterMinutes)).toEqual([15, 30, 60]);
      expect(DEFAULT_ESCALATION_MATRIX.critical.map(l => l.notifyRoles)).toEqual([
        ['manager'],
        ['it_admin'],
        ['admin'],
      ]);
      expect(DEFAULT_ESCALATION_MATRIX.low[0].afterMinutes).toBe(480);
      Object.values(DEFAULT_ESCALATION_MATRIX)
        .flat()
        .forEach(level => {
          expect(level).toEqual({
            level: expect.any(Number),
            afterMinutes: expect.any(Number),
            notifyRoles: expect.any(Array),
            notifyUserIDs: expect.any(Array),
            description: expect.any(String),
          });
        });
    });
  });
});

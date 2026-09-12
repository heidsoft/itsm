jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    info: jest.fn(),
    warning: jest.fn(),
    error: jest.fn(),
  },
  notification: {
    error: jest.fn(),
    destroy: jest.fn(),
  },
  Button: ({ children, onClick }: { children: React.ReactNode; onClick: () => void }) => (
    <button onClick={onClick}>{children}</button>
  ),
}));

jest.mock('@/lib/api/base-api-handler', () => ({
  getFriendlyErrorMessage: jest.fn((error: unknown) => {
    if (error instanceof Error) return error.message;
    if (typeof error === 'string') return error;
    return '操作失败，请稍后重试';
  }),
}));

import { message, notification } from 'antd';
import { notify } from '../notify';

const mockMessage = message as jest.Mocked<typeof message>;
const mockNotification = notification as jest.Mocked<typeof notification>;

describe('notify', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('success', () => {
    it('calls message.success with text and default duration 3s', () => {
      notify.success('保存成功');
      expect(mockMessage.success).toHaveBeenCalledWith('保存成功', 3);
    });

    it('respects custom duration', () => {
      notify.success('完成', { duration: 5 });
      expect(mockMessage.success).toHaveBeenCalledWith('完成', 5);
    });
  });

  describe('info', () => {
    it('calls message.info with text and default duration 3s', () => {
      notify.info('已同步');
      expect(mockMessage.info).toHaveBeenCalledWith('已同步', 3);
    });
  });

  describe('warning', () => {
    it('calls message.warning with text and default duration 4s', () => {
      notify.warning('注意冲突');
      expect(mockMessage.warning).toHaveBeenCalledWith('注意冲突', 4);
    });
  });

  describe('error — guidance table', () => {
    it('resolves 403 from error.status to actionable guidance', () => {
      const err = new Error('Forbidden');
      (err as Error & { status: number }).status = 403;

      notify.error(err);

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toContain('没有权限');
      expect(text).toContain('联系管理员');
    });

    it('resolves 409 from error.code to actionable guidance', () => {
      notify.error({ code: 409 });

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toContain('数据已被其他人修改');
    });

    it('resolves 500 from response.status to actionable guidance', () => {
      notify.error({ response: { status: 500, data: {} } });

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toContain('服务暂时不可用');
    });

    it('resolves Network Error from error.message text match', () => {
      notify.error(new Error('Network Error'));

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toContain('网络连接失败');
    });

    it('resolves timeout from error.message text match', () => {
      notify.error(new Error('Request timeout'));

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toContain('请求超时');
    });
  });

  describe('error — context option', () => {
    it('prepends context to guidance title', () => {
      notify.error({ status: 400 }, { context: '提交变更' });

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toMatch(/^提交变更失败：/);
    });
  });

  describe('error — fallback', () => {
    it('uses fallback option when getFriendlyErrorMessage returns generic text', () => {
      notify.error({}, { fallback: '自定义提示' });

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toBe('自定义提示');
    });

    it('uses context + generic fallback when no guidance matches and no custom fallback', () => {
      notify.error({}, { context: '保存' });

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toBe('保存失败：操作失败，请稍后重试');
    });

    it('falls through to friendly message when error is an Error instance', () => {
      notify.error(new Error('unknown'));

      const text = mockMessage.error.mock.calls[0][0] as string;
      expect(text).toBe('unknown');
    });
  });

  describe('error — action option uses notification', () => {
    it('calls notification.error with action button when action is provided', () => {
      const onClick = jest.fn();
      notify.error(new Error('Network Error'), {
        context: '加载数据',
        action: { label: '重试', onClick },
      });

      expect(mockNotification.error).toHaveBeenCalledTimes(1);
      const call = mockNotification.error.mock.calls[0][0] as Record<string, unknown>;
      expect(call.message).toBe('加载数据失败');
      expect(call.description).toContain('网络连接失败');
      expect(call.duration).toBe(6);
      expect(call.actions).toBeDefined();
    });

    it('does NOT call message.error when action is provided', () => {
      notify.error(new Error('Network Error'), {
        action: { label: '重试', onClick: jest.fn() },
      });

      expect(mockMessage.error).not.toHaveBeenCalled();
    });
  });

  describe('error — duration option', () => {
    it('passes custom duration to message.error', () => {
      notify.error(new Error('Network Error'), { duration: 10 });
      expect(mockMessage.error).toHaveBeenCalledWith(expect.any(String), 10);
    });
  });

  describe('aiUnavailable', () => {
    it('shows AI degradation message via message.warning by default', () => {
      notify.aiUnavailable();

      const text = mockMessage.warning.mock.calls[0][0] as string;
      expect(text).toContain('AI 助手');
      expect(text).toContain('暂时不可用');
      expect(text).toContain('手动填写');
      expect(text).not.toContain('操作失败');
    });

    it('uses custom context', () => {
      notify.aiUnavailable('智能分派');

      const text = mockMessage.warning.mock.calls[0][0] as string;
      expect(text).toContain('智能分派');
    });

    it('uses notification when action is provided', () => {
      notify.aiUnavailable('AI 助手', {
        action: { label: '查看日志', onClick: jest.fn() },
      });

      expect(mockNotification.error).toHaveBeenCalledTimes(1);
      expect(mockMessage.warning).not.toHaveBeenCalled();
    });
  });

  describe('aiLowConfidence', () => {
    it('shows low confidence message via message.info', () => {
      notify.aiLowConfidence();

      const text = mockMessage.info.mock.calls[0][0] as string;
      expect(text).toContain('AI 建议');
      expect(text).toContain('置信度较低');
    });

    it('uses custom context', () => {
      notify.aiLowConfidence('推荐方案');

      const text = mockMessage.info.mock.calls[0][0] as string;
      expect(text).toContain('推荐方案');
    });
  });
});

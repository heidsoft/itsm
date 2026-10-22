import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import ConnectorsAdminPage from '../page';
import connectorService from '@/lib/services/connector-service';

/**
 * 连接器健康列的回归。
 *
 * 真实缺陷：页面用 `health[`${0}/${name}/${provider}`]` 取健康状态。后端健康结果的 key 是
 * `tenantID/name/provider`，而管理员的租户号恒为 1 —— 硬编码的 0 永远匹配不上，
 * 于是「已启用」永远显示成绿色，即使实例正在报错。
 * 后端同时改成只回本租户、按 name/provider 标识（见 handlers/connector/health_scope_test.go）。
 */

jest.mock('@/lib/services/connector-service', () => ({
  __esModule: true,
  default: {
    list: jest.fn(),
    configs: jest.fn(),
    health: jest.fn(),
    provision: jest.fn(),
    revoke: jest.fn(),
    test: jest.fn(),
    send: jest.fn(),
  },
}));

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key, locale: 'zh-CN' }),
}));

jest.mock('@/app/components/PageContainer', () => ({
  PageContainer: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/components/common/UsageGuideCard', () => ({
  UsageGuideCard: () => null,
}));

const mocked = connectorService as jest.Mocked<typeof connectorService>;

const manifest = {
  name: 'feishu',
  version: '1.0.0',
  title: '飞书',
  provider: 'feishu',
  type: 'im',
  capabilities: ['send_message'],
  local: true,
  installed: true,
  category: 'im',
  requiredPermissions: ['connector.read'],
  checksum: 'sha256:x',
};

describe('连接器健康列', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mocked.list.mockResolvedValue([manifest] as never);
    mocked.configs.mockResolvedValue([
      { name: 'feishu', provider: 'feishu', type: 'im', enabled: true },
    ] as never);
  });

  it('健康检查失败时显示异常状态（不再因为硬编码租户号而永远显示绿色）', async () => {
    mocked.health.mockResolvedValue([
      { name: 'feishu', provider: 'feishu', ok: false, message: 'callback token mismatch' },
    ] as never);

    render(
      <App>
        <ConnectorsAdminPage />
      </App>,
    );

    await waitFor(() => expect(mocked.health).toHaveBeenCalled());
    expect(await screen.findByText('connectors.status.error')).toBeInTheDocument();
    expect(screen.queryByText('connectors.status.enabled')).not.toBeInTheDocument();
  });

  it('健康检查正常时显示运行中', async () => {
    mocked.health.mockResolvedValue([
      { name: 'feishu', provider: 'feishu', ok: true, latencyMs: 12 },
    ] as never);

    render(
      <App>
        <ConnectorsAdminPage />
      </App>,
    );

    await waitFor(() => expect(mocked.health).toHaveBeenCalled());
    expect(await screen.findByText('connectors.status.running')).toBeInTheDocument();
  });

  it('同一 name 但 provider 不同的实例不会串用健康结果', async () => {
    mocked.list.mockResolvedValue([
      manifest,
      { ...manifest, provider: 'lark', title: 'Lark' },
    ] as never);
    mocked.health.mockResolvedValue([
      { name: 'feishu', provider: 'lark', ok: false, message: 'only lark is down' },
    ] as never);

    render(
      <App>
        <ConnectorsAdminPage />
      </App>,
    );

    await waitFor(() => expect(mocked.health).toHaveBeenCalled());
    // 飞书实例没有健康结果 → 回落到「已启用」；只有 lark 显示异常
    expect(await screen.findByText('connectors.status.error')).toBeInTheDocument();
    expect(screen.getByText('connectors.status.enabled')).toBeInTheDocument();
  });
});

import { render, screen } from '@testing-library/react';
import { App } from 'antd';

import MarketplaceDetailPage from '../page';
import marketplaceService from '@/lib/services/marketplace-service';
import type { MarketplaceItem } from '@/lib/services/marketplace-service';

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '5' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/services/marketplace-service', () => ({
  __esModule: true,
  default: {
    getItem: jest.fn(),
    getInstallation: jest.fn(),
    installItem: jest.fn(),
    uninstallItem: jest.fn(),
  },
}));

const mockHasPermission = jest.fn();
jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: mockHasPermission }),
}));

jest.mock('@/lib/hooks/useCapabilities', () => ({
  useCapabilities: () => ({
    allows: () => true,
    find: () => undefined,
  }),
}));

const mockedService = marketplaceService as jest.Mocked<typeof marketplaceService>;

const baseItem: MarketplaceItem = {
  id: 5,
  name: 'triage-skill',
  title: '智能分诊技能',
  type: 'skill',
  provider: 'itsm',
  latestVersion: '1.0.0',
  category: 'AI',
};

function renderPage() {
  return render(
    <App>
      <MarketplaceDetailPage />
    </App>,
  );
}

// 可点击的「立即安装」= 未 disabled 的按钮；页头与侧栏各一处，需同口径。
const enabledInstallButtons = () =>
  screen
    .queryAllByRole('button', { name: '立即安装' })
    .filter(btn => !(btn as HTMLButtonElement).disabled);

beforeEach(() => {
  jest.clearAllMocks();
  mockHasPermission.mockReturnValue(true);
  mockedService.getInstallation.mockResolvedValue(null);
});

describe('市场详情安装入口门控（D11 回归）', () => {
  it('skill 条目：页头与侧栏均禁用「暂不支持安装」，无可点的「立即安装」', async () => {
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'skill' });
    renderPage();

    const disabled = await screen.findAllByRole('button', { name: '暂不支持安装' });
    // 页头 + 侧栏各一处，且都不可点
    expect(disabled.length).toBeGreaterThanOrEqual(2);
    disabled.forEach(btn => expect(btn).toBeDisabled());
    expect(enabledInstallButtons()).toHaveLength(0);
  });

  it('connector 条目但无 marketplace:write：侧栏显示禁用的「仅管理员可安装」', async () => {
    mockHasPermission.mockReturnValue(false);
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'connector' });
    renderPage();

    const sidebarBtn = await screen.findByRole('button', { name: '仅管理员可安装' });
    expect(sidebarBtn).toBeDisabled();
    expect(enabledInstallButtons()).toHaveLength(0);
  });

  it('connector 条目且有写权限：保留可点击的「立即安装」（防止过度禁用）', async () => {
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'connector' });
    renderPage();

    const buttons = await screen.findAllByRole('button', { name: '立即安装' });
    expect(buttons.length).toBeGreaterThanOrEqual(1);
    expect(enabledInstallButtons().length).toBeGreaterThanOrEqual(1);
  });
});

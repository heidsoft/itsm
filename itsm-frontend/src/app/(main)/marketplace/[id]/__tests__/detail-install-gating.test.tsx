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
  // 交付判定来自服务端；界面不再按 type 自行推断（D11 的门控口径已随之改变）。
  deliverable: true,
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
    .queryAllByRole('button', { name: "Install now" })
    .filter(btn => !(btn as HTMLButtonElement).disabled);

beforeEach(() => {
  jest.clearAllMocks();
  mockHasPermission.mockReturnValue(true);
  mockedService.getInstallation.mockResolvedValue(null);
});

describe('市场详情安装入口门控（按服务端交付判定）', () => {
  it('skill 且 deliverable：页头与侧栏都给出可点的「立即安装」', async () => {
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'skill', deliverable: true });
    renderPage();

    const buttons = await screen.findAllByRole('button', { name: "Install now" });
    expect(buttons.length).toBeGreaterThanOrEqual(2);
    expect(enabledInstallButtons().length).toBeGreaterThanOrEqual(2);
    expect(screen.queryByRole('button', { name: "The platform does not deliver this type" })).toBeNull();
  });

  it('plugin 且 deliverable=false：两处入口禁用，且文案不再写成「尚未落地」', async () => {
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'plugin', deliverable: false });
    renderPage();

    const disabled = await screen.findAllByRole('button', { name: "The platform does not deliver this type" });
    expect(disabled.length).toBeGreaterThanOrEqual(2);
    disabled.forEach(btn => expect(btn).toBeDisabled());
    expect(enabledInstallButtons()).toHaveLength(0);
    expect(screen.queryByText(/Installation与分发机制尚未落地/)).toBeNull();
  });

  it('connector 且 deliverable=true：保留可点击的「立即安装」', async () => {
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'connector', deliverable: true });
    renderPage();

    await screen.findAllByRole('button', { name: "Install now" });
    expect(enabledInstallButtons().length).toBeGreaterThanOrEqual(1);
  });

  it('有交付能力但无 marketplace:write：侧栏显示禁用的「仅管理员可安装」', async () => {
    mockHasPermission.mockReturnValue(false);
    mockedService.getItem.mockResolvedValue({ ...baseItem, type: 'connector', deliverable: true });
    renderPage();

    const sidebarBtn = await screen.findByRole('button', { name: "Only administrators can install" });
    expect(sidebarBtn).toBeDisabled();
    expect(enabledInstallButtons()).toHaveLength(0);
  });
});

describe('市场详情版本历史（此前长期读不存在的 edges.versions）', () => {
  it('渲染服务端返回的已发布版本，按返回顺序展示版本号与说明', async () => {
    mockedService.getItem.mockResolvedValue({
      ...baseItem,
      versions: [
        { version: '1.2.0', changelog: '支持多语言关键词', releasedAt: '2026-11-01T10:00:00Z' },
        { version: '1.1.0', releasedAt: '2026-10-01T10:00:00Z' },
      ],
    });
    renderPage();

    const tab = await screen.findByRole('tab', { name: "Version History" });
    tab.click();

    expect(await screen.findByText('v1.2.0')).toBeInTheDocument();
    expect(screen.getByText('v1.1.0')).toBeInTheDocument();
    expect(screen.getByText('支持多语言关键词')).toBeInTheDocument();
    // 缺 changelog 的版本给出占位文案，而不是空白行。
    expect(screen.getByText("No update instructions yet")).toBeInTheDocument();
  });

  it('详情未带 versions 时给出空态文案，不得伪造历史', async () => {
    mockedService.getItem.mockResolvedValue({ ...baseItem });
    renderPage();

    const tab = await screen.findByRole('tab', { name: "Version History" });
    tab.click();

    expect(await screen.findByText("No version history yet")).toBeInTheDocument();
  });
});

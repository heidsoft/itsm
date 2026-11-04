import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';

import InstallationsPage from '../page';
import marketplaceService from '@/lib/services/marketplace-service';
import connectorService from '@/lib/services/connector-service';
import type { MarketplaceDependencyReport, TenantInstallation } from '@/lib/services/marketplace-service';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn() },
  serverErrorMessage: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
}));

jest.mock('@/lib/services/marketplace-service', () => ({
  __esModule: true,
  default: {
    listInstallations: jest.fn(),
    dependencyStatus: jest.fn(),
    upgradeItem: jest.fn(),
    setAutoUpgrade: jest.fn(),
    uninstallItem: jest.fn(),
    updateInstallationConfig: jest.fn(),
  },
}));

jest.mock('@/lib/services/connector-service', () => ({
  __esModule: true,
  default: { configs: jest.fn() },
}));

const mockedListItemVersions = jest.fn();
jest.mock('@/lib/api/marketplace-catalog-api', () => ({
  listItemVersions: (...args: unknown[]) => mockedListItemVersions(...args),
}));

const mockHasPermission = jest.fn();
jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: mockHasPermission }),
}));

jest.mock('@/lib/hooks/useCapabilities', () => ({
  useCapabilities: () => ({ allows: () => true, find: () => undefined }),
}));

const mockedService = marketplaceService as jest.Mocked<typeof marketplaceService>;
const mockedConnector = connectorService as jest.Mocked<typeof connectorService>;

// 商品展示字段来自安装列表的扁平 DTO；edges 形态在服务端契约里不存在。
const skillInstallation: TenantInstallation = {
  id: 10,
  tenantId: 1,
  itemId: 5,
  installedVersion: '1.0.0',
  status: 'active',
  autoUpgrade: true,
  installedAt: '2026-10-01T00:00:00Z',
  itemName: 'triage-skill',
  itemTitle: '智能分诊技能',
  itemDescription: '按标题自动分诊',
  itemType: 'skill',
  deliverable: true,
  latestVersion: '1.1.0',
  upgradeAvailable: true,
};

const cleanReport: MarketplaceDependencyReport = {
  tenantId: 1,
  checkedAt: '2026-11-03T00:00:00Z',
  total: 1,
  satisfied: 1,
  unsatisfied: 0,
  declarationMissing: 0,
  findings: [],
};

function renderPage() {
  return render(
    <App>
      <InstallationsPage />
    </App>,
  );
}

beforeEach(() => {
  jest.clearAllMocks();
  mockHasPermission.mockReturnValue(true);
  mockedConnector.configs.mockResolvedValue([]);
  mockedService.dependencyStatus.mockResolvedValue(cleanReport);
  mockedService.listInstallations.mockResolvedValue([skillInstallation]);
  mockedListItemVersions.mockResolvedValue([
    { version: '1.1.0', status: 'published', downloadCount: 0, releasedAt: '2026-11-01T00:00:00Z', createdAt: '2026-11-01T00:00:00Z', updatedAt: '2026-11-01T00:00:00Z' },
    { version: '1.0.0', status: 'published', downloadCount: 0, releasedAt: '2026-10-01T00:00:00Z', createdAt: '2026-10-01T00:00:00Z', updatedAt: '2026-10-01T00:00:00Z' },
    { version: '0.9.0', status: 'published', downloadCount: 0, releasedAt: '2026-09-01T00:00:00Z', createdAt: '2026-09-01T00:00:00Z', updatedAt: '2026-09-01T00:00:00Z' },
  ]);
});

describe('我的应用页读取扁平安装 DTO', () => {
  it('卡片用 itemTitle/描述/目录最新版本渲染，而不是「应用 #id」', async () => {
    renderPage();

    expect(await screen.findByText('智能分诊技能')).toBeInTheDocument();
    expect(screen.getByText('按标题自动分诊')).toBeInTheDocument();
    expect(screen.queryByText(/^应用 #/)).toBeNull();
    expect(screen.getByText('目录最新版本')).toBeInTheDocument();
    expect(screen.getByText('v1.1.0')).toBeInTheDocument();
  });

  it('技能不显示「配置」入口：技能没有租户级配置面，后端会拒绝该请求', async () => {
    renderPage();

    await screen.findByText('智能分诊技能');
    expect(screen.queryByRole('button', { name: '配置' })).toBeNull();
  });

  it('搜索按标题命中真实字段（此前读 edges 时任何关键词都查不到）', async () => {
    renderPage();

    await screen.findByText('智能分诊技能');
    fireEvent.change(screen.getByPlaceholderText('搜索应用名称或描述...'), {
      target: { value: '分诊' },
    });
    await waitFor(() => expect(screen.queryByText('没有找到匹配的应用')).toBeNull());
    expect(screen.getByText('智能分诊技能')).toBeInTheDocument();
  });
});

describe('升级与自动升级', () => {
  it('upgradeAvailable=true 时打开升级弹窗，确认后调用 items/:id/upgrade 并就地更新版本号', async () => {
    const upgraded = { ...skillInstallation, installedVersion: '1.1.0', upgradeAvailable: false };
    mockedService.upgradeItem.mockResolvedValue(upgraded);
    renderPage();

    const button = await screen.findByRole('button', { name: '升级到 v1.1.0' });
    fireEvent.click(button);

    // 二次确认必须先出现，且写明从哪个版本到哪个版本（换版本会重载运行态）。
    expect(await screen.findByText(/升级 \/ 回滚/)).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByText('v1.0.0').length).toBeGreaterThan(0));

    fireEvent.click(screen.getByRole('button', { name: '确认执行' }));

    await waitFor(() => expect(mockedService.upgradeItem).toHaveBeenCalledWith(5, '1.1.0'));
    // 升完之后「已安装版本」与「目录最新版本」同为 1.1.0，两处都要一致。
    await waitFor(() => expect(screen.getAllByText('v1.1.0')).toHaveLength(2));
    expect(screen.queryByRole('button', { name: '升级到 v1.1.0' })).toBeNull();
    // 换版本会重新过依赖闸门，报告要随之刷新。
    expect(mockedService.dependencyStatus).toHaveBeenCalledTimes(2);
  });

  it('可指名更旧的已发布版本实现回滚（后端 upgrade 支持 targetVersion）', async () => {
    const rolledBack = { ...skillInstallation, installedVersion: '0.9.0', upgradeAvailable: true };
    mockedService.upgradeItem.mockResolvedValue(rolledBack);
    renderPage();

    fireEvent.click(await screen.findByRole('button', { name: '升级到 v1.1.0' }));
    await screen.findByText(/升级 \/ 回滚/);

    // 选一个比当前更旧的版本 → 必须带 targetVersion 发给后端。
    // 弹窗内的目标版本下拉（页面上还有筛选用的 combobox，必须限定在弹窗内取）。
    const dialog = await screen.findByRole('dialog');
    fireEvent.mouseDown(within(dialog).getByRole('combobox'));
    // antd Select 的 popup 挂在 body 下；选项没有 title 属性，按文本匹配。
    const olderOption = await waitFor(() => {
      const opt = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
        el => el.textContent?.trim().startsWith('v0.9.0'),
      );
      if (!opt) throw new Error('option not rendered');
      return opt as HTMLElement;
    });
    fireEvent.click(olderOption);
    fireEvent.click(screen.getByRole('button', { name: '确认执行' }));

    await waitFor(() => expect(mockedService.upgradeItem).toHaveBeenCalledWith(5, '0.9.0'));
  });

  it('已最新时不摆升级按钮', async () => {
    mockedService.listInstallations.mockResolvedValue([
      { ...skillInstallation, upgradeAvailable: false },
    ]);
    renderPage();

    await screen.findByText('智能分诊技能');
    expect(screen.queryByRole('button', { name: /升级到/ })).toBeNull();
  });

  it('自动升级按钮按当前状态显示并把取反后的布尔发给后端', async () => {
    mockedService.setAutoUpgrade.mockResolvedValue({ ...skillInstallation, autoUpgrade: false });
    renderPage();

    const button = await screen.findByRole('button', { name: '自动升级：开' });
    fireEvent.click(button);

    await waitFor(() => expect(mockedService.setAutoUpgrade).toHaveBeenCalledWith(5, false));
    await waitFor(() => expect(screen.getByRole('button', { name: '自动升级：关' })).toBeInTheDocument());
  });

  it('升级失败时展示后端原因（409 会说升级未完成、仍在旧版本）', async () => {
    mockedService.upgradeItem.mockRejectedValue(new Error('升级未完成，仍在使用当前版本'));
    renderPage();

    fireEvent.click(await screen.findByRole('button', { name: '升级到 v1.1.0' }));
    fireEvent.click(await screen.findByRole('button', { name: '确认执行' }));

    expect(await screen.findByText('升级未完成，仍在使用当前版本')).toBeInTheDocument();
    expect(screen.getAllByText('v1.0.0').length).toBeGreaterThan(0);
  });
});

describe('依赖对齐视图', () => {
  it('有缺口时列出三态计数与逐条明细', async () => {
    mockedService.dependencyStatus.mockResolvedValue({
      ...cleanReport,
      satisfied: 0,
      unsatisfied: 1,
      findings: [
        {
          itemId: 5,
          itemName: 'triage-skill',
          itemTitle: '智能分诊技能',
          itemType: 'skill',
          status: 'unsatisfied',
          installedVersion: '1.0.0',
          installationStatus: 'active',
          problems: ['缺少前置组件 base-skill（>=1.0.0）'],
        },
      ],
    });
    renderPage();

    expect(await screen.findByText(/1 个前置未满足/)).toBeInTheDocument();
    expect(screen.getByText('缺少前置组件 base-skill（>=1.0.0）')).toBeInTheDocument();
  });

  it('查询失败时说明「暂不可查」，不得显示成依赖健康', async () => {
    mockedService.dependencyStatus.mockRejectedValue(new Error('依赖服务不可用'));
    renderPage();

    expect(await screen.findByText(/依赖对齐状态暂不可查：依赖服务不可用/)).toBeInTheDocument();
  });

  it('全部对齐时不打扰用户', async () => {
    renderPage();

    await screen.findByText('智能分诊技能');
    expect(screen.queryByText(/前置未满足/)).toBeNull();
  });
});

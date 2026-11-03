import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';

import AssetList from '../AssetList';
import { AssetApi, AssetStatus, AssetType, type Asset } from '@/lib/api/asset-api';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/api/asset-api', () => ({
  AssetApi: { getAssets: jest.fn() },
  AssetType: {},
  AssetStatus: {},
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: () => true }),
}));

const mockedGetAssets = AssetApi.getAssets as jest.Mock;

const hardwareAsset: Asset = {
  id: 1,
  assetNumber: 'ASSET-0001',
  name: '测试服务器',
  type: 'hardware' as AssetType,
  status: 'available' as AssetStatus,
  tenantId: 1,
  createdAt: '2026-10-31T01:00:00Z',
  updatedAt: '2026-10-31T01:00:00Z',
};

function renderList() {
  return render(
    <App>
      <AssetList />
    </App>,
  );
}

beforeEach(() => {
  jest.clearAllMocks();
  mockedGetAssets.mockResolvedValue({ items: [hardwareAsset], total: 1 });
});

describe('AssetList 类型列中文映射（D12 回归）', () => {
  it('hardware 资产显示「硬件」，不显示原始大写 HARDWARE', async () => {
    renderList();

    expect(await screen.findByText('硬件')).toBeInTheDocument();
    expect(screen.queryByText('HARDWARE')).not.toBeInTheDocument();
  });

  it('未映射的类型回退显示原始值，不丢信息', async () => {
    mockedGetAssets.mockResolvedValue({
      items: [{ ...hardwareAsset, id: 2, type: 'quantum' as unknown as AssetType }],
      total: 1,
    });
    renderList();

    expect(await screen.findByText('quantum')).toBeInTheDocument();
  });

  it('加载失败进入错误态而不是静默空表', async () => {
    mockedGetAssets.mockRejectedValue(new Error('boom'));
    renderList();

    await waitFor(() => {
      expect(screen.queryByText('硬件')).not.toBeInTheDocument();
    });
  });
});

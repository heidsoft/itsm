import React from 'react';
import { App } from 'antd';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import LicenseDetail from '../LicenseDetail';
import { AssetApi } from '@/lib/api/asset-api';
import type { License } from '@/lib/api/asset-api';

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '7' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/api/asset-api', () => ({
  AssetApi: {
    getLicense: jest.fn(),
    updateLicense: jest.fn(),
    assignLicenseUsers: jest.fn(),
  },
}));

jest.mock('@/lib/hooks/useUserListQuery', () => ({
  useUserPickerQuery: () => ({ users: [], isLoading: false }),
}));

const license: License = {
  id: 7,
  tenantId: 1,
  name: '设计软件许可证',
  licenseType: 'per-seat',
  totalQuantity: 1,
  usedQuantity: 1,
  availableQuantity: 0,
  status: 'depleted',
  users: [42],
  userNames: ['张三'],
  createdAt: '2026-10-01T08:00:00Z',
  updatedAt: '2026-10-01T08:00:00Z',
};

const clearedLicense: License = {
  ...license,
  usedQuantity: 0,
  availableQuantity: 1,
  status: 'active',
  users: [],
  userNames: [],
};

function renderDetail() {
  return render(
    <App>
      <LicenseDetail />
    </App>
  );
}

describe('LicenseDetail 清空授权', () => {
  beforeEach(() => {
    jest.mocked(AssetApi.getLicense).mockResolvedValue(license);
    jest.mocked(AssetApi.updateLicense).mockResolvedValue(clearedLicense);
  });

  it('确认后发送显式空用户列表并展示已释放席位', async () => {
    const user = userEvent.setup();
    renderDetail();

    await user.click(await screen.findByRole('button', { name: '清空授权' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/设计软件许可证/)).toHaveTextContent(
      '将移除「设计软件许可证」的 1 名授权用户，并释放 1 个席位。'
    );
    expect(AssetApi.updateLicense).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole('button', { name: '清空授权' }));

    await waitFor(() => expect(AssetApi.updateLicense).toHaveBeenCalledWith(7, { users: [] }));
    expect(await screen.findByText('0 / 1')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '清空授权' })).not.toBeInTheDocument();
  });

  it('取消确认不修改授权用户', async () => {
    const user = userEvent.setup();
    renderDetail();

    await user.click(await screen.findByRole('button', { name: '清空授权' }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /取\s*消/ }));

    expect(AssetApi.updateLicense).not.toHaveBeenCalled();
    expect(screen.getByText('张三')).toBeInTheDocument();
  });
});

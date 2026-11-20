import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import EscalationChainEditor from '../EscalationChainEditor';
import { EscalationMatrixApi } from '@/lib/api/escalation-matrix-api';
import * as permModule from '@/lib/hooks/use-permissions';

jest.mock('@/lib/api/escalation-matrix-api', () => ({
  EscalationMatrixApi: { getMatrix: jest.fn(), setMatrix: jest.fn() },
  DEFAULT_ESCALATION_MATRIX: {
    critical: [{ level: 1, afterMinutes: 15, notifyRoles: ['manager'], notifyUserIDs: [], description: '' }],
    high: [],
    medium: [],
    low: [],
  },
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: jest.fn(() => ({ hasPermission: () => true })),
}));

const mockedApi = EscalationMatrixApi as jest.Mocked<typeof EscalationMatrixApi>;
const mockedUsePermissions = permModule.usePermissions as unknown as jest.Mock;

const renderEditor = () =>
  render(
    <App>
      <EscalationChainEditor />
    </App>,
  );

describe('EscalationChainEditor 升级链编辑器', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedUsePermissions.mockReturnValue({ hasPermission: () => true });
    mockedApi.getMatrix.mockResolvedValue({
      critical: [{ level: 1, afterMinutes: 15, notifyRoles: ['manager'], notifyUserIDs: [], description: '' }],
    });
    mockedApi.setMatrix.mockResolvedValue(undefined);
  });

  it('保存前校验：存在无通知对象的级别时拦截提交并提示，不调 setMatrix', async () => {
    renderEditor();
    await screen.findByText('15 分钟');

    // 给 high（默认补齐为空链）加一级 → 新级别通知对象为空
    fireEvent.click(screen.getByText('给高加一级'));

    fireEvent.click(screen.getByRole('button', { name: /^保\s*存/ }));

    await waitFor(() =>
      expect(screen.getByText(/未配置任何通知对象/)).toBeInTheDocument(),
    );
    expect(mockedApi.setMatrix).not.toHaveBeenCalled();
  });

  it('删除级别后保存：同优先级重新编号，按编辑后的矩阵全量提交', async () => {
    mockedApi.getMatrix.mockResolvedValue({
      high: [
        { level: 1, afterMinutes: 30, notifyRoles: ['manager'], notifyUserIDs: [], description: '' },
        { level: 2, afterMinutes: 90, notifyRoles: ['it_admin'], notifyUserIDs: [], description: '' },
      ],
    });
    renderEditor();
    // canEdit 时两行都是 InputNumber + 换算小字（0.5 / 1.5 小时）
    await screen.findByText('1.5 小时');

    // 每个 priority 都有 L1 删除钮；渲染顺序 critical→high→…，mock 只回 high，
    // 但 toEditable 会补齐其余优先级（critical 补默认 L1）。high 的 L1 取最后一个…不对：
    // 默认补齐顺序固定 critical/high/medium/low，high 的 L1 删除钮是第二个。
    const deleteButtons = await screen.findAllByLabelText('删除 L1');
    fireEvent.click(deleteButtons[1]);
    await screen.findByText(/删除 高 的 L1/);
    fireEvent.click(document.querySelector('.ant-popover button.ant-btn-dangerous') as HTMLElement);

    // L2(90 分钟) 变为新 L1，0.5 小时那行应消失
    await waitFor(() => expect(screen.queryByText('0.5 小时')).not.toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /^保\s*存/ }));

    await waitFor(() =>
      expect(mockedApi.setMatrix).toHaveBeenCalledWith(
        expect.objectContaining({
          high: [expect.objectContaining({ level: 1, afterMinutes: 90, notifyRoles: ['it_admin'] })],
        }),
      ),
    );
  });

  it('保存失败显式报错，不伪装成功', async () => {
    mockedApi.setMatrix.mockRejectedValue(new Error('system_configs 写入冲突'));
    renderEditor();
    await screen.findByText('15 分钟');

    fireEvent.click(screen.getByText('给致命加一级'));
    // 新增的 L2 角色选择器带 aria-label（表格渲染后出现）；label 元素即 Select 根
    const l2Select = await screen.findByLabelText('L2 通知角色');
    fireEvent.mouseDown(l2Select);
    fireEvent.click(await screen.findByText('IT 管理员（it_admin）'));

    fireEvent.click(screen.getByRole('button', { name: /^保\s*存/ }));
    await waitFor(() => expect(screen.getByText(/system_configs 写入冲突/)).toBeInTheDocument());
  });

  it('无写权限时只读渲染：无保存按钮、无加级按钮', async () => {
    mockedUsePermissions.mockReturnValue({ hasPermission: () => false });
    renderEditor();
    await screen.findByText('15 分钟');
    expect(screen.queryByRole('button', { name: /^保\s*存/ })).not.toBeInTheDocument();
    expect(screen.queryByText('给致命加一级')).not.toBeInTheDocument();
  });
});

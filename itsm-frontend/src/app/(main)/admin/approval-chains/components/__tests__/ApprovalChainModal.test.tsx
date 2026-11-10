/**
 * D52 回归：审批链创建向导分步校验
 * - 「下一步」必须先通过步骤 1（基本信息）字段校验，空名称不得放行；
 * - 「创建」校验失败时，若错误落在隐藏的第一步容器内，必须自动跳回并给出可见提示，
 *   而不是把错误渲染在 display:none 容器里（修复前表现为「点了没反应」）。
 */

import React from 'react';
import { App } from 'antd';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ApprovalChainModal } from '../ApprovalChainModal';

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn().mockResolvedValue({ items: [], total: 0 }) },
}));

jest.mock('@/lib/api/role-api', () => ({
  RoleAPI: { getRoles: jest.fn().mockResolvedValue({ items: [], total: 0 }) },
}));

const renderModal = (onSubmit = jest.fn().mockResolvedValue(undefined)) => {
  const utils = render(
    <App>
      <ApprovalChainModal
        visible
        editingChain={null}
        onCancel={jest.fn()}
        onSubmit={onSubmit}
      />
    </App>
  );
  return { onSubmit, ...utils };
};

// 步骤 1 容器：Modal 渲染在 document.body portal 中，用 screen 查询（display 内联切换）
const getBasicStepContainer = () => screen.getByTestId('approval-chain-step-basic');

describe('ApprovalChainModal 分步校验（D52 回归）', () => {
  it('空名称点「下一步」：停留在步骤 1 并显示字段错误', async () => {
    renderModal();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    // 字段错误可见，且仍停留在第一步
    await screen.findByText('Please enter Approval ChainsName');
    expect(screen.getByText('Basic Info').closest('.ant-steps-item')).toHaveClass(
      'ant-steps-item-process'
    );
    // 步骤 1 容器保持可见
    expect(getBasicStepContainer().style.display).toBe('block');
  });

  it('名称被清空后点「创建」：自动跳回第一步并给出可见提示，且不提交', async () => {
    const onSubmit = jest.fn().mockResolvedValue(undefined);
    renderModal(onSubmit);

    const nameInput = screen.getByPlaceholderText('Please enter Approval ChainsName');

    // 填入合法名称进入第二步
    fireEvent.change(nameInput, { target: { value: '测试审批链' } });
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() => {
      expect(getBasicStepContainer().style.display).toBe('none');
    });

    // 名称在隐藏状态下被清空（等价于用户回退改坏数据后直接创建）
    fireEvent.change(nameInput, { target: { value: '' } });

    fireEvent.click(screen.getByRole('button', { name: 'Add Step' }));
    // Submit after adding a step; invalid hidden basic fields must become visible.
    fireEvent.click(screen.getByRole('button', { name: 'Create' }));

    // 自动跳回第一步 + 可见错误 + 顶部提示，onSubmit 未被调用
    await screen.findByText('Basic information is invalid. Please review it and try again.');
    await waitFor(() => {
      expect(getBasicStepContainer().style.display).toBe('block');
    });
    await screen.findByText('Please enter Approval ChainsName');
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('名称合法时「下一步」正常进入步骤配置', async () => {
    renderModal();

    fireEvent.change(screen.getByPlaceholderText('Please enter Approval ChainsName'), {
      target: { value: '测试审批链' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    await waitFor(() => {
      expect(screen.getByText('Step configuration').closest('.ant-steps-item')).toHaveClass(
        'ant-steps-item-process'
      );
    });
    expect(getBasicStepContainer().style.display).toBe('none');
  });
});

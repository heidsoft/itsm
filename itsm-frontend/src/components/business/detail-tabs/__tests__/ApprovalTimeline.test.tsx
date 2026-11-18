/**
 * 审批意见必填口径回归：同意选填、拒绝必填。
 *
 * 背景：审批中心同意意见选填，而详情面板曾对同意也强制必填，两处口径打架。
 * 后端只在拒绝时（且节点配置要求）校验意见，同意从不强制。这里锁住详情面板
 * 与后端/审批中心一致：同意空意见可提交，拒绝空意见被拦下。
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { ApprovalTimeline } from '../ApprovalTimeline';
import type { ApprovalStep } from '../types';

const mockT = jest.fn((key: string) => key);

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: mockT, language: 'zh-CN' }),
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn().mockResolvedValue({ items: [] }) },
}));

const pendingStep: ApprovalStep = { id: 1, level: 1, status: 'pending' };

function renderPanel(overrides: Partial<React.ComponentProps<typeof ApprovalTimeline>> = {}) {
  const onApprove = jest.fn().mockResolvedValue(undefined);
  const onReject = jest.fn().mockResolvedValue(undefined);
  render(
    <App>
      <ApprovalTimeline
        approvals={[pendingStep]}
        canApprove
        showApprovalActions
        onApprove={onApprove}
        onReject={onReject}
        {...overrides}
      />
    </App>
  );
  return { onApprove, onReject };
}

describe('ApprovalTimeline comment requirement', () => {
  beforeEach(() => jest.clearAllMocks());

  it('allows approve with an empty comment', async () => {
    const { onApprove } = renderPanel();
    fireEvent.click(screen.getByText('detailTabs.approve'));
    fireEvent.click(await screen.findByText('common.submit'));
    await waitFor(() => expect(onApprove).toHaveBeenCalledTimes(1));
    expect(onApprove).toHaveBeenCalledWith(expect.objectContaining({ comment: '' }));
  });

  it('blocks reject when comment is empty', async () => {
    const { onReject } = renderPanel();
    fireEvent.click(screen.getByText('detailTabs.reject'));
    fireEvent.click(await screen.findByText('common.submit'));
    await waitFor(() => expect(screen.getByText('detailTabs.approvalCommentRequired')).toBeInTheDocument());
    expect(onReject).not.toHaveBeenCalled();
  });

  it('allows reject once a comment is provided', async () => {
    const { onReject } = renderPanel();
    fireEvent.click(screen.getByText('detailTabs.reject'));
    const textarea = await screen.findByPlaceholderText('detailTabs.approvalCommentPlaceholder');
    fireEvent.change(textarea, { target: { value: '风险过高' } });
    fireEvent.click(screen.getByText('common.submit'));
    await waitFor(() => expect(onReject).toHaveBeenCalledTimes(1));
    expect(onReject).toHaveBeenCalledWith(expect.objectContaining({ comment: '风险过高' }));
  });
});
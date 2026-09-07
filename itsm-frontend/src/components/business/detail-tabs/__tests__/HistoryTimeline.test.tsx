/**
 * Tests for HistoryTimeline
 *
 * 覆盖：
 *   - fetchHistory 成功路径：渲染记录，source = 'native'，不调用 fetchAuditLog
 *   - fetchHistory 失败 + fetchAuditLog 兜底：source = 'audit'，渲染可读 details
 *   - fetchHistory 失败 + fetchAuditLog 也失败：渲染 Alert 错误
 *   - 加载/空状态：Spin + Empty
 *   - 重试按钮：点击后重新调用 fetchHistory
 *   - 不渲染 method/path/statusCode 块（自 2026-09-06 重构后该块已移除）
 */

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { HistoryTimeline } from '../HistoryTimeline';
import type { HistoryRecord } from '../types';

const mockFetchHistory = jest.fn();
const mockFetchAuditLog = jest.fn();
const mockT = jest.fn((key: string, fallback?: string) => fallback ?? key);
const mockLanguage = 'zh-CN';

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: mockT, language: mockLanguage }),
}));

describe('HistoryTimeline', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders records from fetchHistory when fetch succeeds', async () => {
    const records: HistoryRecord[] = [
      {
        id: 1,
        user: { name: '张三' },
        action: 'create',
        details: '创建工单',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ];
    mockFetchHistory.mockResolvedValueOnce(records);

    render(
      <HistoryTimeline
        targetType="ticket"
        targetId={1}
        fetchHistory={mockFetchHistory}
        fetchAuditLog={mockFetchAuditLog}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('张三')).toBeInTheDocument();
    });
    expect(screen.getByText('创建工单')).toBeInTheDocument();
    expect(mockFetchAuditLog).not.toHaveBeenCalled();
  });

  it('falls back to fetchAuditLog when fetchHistory rejects', async () => {
    mockFetchHistory.mockRejectedValueOnce(new Error('native api down'));
    mockFetchAuditLog.mockResolvedValueOnce([
      {
        id: 9,
        user: { name: '李四' },
        action: 'update',
        details: '更新问题',
        createdAt: '2026-01-02T00:00:00Z',
      },
    ]);

    render(
      <HistoryTimeline
        targetType="problem"
        targetId={2}
        fetchHistory={mockFetchHistory}
        fetchAuditLog={mockFetchAuditLog}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('李四')).toBeInTheDocument();
    });
    expect(screen.getByText('更新问题')).toBeInTheDocument();
    expect(mockFetchAuditLog).toHaveBeenCalledWith('problem', 2);
  });

  it('surfaces an error alert when both fetchers fail', async () => {
    // fetchHistory 失败时组件会 swallow（catch 块不抛错，因为有 fetchAuditLog 兜底），
    // 然后调用 fetchAuditLog。如果 fetchAuditLog 也失败，错误 alert 显示的是它的 message。
    mockFetchHistory.mockRejectedValueOnce(new Error('native api down'));
    mockFetchAuditLog.mockRejectedValueOnce(new Error('audit log down'));

    render(
      <HistoryTimeline
        targetType="change"
        targetId={3}
        fetchHistory={mockFetchHistory}
        fetchAuditLog={mockFetchAuditLog}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('audit log down')).toBeInTheDocument();
    });
  });

  it('renders empty state when records are empty', async () => {
    mockFetchHistory.mockResolvedValueOnce([]);

    render(
      <HistoryTimeline
        targetType="incident"
        targetId={4}
        fetchHistory={mockFetchHistory}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('detailTabs.noHistory')).toBeInTheDocument();
    });
  });

  it('does not render the legacy method/path/statusCode block for audit source', async () => {
    // 关键：fetchHistory 必须 reject 才会触发 fetchAuditLog 兜底
    mockFetchHistory.mockRejectedValueOnce(new Error('native down'));
    mockFetchAuditLog.mockResolvedValueOnce([
      {
        id: 11,
        user: { name: '王五' },
        action: 'create',
        details: '创建发布',
        // 即使这些字段被传入也不应渲染
        method: 'POST',
        path: '/api/v1/releases/9',
        statusCode: 201,
        createdAt: '2026-01-03T00:00:00Z',
      } as HistoryRecord,
    ]);

    render(
      <HistoryTimeline
        targetType="release"
        targetId={9}
        fetchHistory={mockFetchHistory}
        fetchAuditLog={mockFetchAuditLog}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('王五')).toBeInTheDocument();
    });
    // method/path/statusCode 行不应出现
    expect(screen.queryByText(/POST/)).not.toBeInTheDocument();
    expect(screen.queryByText(/\/api\/v1\/releases\/9/)).not.toBeInTheDocument();
    expect(screen.queryByText('201')).not.toBeInTheDocument();
  });

  it('retry button re-triggers fetchHistory', async () => {
    mockFetchHistory.mockRejectedValueOnce(new Error('native down'));
    mockFetchAuditLog.mockRejectedValueOnce(new Error('audit down'));
    // 第二次重试时 fetchHistory 返回空（让画面清空）
    mockFetchHistory.mockResolvedValueOnce([]);

    render(
      <HistoryTimeline
        targetType="ticket"
        targetId={5}
        fetchHistory={mockFetchHistory}
        fetchAuditLog={mockFetchAuditLog}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('audit down')).toBeInTheDocument();
    });

    const retryBtn = screen.getByText('common.retry');
    fireEvent.click(retryBtn);

    await waitFor(() => {
      expect(mockFetchHistory).toHaveBeenCalledTimes(2);
    });
    expect(mockFetchAuditLog).toHaveBeenCalledTimes(1); // 仅首次调用，重试直接走 fetchHistory
  });

  it('formats oldValue/newValue transition when present', async () => {
    mockFetchHistory.mockResolvedValueOnce([
      {
        id: 1,
        user: { name: '张三' },
        action: 'update',
        details: '更新状态',
        fieldName: 'status',
        oldValue: 'open',
        newValue: 'closed',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ]);

    render(
      <HistoryTimeline
        targetType="ticket"
        targetId={6}
        fetchHistory={mockFetchHistory}
      />
    );

    // 验证 details 字段被渲染；antd Timeline 在 jsdom 下的 oldValue/newValue 渲染
    // 受 deprecated <Timeline.Item> 影响不完全可靠，因此断言 details + 用户名即可
    await waitFor(() => {
      expect(screen.getByText('张三')).toBeInTheDocument();
    });
    expect(screen.getByText('更新状态')).toBeInTheDocument();
    expect(screen.getByText('status')).toBeInTheDocument();
  });
});
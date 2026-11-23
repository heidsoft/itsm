'use client';

/**
 * 跨业务域审批待办概览。
 *
 * 只做「概览 + 直达」：从三个审批域各自的权威读接口拉真实待办，统一成一张表，
 * 点标题直达原审批页完成操作。审批逻辑不在本组件重复实现，避免双数据源与
 * 双份操作逻辑（原页面的防重入/证据链/策略四态等守卫保持唯一）。
 *
 * 失败态与空态严格分离：拉取失败显示错误 + 重试，绝不渲染成「暂无待办」。
 */

import React, { useCallback, useEffect, useState } from 'react';
import { Table, Button, Tag, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import Link from 'next/link';
import dayjs from 'dayjs';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { aiGetToolApprovals, type ToolApproval } from '@/lib/api/ai-api';
import { ServiceRequestStatus } from '@/types/service-catalog';
import { KnowledgeReviewStatus } from '@/types/knowledge-base';

export type ApprovalDomain = 'service' | 'knowledge' | 'aiTool';

export const APPROVAL_DOMAIN_LINKS: Record<ApprovalDomain, string> = {
  service: '/service-catalog/approvals',
  knowledge: '/knowledge/reviews',
  aiTool: '/ai/approval',
};

export interface OverviewItem {
  key: string;
  title: string;
  subtitle: string;
  time: string;
  status: string;
  statusColor: string;
}

export interface DomainState {
  items: OverviewItem[];
  loading: boolean;
  error: boolean;
}

const EMPTY_STATE: DomainState = { items: [], loading: true, error: false };

const formatTime = (value?: string): string => {
  if (!value) return '';
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value;
};

// 纯映射函数导出以便单测锁定字段契约。状态只展示读接口返回的真实语义，
// 不臆造审批结论——本页是概览，不是第二个审批控制台。
export function mapServiceItems(rows: Array<Record<string, unknown>>): OverviewItem[] {
  return rows.map((row, index) => ({
    key: `service-${row.id ?? index}`,
    title: String(row.serviceName ?? row.title ?? `#${row.id ?? index}`),
    subtitle: String(row.requesterName ?? row.requester ?? ''),
    time: formatTime(row.createdAt as string | undefined),
    status: 'pending',
    statusColor: 'orange',
  }));
}

export function mapKnowledgeItems(rows: Array<Record<string, unknown>>): OverviewItem[] {
  return rows
    .filter((row) => row.reviewStatus === KnowledgeReviewStatus.UNDER_REVIEW)
    .map((row, index) => ({
      key: `knowledge-${row.id ?? index}`,
      title: String(row.title ?? `#${row.id ?? index}`),
      subtitle: String(row.authorName ?? row.author ?? ''),
      time: formatTime((row.submittedAt ?? row.createdAt) as string | undefined),
      status: 'under_review',
      statusColor: 'orange',
    }));
}

export function mapAiToolItems(rows: ToolApproval[]): OverviewItem[] {
  return rows.map((row) => ({
    key: `aiTool-${row.id}`,
    title: row.toolName,
    subtitle: row.riskLevel ? `risk: ${row.riskLevel}` : '',
    time: formatTime(row.createdAt),
    status: row.approvalState || row.status || 'pending',
    statusColor: 'orange',
  }));
}

/** 并行拉三个审批域的待办，作为聚合页的单一数据源。 */
export function useApprovalOverviews(enabledDomains: ApprovalDomain[]) {
  const [states, setStates] = useState<Record<ApprovalDomain, DomainState>>({
    service: EMPTY_STATE,
    knowledge: EMPTY_STATE,
    aiTool: EMPTY_STATE,
  });

  const loadDomain = useCallback(
    async (domain: ApprovalDomain): Promise<DomainState> => {
      try {
        if (domain === 'service') {
          const res = await ServiceCatalogApi.getServiceRequests({
            status: ServiceRequestStatus.PENDING_APPROVAL,
            page: 1,
            pageSize: 100,
          });
          return { items: mapServiceItems(res.items as unknown as Array<Record<string, unknown>>), loading: false, error: false };
        }
        if (domain === 'knowledge') {
          const res = await KnowledgeBaseApi.getArticles({ page: 1, pageSize: 100 });
          return {
            items: mapKnowledgeItems(res.items as unknown as Array<Record<string, unknown>>),
            loading: false,
            error: false,
          };
        }
        const res = await aiGetToolApprovals('pending');
        return { items: mapAiToolItems(res.items ?? []), loading: false, error: false };
      } catch {
        return { items: [], loading: false, error: true };
      }
    },
    []
  );

  const refresh = useCallback(async () => {
    // 先置 loading 再并行拉取；失败由各域独立 error 表达，不互相污染。
    setStates((prev) => {
      const next = { ...prev };
      enabledDomains.forEach((domain) => {
        next[domain] = { ...prev[domain], loading: true, error: false };
      });
      return next;
    });
    const results = await Promise.all(enabledDomains.map((domain) => loadDomain(domain)));
    setStates((prev) => {
      const next = { ...prev };
      enabledDomains.forEach((domain, index) => {
        next[domain] = results[index];
      });
      return next;
    });
  }, [enabledDomains, loadDomain]);

  const enabledKey = enabledDomains.join(',');
  useEffect(() => {
    void refresh();
    // enabledKey 是稳定字符串键，替代数组引用做依赖，避免父级每次 render 重拉。
  }, [enabledKey]);

  return { states, refresh };
}

export interface ApprovalOverviewTableProps {
  domain: ApprovalDomain;
  state: DomainState;
  onRetry: () => void;
  statusLabel: (status: string) => string;
  labels: {
    title: string;
    subtitle: string;
    time: string;
    status: string;
    actions: string;
    goHandle: string;
    empty: string;
    loadFailed: string;
    retry: string;
  };
}

export function ApprovalOverviewTable({ domain, state, onRetry, statusLabel, labels }: ApprovalOverviewTableProps) {
  const columns: ColumnsType<OverviewItem> = [
    {
      title: labels.title,
      dataIndex: 'title',
      render: (text: string, record) => (
        <Link href={APPROVAL_DOMAIN_LINKS[domain]}>{text}</Link>
      ),
    },
    { title: labels.subtitle, dataIndex: 'subtitle', render: (v: string) => v || '—' },
    { title: labels.time, dataIndex: 'time', render: (v: string) => v || '—' },
    {
      title: labels.status,
      dataIndex: 'status',
      render: (status: string, record) => <Tag color={record.statusColor}>{statusLabel(status)}</Tag>,
    },
    {
      title: labels.actions,
      key: 'actions',
      width: 120,
      render: () => (
        <Space>
          <Link href={APPROVAL_DOMAIN_LINKS[domain]}>{labels.goHandle}</Link>
        </Space>
      ),
    },
  ];

  if (state.error) {
    return (
      <div role="alert" className="py-10 text-center">
        <p>{labels.loadFailed}</p>
        <Button onClick={onRetry}>{labels.retry}</Button>
      </div>
    );
  }

  return (
    <Table
      rowKey="key"
      dataSource={state.items}
      columns={columns}
      loading={state.loading}
      pagination={{ pageSize: 10, showSizeChanger: false }}
      locale={{
        emptyText: <div className="py-8 text-center text-gray-500">{labels.empty}</div>,
      }}
    />
  );
}

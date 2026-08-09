'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { Dayjs } from 'dayjs';
import { App, Card, Divider, Table } from 'antd';
import type { TablePaginationConfig, TableProps } from 'antd/es/table';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';

import { TicketApi } from '@/lib/api/ticket-api';
import type { Ticket } from '@/lib/api/types';
import { useTickets } from '@/lib/hooks/useTickets';
import type { TicketQueryFilters } from '@/lib/hooks/useTickets';
import { useRowSelection } from '@/lib/hooks/useRowSelection';
import { useTableKeyboardNav } from '@/lib/hooks/useTableKeyboardNav';
import { useDebounce } from '@/lib/component-utils';

import { buildTicketListColumns } from './TicketListColumns';
import { TicketListToolbar } from './TicketListToolbar';
import { TicketListFilters, type TicketFilterValues } from './TicketListFilters';
import { TicketDeleteModal } from './TicketDeleteModal';
import TicketBatchOperations from './TicketBatchOperations';

interface TicketListProps {
  readonly embedded?: boolean;
  readonly showHeader?: boolean;
  readonly pageSize?: number;
  readonly filters?: Partial<TicketQueryFilters>;
  readonly onTicketSelect?: (ticket: Ticket) => void;
  readonly advancedFilters?: Partial<TicketQueryFilters>;
}

const DEFAULT_FILTER_VALUES: TicketFilterValues = {
  status: undefined,
  priority: undefined,
  type: undefined,
  createdRange: null,
};

/**
 * Container for the tickets list. Composes the presentational pieces
 * (Toolbar, Filters, Columns, DeleteModal) and owns only the cross-cutting
 * state that genuinely belongs at the container level: search input,
 * filter-panel visibility, delete-confirmation target, active row index.
 *
 * Server state (tickets, pagination, domain filters) is delegated to
 * `useTickets`; row-selection state lives in `useRowSelection`; keyboard
 * navigation in `useTableKeyboardNav`. The remaining local state is the
 * minimum needed to wire those pieces together.
 */
const TicketList: React.FC<TicketListProps> = ({
  embedded = false,
  showHeader = true,
  onTicketSelect,
  advancedFilters,
}) => {
  const router = useRouter();
  const { message, modal } = App.useApp();
  const {
    tickets,
    loading,
    pagination,
    filters,
    fetchTickets,
    updateFilters,
    updatePagination,
    deleteTicket,
    batchDeleteTickets,
  } = useTickets();

  const selection = useRowSelection<number>();
  const [searchValue, setSearchValue] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [filterValues, setFilterValues] = useState<TicketFilterValues>(DEFAULT_FILTER_VALUES);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [ticketToDelete, setTicketToDelete] = useState<Ticket | null>(null);
  // Keyboard-driven active row. `hoveredIndex` is independent so mouse hover
  // never tramples the keyboard cursor — they were racing in v1.
  const [activeRowIndex, setActiveRowIndex] = useState(0);
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  // Debounce search input to avoid hammering the API on every keystroke.
  const debouncedSearchValue = useDebounce(searchValue, 300);

  useEffect(() => {
    updateFilters({ keyword: debouncedSearchValue || undefined });
  }, [debouncedSearchValue, updateFilters]);

  // Sync external advanced filters (e.g. from a dashboard deep-link) into the
  // ticket store. Using JSON.stringify as the dep key avoids the infinite loop
  // you would get from referencing the object directly.
  const advancedFiltersKey = useMemo(
    () => JSON.stringify(advancedFilters ?? {}),
    [advancedFilters]
  );
  useEffect(() => {
    if (advancedFilters === undefined) return;
    updateFilters(advancedFilters);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [advancedFiltersKey]);

  const openTicket = useCallback(
    (ticket: Ticket) => {
      if (onTicketSelect) {
        onTicketSelect(ticket);
      } else {
        router.push(`/tickets/${ticket.id}`);
      }
    },
    [onTicketSelect, router]
  );

  const editTicket = useCallback(
    (ticket: Ticket) => router.push(`/tickets/${ticket.id}?mode=edit`),
    [router]
  );

  // `filters` is read through a ref inside Modal callbacks so a confirmation
  // opened before a filter change still refreshes with the user's current view.
  const filtersRef = useRef(filters);
  useEffect(() => {
    filtersRef.current = filters;
  }, [filters]);

  // H1: Clamp the keyboard cursor to the visible row range whenever the list
  // shrinks (filter / page change / deletion). Without this, pressing `o` after
  // a filter change could call `openTicket(undefined)`.
  // Depend on `tickets.length` (not `tickets`) so we only run when the list
  // size actually changes, and guard on prev >= length so we don't push the
  // cursor back on every render.
  const ticketsLength = tickets.length;
  useEffect(() => {
    setActiveRowIndex(prev => (prev >= ticketsLength ? Math.max(0, ticketsLength - 1) : prev));
  }, [ticketsLength]);

  const closeTicket = useCallback(
    (ticket: Ticket) => {
      modal.confirm({
        title: `关闭工单 ${ticket.ticketNumber}？`,
        content: '关闭后工单将进入终态，请确认处理结果已经记录。',
        okText: '确认关闭',
        cancelText: '取消',
        okButtonProps: { danger: true },
        onOk: async () => {
          try {
            await TicketApi.closeTicket(ticket.id);
            message.success('工单已关闭');
            await fetchTickets(filtersRef.current);
          } catch (error) {
            message.error(error instanceof Error ? error.message : '关闭工单失败');
            // antd v5's `modal.confirm` resolves with `false` (not rejection) on
            // throw inside onOk; the dialog closes either way. We swallow here
            // so the confirm flow returns cleanly. The error toast above is the
            // user-visible signal.
          }
        },
      });
    },
    [fetchTickets, modal, message]
  );

  const handleRefresh = useCallback(() => {
    void fetchTickets(filters);
  }, [fetchTickets, filters]);

  const handleBatchDelete = useCallback(async () => {
    if (selection.count === 0) {
      message.warning('请选择要删除的工单');
      return;
    }
    // batchDeleteTickets now uses Promise.allSettled internally and toasts
    // every failure path itself, so this call only throws on the post-delete
    // refresh — and even then, partial success has already been reported.
    const ids = selection.toArray();
    const result = await batchDeleteTickets(ids);
    // Successful deletes are already gone; keep failed ids selected so the
    // operator can retry them. (H4: was previously wiping everything.)
    if (result.failedIds.length > 0) {
      selection.setMany(result.failedIds);
    } else {
      selection.clear();
    }
  }, [selection, batchDeleteTickets, message]);

  const handleExport = useCallback(async () => {
    try {
      const blob = await TicketApi.exportTickets({ format: 'excel', filters });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tickets_${dayjs().format('YYYY-MM-DD')}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success('导出成功');
    } catch {
      message.error('导出失败');
    }
  }, [filters, message]);

  const handleFiltersChange = useCallback(
    (next: TicketFilterValues) => {
      setFilterValues(next);
      const [start, end] = next.createdRange ?? [];
      updateFilters({
        status: next.status,
        priority: next.priority,
        type: next.type,
        dateRange:
          start && end ? [start.format('YYYY-MM-DD'), end.format('YYYY-MM-DD')] : undefined,
      });
    },
    [updateFilters]
  );

  const handleClearFilters = useCallback(() => {
    setFilterValues(DEFAULT_FILTER_VALUES);
    setSearchValue('');
    updateFilters({});
  }, [updateFilters]);

  const handleConfirmDelete = useCallback(
    async (ticket: Ticket) => {
      try {
        await deleteTicket(ticket.id);
        message.success('删除成功');
        setDeleteModalVisible(false);
        setTicketToDelete(null);
      } catch {
        message.error('删除失败');
      }
    },
    [deleteTicket, message]
  );

  const columns = useMemo(
    () => buildTicketListColumns({ onOpen: openTicket, onEdit: editTicket, onClose: closeTicket }),
    [openTicket, editTicket, closeTicket]
  );

  const rowSelection: TableProps<Ticket>['rowSelection'] = useMemo(
    () => ({
      selectedRowKeys: selection.toArray(),
      onChange: (_keys, rows) => selection.setMany(rows.map(r => r.id)),
    }),
    [selection]
  );

  const handleTableChange: TableProps<Ticket>['onChange'] = useCallback(
    (next: TablePaginationConfig) => {
      updatePagination(next.current ?? 1, next.pageSize ?? 20);
    },
    [updatePagination]
  );

  useTableKeyboardNav<Ticket>({
    rows: tickets,
    activeIndex: activeRowIndex,
    onActivate: openTicket,
    onNext: () => setActiveRowIndex(i => Math.min(i + 1, Math.max(tickets.length - 1, 0))),
    onPrev: () => setActiveRowIndex(i => Math.max(i - 1, 0)),
    enabled: !embedded,
  });

  return (
    <div className='ticket-list space-y-4'>
      {showHeader && (
        <>
          <TicketListToolbar
            searchValue={searchValue}
            showFilters={showFilters}
            selectedCount={selection.count}
            loading={loading}
            onSearchChange={setSearchValue}
            onSearchSubmit={setSearchValue}
            onToggleFilters={() => setShowFilters(v => !v)}
            onRefresh={handleRefresh}
            onBatchDelete={handleBatchDelete}
            onExport={handleExport}
            onCreate={() => router.push('/tickets/create')}
          />
          {showFilters && (
            <Card className='rounded-lg shadow-sm'>
              <Divider style={{ marginTop: 0 }} />
              <TicketListFilters
                values={filterValues}
                onChange={handleFiltersChange}
                onClear={handleClearFilters}
              />
            </Card>
          )}
        </>
      )}

      {!embedded && selection.count > 0 && (
        <TicketBatchOperations
          selectedTickets={tickets.filter(t => selection.isSelected(t.id))}
          onOperationComplete={() => void fetchTickets(filters)}
          onSelectionClear={selection.clear}
        />
      )}

      <Card className='rounded-lg shadow-sm'>
        <div className='mb-3 flex justify-end text-xs text-gray-500' aria-label='键盘快捷键'>
          快捷键：<kbd className='mx-1 rounded border bg-gray-50 px-1.5'>j</kbd>/
          <kbd className='mx-1 rounded border bg-gray-50 px-1.5'>k</kbd> 导航，
          <kbd className='mx-1 rounded border bg-gray-50 px-1.5'>o</kbd> 打开
        </div>
        <Table<Ticket>
          columns={columns}
          dataSource={tickets}
          rowKey='id'
          rowSelection={embedded ? undefined : rowSelection}
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
          onChange={handleTableChange}
          scroll={{ x: 1200 }}
          size='middle'
          onRow={(_, index) => ({
            onMouseEnter: () => setHoveredIndex(index ?? null),
            onMouseLeave: () => setHoveredIndex(null),
          })}
          rowClassName={(_, index) =>
            index === activeRowIndex || index === hoveredIndex ? 'bg-blue-50/60' : ''
          }
          getPopupContainer={node => node.parentElement || document.body}
        />
      </Card>

      <TicketDeleteModal
        open={deleteModalVisible}
        ticket={ticketToDelete}
        onConfirm={handleConfirmDelete}
        onCancel={() => {
          setDeleteModalVisible(false);
          setTicketToDelete(null);
        }}
      />
    </div>
  );
};

export default TicketList;

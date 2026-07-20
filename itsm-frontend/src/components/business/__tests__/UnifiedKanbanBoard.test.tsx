import React from 'react';
import { fireEvent, render, screen, waitFor } from '@/lib/test-utils';
import { UnifiedKanbanBoard } from '../UnifiedKanbanBoard';

interface Item {
  id: number;
  title: string;
  status: string;
}

const items: Item[] = [{ id: 1, title: '网络故障', status: 'open' }];
const columns = [
  { key: 'open', title: '待处理', color: '#1677ff' },
  { key: 'resolved', title: '已解决', color: '#52c41a' },
];

function renderBoard(overrides: Partial<React.ComponentProps<typeof UnifiedKanbanBoard<Item>>> = {}) {
  const props: React.ComponentProps<typeof UnifiedKanbanBoard<Item>> = {
    items,
    columnConfigs: columns,
    getItemId: item => item.id,
    getItemStatus: item => item.status,
    getItemTitle: item => item.title,
    showToolbar: false,
    ...overrides,
  };
  return render(<UnifiedKanbanBoard<Item> {...props} />);
}

describe('UnifiedKanbanBoard', () => {
  it('fills the available width while preserving a usable minimum column width', () => {
    renderBoard();

    expect(screen.getByTestId('kanban-grid')).toHaveStyle({
      minWidth: '616px',
      gridTemplateColumns: 'repeat(2, minmax(300px, 1fr))',
    });
    expect(screen.getByTestId('kanban-column-open')).not.toHaveStyle({ width: '280px' });
  });

  it('does not advertise draggable behavior without a status-change handler', () => {
    renderBoard();
    expect(screen.getByTestId('kanban-item-1')).toHaveAttribute('draggable', 'false');
  });

  it('submits a controlled status transition after drop', async () => {
    const onItemStatusChange = jest.fn().mockResolvedValue(undefined);
    renderBoard({ onItemStatusChange });

    fireEvent.dragStart(screen.getByTestId('kanban-item-1'));
    fireEvent.dragOver(screen.getByTestId('kanban-column-resolved'));
    fireEvent.drop(screen.getByTestId('kanban-column-resolved'));

    await waitFor(() => expect(onItemStatusChange).toHaveBeenCalledWith(1, 'resolved'));
    expect(onItemStatusChange).toHaveBeenCalledTimes(1);
  });

  it('blocks transitions rejected by the permission policy', async () => {
    const onItemStatusChange = jest.fn().mockResolvedValue(undefined);
    renderBoard({ onItemStatusChange, canItemStatusChange: () => false });

    fireEvent.dragStart(screen.getByTestId('kanban-item-1'));
    fireEvent.drop(screen.getByTestId('kanban-column-resolved'));

    await waitFor(() => expect(onItemStatusChange).not.toHaveBeenCalled());
  });

  it('keeps the controlled item in its original column when persistence fails', async () => {
    const onItemStatusChange = jest.fn().mockRejectedValue(new Error('状态流转失败'));
    renderBoard({ onItemStatusChange });

    fireEvent.dragStart(screen.getByTestId('kanban-item-1'));
    fireEvent.drop(screen.getByTestId('kanban-column-resolved'));

    await waitFor(() => expect(onItemStatusChange).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId('kanban-column-open')).toHaveTextContent('网络故障');
    expect(screen.getByTestId('kanban-column-resolved')).not.toHaveTextContent('网络故障');
  });
});

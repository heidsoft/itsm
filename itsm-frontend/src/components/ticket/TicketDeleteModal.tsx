'use client';

import { Modal } from 'antd';
import { AlertCircle } from 'lucide-react';

import type { Ticket } from '@/lib/api/types';

interface TicketDeleteModalProps {
  readonly open: boolean;
  readonly ticket: Ticket | null;
  readonly loading?: boolean;
  readonly onConfirm: (ticket: Ticket) => void | Promise<void>;
  readonly onCancel: () => void;
}

/**
 * Controlled delete-confirmation dialog for a single ticket.
 *
 * The parent owns `open`/`ticket` state and the actual deletion logic; this
 * component only handles the visual + accessibility surface. Title and message
 * are derived from the ticket so a missing ticketNumber never crashes the
 * render path.
 */
export function TicketDeleteModal({
  open,
  ticket,
  loading = false,
  onConfirm,
  onCancel,
}: TicketDeleteModalProps) {
  const handleOk = () => {
    if (ticket) {
      void onConfirm(ticket);
    }
  };

  return (
    <Modal
      title='确认删除'
      open={open}
      onOk={handleOk}
      onCancel={onCancel}
      okText='确认'
      cancelText='取消'
      okButtonProps={{ danger: true }}
      confirmLoading={loading}
    >
      <p>
        <AlertCircle style={{ color: '#ff4d4f', marginRight: 8 }} />
        确定要删除工单 <strong>{ticket?.ticketNumber || '-'}</strong> 吗？此操作不可撤销。
      </p>
    </Modal>
  );
}

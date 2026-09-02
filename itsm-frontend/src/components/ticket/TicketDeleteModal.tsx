'use client';

import { Modal } from 'antd';
import { AlertCircle } from 'lucide-react';

import type { Ticket } from '@/lib/api/types';
import { useI18n } from '@/lib/i18n/useI18n';

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
  const { t } = useI18n();

  const handleOk = () => {
    if (ticket) {
      void onConfirm(ticket);
    }
  };

  return (
    <Modal
      title={t('tickets.confirmDelete')}
      open={open}
      onOk={handleOk}
      onCancel={onCancel}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true }}
      confirmLoading={loading}
    >
      <p>
        <AlertCircle style={{ color: '#ff4d4f', marginRight: 8 }} />
        {t('tickets.confirmDeleteContent', {
          ticketNumber: ticket?.ticketNumber || '-',
        })}
      </p>
    </Modal>
  );
}
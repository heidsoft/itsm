import React, { useCallback, useState } from 'react';
import { App } from 'antd';
import { ticketService } from '@/lib/services/ticket-service';
import { TicketModal } from './TicketModal';
import type { TicketModalProps, TicketFormValues } from './types';

type TicketModalContainerProps = Omit<TicketModalProps, 'loading' | 'onSubmit'> & {
  onSuccess?: () => void | Promise<void>;
};

/** Feature container that owns ticket persistence and success/error feedback. */
export function TicketModalContainer({
  editingTicket,
  onSuccess,
  ...viewProps
}: TicketModalContainerProps) {
  const { message } = App.useApp();
  const [submitting, setSubmitting] = useState(false);

  const submit = useCallback(
    async (values: TicketFormValues) => {
      setSubmitting(true);
      try {
        const { attachments: _attachments, estimatedTime: _estimatedTime, ...ticketValues } = values;
        if (editingTicket) {
          await ticketService.updateTicket(editingTicket.id, ticketValues);
        } else {
          await ticketService.createTicket(ticketValues);
        }
        message.success(editingTicket ? '工单更新成功' : '工单创建成功');
        await onSuccess?.();
      } catch (cause) {
        message.error(cause instanceof Error ? cause.message : '保存工单失败');
        throw cause;
      } finally {
        setSubmitting(false);
      }
    },
    [editingTicket, message, onSuccess]
  );

  return (
    <TicketModal
      {...viewProps}
      editingTicket={editingTicket}
      loading={submitting}
      onSubmit={submit}
    />
  );
}

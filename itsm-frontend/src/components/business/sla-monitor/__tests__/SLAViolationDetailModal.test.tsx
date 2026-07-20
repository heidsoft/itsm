import React from 'react';
import { fireEvent, render, screen } from '@/lib/test-utils';
import { SLAViolationDetailModal } from '../components/SLAViolationDetailModal';
import type { SLAViolation } from '../types';

const violation: SLAViolation = {
  id: 7,
  tenantId: 1,
  ticketId: 10,
  slaDefId: 2,
  violationType: 'response',
  severity: 'critical',
  status: 'open',
  expectedTime: '2026-01-01T08:00:00Z',
  actualTime: '2026-01-01T09:00:00Z',
  delayMinutes: 60,
  description: '响应超时',
  createdAt: '2026-01-01T09:00:00Z',
  updatedAt: '2026-01-01T09:00:00Z',
};

describe('SLAViolationDetailModal', () => {
  it('hides mutating actions without write permission', () => {
    render(
      <SLAViolationDetailModal
        violation={violation}
        visible
        onClose={jest.fn()}
        onResolve={jest.fn()}
        onAcknowledge={jest.fn()}
      />
    );

    expect(screen.queryByRole('button', { name: /确\s*认/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /解\s*决/ })).not.toBeInTheDocument();
  });

  it('emits controlled actions when permission is granted', () => {
    const onResolve = jest.fn();
    const onAcknowledge = jest.fn();
    render(
      <SLAViolationDetailModal
        violation={violation}
        visible
        canManage
        onClose={jest.fn()}
        onResolve={onResolve}
        onAcknowledge={onAcknowledge}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    fireEvent.click(screen.getByRole('button', { name: /解\s*决/ }));
    expect(onAcknowledge).toHaveBeenCalledWith(violation);
    expect(onResolve).toHaveBeenCalledWith(violation);
  });
});

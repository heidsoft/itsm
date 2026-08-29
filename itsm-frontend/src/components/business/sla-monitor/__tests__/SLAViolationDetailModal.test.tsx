import React from 'react';
import { fireEvent, render, screen } from '@/lib/test-utils';
import { SLAViolationDetailModal } from '../components/SLAViolationDetailModal';
import type { SLAViolation } from '../types';

const violation: SLAViolation = {
  id: 7,
  createdBy: 1,
  tenantId: 1,
  ticketId: 10,
  ticketNumber: 'TKT-0007',
  ticketTitle: '打印机故障',
  ticketPriority: 'high',
  slaName: '标准服务SLA',
  slaDefinitionId: 2,
  violationType: 'response',
  violationTime: '2026-01-01T09:00:00Z',
  severity: 'critical',
  isResolved: false,
  resolutionNotes: '',
  description: '响应超时',
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

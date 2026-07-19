import { fireEvent, render, screen } from '@/lib/test-utils';
import { AdvancedReportingView } from '../AdvancedReportingView';

describe('AdvancedReportingView', () => {
  it('shows a retry action when loading fails', () => {
    const onReload = jest.fn();
    render(
      <AdvancedReportingView
        reports={[]}
        loading={false}
        error="服务不可用"
        onReload={onReload}
      />
    );

    expect(screen.getByText('服务不可用')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    expect(onReload).toHaveBeenCalledTimes(1);
  });

  it('delegates navigation instead of executing an unsupported report action', () => {
    const onOpenReport = jest.fn();
    const report = { id: 'tickets', name: '工单报表', path: '/reports/tickets' };
    render(
      <AdvancedReportingView
        reports={[report]}
        loading={false}
        error={null}
        onReload={jest.fn()}
        onOpenReport={onOpenReport}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '查看' }));
    expect(onOpenReport).toHaveBeenCalledWith(report);
    expect(screen.queryByText('创建报表')).not.toBeInTheDocument();
  });
});

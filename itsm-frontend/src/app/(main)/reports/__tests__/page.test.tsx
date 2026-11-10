import { render, screen } from '@/lib/test-utils';
import ReportsPage from '../page';

jest.mock('@/components/business/AdvancedReporting', () => ({
  __esModule: true,
  default: () => <div>supported-report-directory</div>,
}));

describe('ReportsPage', () => {
  it('renders only the supported read-only report directory', () => {
    render(<ReportsPage />);

    expect(screen.getByText('supported-report-directory')).toBeInTheDocument();
    expect(screen.queryByText("高级Analysis")).not.toBeInTheDocument();
    expect(screen.queryByText("CustomAnalysis")).not.toBeInTheDocument();
    expect(screen.queryByText("SLA real-time monitoring")).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Export/ })).not.toBeInTheDocument();
  });
});

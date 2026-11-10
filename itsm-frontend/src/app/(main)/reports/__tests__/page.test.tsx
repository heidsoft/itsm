import { render, screen } from '@/lib/test-utils';
import ReportsPage from '../page';

jest.mock('@/components/business/AdvancedReporting', () => ({
  __esModule: true,
  default: () => <div>supported-report-directory</div>,
}));

jest.mock('@/components/business/reports/ServiceQualityOverview', () => ({
  __esModule: true,
  default: () => <div>service-quality-overview</div>,
}));

describe('ReportsPage', () => {
  it('renders the manager service-quality overview above the report directory', () => {
    render(<ReportsPage />);

    expect(screen.getByText('service-quality-overview')).toBeInTheDocument();
    expect(screen.getByText('supported-report-directory')).toBeInTheDocument();
    expect(screen.queryByText("高级Analysis")).not.toBeInTheDocument();
    expect(screen.queryByText("CustomAnalysis")).not.toBeInTheDocument();
    expect(screen.queryByText("SLA real-time monitoring")).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Export/ })).not.toBeInTheDocument();
  });
});

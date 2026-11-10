import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { StandardChangeApi } from '@/lib/api/standard-change-api';
import StandardChangesPage from '../page';

jest.mock('next/navigation', () => ({ useRouter: () => ({ push: jest.fn() }) }));
jest.mock('@/lib/api/standard-change-api', () => ({
  StandardChangeApi: {
    getTemplates: jest.fn(), getCategories: jest.fn(), updateTemplate: jest.fn(),
  },
}));

it('displays legacy categories in English and preserves their value when editing', async () => {
  const api = jest.mocked(StandardChangeApi);
  api.getTemplates.mockResolvedValue({
    items: [{
      id: 7, title: 'Customer template', category: '服务器', riskLevel: 'low',
      impactScope: 'single_service', implementationPlan: 'Apply the patch',
      rollbackPlan: 'Restore backup', expectedDuration: 30,
    }], total: 1,
  } as never);
  api.getCategories.mockResolvedValue({ categories: ['服务器'] });
  api.updateTemplate.mockResolvedValue({} as never);

  render(<StandardChangesPage />);
  expect(await screen.findByText('Servers')).toBeInTheDocument();
  expect(screen.queryByText('服务器')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
  const dialog = await screen.findByRole('dialog');
  expect(within(dialog).getByText('Servers')).toBeInTheDocument();
  fireEvent.click(within(dialog).getByRole('button', { name: 'Update' }));
  await waitFor(() => expect(api.updateTemplate).toHaveBeenCalledWith(
    7, expect.objectContaining({ category: '服务器' })
  ));
}, 30000);

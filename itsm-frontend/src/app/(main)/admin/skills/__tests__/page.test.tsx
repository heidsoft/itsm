import { render, waitFor } from '@testing-library/react';
import { App } from 'antd';
import SkillsAdminPage from '../page';
import skillApi from '@/lib/api/skill-api';

jest.mock('@/lib/api/skill-api', () => ({
  __esModule: true,
  default: {
    list: jest.fn(),
    promote: jest.fn(),
    disable: jest.fn(),
  },
}));

jest.mock('@/app/components/PageContainer', () => ({
  PageContainer: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const mockedApi = skillApi as jest.Mocked<typeof skillApi>;

describe('SkillsAdminPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedApi.list.mockResolvedValue({
      items: [
        {
          code: 'triage_classifier',
          name: 'Triage Classifier',
          version: '1.0.0',
          title: 'Triage Classifier',
          provider: 'itsm',
          description: 'Auto-classifies incoming tickets',
          category: 'experimental',
          tags: ['ai', 'triage'],
          capabilities: ['classify'],
          requiredPermissions: ['ai:read'],
          isOfficial: false,
          checksum: 'abc',
          isBuiltin: false,
          status: 'active',
          manifest: { inputSchema: {}, outputSchema: {} },
          metrics: { totalCalls: 0, successRate: 0, avgLatencyMs: 0, errorCount: 0 },
        },
      ],
      total: 1,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    });
  });

  it('loads skills on mount', async () => {
    render(
      <App>
        <SkillsAdminPage />
      </App>,
    );

    await waitFor(() => {
      expect(mockedApi.list).toHaveBeenCalled();
    });
  });
});

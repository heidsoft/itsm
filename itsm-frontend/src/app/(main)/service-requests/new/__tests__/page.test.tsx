import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import NewServiceRequestPage from '../page';
import { ServiceCategory, ServiceStatus, type ServiceItem } from '@/types/service-catalog';

const mockPush = jest.fn();
let mockCanCreate = true;
const mockGetAllServices = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: () => mockCanCreate }),
}));

jest.mock('@/lib/api/service-catalog-api', () => ({
  ServiceCatalogApi: {
    getAllServices: (...args: unknown[]) => mockGetAllServices(...args),
  },
}));

const cloudHost: ServiceItem = {
  id: '8',
  name: '云主机',
  category: ServiceCategory.IT_SERVICE,
  status: ServiceStatus.PUBLISHED,
  shortDescription: '按需开通的云服务器',
  deliveryDays: 3,
  tags: [],
  createdBy: 1,
  createdByName: 'admin',
  createdAt: new Date('2026-10-01T00:00:00Z'),
  updatedAt: new Date('2026-10-01T00:00:00Z'),
};

describe('新建服务请求选服务页', () => {
  beforeEach(() => {
    mockCanCreate = true;
    mockPush.mockReset();
    mockGetAllServices.mockReset();
    mockGetAllServices.mockResolvedValue({ services: [cloudHost], total: 1, complete: true });
  });

  it('列出已发布服务，点申请进入该服务的申请表单', async () => {
    render(<NewServiceRequestPage />);

    const applyButton = await screen.findByRole('button', { name: '申请 云主机' });
    expect(mockGetAllServices).toHaveBeenCalledWith({ status: ServiceStatus.PUBLISHED }, 500);

    fireEvent.click(applyButton);
    expect(mockPush).toHaveBeenCalledWith('/service-catalog/request/8');
  });

  it('无创建权限时不请求目录，直接提示', async () => {
    mockCanCreate = false;
    render(<NewServiceRequestPage />);

    expect(screen.getByText('没有创建服务请求的权限')).toBeInTheDocument();
    await waitFor(() => expect(mockGetAllServices).not.toHaveBeenCalled());
  });
});

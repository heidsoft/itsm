import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { ServiceCategory, ServiceStatus, ServiceType } from '@/types/service-catalog';
import type { ServiceItem } from '@/types/service-catalog';
import ServiceCatalogManagement from '../page';

jest.mock('@/lib/api/service-catalog-api', () => ({
  ServiceCatalogApi: {
    getServices: jest.fn(),
    getAllServices: jest.fn(),
    getCatalogStats: jest.fn(),
    getService: jest.fn(),
    createService: jest.fn(),
    updateService: jest.fn(),
    deleteService: jest.fn(),
  },
}));

jest.mock('@/lib/api/cmdb-api', () => ({
  CMDBApi: { getCITypes: jest.fn(), getCloudServices: jest.fn() },
}));

const mockedApi = ServiceCatalogApi as jest.Mocked<typeof ServiceCatalogApi>;
const mockedCmdb = CMDBApi as jest.Mocked<typeof CMDBApi>;

// antd Table + Modal 在 jsdom 下的挂载远超默认 5s 断言预算。
jest.setTimeout(40000);

const catalog: ServiceItem = {
  id: '7',
  name: '对象存储开通',
  category: ServiceCategory.IT_SERVICE,
  status: ServiceStatus.PUBLISHED,
  shortDescription: '开通一个私有桶',
  fullDescription: '开通一个私有桶',
  serviceType: ServiceType.OSS,
  deliveryDays: 3,
  price: 12.5,
  unit: '月',
  requiresApproval: true,
  approvalLevel: 2,
  sortOrder: 7,
  slaResponseTime: 30,
  slaResolutionTime: 240,
  ciTypeId: 11,
  cloudServiceId: 21,
  // 后端 DTO 目前没有这三项投影，客户端按空值补齐——夹具照此写实，不伪造创建人。
  tags: [],
  createdBy: 0,
  createdByName: '',
  createdAt: new Date('2026-01-01T00:00:00Z'),
  updatedAt: new Date('2026-01-02T00:00:00Z'),
};

function mockLoaders() {
  mockedApi.getServices.mockResolvedValue({
    services: [catalog],
    total: 1,
    page: 1,
    pageSize: 20,
    totalPages: 1,
  });
  mockedApi.getCatalogStats.mockResolvedValue({
    totalServices: 1,
    publishedServices: 1,
    categories: { it_service: 1 },
  });
  mockedCmdb.getCITypes.mockResolvedValue([
    { id: 11, name: 'OSS Bucket' },
  ] as never);
  mockedCmdb.getCloudServices.mockResolvedValue({
    items: [{ id: 21, serviceName: 'OSS', resourceTypeName: 'Bucket' }],
  } as never);
  mockedApi.updateService.mockResolvedValue(catalog);
  mockedApi.createService.mockResolvedValue(catalog);
}

async function openEditModal() {
  render(
    <App>
      <ServiceCatalogManagement />
    </App>,
  );
  const rowAction = await screen.findByLabelText('编辑');
  fireEvent.click(rowAction);
  return screen.findByText('编辑服务目录');
}

// antd 给两字按钮插入空格，accessible name 是「更 新」而不是「更新」。
async function clickUpdateButton() {
  fireEvent.click(await screen.findByRole('button', { name: /更\s*新/ }));
}

// antd Select 的清除按钮始终在 DOM 里（默认透明），mousedown 即清空表单值。
function clearSelectByInputId(id: string) {
  const input = document.getElementById(id);
  if (!input) throw new Error(`找不到表单控件 ${id}`);
  const root = input.closest('.ant-select');
  const clear = root?.querySelector('.ant-select-clear');
  if (!clear) throw new Error(`${id} 没有清除按钮`);
  fireEvent.mouseDown(clear);
}

describe('ServiceCatalogManagement 编辑表单', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockLoaders();
  });

  it('不改任何字段直接提交也必须带上全部可写字段', async () => {
    await openEditModal();
    await clickUpdateButton();

    await waitFor(() => expect(mockedApi.updateService).toHaveBeenCalledTimes(1));
    const [id, payload] = mockedApi.updateService.mock.calls[0];
    expect(id).toBe('7');
    // 修复前：serviceType/价格/单位/交付天数/审批/排序/SLA/关联 全都被丢掉，
    // 页面显示旧值而库里已被改回默认。
    expect(payload).toEqual(
      expect.objectContaining({
        name: '对象存储开通',
        category: 'it_service',
        serviceType: ServiceType.OSS,
        deliveryDays: 3,
        price: 12.5,
        unit: '月',
        requiresApproval: true,
        approvalLevel: 2,
        sortOrder: 7,
        slaResponseTime: 30,
        slaResolutionTime: 240,
        ciTypeId: 11,
        cloudServiceId: 21,
        status: ServiceStatus.PUBLISHED,
      }),
    );
  });

  it('清空价格/单位/关联要显式提交 0 与空串，省略会让后端保持原值', async () => {
    await openEditModal();

    fireEvent.change(document.getElementById('price') as HTMLElement, {
      target: { value: '' },
    });
    clearSelectByInputId('unit');
    clearSelectByInputId('cloudServiceId');
    clearSelectByInputId('ciTypeId');

    await clickUpdateButton();
    await waitFor(() => expect(mockedApi.updateService).toHaveBeenCalledTimes(1));
    const [, payload] = mockedApi.updateService.mock.calls[0];

    expect(payload.price).toBe(0);
    expect(payload.unit).toBe('');
    expect(payload.ciTypeId).toBe(0);
    expect(payload.cloudServiceId).toBe(0);
  });

  it('后端业务校验的中文提示必须原样透出，不能只剩「保存失败」', async () => {
    mockedApi.updateService.mockRejectedValueOnce(
      new Error('关联云服务时必须选择 CI 类型'),
    );
    await openEditModal();
    await clickUpdateButton();

    expect(
      await screen.findByText('关联云服务时必须选择 CI 类型'),
    ).toBeInTheDocument();
  });
});

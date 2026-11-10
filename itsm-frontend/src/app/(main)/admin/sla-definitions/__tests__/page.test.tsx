import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { SLAApi } from '@/lib/api/sla-api';
import type { SLADefinition } from '@/lib/api/sla-api';
import SLADefinitionsPage from '../page';

jest.mock('@/lib/api/sla-api', () => ({
  SLAApi: {
    getSLADefinitions: jest.fn(),
    createSLADefinition: jest.fn(),
    updateSLADefinition: jest.fn(),
    deleteSLADefinition: jest.fn(),
  },
}));

const mockedApi = SLAApi as jest.Mocked<typeof SLAApi>;

const definitions: SLADefinition[] = [
  {
    id: 1,
    name: 'Incident-P0-紧急',
    description: '核心业务中断',
    serviceType: 'incident',
    priority: 'urgent',
    responseTime: 15,
    resolutionTime: 120,
    businessHours: { timeZone: 'Asia/Shanghai', alwaysOn: true },
    escalationLevels: [{ level: 1, thresholdMinutes: 15, notifyRoles: ['manager'] }],
    isActive: true,
    createdAt: '2026-10-01T08:00:00Z',
    updatedAt: '2026-10-02T08:00:00Z',
  },
  {
    id: 2,
    name: 'Change-普通',
    description: '普通变更评审',
    serviceType: 'change',
    priority: 'medium',
    responseTime: 480,
    resolutionTime: 10080,
    isActive: false,
    createdAt: '2026-10-01T08:00:00Z',
    updatedAt: '2026-10-03T08:00:00Z',
  },
];

function openDropdown(ariaLabel: string): HTMLElement {
  fireEvent.mouseDown(screen.getByLabelText(ariaLabel));
  const dropdown = document.querySelector(
    '.ant-select-dropdown:not(.ant-select-dropdown-hidden)'
  ) as HTMLElement;
  if (!dropdown) throw new Error('优先级下拉未打开');
  return dropdown;
}

function selectFilterOption(label: string) {
  const dropdown = openDropdown('Priority Filter');
  fireEvent.click(within(dropdown).getByText(label));
}

function renderPage() {
  return render(
    <App>
      <SLADefinitionsPage />
    </App>
  );
}

describe('SLADefinitionsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedApi.getSLADefinitions.mockResolvedValue({
      items: definitions,
      total: definitions.length,
      page: 1,
      pageSize: 100,
    });
    mockedApi.createSLADefinition.mockResolvedValue({ ...definitions[0], id: 3 });
    mockedApi.updateSLADefinition.mockResolvedValue({ ...definitions[0], isActive: false });
    mockedApi.deleteSLADefinition.mockResolvedValue(undefined);
  });

  it('按后端上限分页取数并读取 items/total，而不是把响应对象当数组', async () => {
    renderPage();

    await waitFor(() =>
      expect(mockedApi.getSLADefinitions).toHaveBeenCalledWith({ page: 1, pageSize: 100 })
    );
    expect(await screen.findByText('Incident-P0-紧急')).toBeInTheDocument();
    expect(screen.getByText('Change-普通')).toBeInTheDocument();
  });

  it('统计卡片只使用后端真实字段，去掉草稿与伪造的平均可用性', async () => {
    renderPage();
    await screen.findByText('Incident-P0-紧急');

    expect(screen.getByText("Total number of SLA definitions")).toBeInTheDocument();
    expect(screen.getByText("Total number of SLA definitions")).toBeInTheDocument();
    expect(screen.getByText("Override service type")).toBeInTheDocument();
    // 已启用/已停用同时出现在状态列，因此断言至少各有两处
    expect(screen.getAllByText("Enabled").length).toBeGreaterThan(1);
    expect(screen.getAllByText("Disabled").length).toBeGreaterThan(1);
    expect(screen.queryByText("The article will return to draft status after unpublishing.")).not.toBeInTheDocument();
    expect(screen.queryByText("平均Available性")).not.toBeInTheDocument();
  });

  it('服务窗口与升级规则来自后端投影，未配置时明确标注', async () => {
    renderPage();
    await screen.findByText('Incident-P0-紧急');

    expect(screen.getByText("Service window: 7×24")).toBeInTheDocument();
    expect(screen.getByText("L1 · 15 minutes before escalation → manager")).toBeInTheDocument();
    expect(screen.getByText("Service window: Not configured")).toBeInTheDocument();
    expect(screen.getByText("Not configured")).toBeInTheDocument();
    expect(screen.getByText("Response: 8 hours")).toBeInTheDocument();
    expect(screen.getByText("Solution: 7 days")).toBeInTheDocument();
  });

  it('urgent 与 critical 视为同一优先级，筛选不会漏掉存量定义', async () => {
    renderPage();
    await screen.findByText('Incident-P0-紧急');

    selectFilterOption('Urgent');
    await waitFor(() => expect(screen.queryByText('Change-普通')).not.toBeInTheDocument());
    expect(screen.getByText('Incident-P0-紧急')).toBeInTheDocument();

    selectFilterOption('Low');
    await waitFor(() => expect(screen.queryByText('Incident-P0-紧急')).not.toBeInTheDocument());
  });

  it('新建提交分钟单位并显式带 isActive，保存后重新拉取列表', async () => {
    renderPage();
    await screen.findByText('Incident-P0-紧急');

    fireEvent.click(screen.getByRole('button', { name: /Create new SLA/ }));
    const nameInput = await screen.findByLabelText("SLA Name");
    const modal = nameInput.closest('.ant-modal') as HTMLElement;

    fireEvent.change(within(modal).getByLabelText("SLA Name"), { target: { value: 'Incident-P1-高' } });
    fireEvent.click(within(openDropdown('Service Type')).getByText('Incidents'));
    fireEvent.change(within(modal).getByLabelText("Description"), {
      target: { value: '重要业务受影响，需快速恢复' },
    });
    fireEvent.change(within(modal).getByLabelText(/Response time limit/), { target: { value: '30' } });
    fireEvent.change(within(modal).getByLabelText(/Resolution time limit/), { target: { value: '240' } });

    fireEvent.click(within(modal).getByRole('button', { name: /Save/ }));

    await waitFor(() => expect(mockedApi.createSLADefinition).toHaveBeenCalledTimes(1));
    // 表单字段与后端请求体同单位：响应 30 分钟、解决 240 分钟，并显式提交 isActive
    expect(mockedApi.createSLADefinition).toHaveBeenCalledWith({
      name: 'Incident-P1-高',
      description: '重要业务受影响，需快速恢复',
      serviceType: 'incident',
      priority: 'medium',
      responseTime: 30,
      resolutionTime: 240,
      isActive: true,
    });
    // 保存后以服务端数据为准重新加载，不再把表单值拼成本地假行
    expect(mockedApi.getSLADefinitions).toHaveBeenCalledTimes(2);
  });

  it('停用操作提交 isActive 取反并重新加载', async () => {
    renderPage();
    await screen.findByText('Incident-P0-紧急');

    fireEvent.click(await screen.findByRole('button', { name: "Disable" }));

    await waitFor(() =>
      expect(mockedApi.updateSLADefinition).toHaveBeenCalledWith(1, { isActive: false })
    );
    expect(mockedApi.getSLADefinitions).toHaveBeenCalledTimes(2);
  });

  it('加载失败时显示错误与重试入口，而不是伪装成空列表', async () => {
    mockedApi.getSLADefinitions.mockRejectedValueOnce(new Error('SLA 定义服务暂不可用'));

    renderPage();

    expect(await screen.findByText("SLA definition loading failed")).toBeInTheDocument();
    expect(screen.getByText('SLA 定义服务暂不可用')).toBeInTheDocument();
    expect(screen.queryByText("There is no SLA definition yet. Click &quot;New SLA&quot; in the upper right corner to create it.")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Reload/ }));

    await waitFor(() => expect(mockedApi.getSLADefinitions).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('Incident-P0-紧急')).toBeInTheDocument();
  });

  it('服务端总数超过已加载条数时明确提示截断', async () => {
    mockedApi.getSLADefinitions.mockResolvedValue({
      items: definitions,
      total: 137,
      page: 1,
      pageSize: 100,
    });

    renderPage();

    expect(await screen.findByText(/Shared by the server 137 SLA definitions/)).toBeInTheDocument();
  });
});

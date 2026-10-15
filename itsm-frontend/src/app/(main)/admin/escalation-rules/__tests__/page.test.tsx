import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { SLAApi } from '@/lib/api/sla-api';
import type { SLAAlertRule, SLADefinition } from '@/lib/api/sla-api';
import EscalationRulesPage from '../page';

jest.mock('@/lib/api/sla-api', () => ({
  SLAApi: {
    getAlertRules: jest.fn(),
    getSLADefinitions: jest.fn(),
    createAlertRule: jest.fn(),
    updateAlertRule: jest.fn(),
    deleteAlertRule: jest.fn(),
  },
}));

const mockedApi = SLAApi as jest.Mocked<typeof SLAApi>;

const definitions: SLADefinition[] = [
  {
    id: 1,
    name: 'SLA-P0-紧急',
    description: '',
    priority: 'urgent',
    responseTime: 30,
    resolutionTime: 240,
  },
  {
    id: 4,
    name: 'SLA-P3-低',
    description: '',
    priority: 'low',
    responseTime: 240,
    resolutionTime: 1440,
  },
];

const rules: SLAAlertRule[] = [
  {
    id: 11,
    slaDefinitionId: 4,
    name: 'SLA-变更-响应告警',
    thresholdPercentage: 50,
    alertLevel: 'warning',
    notificationChannels: ['email', 'sms'],
    isActive: true,
  },
];

function renderPage() {
  return render(
    <App>
      <EscalationRulesPage />
    </App>,
  );
}

describe('EscalationRulesPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedApi.getAlertRules.mockResolvedValue(rules);
    mockedApi.getSLADefinitions.mockResolvedValue({
      items: definitions,
      total: definitions.length,
      page: 1,
      pageSize: 100,
    });
    mockedApi.createAlertRule.mockResolvedValue({ ...rules[0], id: 99 });
    mockedApi.updateAlertRule.mockResolvedValue({ ...rules[0], isActive: false });
    mockedApi.deleteAlertRule.mockResolvedValue(undefined);
  });

  it('渲染后端真实字段，不再展示伪造的使用统计与创建者', async () => {
    renderPage();

    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());
    // 关联定义名来自 getSLADefinitions，而不是硬编码的「全部」
    expect(await screen.findByText('SLA-P3-低')).toBeInTheDocument();
    expect(screen.queryByText('使用统计')).not.toBeInTheDocument();
    expect(screen.queryByText('次触发')).not.toBeInTheDocument();
    expect(screen.queryByText(/由 系统/)).not.toBeInTheDocument();
  });

  it('创建时提交表单选中的 slaDefinitionId，而不是硬编码定义', async () => {
    renderPage();
    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: /新建规则/ }));
    const nameInput = await screen.findByLabelText('规则名称');
    const modal = nameInput.closest('.ant-modal') as HTMLElement;

    fireEvent.change(within(modal).getByLabelText('规则名称'), { target: { value: 'P3 响应超时预警' } });

    const definitionCombobox = within(modal).getByLabelText('关联 SLA 定义');
    fireEvent.mouseDown(definitionCombobox);
    fireEvent.click(await screen.findByText('SLA-P3-低（low）'));

    fireEvent.click(within(modal).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(mockedApi.createAlertRule).toHaveBeenCalledTimes(1));
    expect(mockedApi.createAlertRule).toHaveBeenCalledWith({
      name: 'P3 响应超时预警',
      slaDefinitionId: 4,
      alertLevel: 'warning',
      thresholdPercentage: 80,
      notificationChannels: ['email'],
      isActive: false,
    });
  });

  it('停用按钮调用更新接口并取反 isActive', async () => {
    renderPage();
    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());

    fireEvent.click(await screen.findByRole('button', { name: '停用' }));

    await waitFor(() =>
      expect(mockedApi.updateAlertRule).toHaveBeenCalledWith(11, { isActive: false }),
    );
  });

  it('删除需二次确认且确认后才调用删除接口', async () => {
    renderPage();
    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());

    fireEvent.click(await screen.findByRole('button', { name: /删除/ }));

    const confirm = await screen.findByText('确定删除该告警规则吗？');
    const popper = confirm.closest('.ant-popover') as HTMLElement;
    expect(within(popper).getByText(/SLA-变更-响应告警/)).toBeInTheDocument();
    expect(mockedApi.deleteAlertRule).not.toHaveBeenCalled();

    fireEvent.click(within(popper).getByRole('button', { name: '删 除' }));

    await waitFor(() => expect(mockedApi.deleteAlertRule).toHaveBeenCalledWith(11));
  });
});

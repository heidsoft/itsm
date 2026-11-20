import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { SLAApi } from '@/lib/api/sla-api';
import type { SLAAlertRule, SLADefinition } from '@/lib/api/sla-api';
import { EscalationMatrixApi } from '@/lib/api/escalation-matrix-api';
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

jest.mock('@/lib/api/escalation-matrix-api', () => ({
  EscalationMatrixApi: {
    getMatrix: jest.fn(),
    setMatrix: jest.fn(),
  },
  DEFAULT_ESCALATION_MATRIX: {
    critical: [{ level: 1, afterMinutes: 15, notifyRoles: ['manager'], notifyUserIDs: [], description: '' }],
    high: [{ level: 1, afterMinutes: 60, notifyRoles: ['manager'], notifyUserIDs: [], description: '' }],
    medium: [{ level: 1, afterMinutes: 240, notifyRoles: ['manager'], notifyUserIDs: [], description: '' }],
    low: [{ level: 1, afterMinutes: 480, notifyRoles: ['manager'], notifyUserIDs: [], description: '' }],
  },
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: () => true }),
  PERM: { sla: { read: 'sla:read', write: 'sla:write' } },
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
    (EscalationMatrixApi.getMatrix as jest.Mock).mockResolvedValue({
      critical: [{ level: 1, afterMinutes: 15, notifyRoles: ['manager'], notifyUserIDs: [], description: '内置默认' }],
    });
  });

  it('渲染后端真实字段，不再展示伪造的使用统计与创建者', async () => {
    renderPage();

    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());
    // 页面合并后标题为「超时升级」，两个 Tab 常驻可见
    expect(screen.getByText('超时升级')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '告警阈值' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '升级链' })).toBeInTheDocument();
    // 关联定义名来自 getSLADefinitions，而不是硬编码的「全部」
    expect(await screen.findByText('SLA-P3-低')).toBeInTheDocument();
    expect(screen.queryByText('使用统计')).not.toBeInTheDocument();
    expect(screen.queryByText('次触发')).not.toBeInTheDocument();
    expect(screen.queryByText(/由 系统/)).not.toBeInTheDocument();
  });

  // 整页 jsdom 渲染 + antd 弹窗，在全量并行下会超过 jest.config 的 10s 默认上限。
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
  }, 30000);

  it('停用按钮调用更新接口并取反 isActive', async () => {
    renderPage();
    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());

    fireEvent.click(await screen.findByRole('button', { name: '停用' }));

    await waitFor(() =>
      expect(mockedApi.updateAlertRule).toHaveBeenCalledWith(11, { isActive: false }),
    );
  }, 30000);

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
  }, 30000);

  it('切到升级链 Tab 后按优先级展示并可编辑服务端矩阵', async () => {
    renderPage();
    await waitFor(() => expect(mockedApi.getAlertRules).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('tab', { name: '升级链' }));

    // 展示服务端矩阵值（15 分钟），缺失的优先级由默认补齐（60 分钟→1.0 小时、480 分钟→8.0 小时）
    await screen.findByText('15 分钟');
    expect(screen.getByText('1.0 小时')).toBeInTheDocument();
    expect(screen.getByText('8.0 小时')).toBeInTheDocument();
    // 权限放行时出现保存与加级入口
    expect(screen.getByRole('button', { name: /保\s*存（无改动）/ })).toBeInTheDocument();
    expect(screen.getByText('给致命加一级')).toBeInTheDocument();
  }, 30000);
});

import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { SLATemplateApi, type SLATemplate } from '@/lib/api/sla-template-api';
import SLATemplatesPage from '../page';

jest.mock('@/lib/api/sla-template-api', () => ({
  SLATemplateApi: {
    listTemplates: jest.fn(),
    installTemplate: jest.fn(),
    installAllRecommended: jest.fn(),
  },
}));

const mockedApi = SLATemplateApi as jest.Mocked<typeof SLATemplateApi>;

const p1: SLATemplate = {
  key: 'incident_p1_critical',
  name: '事件 P1 紧急',
  description: '适用于业务完全中断的紧急事件',
  serviceType: 'incident',
  priority: 'critical',
  responseTime: 15,
  resolutionTime: 240,
  businessHours: {
    timeZone: 'Asia/Shanghai',
    alwaysOn: true,
    workDays: [1, 2, 3, 4, 5],
    startTime: '00:00',
    endTime: '23:59',
  },
  escalationLevels: [
    { level: 1, thresholdMinutes: 15, notifyRoles: ['manager'] },
    { level: 2, thresholdMinutes: 30, notifyRoles: ['it_admin'] },
  ],
  conditions: { ticket_type: ['incident'], priority: ['critical'] },
  industry: 'incident',
  recommended: true,
  installed: true,
  slaDefinitionId: 42,
  installedAt: '2026-10-15T08:00:00Z',
};

const changeNormal: SLATemplate = {
  key: 'change_normal',
  name: '变更 - 普通',
  description: '适用于风险较低的常规变更',
  serviceType: 'change',
  priority: 'medium',
  responseTime: 240,
  resolutionTime: 1440,
  businessHours: {
    timeZone: 'Asia/Shanghai',
    alwaysOn: false,
    workDays: [1, 2, 3, 4, 5],
    startTime: '09:00',
    endTime: '18:00',
  },
  escalationLevels: [{ level: 1, thresholdMinutes: 240, notifyRoles: ['change_manager'] }],
  conditions: { ticket_type: ['change'] },
  industry: 'change',
  recommended: true,
  installed: false,
};

// 用文本定位行，避免 findByRole('row', {name}) 在 antd 表格上付出可及名称全树遍历的代价
async function rowOf(templateName: string): Promise<HTMLElement> {
  const cell = await screen.findByText(templateName);
  const row = cell.closest('tr');
  if (!row) throw new Error(`未找到模板行：${templateName}`);
  return row;
}

async function clickInstall(templateName: string) {
  const row = await rowOf(templateName);
  fireEvent.click(within(row).getByRole('button', { name: /^Installation$/ }));
  fireEvent.click(await screen.findByText("Confirm installation"));
}

async function renderPage() {
  render(
    <App>
      <SLATemplatesPage />
    </App>
  );
  return rowOf(p1.name);
}

describe('SLATemplatesPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedApi.listTemplates.mockResolvedValue([p1, changeNormal]);
    mockedApi.installTemplate.mockResolvedValue({
      templateKey: changeNormal.key,
      slaDefinitionId: 43,
      created: true,
      wasAlreadyExist: false,
      message: 'ok',
    });
    mockedApi.installAllRecommended.mockResolvedValue({ results: [], failures: [] });
  });

  it('安装状态来自服务端而不是本次点击的内存状态', async () => {
    const installedRow = await renderPage();

    // 页面加载即从后端拿到 installed，已装模板刷新后不会退回「未安装」
    expect(within(installedRow).getByText('Already installed')).toBeInTheDocument();
    expect(within(installedRow).queryByText("Not installed")).not.toBeInTheDocument();

    const pendingRow = await rowOf(changeNormal.name);
    expect(within(pendingRow).getByText("Not installed")).toBeInTheDocument();
    expect(screen.getByText("Installed for this tenant")).toBeInTheDocument();
  });

  it('优先级、服务窗口、时限使用与 SLA 定义页相同的展示口径', async () => {
    const row = await renderPage();

    // 后端词表是 critical，不是 P1；旧页面按 P1..P4 上色导致整列空白
    expect(within(row).getByText('Urgent')).toBeInTheDocument();
    expect(within(row).queryByText('critical')).not.toBeInTheDocument();
    expect(within(row).getByText("15 minutes")).toBeInTheDocument();
    expect(within(row).getByText("4 hours")).toBeInTheDocument();
    expect(within(row).getByText('7×24')).toBeInTheDocument();
    expect(within(row).getByText('Incidents')).toBeInTheDocument();
    expect(within(row).getByText("Incident")).toBeInTheDocument();
  });

  it('安装成功后以服务端列表为准重新拉取，而不是本地记一笔', async () => {
    await renderPage();
    expect(mockedApi.listTemplates).toHaveBeenCalledTimes(1);

    await clickInstall(changeNormal.name);

    await waitFor(() => expect(mockedApi.installTemplate).toHaveBeenCalledWith(changeNormal.key));
    await waitFor(() => expect(mockedApi.listTemplates).toHaveBeenCalledTimes(2));
    expect(
      await screen.findByText(`Template "${changeNormal.key}"Already installed in the current tenant"`)
    ).toBeInTheDocument();
  });

  it('模板已存在时如实提示跳过，不谎报新建', async () => {
    mockedApi.installTemplate.mockResolvedValue({
      templateKey: changeNormal.key,
      slaDefinitionId: 43,
      created: false,
      wasAlreadyExist: true,
      message: '该模板已安装过',
    });

    await renderPage();
    await clickInstall(changeNormal.name);

    expect(
      await screen.findByText(`Template "${changeNormal.key}"It has been installed before and has not been re-created.`)
    ).toBeInTheDocument();
  });

  it('单个安装失败要显示后端原因，不伪装成功', async () => {
    mockedApi.installTemplate.mockRejectedValueOnce(new Error('租户上下文缺失'));

    await renderPage();
    await clickInstall(changeNormal.name);

    expect(await screen.findByText("1 Template installation failed")).toBeInTheDocument();
    expect(screen.getByText('change_normal：租户上下文缺失')).toBeInTheDocument();
    expect(mockedApi.listTemplates).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(/Already installed in the current tenant/)).not.toBeInTheDocument();
  });

  it('批量安装逐个提交并点名失败模板', async () => {
    mockedApi.installAllRecommended.mockResolvedValue({
      results: [
        {
          templateKey: p1.key,
          slaDefinitionId: 42,
          created: false,
          wasAlreadyExist: true,
          message: 'ok',
        },
      ],
      failures: [{ templateKey: changeNormal.key, reason: '安装 SLA 模板失败' }],
    });

    await renderPage();

    fireEvent.click(screen.getByText("Install all recommended templates in batches"));
    fireEvent.click(await screen.findByText("Install all"));

    await waitFor(() => expect(mockedApi.installAllRecommended).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("1 Template installation failed")).toBeInTheDocument();
    expect(screen.getByText("change_normal：安装 SLA 模板失败")).toBeInTheDocument();
    expect(mockedApi.listTemplates).toHaveBeenCalledTimes(2);
  });

  it('加载失败显示错误与重试入口，而不是静默空表格', async () => {
    mockedApi.listTemplates.mockRejectedValueOnce(new Error('SLA 模板服务暂不可用'));

    render(
      <App>
        <SLATemplatesPage />
      </App>
    );

    expect(await screen.findByText("SLA template loading failed")).toBeInTheDocument();
    expect(screen.getByText('SLA 模板服务暂不可用')).toBeInTheDocument();
    expect(screen.getByText("Template loading failed")).toBeInTheDocument();

    fireEvent.click(screen.getByText(/Retry/));

    await waitFor(() => expect(mockedApi.listTemplates).toHaveBeenCalledTimes(2));
    expect(await rowOf(p1.name)).toBeInTheDocument();
  });

  it('详情弹窗用后端投影字段渲染升级级别，不再打印原始 JSON', async () => {
    const row = await renderPage();

    fireEvent.click(within(row).getByRole('button', { name: /View/ }));

    const modal = await screen.findByRole('dialog');
    expect(within(modal).getByText("L1 · 15 minutes before escalation → manager")).toBeInTheDocument();
    expect(within(modal).getByText("L2 · 30 minutes before escalation → it_admin")).toBeInTheDocument();
    expect(within(modal).getByText(/Installed in/)).toBeInTheDocument();
    expect(within(modal).queryByText(/threshold_minutes/)).not.toBeInTheDocument();
    expect(within(modal).queryByText(/notify_roles/)).not.toBeInTheDocument();
  });
});

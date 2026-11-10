import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { EscalationMatrixApi } from '@/lib/api/escalation-matrix-api';
import type { EscalationMatrix } from '@/lib/api/escalation-matrix-api';
import EscalationMatricesPage from '../page';

jest.mock('@/lib/api/escalation-matrix-api', () => ({
  EscalationMatrixApi: { getMatrix: jest.fn() },
  DEFAULT_ESCALATION_MATRIX: {
    critical: [
      {
        level: 1,
        afterMinutes: 15,
        notifyRoles: ['manager'],
        notifyUserIDs: [],
        description: '内置默认：15分钟未响应升级至部门经理',
      },
    ],
    high: [],
    medium: [],
    low: [],
  },
}));

const mockedApi = EscalationMatrixApi as jest.Mocked<typeof EscalationMatrixApi>;

// 与内置默认刻意不同的服务端矩阵：阈值、角色、用户、优先级键全部不一样，
// 页面只要漏读一处就会露馅。
const SERVER_MATRIX: EscalationMatrix = {
  critical: [
    {
      level: 1,
      afterMinutes: 7,
      notifyRoles: ['it_admin'],
      notifyUserIDs: [3],
      description: '7分钟未响应升级至IT管理员',
    },
    {
      level: 2,
      afterMinutes: 5400,
      notifyRoles: [],
      notifyUserIDs: [],
      description: '',
    },
  ],
  urgent: [
    {
      level: 1,
      afterMinutes: 30,
      notifyRoles: ['admin', 'manager'],
      notifyUserIDs: [],
      description: '30分钟未响应升级至租户管理员与部门经理',
    },
  ],
};

function renderPage() {
  return render(
    <App>
      <EscalationMatricesPage />
    </App>,
  );
}

function statistic(label: string): HTMLElement {
  const node = screen.getByText(label).closest('.ant-statistic');
  if (!node) throw new Error(`未找到统计卡片：${label}`);
  return node as HTMLElement;
}

describe('EscalationMatricesPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('展示服务端矩阵的阈值、角色与用户，而不是内置默认', async () => {
    mockedApi.getMatrix.mockResolvedValue(SERVER_MATRIX);
    renderPage();

    await waitFor(() => expect(mockedApi.getMatrix).toHaveBeenCalled());

    // 表格视图与分组视图同源，每个服务端值都出现两次；只渲染一处说明另一处仍写死。
    expect(await screen.findAllByText("7 minutes")).toHaveLength(2);
    expect(screen.getAllByText("it_admin · IT Administrator")).toHaveLength(2);
    expect(screen.getAllByText("3.8 days")).toHaveLength(2);
    // 只有 critical L2 两个通知目标都为空，urgent L1 仅用户为空。
    expect(screen.getAllByText("No role specified")).toHaveLength(1);
    expect(screen.getAllByText("No user specified")).toHaveLength(2);
    expect(screen.getAllByText("No description")).toHaveLength(1);

    // 内置默认的阈值 15 分钟与服务端不同，出现即说明页面在渲染假配置。
    expect(screen.queryByText("15 minutes")).not.toBeInTheDocument();
  });

  it('优先级键与通知目标由服务端决定，未知键原样透出', async () => {
    mockedApi.getMatrix.mockResolvedValue(SERVER_MATRIX);
    renderPage();

    expect(await screen.findAllByText("critical · Fatal")).toHaveLength(3);
    // urgent 不在前端词表里，必须原样显示，不能被前端词表覆盖。
    expect(screen.getAllByText('urgent')).toHaveLength(2);

    // 统计值全部由服务端矩阵得出：2 个优先级、3 个级别、4 个通知目标
    expect(statistic("Override priority")).toHaveTextContent(/2\s*items/);
    expect(statistic("Total number of upgrade levels")).toHaveTextContent(/3\s*level/);
    expect(statistic("Number of notification targets")).toHaveTextContent(/4\s*items/);
    expect(screen.getByText("2 level upgrade")).toBeInTheDocument();
    expect(screen.getByText("1 level upgrade")).toBeInTheDocument();
  });

  it('加载失败时回落到内置默认并明确标注不是生产配置', async () => {
    mockedApi.getMatrix.mockRejectedValue(new Error('升级矩阵返回为空，无法确认当前生效配置'));
    renderPage();

    const alert = await screen.findByText("Upgrade matrix loading failed");
    expect(alert).toBeInTheDocument();
    expect(
      screen.getByText(/\. What is shown below is the default policy built into the system, not the production configuration\. Please do not judge online upgrade behavior based on this\./),
    ).toBeInTheDocument();
    expect(screen.getAllByText("15 minutes").length).toBeGreaterThan(0);
  });
});

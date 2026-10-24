import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { teamService } from '@/lib/services/team-service';
import type { Team, TeamListResult } from '@/lib/services/team-service';
import { UserApi } from '@/lib/api/user-api';
import TeamsPage from '../page';

jest.mock('@/lib/services/team-service', () => ({
  teamService: {
    listTeams: jest.fn(),
    getTeam: jest.fn(),
    createTeam: jest.fn(),
    updateTeam: jest.fn(),
    deleteTeam: jest.fn(),
    listMembers: jest.fn(),
    addMember: jest.fn(),
    removeMember: jest.fn(),
  },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn() },
}));

const mockedTeamService = teamService as jest.Mocked<typeof teamService>;
const mockedGetUsers = UserApi.getUsers as jest.Mock;

// antd Table + Modal 在 jsdom 下每次挂载要几秒，默认 5s/10s 预算不够。
jest.setTimeout(30000);

const alice = { id: 10, username: 'alice', name: '爱丽丝', email: 'alice@example.com' };
const bob = { id: 11, username: 'bob', name: '鲍勃', email: 'bob@example.com' };
const carol = { id: 20, username: 'carol', name: '卡罗尔', email: 'carol@example.com' };

const team: Team = {
  id: 1,
  name: '运维一组',
  code: 'OPS-A',
  description: '负责生产运维',
  status: 'active',
  managerId: 10,
  managerName: '爱丽丝',
  tenantId: 7,
  members: [alice, bob],
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
};

function listResult(items: Team[], total = items.length): TeamListResult {
  return { items, total, page: 1, pageSize: 10, totalPages: 1 };
}

async function renderPage(items: Team[] = [team], total = items.length) {
  mockedTeamService.listTeams.mockResolvedValue(listResult(items, total));
  render(
    <App>
      <TeamsPage />
    </App>
  );
  await waitFor(() => expect(mockedTeamService.listTeams).toHaveBeenCalled());
  return screen.findByText('运维一组');
}

async function openMemberModal() {
  await renderPage();
  fireEvent.click(await screen.findByRole('button', { name: '管理成员' }));
  return await screen.findByRole('dialog');
}

function visibleDropdownOption(name: string) {
  return waitFor(() => {
    const option = [...document.querySelectorAll('.ant-select-dropdown:not(.ant-select-dropdown-hidden) .ant-select-item-option')].find(
      element => element.textContent?.includes(name)
    );
    if (!option) throw new Error(`下拉里没有选项 ${name}`);
    return option;
  });
}

describe('团队管理页 — 成员链路', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedGetUsers.mockResolvedValue({
      items: [
        { id: 10, username: 'alice', name: '爱丽丝' },
        { id: 11, username: 'bob', name: '鲍勃' },
        { id: 20, username: 'carol', name: '卡罗尔' },
      ],
    });
  });

  it('成员列渲染服务端返回的 members，而不是 Ent 的 edges 形态', async () => {
    const name = await renderPage();
    const row = name.closest('tr') as HTMLElement;

    // 两名成员的头像 + 成员数
    expect(row.querySelectorAll('.ant-avatar')).toHaveLength(2);
    expect(within(row).getByText('2')).toBeInTheDocument();
    // 负责人姓名取自服务端投影字段
    expect(within(row).getByText('爱丽丝')).toBeInTheDocument();
  }, 30000);

  it('负责人姓名用服务端投影，不在已加载用户目录里的负责人也能显示', async () => {
    const outside = { ...team, managerId: 9999, managerName: '陈工' };
    const name = await renderPage([outside]);
    const row = name.closest('tr') as HTMLElement;

    expect(within(row).getByText('陈工')).toBeInTheDocument();
  }, 30000);

  it('团队总数使用服务端 total，不用当前页条数冒充', async () => {
    await renderPage([team], 37);

    expect(await screen.findByText('37')).toBeInTheDocument();
  });

  it('搜索交给服务端并回到第 1 页', async () => {
    await renderPage();

    fireEvent.change(screen.getByRole('textbox'), { target: { value: '运维' } });

    await waitFor(
      () =>
        expect(mockedTeamService.listTeams).toHaveBeenLastCalledWith({
          page: 1,
          pageSize: 10,
          search: '运维',
        }),
      { timeout: 2000 }
    );
  });

  it('翻页请求带上新的 page', async () => {
    await renderPage([team], 30);

    const next = await waitFor(() => {
      const button = document.querySelector('.ant-pagination-next button');
      if (!button) throw new Error('分页下一页未渲染');
      return button;
    });
    fireEvent.click(next);

    await waitFor(() =>
      expect(mockedTeamService.listTeams).toHaveBeenLastCalledWith({
        page: 2,
        pageSize: 10,
        search: undefined,
      })
    );
  });

  it('添加成员调用真实端点并用服务端返回的列表刷新', async () => {
    const dialog = await openMemberModal();
    mockedTeamService.addMember.mockResolvedValue([alice, bob, carol]);

    fireEvent.mouseDown(within(dialog).getByRole('combobox'));
    fireEvent.click(await visibleDropdownOption('卡罗尔'));
    fireEvent.click(within(dialog).getByRole('button', { name: /添加/ }));

    await waitFor(() => expect(mockedTeamService.addMember).toHaveBeenCalledWith(1, 20));
    expect(await within(dialog).findByText('carol@example.com')).toBeInTheDocument();
  });

  it('已在其他团队的用户加入失败时透出后端冲突原因，成员列表保持不变', async () => {
    const dialog = await openMemberModal();
    mockedTeamService.addMember.mockRejectedValue(
      new Error('用户已属于团队「网络组」，请先从该团队移出')
    );

    fireEvent.mouseDown(within(dialog).getByRole('combobox'));
    fireEvent.click(await visibleDropdownOption('卡罗尔'));
    fireEvent.click(within(dialog).getByRole('button', { name: /添加/ }));

    expect(
      await screen.findByText('用户已属于团队「网络组」，请先从该团队移出')
    ).toBeInTheDocument();
    expect(within(dialog).getByText('alice@example.com')).toBeInTheDocument();
    expect(within(dialog).queryByText('carol@example.com')).not.toBeInTheDocument();
  });

  it('移除成员需要二次确认并按服务端结果更新', async () => {
    const dialog = await openMemberModal();
    mockedTeamService.removeMember.mockResolvedValue([alice]);

    fireEvent.click(within(dialog).getAllByRole('button', { name: '移出团队' })[1]);
    const confirm = await screen.findByRole('tooltip');
    fireEvent.click(within(confirm).getByRole('button', { name: '确 定' }));

    await waitFor(() => expect(mockedTeamService.removeMember).toHaveBeenCalledWith(1, 11));
    expect(await within(dialog).findByText('alice@example.com')).toBeInTheDocument();
    expect(within(dialog).queryByText('bob@example.com')).not.toBeInTheDocument();
  });

  it('编辑时清空负责人和描述要显式提交 0/空串，后端才能与"未传"区分', async () => {
    await renderPage();
    fireEvent.click(await screen.findByRole('button', { name: '编辑团队' }));
    const dialog = await screen.findByRole('dialog');

    const clearIcon = dialog.querySelector('.ant-select-clear');
    expect(clearIcon).not.toBeNull();
    // rc-select 的清空图标绑在 mousedown 上
    fireEvent.mouseDown(clearIcon as Element);
    fireEvent.change(screen.getByDisplayValue('负责生产运维'), { target: { value: '' } });
    mockedTeamService.updateTeam.mockResolvedValue({ ...team, managerId: 0, description: '' });

    fireEvent.click(within(dialog).getByRole('button', { name: '保 存' }));

    await waitFor(() =>
      expect(mockedTeamService.updateTeam).toHaveBeenCalledWith(1, {
        name: '运维一组',
        code: 'OPS-A',
        description: '',
        managerId: 0,
      })
    );
  });

  it('加载失败给出可重试的错误态而不是空列表', async () => {
    mockedTeamService.listTeams.mockRejectedValue(new Error('加载团队数据失败'));
    render(
      <App>
        <TeamsPage />
      </App>
    );

    expect(await screen.findByText('加载团队数据失败')).toBeInTheDocument();

    mockedTeamService.listTeams.mockResolvedValue(listResult([team]));
    fireEvent.click(await screen.findByRole('button', { name: '重 试' }));

    expect(await screen.findByText('运维一组')).toBeInTheDocument();
  });
});

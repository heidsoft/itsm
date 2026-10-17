import { httpClient } from '@/lib/api/http-client';

const TEAMS_PATH = '/api/v1/org/teams';

/** 团队成员投影，与后端 handlers/common.TeamMember 逐字段一致。 */
export interface TeamMember {
  id: number;
  username: string;
  name: string;
  email: string;
}

/** 团队投影，与后端 handlers/common.Team 逐字段一致。 */
export interface Team {
  id: number;
  name: string;
  code: string;
  description: string;
  status: string;
  managerId: number;
  managerName: string;
  tenantId: number;
  members: TeamMember[];
  createdAt: string;
  updatedAt: string;
}

export interface TeamListParams {
  page?: number;
  pageSize?: number;
  search?: string;
}

/** 列表信封：分页与总数由服务端给出，前端不再对当前页做二次业务过滤。 */
export interface TeamListResult {
  items: Team[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface TeamMemberListResult {
  items: TeamMember[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CreateTeamRequest {
  name: string;
  code: string;
  description?: string;
  managerId?: number;
}

export type UpdateTeamRequest = Partial<CreateTeamRequest>;

function toTeam(raw: Partial<Team>): Team {
  return {
    id: raw.id ?? 0,
    name: raw.name ?? '',
    code: raw.code ?? '',
    description: raw.description ?? '',
    status: raw.status ?? '',
    managerId: raw.managerId ?? 0,
    managerName: raw.managerName ?? '',
    tenantId: raw.tenantId ?? 0,
    members: (raw.members ?? []).map(member => ({
      id: member.id,
      username: member.username,
      name: member.name,
      email: member.email,
    })),
    createdAt: raw.createdAt ?? '',
    updatedAt: raw.updatedAt ?? '',
  };
}

class TeamService {
  async listTeams(params: TeamListParams = {}): Promise<TeamListResult> {
    const data = await httpClient.get<TeamListResult>(TEAMS_PATH, params);
    return {
      items: (data.items ?? []).map(toTeam),
      total: data.total,
      page: data.page,
      pageSize: data.pageSize,
      totalPages: data.totalPages,
    };
  }

  async getTeam(id: number): Promise<Team> {
    const data = await httpClient.get<Partial<Team>>(`${TEAMS_PATH}/${id}`);
    return toTeam(data);
  }

  async createTeam(data: CreateTeamRequest): Promise<Team> {
    const team = await httpClient.post<Partial<Team>>(TEAMS_PATH, data);
    return toTeam(team);
  }

  async updateTeam(id: number, data: UpdateTeamRequest): Promise<Team> {
    const team = await httpClient.put<Partial<Team>>(`${TEAMS_PATH}/${id}`, data);
    return toTeam(team);
  }

  async deleteTeam(id: number): Promise<void> {
    return httpClient.delete<void>(`${TEAMS_PATH}/${id}`);
  }

  async listMembers(teamId: number, params: TeamListParams = {}): Promise<TeamMemberListResult> {
    return httpClient.get<TeamMemberListResult>(`${TEAMS_PATH}/${teamId}/members`, params);
  }

  /** 添加成员返回服务端最新成员列表，前端不需要自己拼本地状态。 */
  async addMember(teamId: number, userId: number): Promise<TeamMember[]> {
    return httpClient.post<TeamMember[]>(`${TEAMS_PATH}/${teamId}/members`, { userId });
  }

  async removeMember(teamId: number, userId: number): Promise<TeamMember[]> {
    return httpClient.delete<TeamMember[]>(`${TEAMS_PATH}/${teamId}/members`, { userId });
  }
}

export const teamService = new TeamService();

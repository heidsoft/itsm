import { httpClient } from '@/lib/api/http-client';
import type { User } from '@/lib/api/api-config'; // Reuse User interface if possible, or define local

export interface Team {
  id: number;
  name: string;
  code: string;
  description?: string;
  managerId?: number;
  createdAt?: string;
  updatedAt?: string;
  edges?: {
    users?: User[];
  };
}

type RawTeam = Partial<Team> & {
  managerId?: number;
  createdAt?: string;
  updatedAt?: string;
};

export interface CreateTeamRequest {
  name: string;
  code: string;
  description?: string;
  managerId?: number;
}

export interface AddMemberRequest {
  teamId: number;
  userId: number;
}

class TeamService {
  private readonly baseUrl = '/api/v1/org/teams';

  private normalizeTeam(team: RawTeam): Team {
    return {
      id: team.id || 0,
      name: team.name || '',
      code: team.code || '',
      description: team.description,
      managerId: team.managerId,
      createdAt: team.createdAt,
      updatedAt: team.updatedAt,
      edges: team.edges,
    };
  }

  async listTeams(): Promise<Team[]> {
    const data = await httpClient.get<RawTeam[]>(this.baseUrl);
    return data.map(team => this.normalizeTeam(team));
  }

  async createTeam(data: CreateTeamRequest): Promise<Team> {
    return httpClient.post<Team>(this.baseUrl, data);
  }

  async addMember(data: AddMemberRequest): Promise<void> {
    return httpClient.post<void>(`${this.baseUrl}/members`, data);
  }

  async updateTeam(id: number, data: Partial<CreateTeamRequest>): Promise<Team> {
    return httpClient.put<Team>(`${this.baseUrl}/${id}`, data);
  }

  async deleteTeam(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }
}

export const teamService = new TeamService();

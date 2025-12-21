import { httpClient } from '@/lib/api/http-client';
import { User } from '@/app/lib/api-config'; // Reuse User interface if possible, or define local

export interface Team {
  id: number;
  name: string;
  code: string;
  description?: string;
  manager_id?: number;
  created_at?: string;
  updated_at?: string;
  edges?: {
    users?: User[];
  };
}

export interface CreateTeamRequest {
  name: string;
  code: string;
  description?: string;
  manager_id?: number;
}

export interface AddMemberRequest {
  team_id: number;
  user_id: number;
}

class TeamService {
  private readonly baseUrl = '/api/v1/teams';

  async listTeams(): Promise<Team[]> {
    return httpClient.get<Team[]>(this.baseUrl);
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


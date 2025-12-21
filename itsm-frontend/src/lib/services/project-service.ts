import { httpClient } from '@/lib/api/http-client';

export interface Project {
  id: number;
  name: string;
  code: string;
  department_id?: number;
  manager_id?: number;
  start_date?: string;
  end_date?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateProjectRequest {
  name: string;
  code: string;
  department_id?: number;
  manager_id?: number;
}

class ProjectService {
  private readonly baseUrl = '/api/v1/projects';

  async listProjects(): Promise<Project[]> {
    return httpClient.get<Project[]>(this.baseUrl);
  }

  async createProject(data: CreateProjectRequest): Promise<Project> {
    return httpClient.post<Project>(this.baseUrl, data);
  }

  async updateProject(id: number, data: Partial<CreateProjectRequest>): Promise<Project> {
    return httpClient.put<Project>(`${this.baseUrl}/${id}`, data);
  }

  async deleteProject(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }
}

export const projectService = new ProjectService();


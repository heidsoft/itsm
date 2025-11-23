import { httpClient } from '@/lib/api/http-client';

export interface Department {
  id: number;
  name: string;
  code: string;
  description?: string;
  manager_id?: number;
  parent_id?: number;
  children?: Department[];
  created_at?: string;
  updated_at?: string;
  // Frontend specific helper
  key?: number; 
  title?: string;
  value?: number;
}

export interface CreateDepartmentRequest {
  name: string;
  code: string;
  description?: string;
  manager_id?: number;
  parent_id?: number;
}

export interface UpdateDepartmentRequest {
  name?: string;
  code?: string;
  description?: string;
  manager_id?: number;
  parent_id?: number;
}

class DepartmentService {
  private readonly baseUrl = '/api/v1/departments';

  async getDepartmentTree(): Promise<Department[]> {
    return httpClient.get<Department[]>(`${this.baseUrl}/tree`);
  }

  async createDepartment(data: CreateDepartmentRequest): Promise<Department> {
    return httpClient.post<Department>(this.baseUrl, data);
  }
  
  async updateDepartment(id: number, data: UpdateDepartmentRequest): Promise<Department> {
    return httpClient.put<Department>(`${this.baseUrl}/${id}`, data);
  }

  async deleteDepartment(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }
}

export const departmentService = new DepartmentService();


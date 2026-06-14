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

type RawDepartment = Partial<Department> & {
  managerId?: number;
  parentId?: number;
  createdAt?: string;
  updatedAt?: string;
};

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
  private readonly baseUrl = '/api/v1/org/departments';

  private normalizeDepartment(department: RawDepartment): Department {
    return {
      id: department.id || 0,
      name: department.name || '',
      code: department.code || '',
      description: department.description,
      manager_id: department.manager_id ?? department.managerId,
      parent_id: department.parent_id ?? department.parentId,
      children: department.children?.map(child => this.normalizeDepartment(child)),
      created_at: department.created_at || department.createdAt,
      updated_at: department.updated_at || department.updatedAt,
    };
  }

  async getDepartmentTree(): Promise<Department[]> {
    const data = await httpClient.get<RawDepartment[]>(`${this.baseUrl}/tree`);
    return data.map(department => this.normalizeDepartment(department));
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

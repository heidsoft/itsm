import { httpClient } from '@/lib/api/http-client';

export interface Department {
  id: number;
  name: string;
  code: string;
  description?: string;
  managerId?: number;
  parentId?: number;
  children?: Department[];
  createdAt?: string;
  updatedAt?: string;
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
  managerId?: number;
  parentId?: number;
}

export interface UpdateDepartmentRequest {
  name?: string;
  code?: string;
  description?: string;
  managerId?: number;
  parentId?: number;
}

class DepartmentService {
  private readonly baseUrl = '/api/v1/org/departments';

  private normalizeDepartment(department: RawDepartment): Department {
    return {
      id: department.id || 0,
      name: department.name || '',
      code: department.code || '',
      description: department.description,
      managerId: department.managerId,
      parentId: department.parentId,
      children: department.children?.map(child => this.normalizeDepartment(child)),
      createdAt: department.createdAt,
      updatedAt: department.updatedAt,
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

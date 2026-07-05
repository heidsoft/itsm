/**
 * 通用系统类型定义
 */

export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
  role: string;
  department?: string;
  departmentId?: number;
  phone?: string;
  active: boolean;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

export interface Department {
  id: number;
  name: string;
  code: string;
  description?: string;
  managerId?: number;
  parentId?: number;
  tenantId: number;
  children?: Department[];
  createdAt: string;
  updatedAt: string;
}

export interface Team {
  id: number;
  name: string;
  code: string;
  description?: string;
  status: string;
  managerId?: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

export interface Tag {
  id: number;
  name: string;
  code: string;
  description?: string;
  color: string;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

export interface AuditLog {
  id: number;
  createdAt: string;
  tenantId: number;
  userId: number;
  requestId: string;
  ip: string;
  resource: string;
  action: string;
  path: string;
  method: string;
  statusCode: number;
  requestBody?: string;
}

export interface AuthResult {
  accessToken: string;
  refreshToken: string;
  user: User;
}

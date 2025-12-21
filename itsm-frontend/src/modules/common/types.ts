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
    department_id?: number;
    phone?: string;
    active: boolean;
    tenant_id: number;
    created_at: string;
    updated_at: string;
}

export interface Department {
    id: number;
    name: string;
    code: string;
    description?: string;
    manager_id?: number;
    parent_id?: number;
    tenant_id: number;
    children?: Department[];
    created_at: string;
    updated_at: string;
}

export interface Team {
    id: number;
    name: string;
    code: string;
    description?: string;
    status: string;
    manager_id?: number;
    tenant_id: number;
    created_at: string;
    updated_at: string;
}

export interface Tag {
    id: number;
    name: string;
    code: string;
    description?: string;
    color: string;
    tenant_id: number;
    created_at: string;
    updated_at: string;
}

export interface AuditLog {
    id: number;
    created_at: string;
    tenant_id: number;
    user_id: number;
    request_id: string;
    ip: string;
    resource: string;
    action: string;
    path: string;
    method: string;
    status_code: number;
    request_body?: string;
}

export interface AuthResult {
    access_token: string;
    refresh_token: string;
    user: User;
}

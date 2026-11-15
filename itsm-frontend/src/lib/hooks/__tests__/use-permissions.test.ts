import { renderHook } from '@testing-library/react';
import {
  usePermissions, useRoutePermissions, useOperationPermissions,
  usePermissionGuard,
} from '../use-permissions';
import { PERM } from '@/lib/permissions';

// 契约（2026-11-14 权限码单点化批次）：
//  1. 后端下发的 user.permissions 是唯一事实来源，缺权限即 fail-closed；
//  2. hasPermission 只接受生成的权限码（PERM.*），不接受 (resource, action) 两参；
//  3. fixture 只允许出现后端码空间里的真实码——幽灵码（ticket:reopen、
//     incident:declare_major 等）在真实环境永远不可能被授予，出现在 fixture
//     里会掩盖「按钮永远不渲染」的缺陷。

const adminPermissions = [
  'ticket:read', 'ticket:create', 'ticket:update', 'ticket:delete',
  'ticket:assign', 'ticket:escalate', 'ticket:resolve', 'ticket:close',
  'ticket:export',
  'incident:read', 'incident:write',
  'problem:read', 'problem:write',
  'change:read', 'change:write', 'change:approve',
  'knowledge:read', 'knowledge:write',
  'cmdb:read', 'cmdb:write',
  'user:read', 'user:write',
  'role:read', 'role:write',
  'system_config:read', 'system_config:write',
  'audit:read',
  'report:read',
];

const mockUser = {
  role: 'admin',
  id: 1,
  name: 'Admin User',
  permissions: adminPermissions,
};

const makeState = (user: typeof mockUser | null) => ({
  user,
  hasPermission: (permission: string) =>
    !!user?.permissions?.includes(permission),
});

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: Object.assign(
    jest.fn(() => makeState(mockUser)),
    { getState: () => makeState(mockUser) },
  ),
}));

import { useAuthStore } from '@/lib/store/auth-store';

const mockStore = (user: typeof mockUser | null) => {
  (useAuthStore as unknown as jest.Mock).mockReturnValue(makeState(user));
  (useAuthStore as unknown as { getState: () => unknown }).getState = () => makeState(user);
};

describe('usePermissions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockStore(mockUser);
  });

  it('should return permissions for admin role', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.userPermissions.length).toBeGreaterThan(0);
    expect(result.current.userRoles).toEqual(['admin']);
  });

  it('should check hasPermission with generated codes', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasPermission(PERM.ticket.read)).toBe(true);
    expect(result.current.hasPermission(PERM.ticket.create)).toBe(true);
    // fixture 未含 user:delete → fail-closed
    expect(result.current.hasPermission(PERM.user.delete)).toBe(false);
  });

  it('should fail closed for malformed codes', () => {
    const { result } = renderHook(() => usePermissions());
    // 类型层已拦截非法码；运行时对畸形串兜底拒绝
    expect(result.current.hasPermission('ticket' as never)).toBe(false);
  });

  it('should honor wildcard grants like the backend matcher', () => {
    const wildcardUser = {
      role: 'super_admin',
      id: 9,
      name: 'Super',
      permissions: ['*'],
    };
    mockStore(wildcardUser as unknown as typeof mockUser);
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasPermission(PERM.ticket.read)).toBe(true);
    expect(result.current.hasPermission(PERM.msp.write)).toBe(true);
  });

  it('should check hasRole correctly', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasRole('admin')).toBe(true);
    expect(result.current.hasRole('end_user')).toBe(false);
  });

  it('should check hasAnyRole', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasAnyRole(['admin', 'end_user'])).toBe(true);
    expect(result.current.hasAnyRole(['end_user', 'technician'])).toBe(false);
  });

  it('should check hasAllRoles', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasAllRoles(['admin'])).toBe(true);
    expect(result.current.hasAllRoles(['admin', 'end_user'])).toBe(false);
  });

  it('should check isAdmin correctly', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.isAdmin()).toBe(true);
  });

  it('should check isSuperAdmin correctly', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.isSuperAdmin()).toBe(false);
  });

  it('should return true for isSuperAdmin with super_admin role', () => {
    mockStore({ ...mockUser, role: 'super_admin' });
    const { result } = renderHook(() => usePermissions());
    expect(result.current.isSuperAdmin()).toBe(true);
    expect(result.current.isAdmin()).toBe(true);
  });

  it('should return empty permissions when no user (fail-closed)', () => {
    mockStore(null);
    const { result } = renderHook(() => usePermissions());
    expect(result.current.userPermissions).toEqual([]);
    expect(result.current.userRoles).toEqual([]);
  });

  it('should return empty userRoles when user has no role', () => {
    const noRoleUser = { ...mockUser, role: '' };
    mockStore(noRoleUser as typeof mockUser);
    const { result } = renderHook(() => usePermissions());
    // 权限来自后端下发，与角色名无关，仍保留
    expect(result.current.userPermissions.length).toBeGreaterThan(0);
    expect(result.current.userRoles).toEqual([]);
  });

  it('should get available actions for resource', () => {
    const { result } = renderHook(() => usePermissions());
    const ticketActions = result.current.getAvailableActions('ticket');
    expect(ticketActions).toContain('read');
    expect(ticketActions).toContain('create');
  });

  it('should return empty actions for nonexistent resource', () => {
    const { result } = renderHook(() => usePermissions());
    const actions = result.current.getAvailableActions('nonexistent');
    expect(actions).toEqual([]);
  });

  it('should check canAccessRoute with route permissions', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.canAccessRoute([{ resource: 'ticket', action: 'read' }])).toBe(true);
    expect(result.current.canAccessRoute([{ resource: 'ticket', action: 'write' }])).toBe(false);
  });

  it('should check canAccessRoute with required roles', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.canAccessRoute([], ['admin'])).toBe(true);
    expect(result.current.canAccessRoute([], ['super_admin'])).toBe(false);
  });

  it('should check canAccessRoute with empty permissions and roles', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.canAccessRoute([])).toBe(true);
  });

  it('should grant only backend-issued permissions (no static role table)', () => {
    // 新契约：角色名不再决定权限，只有后端下发的 permissions 生效
    const limitedUser = {
      role: 'technician',
      id: 2,
      name: 'Tech',
      permissions: ['ticket:read', 'ticket:update'],
    };
    mockStore(limitedUser as unknown as typeof mockUser);
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasPermission(PERM.ticket.read)).toBe(true);
    expect(result.current.hasPermission(PERM.ticket.update)).toBe(true);
    // 未下发的权限一律拒绝，即使角色听起来应该有
    expect(result.current.hasPermission(PERM.ticket.create)).toBe(false);
    expect(result.current.hasPermission(PERM.user.write)).toBe(false);
  });

  it('should be fail-closed when permissions missing entirely', () => {
    const noPermUser = { role: 'admin', id: 3, name: 'No Perm' };
    (useAuthStore as unknown as jest.Mock).mockReturnValue(makeState(noPermUser as unknown as typeof mockUser));
    (useAuthStore as unknown as { getState: () => unknown }).getState = () => makeState(noPermUser as unknown as typeof mockUser);
    const { result } = renderHook(() => usePermissions());
    expect(result.current.userPermissions).toEqual([]);
    expect(result.current.hasPermission(PERM.ticket.read)).toBe(false);
  });
});

describe('useRoutePermissions', () => {
  beforeEach(() => {
    mockStore(mockUser);
  });

  it('should check route permission when no requirements', () => {
    const { result } = renderHook(() => useRoutePermissions());
    expect(result.current.checkRoutePermission()).toBe(true);
  });

  it('should check route with permissions', () => {
    const { result } = renderHook(() => useRoutePermissions());
    expect(result.current.checkRoutePermission([{ resource: 'ticket', action: 'read' }])).toBe(true);
  });

  it('should check route with roles', () => {
    const { result } = renderHook(() => useRoutePermissions());
    expect(result.current.checkRoutePermission(undefined, ['admin'])).toBe(true);
    expect(result.current.checkRoutePermission(undefined, ['super_admin'])).toBe(false);
  });

  it('should return userPermissions and userRoles', () => {
    const { result } = renderHook(() => useRoutePermissions());
    expect(result.current.userPermissions.length).toBeGreaterThan(0);
    expect(result.current.userRoles).toEqual(['admin']);
  });
});

describe('useOperationPermissions', () => {
  beforeEach(() => {
    mockStore(mockUser);
  });

  it('should return ticket permissions aligned with backend routes', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.ticket.canView()).toBe(true);
    expect(result.current.ticket.canCreate()).toBe(true);
    expect(result.current.ticket.canDelete()).toBe(true);
    expect(result.current.ticket.canAssign()).toBe(true);
    expect(result.current.ticket.canEscalate()).toBe(true);
    // resolve/close 双路由族：update 或专属码任一命中
    expect(result.current.ticket.canResolve()).toBe(true);
    expect(result.current.ticket.canClose()).toBe(true);
    expect(result.current.ticket.canExport()).toBe(true);
  });

  it('should return incident permissions (write-family codes)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.incident.canView()).toBe(true);
    expect(result.current.incident.canCreate()).toBe(true);
    expect(result.current.incident.canDeclareAsMajor()).toBe(true);
  });

  it('should return problem permissions (write-family codes)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.problem.canView()).toBe(true);
    expect(result.current.problem.canCreate()).toBe(true);
  });

  it('should return change permissions (reject rides approve)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.change.canView()).toBe(true);
    expect(result.current.change.canApprove()).toBe(true);
    expect(result.current.change.canReject()).toBe(true);
  });

  it('should return knowledge permissions (publish rides write)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.knowledge.canView()).toBe(true);
    expect(result.current.knowledge.canCreate()).toBe(true);
    expect(result.current.knowledge.canPublish()).toBe(true);
  });

  it('should return cmdb permissions (manage/import/export ride write)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.cmdb.canView()).toBe(true);
    expect(result.current.cmdb.canManage()).toBe(true);
  });

  it('should return user management permissions (write-family codes)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.user.canView()).toBe(true);
    expect(result.current.user.canCreate()).toBe(true);
    expect(result.current.user.canManage()).toBe(true);
    expect(result.current.user.canResetPassword()).toBe(true);
  });

  it('should return role management permissions (write-family codes)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.role.canView()).toBe(true);
    expect(result.current.role.canManage()).toBe(true);
  });

  it('should return system permissions on system_config/audit codes', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.system.canViewSettings()).toBe(true);
    expect(result.current.system.canManage()).toBe(true);
    expect(result.current.system.canViewLogs()).toBe(true);
  });

  it('should return report permissions (export rides read)', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.report.canView()).toBe(true);
    expect(result.current.report.canExport()).toBe(true);
  });

  it('should limit permissions to backend-issued set for restricted users', () => {
    const limitedUser = {
      role: 'end_user',
      id: 4,
      name: 'End User',
      permissions: ['ticket:read', 'ticket:create'],
    };
    mockStore(limitedUser as unknown as typeof mockUser);
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.ticket.canView()).toBe(true);
    expect(result.current.ticket.canCreate()).toBe(true);
    expect(result.current.ticket.canDelete()).toBe(false);
    expect(result.current.change.canApprove()).toBe(false);
    expect(result.current.user.canManage()).toBe(false);
    expect(result.current.system.canManage()).toBe(false);
  });

  it('should hide reopen for holders without ticket:close (phantom-code regression)', () => {
    // 回归锁：reopen 曾挂在前端臆造的 ticket:reopen 上，持有真实路由码
    // ticket:close 的用户反而看不到按钮；现在 reopen 与后端路由同码。
    const closeOnlyUser = {
      role: 'agent',
      id: 5,
      name: 'Agent',
      permissions: ['ticket:read', 'ticket:close'],
    };
    mockStore(closeOnlyUser as unknown as typeof mockUser);
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.ticket.canReopen()).toBe(true);
  });
});

describe('usePermissionGuard', () => {
  beforeEach(() => {
    mockStore(mockUser);
  });

  it('should return true when has permission', () => {
    const { result } = renderHook(() => usePermissionGuard());
    expect(result.current.guard([{ resource: 'ticket', action: 'read' }])).toBe(true);
  });

  it('should return false and call fallback when no permission', () => {
    const { result } = renderHook(() => usePermissionGuard());
    const fallback = jest.fn();
    // fixture 未含 ticket:import
    expect(result.current.guard([{ resource: 'ticket', action: 'import' }], undefined, fallback)).toBe(false);
    expect(fallback).toHaveBeenCalled();
  });

  it('should not call fallback when has access', () => {
    const { result } = renderHook(() => usePermissionGuard());
    const fallback = jest.fn();
    result.current.guard([{ resource: 'ticket', action: 'read' }], undefined, fallback);
    expect(fallback).not.toHaveBeenCalled();
  });

  it('guardOperation should check generated codes', () => {
    const { result } = renderHook(() => usePermissionGuard());
    expect(result.current.guardOperation(PERM.ticket.read)).toBe(true);
    expect(result.current.guardOperation(PERM.ticket.import)).toBe(false);
  });

  it('guardOperation should call onDenied when no access', () => {
    const { result } = renderHook(() => usePermissionGuard());
    const onDenied = jest.fn();
    result.current.guardOperation(PERM.ticket.import, onDenied);
    expect(onDenied).toHaveBeenCalled();
  });

  it('guardRole should check roles', () => {
    const { result } = renderHook(() => usePermissionGuard());
    expect(result.current.guardRole(['admin'])).toBe(true);
    expect(result.current.guardRole(['super_admin'])).toBe(false);
  });

  it('guardRole should call onDenied when no access', () => {
    const { result } = renderHook(() => usePermissionGuard());
    const onDenied = jest.fn();
    result.current.guardRole(['super_admin'], onDenied);
    expect(onDenied).toHaveBeenCalled();
  });
});

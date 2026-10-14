import { renderHook } from '@testing-library/react';
import {
  usePermissions, useRoutePermissions, useOperationPermissions,
  usePermissionGuard,
} from '../use-permissions';

// 契约：后端下发的 user.permissions 是唯一事实来源，缺权限即 fail-closed，
// 前端不维护角色到权限的静态表。因此：
//  1. mock 用户必须携带 permissions 数组（模拟后端 login/capabilities 下发）；
//  2. hasPermission 走 useAuthStore.getState()，mock 工厂必须提供 getState；
//  3. 「角色 X 应有权限 Y」的旧断言不再成立——权限由后端 role_permission 配置决定，
//     前端只负责透传与 fail-closed。

const adminPermissions = [
  'ticket:read', 'ticket:create', 'ticket:update', 'ticket:delete',
  'ticket:assign', 'ticket:escalate', 'ticket:resolve', 'ticket:close',
  'ticket:reopen', 'ticket:export',
  'incident:read', 'incident:create', 'incident:declare_major',
  'problem:read', 'problem:create',
  'change:read', 'change:approve', 'change:reject', 'change:review',
  'knowledge:read', 'knowledge:create', 'knowledge:publish',
  'cmdb:read', 'cmdb:manage',
  'user:read', 'user:create', 'user:manage',
  'role:read', 'role:manage',
  'system:read', 'system:manage',
  'report:read', 'report:export', 'report:create',
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

  it('should check hasPermission correctly', () => {
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasPermission('ticket', 'read')).toBe(true);
    expect(result.current.hasPermission('ticket', 'create')).toBe(true);
    expect(result.current.hasPermission('nonexistent', 'action')).toBe(false);
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
    expect(result.current.canAccessRoute([{ resource: 'nonexistent', action: 'do' }])).toBe(false);
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

  it('should check canBatchOperate', () => {
    const { result } = renderHook(() => usePermissions());
    // mock 权限集未下发 batch_delete，fail-closed 应拒绝
    expect(result.current.canBatchOperate('ticket', 'delete')).toBe(false);
  });

  it('should grant only backend-issued permissions (no static role table)', () => {
    // 新契约：角色名不再决定权限，只有后端下发的 permissions 生效
    const limitedUser = {
      role: 'technician',
      id: 2,
      name: 'Tech',
      permissions: ['ticket:read', 'ticket:resolve'],
    };
    mockStore(limitedUser as unknown as typeof mockUser);
    const { result } = renderHook(() => usePermissions());
    expect(result.current.hasPermission('ticket', 'read')).toBe(true);
    expect(result.current.hasPermission('ticket', 'resolve')).toBe(true);
    // 未下发的权限一律拒绝，即使角色听起来应该有
    expect(result.current.hasPermission('ticket', 'create')).toBe(false);
    expect(result.current.hasPermission('user', 'manage')).toBe(false);
  });

  it('should be fail-closed when permissions missing entirely', () => {
    const noPermUser = { role: 'admin', id: 3, name: 'No Perm' };
    (useAuthStore as unknown as jest.Mock).mockReturnValue(makeState(noPermUser as unknown as typeof mockUser));
    (useAuthStore as unknown as { getState: () => unknown }).getState = () => makeState(noPermUser as unknown as typeof mockUser);
    const { result } = renderHook(() => usePermissions());
    expect(result.current.userPermissions).toEqual([]);
    expect(result.current.hasPermission('ticket', 'read')).toBe(false);
  });

  it('should handle superadmin role alias (permissions still required)', () => {
    mockStore({ ...mockUser, role: 'superadmin' });
    const { result } = renderHook(() => usePermissions());
    // 角色别名不影响权限判断：权限来自后端下发
    expect(result.current.hasPermission('user', 'manage')).toBe(true);
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

  it('should return ticket permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.ticket.canView()).toBe(true);
    expect(result.current.ticket.canCreate()).toBe(true);
    expect(result.current.ticket.canDelete()).toBe(true);
    expect(result.current.ticket.canAssign()).toBe(true);
    expect(result.current.ticket.canEscalate()).toBe(true);
    expect(result.current.ticket.canResolve()).toBe(true);
    expect(result.current.ticket.canClose()).toBe(true);
    expect(result.current.ticket.canReopen()).toBe(true);
    expect(result.current.ticket.canExport()).toBe(true);
  });

  it('should return incident permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.incident.canView()).toBe(true);
    expect(result.current.incident.canCreate()).toBe(true);
    expect(result.current.incident.canDeclareAsMajor()).toBe(true);
  });

  it('should return problem permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.problem.canView()).toBe(true);
    expect(result.current.problem.canCreate()).toBe(true);
  });

  it('should return change permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.change.canView()).toBe(true);
    expect(result.current.change.canApprove()).toBe(true);
    expect(result.current.change.canReject()).toBe(true);
    expect(result.current.change.canReview()).toBe(true);
  });

  it('should return knowledge permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.knowledge.canView()).toBe(true);
    expect(result.current.knowledge.canCreate()).toBe(true);
    expect(result.current.knowledge.canPublish()).toBe(true);
  });

  it('should return cmdb permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.cmdb.canView()).toBe(true);
    expect(result.current.cmdb.canManage()).toBe(true);
  });

  it('should return user management permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.user.canView()).toBe(true);
    expect(result.current.user.canCreate()).toBe(true);
    expect(result.current.user.canManage()).toBe(true);
  });

  it('should return role management permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.role.canView()).toBe(true);
    expect(result.current.role.canManage()).toBe(true);
  });

  it('should return system permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.system.canViewSettings()).toBe(true);
    expect(result.current.system.canManage()).toBe(true);
  });

  it('should return report permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());
    expect(result.current.report.canView()).toBe(true);
    expect(result.current.report.canExport()).toBe(true);
    expect(result.current.report.canCreate()).toBe(true);
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
    expect(result.current.guard([{ resource: 'nonexistent', action: 'do' }], undefined, fallback)).toBe(false);
    expect(fallback).toHaveBeenCalled();
  });

  it('should not call fallback when has access', () => {
    const { result } = renderHook(() => usePermissionGuard());
    const fallback = jest.fn();
    result.current.guard([{ resource: 'ticket', action: 'read' }], undefined, fallback);
    expect(fallback).not.toHaveBeenCalled();
  });

  it('guardOperation should check resource/action', () => {
    const { result } = renderHook(() => usePermissionGuard());
    expect(result.current.guardOperation('ticket', 'read')).toBe(true);
    expect(result.current.guardOperation('nonexistent', 'do')).toBe(false);
  });

  it('guardOperation should call onDenied when no access', () => {
    const { result } = renderHook(() => usePermissionGuard());
    const onDenied = jest.fn();
    result.current.guardOperation('nonexistent', 'do', onDenied);
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

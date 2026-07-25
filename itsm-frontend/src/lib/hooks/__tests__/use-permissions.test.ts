import { renderHook } from '@testing-library/react';
import { usePermissions, useRoutePermissions, useOperationPermissions, PERMISSIONS, ROLES } from '../use-permissions';

// Mock the auth store
const mockUser = { role: 'admin', id: 1, name: 'Admin User' };

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: jest.fn(() => ({ user: mockUser })),
}));

import { useAuthStore } from '@/lib/store/auth-store';

describe('usePermissions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (useAuthStore as unknown as jest.Mock).mockReturnValue({ user: mockUser });
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

  it('should check isAdmin correctly', () => {
    const { result } = renderHook(() => usePermissions());

    expect(result.current.isAdmin()).toBe(true);
  });

  it('should check isSuperAdmin correctly', () => {
    const { result } = renderHook(() => usePermissions());

    expect(result.current.isSuperAdmin()).toBe(false);
  });

  it('should return empty permissions when no user', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({ user: null });

    const { result } = renderHook(() => usePermissions());

    expect(result.current.userPermissions).toEqual([]);
    expect(result.current.userRoles).toEqual([]);
  });

  it('should get available actions for resource', () => {
    const { result } = renderHook(() => usePermissions());

    const ticketActions = result.current.getAvailableActions('ticket');
    expect(ticketActions).toContain('read');
    expect(ticketActions).toContain('create');
  });

  it('should check canAccessRoute', () => {
    const { result } = renderHook(() => usePermissions());

    expect(
      result.current.canAccessRoute([{ resource: 'ticket', action: 'read' }])
    ).toBe(true);

    expect(
      result.current.canAccessRoute([{ resource: 'nonexistent', action: 'do' }])
    ).toBe(false);
  });
});

describe('useOperationPermissions', () => {
  beforeEach(() => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({ user: mockUser });
  });

  it('should return ticket permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());

    expect(result.current.ticket.canView()).toBe(true);
    expect(result.current.ticket.canCreate()).toBe(true);
    expect(result.current.ticket.canDelete()).toBe(true);
  });

  it('should return knowledge permissions', () => {
    const { result } = renderHook(() => useOperationPermissions());

    expect(result.current.knowledge.canView()).toBe(true);
    expect(result.current.knowledge.canCreate()).toBe(true);
  });
});

describe('PERMISSIONS constants', () => {
  it('should have correct ticket permissions', () => {
    expect(PERMISSIONS.TICKET.READ).toEqual({ resource: 'ticket', action: 'read' });
    expect(PERMISSIONS.TICKET.CREATE).toEqual({ resource: 'ticket', action: 'create' });
  });
});

describe('ROLES constants', () => {
  it('should have correct role values', () => {
    expect(ROLES.SUPER_ADMIN).toBe('super_admin');
    expect(ROLES.ADMIN).toBe('admin');
    expect(ROLES.END_USER).toBe('end_user');
  });
});

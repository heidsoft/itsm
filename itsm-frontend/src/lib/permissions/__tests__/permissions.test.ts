/**
 * 权限模块单元测试
 * TDD RED 阶段 - 先写测试
 */

import {
  parsePermissions,
  hasWildcardPermission,
  matchesPermission,
  hasAllPermissions,
  hasAnyPermission,
} from '../utils';
import type { Permission } from '../types';

describe('parsePermissions', () => {
  it('should parse permission strings into a Set', () => {
    const permissions = ['ticket:read', 'ticket:create', 'incident:read'];
    const result = parsePermissions(permissions);

    expect(result).toBeInstanceOf(Set);
    expect(result.size).toBe(3);
    expect(result.has('ticket:read')).toBe(true);
    expect(result.has('ticket:create')).toBe(true);
    expect(result.has('incident:read')).toBe(true);
  });

  it('should return empty Set for empty array', () => {
    const result = parsePermissions([]);
    expect(result.size).toBe(0);
  });
});

describe('hasWildcardPermission', () => {
  it('should return true for "*" wildcard', () => {
    const permissions = new Set(['*']);
    expect(hasWildcardPermission(permissions)).toBe(true);
  });

  it('should return true for "*:*" wildcard', () => {
    const permissions = new Set(['*:*']);
    expect(hasWildcardPermission(permissions)).toBe(true);
  });

  it('should return false for non-wildcard permissions', () => {
    const permissions = new Set(['ticket:read', 'ticket:create']);
    expect(hasWildcardPermission(permissions)).toBe(false);
  });

  it('should return false for empty Set', () => {
    const permissions = new Set<string>();
    expect(hasWildcardPermission(permissions)).toBe(false);
  });
});

describe('matchesPermission', () => {
  describe('exact match', () => {
    it('should return true when permission matches exactly', () => {
      const permissions = new Set(['ticket:read', 'ticket:create']);
      expect(matchesPermission(permissions, 'ticket', 'read')).toBe(true);
      expect(matchesPermission(permissions, 'ticket', 'create')).toBe(true);
    });

    it('should return false when permission does not match', () => {
      const permissions = new Set(['ticket:read']);
      expect(matchesPermission(permissions, 'ticket', 'delete')).toBe(false);
      expect(matchesPermission(permissions, 'incident', 'read')).toBe(false);
    });
  });

  describe('wildcard match', () => {
    it('should match "*" wildcard for any permission', () => {
      const permissions = new Set(['*']);
      expect(matchesPermission(permissions, 'ticket', 'read')).toBe(true);
      expect(matchesPermission(permissions, 'incident', 'delete')).toBe(true);
      expect(matchesPermission(permissions, 'any', 'action')).toBe(true);
    });

    it('should match "*:*" wildcard for any permission', () => {
      const permissions = new Set(['*:*']);
      expect(matchesPermission(permissions, 'ticket', 'read')).toBe(true);
    });

    it('should match resource wildcard "resource:*"', () => {
      const permissions = new Set(['ticket:*']);
      expect(matchesPermission(permissions, 'ticket', 'read')).toBe(true);
      expect(matchesPermission(permissions, 'ticket', 'create')).toBe(true);
      expect(matchesPermission(permissions, 'ticket', 'delete')).toBe(true);
      expect(matchesPermission(permissions, 'incident', 'read')).toBe(false);
    });

    it('should match action wildcard "*:action"', () => {
      const permissions = new Set(['*:read']);
      expect(matchesPermission(permissions, 'ticket', 'read')).toBe(true);
      expect(matchesPermission(permissions, 'incident', 'read')).toBe(true);
      expect(matchesPermission(permissions, 'ticket', 'create')).toBe(false);
    });
  });
});

describe('hasAllPermissions', () => {
  it('should return true when all permissions are present', () => {
    const permissions = new Set(['ticket:read', 'ticket:create', 'ticket:update']);
    const required: Permission[] = [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'create' },
    ];
    expect(hasAllPermissions(permissions, required)).toBe(true);
  });

  it('should return false when any permission is missing', () => {
    const permissions = new Set(['ticket:read']);
    const required: Permission[] = [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'delete' },
    ];
    expect(hasAllPermissions(permissions, required)).toBe(false);
  });

  it('should return true for empty required array', () => {
    const permissions = new Set(['ticket:read']);
    expect(hasAllPermissions(permissions, [])).toBe(true);
  });

  it('should work with wildcard permissions', () => {
    const permissions = new Set(['*']);
    const required: Permission[] = [
      { resource: 'ticket', action: 'read' },
      { resource: 'incident', action: 'create' },
    ];
    expect(hasAllPermissions(permissions, required)).toBe(true);
  });
});

describe('hasAnyPermission', () => {
  it('should return true when any permission is present', () => {
    const permissions = new Set(['ticket:read']);
    const required: Permission[] = [
      { resource: 'ticket', action: 'read' },
      { resource: 'ticket', action: 'delete' },
    ];
    expect(hasAnyPermission(permissions, required)).toBe(true);
  });

  it('should return false when no permissions are present', () => {
    const permissions = new Set(['ticket:read']);
    const required: Permission[] = [
      { resource: 'ticket', action: 'delete' },
      { resource: 'incident', action: 'read' },
    ];
    expect(hasAnyPermission(permissions, required)).toBe(false);
  });

  it('should return false for empty required array', () => {
    const permissions = new Set(['ticket:read']);
    expect(hasAnyPermission(permissions, [])).toBe(false);
  });

  it('should work with wildcard permissions', () => {
    const permissions = new Set(['ticket:*']);
    const required: Permission[] = [
      { resource: 'ticket', action: 'read' },
      { resource: 'incident', action: 'create' },
    ];
    expect(hasAnyPermission(permissions, required)).toBe(true);
  });
});

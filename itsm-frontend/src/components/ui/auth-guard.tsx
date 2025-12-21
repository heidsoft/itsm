'use client';

import React from 'react';
import { useAuthStore } from '@/lib/store/auth-store';

export interface AuthGuardProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  redirectTo?: string;
  requireAuth?: boolean;
  requiredRoles?: string[];
  requiredPermissions?: string[];
  requireAll?: boolean;
}

export const AuthGuard: React.FC<AuthGuardProps> = ({
  children,
  fallback = null,
  redirectTo = '/login',
  requireAuth = true,
  requiredRoles = [],
  requiredPermissions = [],
  requireAll = true,
}) => {
  const { user, isAuthenticated } = useAuthStore();

  if (requireAuth && !isAuthenticated) {
    if (typeof window !== 'undefined') window.location.href = redirectTo;
    return fallback;
  }

  if (requiredRoles.length > 0) {
    const ok = requireAll
      ? requiredRoles.every(r => user?.role === r)
      : requiredRoles.some(r => user?.role === r);
    if (!ok) return fallback;
  }

  if (requiredPermissions.length > 0) {
    const perms = user?.permissions || [];
    const ok = requireAll
      ? requiredPermissions.every(p => perms.includes(p))
      : requiredPermissions.some(p => perms.includes(p));
    if (!ok) return fallback;
  }

  return <>{children}</>;
};



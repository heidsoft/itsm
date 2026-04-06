// Re-export from authoritative location
// This file exists for backwards compatibility - import from @/components/auth/AuthGuard instead
export {
  AuthGuard,
  PermissionGuard,
  RoleGuard,
  OperationGuard,
  AdminGuard,
  SuperAdminGuard,
  ConditionalGuard,
  withAuth,
  withPermission,
  withRole,
  withAdmin,
} from '@/components/auth/AuthGuard';

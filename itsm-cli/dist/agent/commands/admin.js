import { cmd } from '../spec.js';
import { requirePositional, withPagination } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
/** Admin/observability surface for agents: dashboard, users, tenants, RBAC, audit, MSP, command bus. */
export function adminCommands() {
    return [
        cmd('admin.overview', 'Dashboard overview', 'itsm admin overview', ['GET /api/v1/dashboard/overview'], async () => apiClient.getDashboardOverview()),
        cmd('admin.users', 'List users', 'itsm admin users [--page N]', ['GET /api/v1/users'], async (ctx) => apiClient.listUsers(withPagination(ctx.args))),
        cmd('admin.user', 'User detail', 'itsm admin user <id>', ['GET /api/v1/users/:id'], async (ctx) => apiClient.getUser(requirePositional(ctx.args, 0, 'userId'))),
        cmd('admin.teams', 'List teams', 'itsm admin teams', ['GET /api/v1/teams'], async () => apiClient.listTeams()),
        cmd('admin.departments', 'List departments', 'itsm admin departments', ['GET /api/v1/departments'], async () => apiClient.listDepartments()),
        cmd('admin.roles', 'List roles', 'itsm admin roles', ['GET /api/v1/roles'], async (ctx) => apiClient.listRoles(withPagination(ctx.args))),
        cmd('admin.permissions', 'List permission codes', 'itsm admin permissions', ['GET /api/v1/permissions'], async () => apiClient.listPermissions()),
        cmd('admin.tenants', 'List tenants', 'itsm admin tenants', ['GET /api/v1/tenants'], async (ctx) => apiClient.listTenants(withPagination(ctx.args))),
        cmd('admin.tenant', 'Tenant detail', 'itsm admin tenant <id>', ['GET /api/v1/tenants/:id'], async (ctx) => apiClient.getTenant(requirePositional(ctx.args, 0, 'tenantId'))),
        cmd('admin.audit-logs', 'Audit trail (filterable)', 'itsm admin audit-logs [--page N]', ['GET /api/v1/audit-logs'], async (ctx) => apiClient.listAuditLogs(withPagination(ctx.args))),
        cmd('admin.msp-status', 'MSP mode status', 'itsm admin msp-status', ['GET /api/v1/msp/status'], async () => apiClient.getMSPStatus()),
        cmd('admin.msp-context', 'MSP operating context (operator/customer/delegated scope)', 'itsm admin msp-context', ['GET /api/v1/msp/context'], async () => apiClient.getMSPContext()),
        cmd('admin.msp-customers', 'MSP customer list', 'itsm admin msp-customers', ['GET /api/v1/msp/customers'], async () => apiClient.listMSPCustomers()),
        cmd('admin.msp-allocations', 'Allocated feed entries', 'itsm admin msp-allocations', ['GET /api/v1/msp/allocations'], async () => apiClient.listMSPAllocations()),
    ];
}

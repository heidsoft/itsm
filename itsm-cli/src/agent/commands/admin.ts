import { cmd } from '../spec.js';
import { requirePositional, withPagination, stringFlag, numberFlag, booleanFlag, optionalString } from '../helpers.js';
import type { ArgContext } from '../args.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/** Admin/observability surface for agents: dashboard, users, tenants, RBAC, audit, MSP, command bus. */

export function adminCommands() {
  return [
    cmd('admin.overview', 'Dashboard overview', 'itsm admin overview', ['GET /api/v1/dashboard/overview'], async () => apiClient.getDashboardOverview()),
    cmd('admin.users', 'List users', 'itsm admin users [--page N] [--pageSize N]', ['GET /api/v1/users'], async (ctx) => apiClient.listUsers(withPagination(ctx.args))),
    cmd('admin.user', 'User detail', 'itsm admin user <id>', ['GET /api/v1/users/:id'], async (ctx) => apiClient.getUser(requirePositional(ctx.args, 0, 'userId'))),
    cmd('admin.teams', 'List teams', 'itsm admin teams', ['GET /api/v1/teams'], async () => apiClient.listTeams()),
    cmd('admin.departments', 'List departments', 'itsm admin departments', ['GET /api/v1/departments'], async () => apiClient.listDepartments()),
    cmd('admin.roles', 'List roles', 'itsm admin roles [--page N] [--pageSize N]', ['GET /api/v1/roles'], async (ctx) => apiClient.listRoles(withPagination(ctx.args))),
    cmd('admin.permissions', 'List permission codes', 'itsm admin permissions', ['GET /api/v1/permissions'], async () => apiClient.listPermissions()),
    cmd('admin.tenants', 'List tenants', 'itsm admin tenants [--page N] [--pageSize N]', ['GET /api/v1/tenants'], async (ctx) => apiClient.listTenants(withPagination(ctx.args))),
    cmd('admin.tenant', 'Tenant detail', 'itsm admin tenant <id>', ['GET /api/v1/tenants/:id'], async (ctx) => apiClient.getTenant(requirePositional(ctx.args, 0, 'tenantId'))),
    cmd('admin.audit-logs', 'Audit trail (filterable)', 'itsm admin audit-logs [--page N] [--pageSize N]', ['GET /api/v1/audit-logs'], async (ctx) => apiClient.listAuditLogs(withPagination(ctx.args))),
    cmd('admin.msp-status', 'MSP mode status', 'itsm admin msp-status', ['GET /api/v1/msp/status'], async () => apiClient.getMSPStatus()),
    cmd('admin.msp-context', 'MSP operating context (operator/customer/delegated scope)', 'itsm admin msp-context', ['GET /api/v1/msp/context'], async () => apiClient.getMSPContext()),
    cmd('admin.msp-customers', 'MSP customer list', 'itsm admin msp-customers', ['GET /api/v1/msp/customers'], async () => apiClient.listMSPCustomers()),
    cmd('admin.msp-allocations', 'Allocated feed entries', 'itsm admin msp-allocations', ['GET /api/v1/msp/allocations'], async () => apiClient.listMSPAllocations()),

    // Durable command bus: inspect and recover side effects (notifications, webhooks, BPMN).
    cmd(
      'admin.commands',
      'List durable operational commands (filter by status/commandType/aggregateType)',
      'itsm admin commands [--status s] [--commandType t] [--aggregateType t] [--page N] [--pageSize N]',
      ['GET /api/v1/admin/operations/commands'],
      async (ctx) =>
        apiClient.listOperationalCommands({
          ...withPagination(ctx.args),
          status: optionalString(ctx.args.flags, 'status'),
          commandType: optionalString(ctx.args.flags, 'commandType'),
          aggregateType: optionalString(ctx.args.flags, 'aggregateType'),
        })
    ),
    cmd('admin.command', 'One durable command (attempt, lease, last safe error)', 'itsm admin command <id>', ['GET /api/v1/admin/operations/commands/:id'], async (ctx) =>
      apiClient.getOperationalCommand(requirePositional(ctx.args, 0, 'commandId'))
    ),
    cmd('admin.command-replay', 'Replay a dead-letter/failed command (preserves idempotency key)', 'itsm admin command-replay <id>', ['POST /api/v1/admin/operations/commands/:id/replay'], async (ctx) =>
      apiClient.replayOperationalCommand(requirePositional(ctx.args, 0, 'commandId'))
    ),
    cmd('admin.command-cancel', 'Cancel a pending command', 'itsm admin command-cancel <id>', ['POST /api/v1/admin/operations/commands/:id/cancel'], async (ctx) =>
      apiClient.cancelOperationalCommand(requirePositional(ctx.args, 0, 'commandId'))
    ),
    cmd(
      'admin.commands-bulk-replay',
      'Bulk replay by filter (status/commandType/aggregateType, optional --leaseExpired)',
      'itsm admin commands-bulk-replay [--status s] [--commandType t] [--aggregateType t] [--leaseExpired] [--pageSize N]',
      ['POST /api/v1/admin/operations/commands/bulk-replay'],
      async (ctx) => apiClient.bulkReplayOperationalCommands(bulkFilter(ctx.args.flags))
    ),
    cmd(
      'admin.commands-bulk-cancel',
      'Bulk cancel by filter (same filters as bulk-replay)',
      'itsm admin commands-bulk-cancel [--status s] [--commandType t] [--aggregateType t] [--leaseExpired] [--pageSize N]',
      ['POST /api/v1/admin/operations/commands/bulk-cancel'],
      async (ctx) => apiClient.bulkCancelOperationalCommands(bulkFilter(ctx.args.flags))
    ),

    // Durable timers behind SLA and escalation.
    cmd('admin.timers', 'List durable timers', 'itsm admin timers [--page N] [--pageSize N]', ['GET /api/v1/timers'], async (ctx) => apiClient.listTimers(withPagination(ctx.args))),
    cmd('admin.timer', 'Timer detail', 'itsm admin timer <id>', ['GET /api/v1/timers/:id'], async (ctx) => apiClient.getTimer(requirePositional(ctx.args, 0, 'timerId'))),
    cmd('admin.timer-stats', 'Timer aggregate statistics', 'itsm admin timer-stats', ['GET /api/v1/timers/stats'], async () => apiClient.getTimerStats()),
  ];
}

/**
 * Bulk replay/cancel filter. Both handlers decode a JSON body and treat an
 * empty body as "no filter", so only send flags the caller actually passed.
 */
function bulkFilter(flags: ArgContext['flags']): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  const status = stringFlag(flags, 'status');
  const commandType = stringFlag(flags, 'commandType');
  const aggregateType = stringFlag(flags, 'aggregateType');
  const pageSize = numberFlag(flags, 'pageSize');
  const leaseExpired = booleanFlag(flags, 'leaseExpired');
  if (status !== undefined) payload['status'] = status;
  if (commandType !== undefined) payload['commandType'] = commandType;
  if (aggregateType !== undefined) payload['aggregateType'] = aggregateType;
  if (pageSize !== undefined) payload['pageSize'] = pageSize;
  if (leaseExpired !== undefined) payload['leaseExpired'] = leaseExpired;
  if (Object.keys(payload).length === 0) {
    throw usageError('Refusing to bulk-mutate without a filter. Pass at least one of --status, --commandType, --aggregateType, --leaseExpired.');
  }
  return payload;
}

import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, body, numberFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/** Change lifecycle — bodies mirror dto/change_dto.go. Status verbs map directly. */

const ACTIONS = ['submit', 'schedule', 'start', 'complete', 'close', 'rollback', 'cancel'] as const;

export function changeCommands() {
  return [
    cmd('change.list', 'List changes', 'itsm change list [--status s] [--riskLevel r]', ['GET /api/v1/changes'], async (ctx) =>
      apiClient.listChanges({
        ...withPagination(ctx.args),
        status: optionalString(ctx.args.flags, 'status'),
        riskLevel: optionalString(ctx.args.flags, 'riskLevel'),
      } as never)
    ),
    cmd('change.get', 'Change detail', 'itsm change get <id>', ['GET /api/v1/changes/:id'], async (ctx) => apiClient.getChange(requirePositional(ctx.args, 0, 'changeId'))),
    cmd('change.create', 'Create a change request (title required)', 'itsm change create --data <json>', ['POST /api/v1/changes'], async (ctx) => apiClient.createChange(body(ctx.args) ?? {})),
    cmd('change.update', 'Pointer-field update', 'itsm change update <id> --data <json>', ['PUT /api/v1/changes/:id'], async (ctx) => apiClient.updateChange(requirePositional(ctx.args, 0, 'changeId'), body(ctx.args) ?? {})),
    cmd('change.assign', 'Assign (assigneeId required)', 'itsm change assign <id> --assigneeId <n>', ['POST /api/v1/changes/:id/assign'], async (ctx) => {
      const assigneeId = numberFlag(ctx.args.flags, 'assigneeId');
      if (!assigneeId) throw usageError('Missing required flag --assigneeId <number>');
      return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), 'assign', { assigneeId });
    }),
    ...ACTIONS.map((action) =>
      cmd(
        `change.${action}`,
        `Transition: ${action} ({comment, reason} — empty body allowed)`,
        `itsm change ${action} <id> [--comment text] [--reason text]`,
        [`POST /api/v1/changes/:id/${action}`],
        async (ctx) => {
          const data: Record<string, unknown> = {};
          const comment = optionalString(ctx.args.flags, 'comment');
          const reason = optionalString(ctx.args.flags, 'reason');
          if (comment) data['comment'] = comment;
          if (reason) data['reason'] = reason;
          return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), action, Object.keys(data).length ? data : undefined);
        }
      )
    ),
    cmd('change.approve', 'Approve transition', 'itsm change approve <id> [--comment text]', ['POST /api/v1/changes/:id/approve'], async (ctx) => {
      const comment = optionalString(ctx.args.flags, 'comment');
      return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), 'approve', comment ? { comment } : undefined);
    }),
    cmd('change.reject', 'Reject transition (comment/reason)', 'itsm change reject <id> [--reason text]', ['POST /api/v1/changes/:id/reject'], async (ctx) => {
      const reason = optionalString(ctx.args.flags, 'reason') ?? optionalString(ctx.args.flags, 'comment');
      return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), 'reject', reason ? { reason } : undefined);
    }),
    cmd('change.approvals', 'Change approval chain state', 'itsm change approvals <id>', ['GET /api/v1/changes/:id/approvals'], async (ctx) => apiClient.getChangeApprovals(requirePositional(ctx.args, 0, 'changeId'))),
    cmd('change.risk', 'Change risk assessment', 'itsm change risk <id>', ['GET /api/v1/changes/:id/risk'], async (ctx) => apiClient.getChangeRisk(requirePositional(ctx.args, 0, 'changeId'))),
    cmd('change.risk-update', 'Update risk assessment', 'itsm change risk-update <id> --data <json>', ['PUT /api/v1/changes/:id/risk'], async (ctx) => apiClient.updateChangeRisk(requirePositional(ctx.args, 0, 'changeId'), body(ctx.args) ?? {})),
    cmd('change.pir', 'Post-implementation review', 'itsm change pir <id>', ['GET /api/v1/changes/:id/pir'], async (ctx) => apiClient.getChangePIR(requirePositional(ctx.args, 0, 'changeId'))),
    cmd('change.stats', 'Change aggregate statistics', 'itsm change stats', ['GET /api/v1/changes/stats'], async () => apiClient.getChangeStats()),
  ];
}

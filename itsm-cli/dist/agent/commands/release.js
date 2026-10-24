import { cmd } from '../spec.js';
import { requirePositional, optionalString, withPagination, body } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/** Release management — bodies mirror dto/release_dto.go (reject/rollback need reason). */
export function releaseCommands() {
    return [
        cmd('release.list', 'List releases (paged; backend binds page/pageSize/status/type)', 'itsm release list [--page N] [--pageSize N] [--status value] [--type value]', ['GET /api/v1/releases'], async (ctx) => apiClient.listReleases({
            ...withPagination(ctx.args),
            status: optionalString(ctx.args, 'status'),
            type: optionalString(ctx.args, 'type'),
        })),
        cmd('release.get', 'Release detail', 'itsm release get <id>', ['GET /api/v1/releases/:id'], async (ctx) => apiClient.getRelease(requirePositional(ctx.args, 0, 'releaseId'))),
        cmd('release.create', 'Create a release (releaseNumber + title required)', 'itsm release create --data <json>', ['POST /api/v1/releases'], async (ctx) => apiClient.createRelease(body(ctx.args) ?? {})),
        cmd('release.update', 'Pointer-field update', 'itsm release update <id> --data <json>', ['PUT /api/v1/releases/:id'], async (ctx) => apiClient.updateRelease(requirePositional(ctx.args, 0, 'releaseId'), body(ctx.args) ?? {})),
        cmd('release.status', 'Update release status with comment', 'itsm release status <id> --status <value> [--comment text]', ['PUT /api/v1/releases/:id/status'], async (ctx) => {
            const releaseId = requirePositional(ctx.args, 0, 'releaseId');
            const status = optionalString(ctx.args.flags, 'status');
            if (!status)
                throw usageError('Missing required flag --status');
            const comment = optionalString(ctx.args.flags, 'comment') ?? body(ctx.args)?.comment;
            return apiClient.updateReleaseStatus(releaseId, status, comment);
        }),
        cmd('release.approve', 'Approve a release (comment optional)', 'itsm release approve <id> [--comment text]', ['POST /api/v1/releases/:id/approve'], async (ctx) => {
            const comment = optionalString(ctx.args.flags, 'comment') ?? body(ctx.args)?.comment;
            return apiClient.releaseLifecycle(requirePositional(ctx.args, 0, 'releaseId'), 'approve', comment ? { comment } : undefined);
        }),
        cmd('release.reject', 'Reject a release (reason required)', 'itsm release reject <id> --reason <text>', ['POST /api/v1/releases/:id/reject'], async (ctx) => {
            const reason = optionalString(ctx.args.flags, 'reason') ?? body(ctx.args)?.reason;
            if (typeof reason !== 'string' || !reason)
                throw usageError('Missing required --reason');
            return apiClient.releaseLifecycle(requirePositional(ctx.args, 0, 'releaseId'), 'reject', { reason });
        }),
        cmd('release.rollback', 'Roll back a released deployment (reason required)', 'itsm release rollback <id> --reason <text>', ['POST /api/v1/releases/:id/rollback'], async (ctx) => {
            const reason = optionalString(ctx.args.flags, 'reason') ?? body(ctx.args)?.reason;
            if (typeof reason !== 'string' || !reason)
                throw usageError('Missing required --reason');
            return apiClient.releaseLifecycle(requirePositional(ctx.args, 0, 'releaseId'), 'rollback', { reason });
        }),
        cmd('release.delete', 'Delete a release', 'itsm release delete <id>', ['DELETE /api/v1/releases/:id'], async (ctx) => apiClient.deleteRelease(requirePositional(ctx.args, 0, 'releaseId'))),
        cmd('release.stats', 'Release statistics', 'itsm release stats', ['GET /api/v1/releases/stats'], async () => apiClient.getReleaseStats()),
    ];
}

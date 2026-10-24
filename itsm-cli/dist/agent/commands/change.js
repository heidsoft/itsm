import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, optionalBool, body, numberFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/** Change lifecycle — bodies mirror dto/change_dto.go. Status verbs map directly. */
const ACTIONS = ['submit', 'schedule', 'start', 'complete', 'close', 'rollback', 'cancel', 'fail'];
export function changeCommands() {
    return [
        cmd('change.list', 'List changes', 'itsm change list [--status s] [--riskLevel r] [--page N] [--pageSize N]', ['GET /api/v1/changes'], async (ctx) => apiClient.listChanges({
            ...withPagination(ctx.args),
            status: optionalString(ctx.args.flags, 'status'),
            riskLevel: optionalString(ctx.args.flags, 'riskLevel'),
        })),
        cmd('change.get', 'Change detail', 'itsm change get <id>', ['GET /api/v1/changes/:id'], async (ctx) => apiClient.getChange(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.create', 'Create a change request (title required)', 'itsm change create --data <json>', ['POST /api/v1/changes'], async (ctx) => apiClient.createChange(body(ctx.args) ?? {})),
        cmd('change.update', 'Pointer-field update', 'itsm change update <id> --data <json>', ['PUT /api/v1/changes/:id'], async (ctx) => apiClient.updateChange(requirePositional(ctx.args, 0, 'changeId'), body(ctx.args) ?? {})),
        cmd('change.delete', 'Delete a change request', 'itsm change delete <id>', ['DELETE /api/v1/changes/:id'], async (ctx) => apiClient.deleteChange(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.assign', 'Assign (assigneeId required)', 'itsm change assign <id> --assigneeId <n>', ['POST /api/v1/changes/:id/assign'], async (ctx) => {
            const assigneeId = numberFlag(ctx.args.flags, 'assigneeId');
            if (!assigneeId)
                throw usageError('Missing required flag --assigneeId <number>');
            return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), 'assign', { assigneeId });
        }),
        ...ACTIONS.map((action) => cmd(`change.${action}`, `Transition: ${action} ({comment, reason} — empty body allowed)`, `itsm change ${action} <id> [--comment text] [--reason text]`, [`POST /api/v1/changes/:id/${action}`], async (ctx) => {
            const data = {};
            const comment = optionalString(ctx.args.flags, 'comment');
            const reason = optionalString(ctx.args.flags, 'reason');
            if (comment)
                data['comment'] = comment;
            if (reason)
                data['reason'] = reason;
            return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), action, Object.keys(data).length ? data : undefined);
        })),
        cmd('change.approve', 'Approve transition', 'itsm change approve <id> [--comment text]', ['POST /api/v1/changes/:id/approve'], async (ctx) => {
            const comment = optionalString(ctx.args.flags, 'comment');
            return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), 'approve', comment ? { comment } : undefined);
        }),
        cmd('change.reject', 'Reject transition (comment/reason)', 'itsm change reject <id> [--reason text] [--comment text]', ['POST /api/v1/changes/:id/reject'], async (ctx) => {
            const reason = optionalString(ctx.args.flags, 'reason') ?? optionalString(ctx.args.flags, 'comment');
            return apiClient.changeLifecycle(requirePositional(ctx.args, 0, 'changeId'), 'reject', reason ? { reason } : undefined);
        }),
        cmd('change.approvals', 'Change approval chain state', 'itsm change approvals <id>', ['GET /api/v1/changes/:id/approvals'], async (ctx) => apiClient.getChangeApprovals(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.risk', 'Change risk assessment', 'itsm change risk <id>', ['GET /api/v1/changes/:id/risk'], async (ctx) => apiClient.getChangeRisk(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.risk-update', 'Update risk assessment', 'itsm change risk-update <id> --data <json>', ['PUT /api/v1/changes/:id/risk'], async (ctx) => apiClient.updateChangeRisk(requirePositional(ctx.args, 0, 'changeId'), body(ctx.args) ?? {})),
        cmd('change.pir', 'Post-implementation review', 'itsm change pir <id>', ['GET /api/v1/changes/:id/pir'], async (ctx) => apiClient.getChangePIR(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.pir-create', 'Create a PIR (overallResult required: successful|partially_successful|failed)', 'itsm change pir-create <changeId> --overallResult <r> [--objectivesAchieved true|false] [--successSummary t] [--issuesEncountered t] [--lessonsLearned t] [--improvementRecommendations t] | --data <json>', ['POST /api/v1/changes/:id/pir'], async (ctx) => {
            // Backend gate: a PIR only exists after the change is implemented/completed
            // ("尚未实施完成，不能做实施后回顾"). Flat args first so the required
            // overallResult does not force hand-escaped JSON.
            const data = body(ctx.args);
            if (data)
                return apiClient.createChangePIR(requirePositional(ctx.args, 0, 'changeId'), data);
            const overallResult = optionalString(ctx.args.flags, 'overallResult');
            if (!overallResult)
                throw usageError('Missing --overallResult (successful|partially_successful|failed)');
            const payload = { overallResult };
            const objectives = optionalBool(ctx.args.flags, 'objectivesAchieved');
            if (objectives !== undefined)
                payload['objectivesAchieved'] = objectives;
            for (const f of ['successSummary', 'issuesEncountered', 'lessonsLearned', 'improvementRecommendations']) {
                const v = optionalString(ctx.args.flags, f);
                if (v)
                    payload[f] = v;
            }
            return apiClient.createChangePIR(requirePositional(ctx.args, 0, 'changeId'), payload);
        }),
        cmd('change.pir-list', 'List PIRs across changes', 'itsm change pir-list [--page N] [--pageSize N]', ['GET /api/v1/changes/pirs'], async (ctx) => apiClient.listChangePIRs(withPagination(ctx.args))),
        cmd('change.pir-update', 'Update a PIR (keyed by pirId, not changeId)', 'itsm change pir-update <pirId> --data <json>', ['PUT /api/v1/changes/pir/:id'], async (ctx) => apiClient.updateChangePIR(requirePositional(ctx.args, 0, 'pirId'), body(ctx.args) ?? {})),
        cmd('change.approval-summary', 'Approval chain summary for one change', 'itsm change approval-summary <id>', ['GET /api/v1/changes/:id/approval-summary'], async (ctx) => apiClient.getChangeApprovalSummary(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.risk-assessment', 'Full risk assessment (richer than /risk)', 'itsm change risk-assessment <id>', ['GET /api/v1/changes/:id/risk-assessment'], async (ctx) => apiClient.getChangeRiskAssessment(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.cmdb-impact', 'CI blast radius for one change', 'itsm change cmdb-impact <id>', ['GET /api/v1/changes/:id/cmdb-impact'], async (ctx) => apiClient.getChangeCMDBImpact(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.rollback-steps', 'Rollback checklist', 'itsm change rollback-steps <id>', ['GET /api/v1/changes/:id/rollback-steps'], async (ctx) => apiClient.getChangeRollbackSteps(requirePositional(ctx.args, 0, 'changeId'))),
        cmd('change.calendar', 'Change calendar view (startDate/endDate REQUIRED, format YYYY-MM-DD)', 'itsm change calendar --startDate <YYYY-MM-DD> --endDate <YYYY-MM-DD> [--status s]', ['GET /api/v1/changes/calendar'], async (ctx) => {
            // dto.ChangeCalendarRequest binds startDate/endDate as required; no page/pageSize
            // exists on this endpoint, so passing pagination would 400 on ShouldBindQuery.
            const startDate = optionalString(ctx.args.flags, 'startDate');
            const endDate = optionalString(ctx.args.flags, 'endDate');
            if (!startDate || !endDate)
                throw usageError('Missing --startDate and/or --endDate (both required, format YYYY-MM-DD)');
            return apiClient.listChangeCalendar({ startDate, endDate, status: optionalString(ctx.args.flags, 'status') });
        }),
        cmd('change.approvals-create', 'Record an approval decision for a change (changeId comes from the path)', 'itsm change approvals-create <id> [--comment <text>]', ['POST /api/v1/changes/:id/approvals'], async (ctx) => {
            // dto.CreateChangeApprovalRequest binds with ShouldBindJSON, so an empty
            // request body is EOF → 400. Always send an object.
            const comment = optionalString(ctx.args.flags, 'comment');
            const overrides = body(ctx.args) ?? {};
            const payload = { ...overrides };
            if (comment !== undefined)
                payload['comment'] = comment;
            payload['changeId'] = requirePositional(ctx.args, 0, 'changeId');
            return apiClient.createChangeApproval(requirePositional(ctx.args, 0, 'changeId'), payload);
        }),
        cmd('change.rollback-steps-replace', 'Overwrite the ordered rollback checklist (empty array clears it; before implementation only)', 'itsm change rollback-steps-replace <id> --data \'{"steps":["stop app","restore db"]}\'', ['PUT /api/v1/changes/:id/rollback-steps'], async (ctx) => {
            // dto.ReplaceRollbackStepsRequest.Steps is string[] and is NOT binding:"required",
            // so an explicit [] is the documented way to clear the checklist.
            const payload = body(ctx.args);
            if (!payload || !Array.isArray(payload['steps'])) {
                throw usageError('Missing --data with a "steps" array of strings. Usage: itsm change rollback-steps-replace <id> --data \'{"steps":[...]}\'');
            }
            if (!payload['steps'].every((s) => typeof s === 'string')) {
                throw usageError('"steps" must be an array of strings');
            }
            return apiClient.replaceChangeRollbackSteps(requirePositional(ctx.args, 0, 'changeId'), payload['steps']);
        }),
        cmd('change.rollback-step-resolve', 'Confirm one rollback step result (note required for skipped/failed)', 'itsm change rollback-step-resolve <id> <stepId> --status done [--note <text>]', ['POST /api/v1/changes/:id/rollback-steps/:stepId'], async (ctx) => {
            const status = optionalString(ctx.args.flags, 'status');
            if (!status)
                throw usageError('Missing --status (pending|done|skipped|failed)');
            const note = optionalString(ctx.args.flags, 'note');
            if ((status === 'skipped' || status === 'failed') && !note) {
                throw usageError(`--note is required when --status ${status}`);
            }
            const payload = { status };
            if (note !== undefined)
                payload['note'] = note;
            return apiClient.resolveChangeRollbackStep(requirePositional(ctx.args, 0, 'changeId'), requirePositional(ctx.args, 1, 'stepId'), payload);
        }),
        cmd('change.stats', 'Change aggregate statistics', 'itsm change stats', ['GET /api/v1/changes/stats'], async () => apiClient.getChangeStats()),
    ];
}

import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, optionalNumber, body, numberFlag, stringFlag, booleanFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/** Ticket lifecycle — the primary agent debugging surface. Bodies mirror dto/ticket_dto.go. */
function id(ctx) {
    return requirePositional(ctx.args, 0, 'ticketId');
}
function listQuery(ctx) {
    return {
        ...withPagination(ctx.args),
        status: optionalString(ctx.args.flags, 'status'),
        priority: optionalString(ctx.args.flags, 'priority'),
        assigneeId: optionalNumber(ctx.args.flags, 'assigneeId'),
        requesterId: optionalNumber(ctx.args.flags, 'requesterId'),
        keyword: optionalString(ctx.args.flags, 'keyword'),
    };
}
function ticketIdFlag(ctx) {
    const fromPositional = ctx.args.positional[0];
    const raw = stringFlag(ctx.args.flags, 'ticketId') ?? fromPositional;
    const n = Number(raw);
    if (!raw || Number.isNaN(n))
        throw usageError('Missing ticketId (positional <id> or --ticketId)');
    return n;
}
/** `--tagIds 1,2,3` / `--tags a,b` → typed arrays (no silent skips: bad input fails). */
function csvNumbers(ctx, name) {
    const raw = stringFlag(ctx.args.flags, name);
    if (raw === undefined)
        return undefined;
    const parts = raw.split(',').map((p) => p.trim()).filter(Boolean);
    const out = parts.map((p) => Number(p));
    if (out.some((n) => !Number.isInteger(n) || n <= 0)) {
        throw usageError(`--${name} must be a comma-separated list of positive integers`);
    }
    return out;
}
function csvStrings(ctx, name) {
    const raw = stringFlag(ctx.args.flags, name);
    if (raw === undefined)
        return undefined;
    const parts = raw.split(',').map((p) => p.trim()).filter(Boolean);
    return parts.length ? parts : undefined;
}
export function ticketCommands() {
    return [
        cmd('ticket.list', 'List tickets', 'itsm ticket list [--status s] [--priority p] [--page N]', ['GET /api/v1/tickets'], async (ctx) => apiClient.listTickets(listQuery(ctx))),
        cmd('ticket.get', 'Get ticket detail', 'itsm ticket get <id>', ['GET /api/v1/tickets/:id'], async (ctx) => apiClient.getTicket(id(ctx))),
        cmd('ticket.create', 'Create a ticket (priority required: low|medium|high|critical|urgent)', 'itsm ticket create <title> <description> --priority <p> | --data <json>', ['POST /api/v1/tickets'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.createTicket(data);
            const title = requirePositional(ctx.args, 0, 'title');
            const description = requirePositional(ctx.args, 1, 'description');
            const priority = stringFlag(ctx.args.flags, 'priority');
            if (!priority)
                throw usageError('Missing required flag --priority (low|medium|high|critical|urgent)');
            return apiClient.createTicket({ title, description, priority });
        }),
        cmd('ticket.update', 'Partial update (PATCH passthrough; version carries the optimistic lock)', 'itsm ticket update <id> --data <json>', ['PATCH /api/v1/tickets/:id'], async (ctx) => apiClient.updateTicket(id(ctx), body(ctx.args) ?? {})),
        cmd('ticket.delete', 'Delete a ticket', 'itsm ticket delete <id>', ['DELETE /api/v1/tickets/:id'], async (ctx) => apiClient.deleteTicket(id(ctx))),
        cmd('ticket.search', 'Ticket search (backend param: keyword)', 'itsm ticket search <query>', ['GET /api/v1/tickets/search'], async (ctx) => {
            const q = requirePositional(ctx.args, 0, 'query');
            return apiClient.searchTickets(q, withPagination(ctx.args));
        }),
        cmd('ticket.stats', 'Ticket aggregate statistics', 'itsm ticket stats', ['GET /api/v1/tickets/stats'], async () => apiClient.getTicketStats()),
        cmd('ticket.assign', 'Assign to a technician', 'itsm ticket assign <id> --assigneeId <n>', ['POST /api/v1/tickets/:id/assign'], async (ctx) => {
            const assigneeId = numberFlag(ctx.args.flags, 'assigneeId');
            if (!assigneeId)
                throw usageError('Missing required flag --assigneeId <number>');
            return apiClient.assignTicket(id(ctx), String(assigneeId));
        }),
        cmd('ticket.resolve', 'Resolve with resolution notes', 'itsm ticket resolve <id> [--resolution text] [--data <json>]', ['POST /api/v1/tickets/:id/resolve'], async (ctx) => {
            const resolution = stringFlag(ctx.args.flags, 'resolution') || body(ctx.args)?.resolution;
            return apiClient.resolveTicket(id(ctx), typeof resolution === 'string' ? { resolution } : body(ctx.args));
        }),
        cmd('ticket.close', 'Close with close reason', 'itsm ticket close <id> [--closeReason text] [--data <json>]', ['POST /api/v1/tickets/:id/close'], async (ctx) => {
            const closeReason = stringFlag(ctx.args.flags, 'closeReason') || body(ctx.args)?.closeReason;
            return apiClient.closeTicket(id(ctx), typeof closeReason === 'string' ? { closeReason } : body(ctx.args));
        }),
        cmd('ticket.escalate', 'Escalate (reason required)', 'itsm ticket escalate <id> --reason <text>', ['POST /api/v1/tickets/:id/escalate'], async (ctx) => {
            const reason = stringFlag(ctx.args.flags, 'reason') ?? body(ctx.args)?.reason;
            if (typeof reason !== 'string' || !reason)
                throw usageError('Missing required flag --reason');
            return apiClient.escalateTicket(id(ctx), { reason });
        }),
        cmd('ticket.history', 'Activity history', 'itsm ticket history <id>', ['GET /api/v1/tickets/:id/history'], async (ctx) => apiClient.getTicketHistory(id(ctx))),
        cmd('ticket.workflow-state', 'Current workflow state machine view', 'itsm ticket workflow-state <id>', ['GET /api/v1/tickets/:id/workflow/state'], async (ctx) => apiClient.getTicketWorkflowState(id(ctx))),
        cmd('ticket.sla', 'SLA timers for one ticket', 'itsm ticket sla <id>', ['GET /api/v1/tickets/:id/sla'], async (ctx) => apiClient.getTicketSLA(id(ctx))),
        cmd('ticket.comments.list', 'List comments', 'itsm ticket comments list <id>', ['GET /api/v1/tickets/:id/comments'], async (ctx) => apiClient.listTicketComments(requirePositional(ctx.args, 0, 'ticketId'))),
        cmd('ticket.comments.add', 'Add comment (content 1-5000)', 'itsm ticket comments add <id> --content <text> | --data <json>', ['POST /api/v1/tickets/:id/comments'], async (ctx) => {
            const tid = requirePositional(ctx.args, 0, 'ticketId');
            const content = stringFlag(ctx.args.flags, 'content') ?? body(ctx.args)?.content;
            if (typeof content !== 'string' || !content)
                throw usageError('Missing required --content');
            const isInternal = booleanFlag(ctx.args.flags, 'internal');
            return apiClient.createTicketComment(tid, isInternal === undefined ? { content } : { content, isInternal });
        }),
        cmd('ticket.ai-analyze', 'AI impact/priority analysis for one ticket', 'itsm ticket ai-analyze <id>', ['POST /api/v1/ai/tickets/:id/analyze'], async (ctx) => apiClient.aiAnalyzeTicket(id(ctx))),
        cmd('ticket.ai-summarize', 'AI summary of one ticket (audit-logged)', 'itsm ticket ai-summarize <id>', ['GET /api/v1/ai/tickets/:id/summary'], async (ctx) => apiClient.aiSummarizeTicket(id(ctx))),
        cmd('ticket.overdue', 'Tickets breaching SLA', 'itsm ticket overdue', ['GET /api/v1/tickets/overdue'], async () => apiClient.getOverdueTickets()),
        // ---- /tickets/workflow/* state machine (body carries ticketId) ----
        cmd('ticket.workflow-accept', 'Accept/claim the ticket assignment', 'itsm ticket workflow-accept <id> [--comment text]', ['POST /api/v1/tickets/workflow/accept'], async (ctx) => apiClient.ticketWorkflowAction('accept', { ticketId: ticketIdFlag(ctx), comment: optionalString(ctx.args.flags, 'comment') })),
        cmd('ticket.workflow-reject', 'Reject the flow back', 'itsm ticket workflow-reject <id> --reason text --comment text', ['POST /api/v1/tickets/workflow/reject'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const reason = optionalString(ctx.args.flags, 'reason');
            const comment = optionalString(ctx.args.flags, 'comment');
            if (!reason || !comment)
                throw usageError('workflow-reject needs --reason and --comment');
            return apiClient.ticketWorkflowAction('reject', { ticketId, reason, comment });
        }),
        cmd('ticket.workflow-withdraw', 'Withdraw', 'itsm ticket workflow-withdraw <id> --reason text', ['POST /api/v1/tickets/workflow/withdraw'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const reason = optionalString(ctx.args.flags, 'reason');
            if (!reason)
                throw usageError('workflow-withdraw needs --reason');
            return apiClient.ticketWorkflowAction('withdraw', { ticketId, reason });
        }),
        cmd('ticket.workflow-forward', 'Forward to another user', 'itsm ticket workflow-forward <id> --toUserId <n> --comment text', ['POST /api/v1/tickets/workflow/forward'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const toUserId = numberFlag(ctx.args.flags, 'toUserId');
            const comment = optionalString(ctx.args.flags, 'comment');
            if (!toUserId || !comment)
                throw usageError('workflow-forward needs --toUserId and --comment');
            return apiClient.ticketWorkflowAction('forward', { ticketId, toUserId, comment });
        }),
        cmd('ticket.workflow-cc', 'Add CCs', 'itsm ticket workflow-cc <id> --ccUsers 1,2,3', ['POST /api/v1/tickets/workflow/cc'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const raw = optionalString(ctx.args.flags, 'ccUsers');
            if (!raw)
                throw usageError('workflow-cc needs --ccUsers <comma-separated user ids>');
            const ccUsers = raw.split(',').map((s) => Number(s.trim())).filter((n) => !Number.isNaN(n));
            if (!ccUsers.length)
                throw usageError('--ccUsers needs at least one numeric id');
            return apiClient.ticketWorkflowAction('cc', { ticketId, ccUsers });
        }),
        cmd('ticket.workflow-approve', 'Approve workflow step (needs pending approvalId)', 'itsm ticket workflow-approve <id> --approvalId <n> --action approve|reject|delegate', ['POST /api/v1/tickets/workflow/approve'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const approvalId = numberFlag(ctx.args.flags, 'approvalId');
            const action = optionalString(ctx.args.flags, 'action') ?? 'approve';
            if (!approvalId)
                throw usageError('workflow-approve needs --approvalId');
            if (!['approve', 'reject', 'delegate'].includes(action))
                throw usageError('--action must be approve|reject|delegate');
            const bodyPayload = { ticketId, approvalId, action };
            const comment = optionalString(ctx.args.flags, 'comment');
            if (comment)
                bodyPayload['comment'] = comment;
            const delegateToUserId = numberFlag(ctx.args.flags, 'delegateToUserId');
            if (action === 'delegate') {
                if (!delegateToUserId)
                    throw usageError('delegate needs --delegateToUserId');
                bodyPayload['delegateToUserId'] = delegateToUserId;
            }
            return apiClient.ticketWorkflowAction('approve', bodyPayload);
        }),
        cmd('ticket.workflow-resolve', 'Resolve through the workflow path', 'itsm ticket workflow-resolve <id> [--resolution text]', ['POST /api/v1/tickets/workflow/resolve'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const resolution = optionalString(ctx.args.flags, 'resolution');
            return apiClient.ticketWorkflowAction('resolve', resolution ? { ticketId, resolution } : { ticketId });
        }),
        cmd('ticket.workflow-close', 'Close through the workflow path', 'itsm ticket workflow-close <id> [--closeReason text]', ['POST /api/v1/tickets/workflow/close'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const closeReason = optionalString(ctx.args.flags, 'closeReason');
            return apiClient.ticketWorkflowAction('close', closeReason ? { ticketId, closeReason } : { ticketId });
        }),
        cmd('ticket.workflow-reopen', 'Reopen (reason required)', 'itsm ticket workflow-reopen <id> --reason text', ['POST /api/v1/tickets/workflow/reopen'], async (ctx) => {
            const ticketId = ticketIdFlag(ctx);
            const reason = optionalString(ctx.args.flags, 'reason');
            if (!reason)
                throw usageError('workflow-reopen needs --reason');
            return apiClient.ticketWorkflowAction('reopen', { ticketId, reason });
        }),
        // ---- status, SLA, structure, associations, tags, rating, side data ----
        cmd('ticket.status', 'Transition status through the ticket state machine (server-validated)', 'itsm ticket status <id> --status <new status>', ['PUT /api/v1/tickets/:id/status'], async (ctx) => {
            const status = optionalString(ctx.args.flags, 'status');
            if (!status)
                throw usageError('Missing --status (e.g. in_progress, resolved, closed)');
            return apiClient.updateTicketStatus(id(ctx), status);
        }),
        cmd('ticket.sla-pause', 'Pause the SLA clock (audited)', 'itsm ticket sla-pause <id>', ['PUT /api/v1/tickets/:id/sla/pause'], async (ctx) => apiClient.pauseTicketSLA(id(ctx))),
        cmd('ticket.sla-resume', 'Resume the SLA clock', 'itsm ticket sla-resume <id>', ['PUT /api/v1/tickets/:id/sla/resume'], async (ctx) => apiClient.resumeTicketSLA(id(ctx))),
        cmd('ticket.subtasks.list', 'List child tickets', 'itsm ticket subtasks list <id>', ['GET /api/v1/tickets/:id/subtasks'], async (ctx) => apiClient.listSubtasks(id(ctx))),
        cmd('ticket.subtasks.create', 'Create a child ticket (title required)', 'itsm ticket subtasks create <id> --title <t> [--priority p] [--assigneeId n] | --data <json>', ['POST /api/v1/tickets/:id/subtasks'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.createSubtask(id(ctx), data);
            const title = optionalString(ctx.args.flags, 'title');
            if (!title)
                throw usageError('Missing --title');
            const payload = { title };
            const description = optionalString(ctx.args.flags, 'description');
            const priority = optionalString(ctx.args.flags, 'priority');
            const type = optionalString(ctx.args.flags, 'type');
            const assigneeId = numberFlag(ctx.args.flags, 'assigneeId');
            if (description)
                payload['description'] = description;
            if (priority)
                payload['priority'] = priority;
            if (type)
                payload['type'] = type;
            if (assigneeId)
                payload['assigneeId'] = assigneeId;
            return apiClient.createSubtask(id(ctx), payload);
        }),
        cmd('ticket.subtasks.update', 'Update a child ticket (version optimistic lock via --data)', 'itsm ticket subtasks update <id> <subtaskId> [--data <json>]', ['PATCH /api/v1/tickets/:id/subtasks/:subtaskId'], async (ctx) => {
            const subtaskId = requirePositional(ctx.args, 1, 'subtaskId');
            return apiClient.updateSubtask(id(ctx), subtaskId, body(ctx.args) ?? {});
        }),
        cmd('ticket.subtasks.delete', 'Delete a child ticket', 'itsm ticket subtasks delete <id> <subtaskId>', ['DELETE /api/v1/tickets/:id/subtasks/:subtaskId'], async (ctx) => {
            const subtaskId = requirePositional(ctx.args, 1, 'subtaskId');
            return apiClient.deleteSubtask(id(ctx), subtaskId);
        }),
        cmd('ticket.relations', 'Parent / related / child association graph', 'itsm ticket relations <id>', ['GET /api/v1/tickets/:id/relations'], async (ctx) => apiClient.getTicketRelations(id(ctx))),
        cmd('ticket.relations-update', 'Replace associations (parentId, relatedIds, tagIds; all validated in-tenant)', 'itsm ticket relations-update <id> --data <json>', ['PATCH /api/v1/tickets/:id/relations'], async (ctx) => {
            const data = body(ctx.args);
            if (!data)
                throw usageError('Provide --data \'{"parentId":1,"relatedIds":[2],"tagIds":[3]}\'');
            return apiClient.updateTicketRelations(id(ctx), data);
        }),
        cmd('ticket.relations-stats', 'Association counters (parents/children/related)', 'itsm ticket relations-stats <id>', ['GET /api/v1/tickets/:id/relations/stats'], async (ctx) => apiClient.getTicketRelationStats(id(ctx))),
        cmd('ticket.cc', 'CC (watcher) records', 'itsm ticket cc <id>', ['GET /api/v1/tickets/:id/cc'], async (ctx) => apiClient.getTicketCC(id(ctx))),
        cmd('ticket.tags-add', 'Attach tags by id or name', 'itsm ticket tags-add <id> --tagIds 1,2 | --tags a,b', ['POST /api/v1/tickets/:id/tags'], async (ctx) => {
            const tagIds = csvNumbers(ctx, 'tagIds');
            const tags = csvStrings(ctx, 'tags');
            if (!tagIds && !tags)
                throw usageError('Provide --tagIds or --tags');
            return apiClient.addTicketTags(id(ctx), tagIds ? { tagIds } : { tags });
        }),
        cmd('ticket.tags-remove', 'Detach tags by id or name', 'itsm ticket tags-remove <id> --tagIds 1,2 | --tags a,b', ['DELETE /api/v1/tickets/:id/tags'], async (ctx) => {
            const tagIds = csvNumbers(ctx, 'tagIds');
            const tags = csvStrings(ctx, 'tags');
            if (!tagIds && !tags)
                throw usageError('Provide --tagIds or --tags');
            return apiClient.removeTicketTags(id(ctx), tagIds ? { tagIds } : { tags });
        }),
        cmd('ticket.rating', 'Read the satisfaction rating', 'itsm ticket rating <id>', ['GET /api/v1/tickets/:id/rating'], async (ctx) => apiClient.getTicketRating(id(ctx))),
        cmd('ticket.rating-submit', 'Submit a 1–5 satisfaction rating', 'itsm ticket rating-submit <id> --rating <1-5> [--comment text]', ['POST /api/v1/tickets/:id/rating'], async (ctx) => {
            const rating = numberFlag(ctx.args.flags, 'rating');
            if (!rating || rating < 1 || rating > 5)
                throw usageError('Missing --rating (integer 1–5)');
            const comment = optionalString(ctx.args.flags, 'comment');
            return apiClient.submitTicketRating(id(ctx), comment ? { rating, comment } : { rating });
        }),
        cmd('ticket.rating-stats', 'Rating distribution statistics', 'itsm ticket rating-stats', ['GET /api/v1/tickets/rating-stats'], async (ctx) => apiClient.getRatingStats(withPagination(ctx.args))),
        cmd('ticket.configuration-items', 'CIs linked to the ticket', 'itsm ticket configuration-items <id>', ['GET /api/v1/tickets/:id/configuration-items'], async (ctx) => apiClient.getTicketConfigurationItems(id(ctx))),
        cmd('ticket.dependencies', 'Dependency impact (parent chain, children, related)', 'itsm ticket dependencies <id>', ['GET /api/v1/tickets/:id/dependencies'], async (ctx) => apiClient.getTicketDependencies(id(ctx))),
        cmd('ticket.notifications', 'Ticket-scoped notification history', 'itsm ticket notifications <id>', ['GET /api/v1/tickets/:id/notifications'], async (ctx) => apiClient.listTicketNotifications(id(ctx))),
        cmd('ticket.notify', 'Send a ticket notification (email|sms|webhook)', 'itsm ticket notify <id> --type email|sms|webhook [--recipients a@x,b@y] [--template t] [--data <json>]', ['POST /api/v1/tickets/:id/notifications'], async (ctx) => {
            const type = optionalString(ctx.args.flags, 'type');
            if (type !== 'email' && type !== 'sms' && type !== 'webhook')
                throw usageError('Missing --type (email|sms|webhook)');
            const recipients = csvStrings(ctx, 'recipients');
            const template = optionalString(ctx.args.flags, 'template');
            const templateData = body(ctx.args);
            return apiClient.sendTicketNotification(id(ctx), {
                type,
                ...(recipients ? { recipients } : {}),
                ...(template ? { template } : {}),
                ...(templateData ? { data: templateData } : {}),
            });
        }),
        cmd('ticket.workflow-history', 'Workflow transition history', 'itsm ticket workflow-history <id>', ['GET /api/v1/tickets/:id/workflow-history'], async (ctx) => apiClient.getTicketWorkflowHistory(id(ctx))),
        cmd('ticket.auto-assign', 'Smart assignment (rules + workload)', 'itsm ticket auto-assign <id> [--data <json>]', ['POST /api/v1/tickets/:id/auto-assign'], async (ctx) => apiClient.autoAssignTicket(id(ctx), body(ctx.args))),
    ];
}

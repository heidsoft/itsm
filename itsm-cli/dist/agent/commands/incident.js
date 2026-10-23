import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, body, numberFlag, stringFlag, booleanFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/** Incident lifecycle — bodies mirror dto/incident_dto.go. */
function id(ctx) {
    return requirePositional(ctx.args, 0, 'incidentId');
}
export function incidentCommands() {
    return [
        cmd('incident.list', 'List incidents', 'itsm incident list [--status s] [--priority p] [--keyword k]', ['GET /api/v1/incidents'], async (ctx) => apiClient.listIncidents({
            ...withPagination(ctx.args),
            status: optionalString(ctx.args.flags, 'status'),
            priority: optionalString(ctx.args.flags, 'priority'),
            keyword: optionalString(ctx.args.flags, 'keyword'),
        })),
        cmd('incident.get', 'Get incident detail', 'itsm incident get <id>', ['GET /api/v1/incidents/:id'], async (ctx) => apiClient.getIncident(id(ctx))),
        cmd('incident.create', 'Create an incident (title required)', 'itsm incident create <title> [--priority p] | --data <json>', ['POST /api/v1/incidents'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.createIncident(data);
            const title = requirePositional(ctx.args, 0, 'title');
            const dataFlat = { title };
            const priority = optionalString(ctx.args.flags, 'priority');
            const severity = optionalString(ctx.args.flags, 'severity');
            const category = optionalString(ctx.args.flags, 'category');
            if (priority)
                dataFlat['priority'] = priority;
            if (severity)
                dataFlat['severity'] = severity;
            if (category)
                dataFlat['category'] = category;
            return apiClient.createIncident(dataFlat);
        }),
        cmd('incident.update', 'Full-object update (pointer fields; version optimistic lock)', 'itsm incident update <id> --data <json>', ['PUT /api/v1/incidents/:id'], async (ctx) => apiClient.updateIncident(id(ctx), body(ctx.args) ?? {})),
        cmd('incident.delete', 'Delete an incident', 'itsm incident delete <id>', ['DELETE /api/v1/incidents/:id'], async (ctx) => apiClient.deleteIncident(id(ctx))),
        cmd('incident.acknowledge', 'Acknowledge response started (no body)', 'itsm incident acknowledge <id>', ['POST /api/v1/incidents/:id/acknowledge'], async (ctx) => apiClient.incidentLifecycle(id(ctx), 'acknowledge')),
        cmd('incident.assign', 'Assign to a technician', 'itsm incident assign <id> --assigneeId <n>', ['POST /api/v1/incidents/:id/assign'], async (ctx) => {
            const assigneeId = numberFlag(ctx.args.flags, 'assigneeId');
            if (!assigneeId)
                throw usageError('Missing required flag --assigneeId <number>');
            return apiClient.incidentLifecycle(id(ctx), 'assign', { assigneeId });
        }),
        cmd('incident.resolve', 'Resolve with resolution + rootCause', 'itsm incident resolve <id> [--resolution text] [--rootCause text]', ['POST /api/v1/incidents/:id/resolve'], async (ctx) => {
            const resolution = optionalString(ctx.args.flags, 'resolution');
            const rootCause = optionalString(ctx.args.flags, 'rootCause');
            const data = {};
            if (resolution)
                data['resolution'] = resolution;
            if (rootCause)
                data['rootCause'] = rootCause;
            return apiClient.incidentLifecycle(id(ctx), 'resolve', data);
        }),
        cmd('incident.close', 'Close with closeNotes (backend rejects an empty note)', 'itsm incident close <id> --closeNotes <text>', ['POST /api/v1/incidents/:id/close'], async (ctx) => {
            const closeNotes = optionalString(ctx.args.flags, 'closeNotes');
            if (!closeNotes)
                throw usageError('Missing --closeNotes <text> (the backend requires a close note)');
            return apiClient.incidentLifecycle(id(ctx), 'close', { closeNotes });
        }),
        cmd('incident.reopen', 'Reopen a closed incident (no body)', 'itsm incident reopen <id>', ['POST /api/v1/incidents/:id/reopen'], async (ctx) => apiClient.incidentLifecycle(id(ctx), 'reopen')),
        cmd('incident.escalate', 'Escalate (escalationLevel required)', 'itsm incident escalate <id> --escalationLevel <n> [--reason text]', ['POST /api/v1/incidents/:id/escalate'], async (ctx) => {
            const level = numberFlag(ctx.args.flags, 'escalationLevel');
            if (!level)
                throw usageError('Missing required flag --escalationLevel <number>');
            const data = { escalationLevel: level };
            const reason = optionalString(ctx.args.flags, 'reason');
            if (reason)
                data['reason'] = reason;
            return apiClient.incidentLifecycle(id(ctx), 'escalate', data);
        }),
        cmd('incident.convert-to-problem', 'Promote an incident into a problem', 'itsm incident convert-to-problem <id> [--data <json>]', ['POST /api/v1/incidents/:id/convert-to-problem'], async (ctx) => apiClient.incidentLifecycle(id(ctx), 'convert-to-problem', body(ctx.args))),
        cmd('incident.root-cause', 'Set free-form root cause (PUT contract)', 'itsm incident root-cause <id> --data <json>', ['PUT /api/v1/incidents/:id/root-cause'], async (ctx) => {
            const data = body(ctx.args);
            if (!data)
                throw usageError('Provide --data \'{"rootCause": {...}}\'');
            return apiClient.updateIncidentRootCause(id(ctx), data);
        }),
        cmd('incident.stats', 'Incident aggregate statistics', 'itsm incident stats', ['GET /api/v1/incidents/stats'], async () => apiClient.getIncidentStats()),
        cmd('incident.comments.list', 'Incident comments', 'itsm incident comments list <id>', ['GET /api/v1/incidents/:id/comments'], async (ctx) => apiClient.listIncidentComments(requirePositional(ctx.args, 0, 'incidentId'))),
        cmd('incident.comments.add', 'Add incident comment', 'itsm incident comments add <id> --content <text> | --data <json>', ['POST /api/v1/incidents/:id/comments'], async (ctx) => {
            const tid = requirePositional(ctx.args, 0, 'incidentId');
            const content = stringFlag(ctx.args.flags, 'content') ?? body(ctx.args)?.content;
            if (typeof content !== 'string' || !content)
                throw usageError('Missing required --content');
            const isInternal = booleanFlag(ctx.args.flags, 'internal');
            return apiClient.createIncidentComment(tid, isInternal === undefined ? { content } : { content, isInternal });
        }),
        cmd('incident.triage', 'AI triage from title+description (decision support, never auto-executes)', 'itsm incident triage --title X --description Y', ['POST /api/v1/ai/triage'], async (ctx) => {
            const title = optionalString(ctx.args.flags, 'title') ?? ctx.args.positional[0] ?? '';
            const description = optionalString(ctx.args.flags, 'description', 'desc') ?? ctx.args.positional[1] ?? title;
            if (!title)
                throw usageError('Usage: itsm incident triage --title X --description Y');
            return apiClient.aiTriage({ title, description });
        }),
        // ---- status, analysis, classification, impact, SLA, observability ----
        cmd('incident.status', 'Update status through the incident state machine', 'itsm incident status <id> --status <s> [--data <json>]', ['PUT /api/v1/incidents/:id/status'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.updateIncidentStatus(id(ctx), data);
            const status = optionalString(ctx.args.flags, 'status');
            if (!status)
                throw usageError('Missing --status');
            return apiClient.updateIncidentStatus(id(ctx), { status });
        }),
        cmd('incident.root-cause-get', 'Read the root-cause analysis document', 'itsm incident root-cause-get <id>', ['GET /api/v1/incidents/:id/root-cause'], async (ctx) => apiClient.getIncidentRootCause(id(ctx))),
        cmd('incident.classification', 'Read classification (category, serviceType, urgency, impact)', 'itsm incident classification <id>', ['GET /api/v1/incidents/:id/classification'], async (ctx) => apiClient.getIncidentClassification(id(ctx))),
        cmd('incident.classification-update', 'Update classification fields', 'itsm incident classification-update <id> [--category c] [--subcategory s] [--serviceType t] [--failureType f] [--urgency u] [--impact i] | --data <json>', ['PUT /api/v1/incidents/:id/classification'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.updateIncidentClassification(id(ctx), data);
            const payload = {};
            for (const key of ['category', 'subcategory', 'serviceType', 'failureType', 'urgency', 'impact']) {
                const value = optionalString(ctx.args.flags, key);
                if (value)
                    payload[key] = value;
            }
            if (!Object.keys(payload).length)
                throw usageError('Provide at least one classification field or --data');
            return apiClient.updateIncidentClassification(id(ctx), payload);
        }),
        cmd('incident.impact-assessment', 'Read the impact assessment document', 'itsm incident impact-assessment <id>', ['GET /api/v1/incidents/:id/impact-assessment'], async (ctx) => apiClient.getIncidentImpactAssessment(id(ctx))),
        cmd('incident.impact-assessment-update', 'Replace the impact assessment document', 'itsm incident impact-assessment-update <id> --data \'{"impactAnalysis":{...}}\'', ['PUT /api/v1/incidents/:id/impact-assessment'], async (ctx) => {
            const data = body(ctx.args);
            if (!data)
                throw usageError('Provide --data with an impactAnalysis object');
            return apiClient.updateIncidentImpactAssessment(id(ctx), data);
        }),
        cmd('incident.major-incident', 'Escalate to major incident (impactScope + businessImpact required)', 'itsm incident major-incident <id> --impactScope low|medium|high|critical --businessImpact text [--communicationPlan text]', ['POST /api/v1/incidents/:id/major-incident'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.escalateMajorIncident(id(ctx), data);
            const impactScope = optionalString(ctx.args.flags, 'impactScope');
            const businessImpact = optionalString(ctx.args.flags, 'businessImpact');
            if (!impactScope || !businessImpact)
                throw usageError('Missing --impactScope and --businessImpact');
            const communicationPlan = optionalString(ctx.args.flags, 'communicationPlan');
            return apiClient.escalateMajorIncident(id(ctx), {
                impactScope,
                businessImpact,
                ...(communicationPlan ? { communicationPlan } : {}),
            });
        }),
        cmd('incident.events', 'Events correlated with the incident', 'itsm incident events <id>', ['GET /api/v1/incidents/:id/events'], async (ctx) => apiClient.getIncidentEvents(id(ctx))),
        cmd('incident.alerts', 'Alerts fired for the incident', 'itsm incident alerts <id>', ['GET /api/v1/incidents/:id/alerts'], async (ctx) => apiClient.getIncidentAlerts(id(ctx))),
        cmd('incident.metrics', 'Runtime metrics attached to the incident', 'itsm incident metrics <id>', ['GET /api/v1/incidents/:id/metrics'], async (ctx) => apiClient.getIncidentMetrics(id(ctx))),
        cmd('incident.sla-pause', 'Pause the incident SLA clock', 'itsm incident sla-pause <id>', ['PUT /api/v1/incidents/:id/sla/pause'], async (ctx) => apiClient.pauseIncidentSLA(id(ctx))),
        cmd('incident.sla-resume', 'Resume the incident SLA clock', 'itsm incident sla-resume <id>', ['PUT /api/v1/incidents/:id/sla/resume'], async (ctx) => apiClient.resumeIncidentSLA(id(ctx))),
        cmd('incident.report', 'Windowed incident distribution and daily trend', 'itsm incident report [--dateFrom d] [--dateTo d]', ['GET /api/v1/incidents/stats/report'], async (ctx) => apiClient.incidentReport({
            dateFrom: optionalString(ctx.args.flags, 'dateFrom'),
            dateTo: optionalString(ctx.args.flags, 'dateTo'),
        })),
    ];
}

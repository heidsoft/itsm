import { cmd } from '../spec.js';
import { requirePositional, body, stringFlag } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/** BPMN workflow engine — bodies mirror handlers/bpmn inline contracts. */
function taskId(ctx) {
    return requirePositional(ctx.args, 0, 'taskId');
}
export function workflowCommands() {
    return [
        cmd('workflow.definitions', 'List process definitions', 'itsm workflow definitions', ['GET /api/v1/bpmn/process-definitions'], async (ctx) => apiClient.listBPMNDefinitions()),
        cmd('workflow.definition', 'Process definition detail (by key)', 'itsm workflow definition <key>', ['GET /api/v1/bpmn/process-definitions/:key'], async (ctx) => apiClient.getBPMNDefinition(requirePositional(ctx.args, 0, 'key'))),
        cmd('workflow.instances', 'List process instances', 'itsm workflow instances', ['GET /api/v1/bpmn/process-instances'], async (ctx) => apiClient.listProcessInstances()),
        cmd('workflow.instance', 'Process instance detail', 'itsm workflow instance <id>', ['GET /api/v1/bpmn/process-instances/:id'], async (ctx) => apiClient.getProcessInstance(taskId(ctx))),
        cmd('workflow.start', 'Start an instance (processDefinitionKey required)', 'itsm workflow start <processDefinitionKey> [--businessKey k] [--variables <json>] | --data <json>', ['POST /api/v1/bpmn/process-instances'], async (ctx) => {
            const data = body(ctx.args);
            if (data)
                return apiClient.startProcess(data);
            const key = requirePositional(ctx.args, 0, 'processDefinitionKey');
            const payload = { processDefinitionKey: key };
            const businessKey = stringFlag(ctx.args.flags, 'businessKey');
            if (businessKey)
                payload['businessKey'] = businessKey;
            const variables = stringFlag(ctx.args.flags, 'variables');
            if (variables) {
                try {
                    payload['variables'] = JSON.parse(variables);
                }
                catch {
                    throw usageError('--variables must be a JSON object');
                }
            }
            return apiClient.startProcess(payload);
        }),
        cmd('workflow.variables', 'Set instance variables (required)', 'itsm workflow variables <instanceId> --variables <json>', ['PUT /api/v1/bpmn/process-instances/:id/variables'], async (ctx) => apiClient.setProcessInstanceVariables(requirePositional(ctx.args, 0, 'instanceId'), variablesOf(ctx.args.flags))),
        cmd('workflow.suspend', 'Suspend an instance (reason required)', 'itsm workflow suspend <instanceId> --reason text', ['PUT /api/v1/bpmn/process-instances/:id/suspend'], async (ctx) => {
            const reason = requireReason(ctx.args.flags);
            return apiClient.processInstanceAction(requirePositional(ctx.args, 0, 'instanceId'), 'suspend', { reason });
        }),
        cmd('workflow.resume', 'Resume an instance (no body)', 'itsm workflow resume <instanceId>', ['PUT /api/v1/bpmn/process-instances/:id/resume'], async (ctx) => apiClient.processInstanceAction(requirePositional(ctx.args, 0, 'instanceId'), 'resume')),
        cmd('workflow.terminate', 'Terminate an instance (reason required, irreversible)', 'itsm workflow terminate <instanceId> --reason text', ['PUT /api/v1/bpmn/process-instances/:id/terminate'], async (ctx) => {
            const reason = requireReason(ctx.args.flags);
            return apiClient.processInstanceAction(requirePositional(ctx.args, 0, 'instanceId'), 'terminate', { reason });
        }),
        cmd('workflow.tasks', 'My active human tasks', 'itsm workflow tasks', ['GET /api/v1/bpmn/tasks'], async () => apiClient.listUserTasks()),
        cmd('workflow.task', 'Task detail', 'itsm workflow task <taskId>', ['GET /api/v1/bpmn/tasks/:id'], async (ctx) => apiClient.getTask(taskId(ctx))),
        cmd('workflow.claim', 'Claim an unassigned task (no body; user from auth)', 'itsm workflow claim <taskId>', ['PUT /api/v1/bpmn/tasks/:id/claim'], async (ctx) => apiClient.claimTask(taskId(ctx))),
        cmd('workflow.assign', 'Assign a task to a user', 'itsm workflow assign <taskId> --assigneeId <n>', ['PUT /api/v1/bpmn/tasks/:id/assign'], async (ctx) => apiClient.assignTask(taskId(ctx), requireNumber(ctx.args.flags, 'assigneeId'))),
        cmd('workflow.complete', 'Complete task with optional variables', 'itsm workflow complete <taskId> [--variables <json>]', ['PUT /api/v1/bpmn/tasks/:id/complete'], async (ctx) => {
            const v = stringFlag(ctx.args.flags, 'variables');
            return apiClient.completeTask(taskId(ctx), v ? variablesOf(ctx.args.flags) : undefined);
        }),
        cmd('workflow.decision', 'Submit an approval decision: approve | reject | delegate | add_approver', 'itsm workflow decision <taskId> --action approve [--comment text]', ['POST /api/v1/bpmn/tasks/:id/decisions'], async (ctx) => apiClient.decideTask(taskId(ctx), decisionBody(ctx.args.flags))),
        cmd('workflow.cancel', 'Cancel a task (reason required)', 'itsm workflow cancel <taskId> --reason text', ['PUT /api/v1/bpmn/tasks/:id/cancel'], async (ctx) => apiClient.cancelTask(taskId(ctx), requireReason(ctx.args.flags))),
    ];
}
function requireReason(flags) {
    const v = stringFlag(flags, 'reason');
    if (typeof v === 'string' && v)
        return v;
    throw usageError('Missing required flag --reason');
}
function requireNumber(flags, name) {
    const v = stringFlag(flags, name) ?? flags[name];
    const n = Number(v);
    if (!v || Number.isNaN(n))
        throw usageError(`Missing required flag --${name} <number>`);
    return n;
}
/** Builds the inline decision body: {action required, comment?, variables?, delegateToUserId?, addApproverUserId?}. */
function decisionBody(flags) {
    const action = (stringFlag(flags, 'action') ?? 'approve').toLowerCase();
    if (!['approve', 'reject', 'delegate', 'add_approver'].includes(action)) {
        throw usageError('--action must be one of approve|reject|delegate|add_approver');
    }
    const out = { action };
    const comment = stringFlag(flags, 'comment');
    if (typeof comment === 'string' && comment)
        out['comment'] = comment;
    const variables = stringFlag(flags, 'variables');
    if (variables)
        out['variables'] = JSON.parse(variables);
    if (action === 'delegate') {
        const delegateToUserId = flags['delegateToUserId'];
        if (delegateToUserId === undefined)
            throw usageError('delegate needs --delegateToUserId <n>');
        out['delegateToUserId'] = Number(delegateToUserId);
    }
    if (action === 'add_approver') {
        const addApproverUserId = flags['addApproverUserId'];
        if (addApproverUserId === undefined)
            throw usageError('add_approver needs --addApproverUserId <n>');
        out['addApproverUserId'] = Number(addApproverUserId);
    }
    return out;
}
function variablesOf(flags) {
    const v = stringFlag(flags, 'variables');
    try {
        return v ? JSON.parse(v) : {};
    }
    catch {
        throw usageError('--variables must be a JSON object');
    }
}

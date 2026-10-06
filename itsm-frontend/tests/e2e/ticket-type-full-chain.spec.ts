import { randomBytes } from 'node:crypto';
import { test, expect, type Page } from '@playwright/test';
import { requireIsolatedStack } from '../../playwright.ticket-type.config';
import {
  assertPage,
  get,
  login,
  mutate,
  single,
  write,
  type PageDTO,
  type SessionUser,
} from './ticket-type-full-chain.support';

const baseURL = requireIsolatedStack();
const TYPES = '/api/v1/ticket-types';
const TICKETS = '/api/v1/tickets';
const INSTANCES = '/api/v1/bpmn/process-instances';
const TASKS = '/api/v1/bpmn/tasks';
const COMMANDS = '/api/v1/admin/operations/commands';

interface TicketType {
  id: number;
  code: string;
  tenantId: number;
  createdBy: number;
  workflowDefinitionKey: string;
  defaultSlaId: number;
  assignmentRuleId: number;
}
interface Ticket {
  id: number;
  tenantId: number;
  requesterId: number;
  assigneeId: number;
  ticketTypeId: number;
  createdAt: string;
  slaResponseDeadline?: string;
  slaResolutionDeadline?: string;
}
interface Instance {
  id: string;
  instanceId: string;
  businessKey: string;
  processDefinitionKey: string;
  currentActivityId: string;
  status: string;
  tenantId: number;
  startTime: string;
  endTime: string;
}
interface Task {
  id: number;
  taskId: string;
  taskDefinitionKey: string;
  processInstanceId: number;
  processInstanceKey: string;
  businessKey: string;
  assignee: string;
  status: string;
}
interface Command {
  id: number;
  tenantId: number;
  commandType: string;
  aggregateType: string;
  aggregateId: number;
  idempotencyKey: string;
  status: string;
  attempt: number;
  fencingToken: number;
  completedAt?: string;
}
interface CommandPage {
  items: Command[];
  total: number;
  page: number;
  pageSize: number;
}
interface Audit {
  tenantId: number;
  userId: number;
  resource: string;
  action: string;
  requestBody: string;
}
interface Timeline {
  processInstanceId: string;
  total: number;
  entries: {
    id: string;
    eventType: string;
    activityId: string;
    activityType: string;
    tenantId: number;
    userId: number;
  }[];
}

async function workflowCommands(page: Page, ticketId: number): Promise<Command[]> {
  const result = await get<CommandPage>(
    page,
    `${COMMANDS}?commandType=workflow.start&aggregateType=ticket&page=1&pageSize=100`
  );
  expect(result).toMatchObject({ page: 1, pageSize: 100, items: expect.any(Array) });
  expect(result.total).toBeLessThanOrEqual(100);
  expect(result.items).toHaveLength(result.total);
  return result.items.filter(command => command.aggregateId === ticketId);
}

async function ticketTasks(page: Page, businessKey: string): Promise<Task[]> {
  // Use the typed list view, not GET /tasks/:id (which still serializes Ent).
  const result = await get<PageDTO<Task>>(page, `${TASKS}?page=1&pageSize=20`);
  assertPage(result);
  return result.items.filter(task => task.businessKey === businessKey);
}

function timestamp(value: string | undefined): number {
  expect(value).toEqual(expect.any(String));
  if (value === undefined) throw new Error('Required timestamp is missing');
  const parsed = Date.parse(value);
  expect(Number.isFinite(parsed)).toBe(true);
  return parsed;
}

function assertSLA(ticket: Ticket): void {
  // Compare to the authoritative persisted creation time, never browser time.
  const created = timestamp(ticket.createdAt);
  const response = timestamp(ticket.slaResponseDeadline);
  const resolution = timestamp(ticket.slaResolutionDeadline);
  for (const [deadline, minutes] of [
    [response, 37],
    [resolution, 247],
  ]) {
    const calculationDelay = deadline - created - minutes * 60_000;
    expect(calculationDelay).toBeGreaterThanOrEqual(-1_000);
    expect(calculationDelay).toBeLessThan(60_000);
  }
  expect(resolution - response).toBe(210 * 60_000);
}

test('TicketType preset → HTTP ticket → real outbox worker → task → audit, with tenant isolation', async ({
  page,
  browser,
}, testInfo) => {
  test.setTimeout(180_000);
  expect(testInfo.project.use.baseURL).toBe(baseURL);
  const password = process.env.E2E_ADMIN_PASSWORD;
  if (!password) throw new Error('E2E_ADMIN_PASSWORD is required for the disposable stack');
  const suffix = randomBytes(6).toString('hex');
  const code = `e2e_pacs_${suffix}`;
  const workflowKey = `e2e_ticket_type_${suffix}`;
  const admin = await login(page, 'admin', password);

  // Fixtures are real API writes. Teardown is Compose down -v, not partial
  // domain deletes that erase failure evidence or leave outbox jobs orphaned.
  const installed =
    await test.step('Install the PACS preset through its production route', async () => {
      const presets = await get<{ id: string }[]>(page, '/api/v1/ticket-type-presets');
      expect(presets.some(preset => preset.id === 'pacs-incident')).toBe(true);
      const result = await write<TicketType>(
        page,
        'POST',
        '/api/v1/ticket-type-presets/pacs-incident/install',
        {
          code,
          name: `E2E PACS ${suffix}`,
        }
      );
      expect(result).toMatchObject({
        id: expect.any(Number),
        code,
        tenantId: admin.tenantId,
        createdBy: admin.id,
        status: 'active',
        defaultPriority: 'high',
        customFields: expect.arrayContaining([
          expect.objectContaining({ name: 'pacsNode', required: true }),
          expect.objectContaining({ name: 'affectedDepartment', required: true }),
        ]),
      });
      return result;
    });

  const department = await write<{ id: number; tenantId: number }>(
    page,
    'POST',
    '/api/v1/org/departments',
    {
      name: `E2E Radiology ${suffix}`,
      code: `radiology-${suffix}`,
    }
  );
  expect(department.tenantId).toBe(admin.tenantId);
  const assignee = await write<SessionUser>(page, 'POST', '/api/v1/users', {
    username: `agent${suffix}`,
    name: `E2E Agent ${suffix}`,
    email: `agent${suffix}@example.invalid`,
    password: `E2e!${randomBytes(16).toString('hex')}`,
    role: 'agent',
  });
  expect(assignee.tenantId).toBe(admin.tenantId);
  expect(assignee.id).not.toBe(admin.id);

  const { sla, assignment } =
    await test.step('Publish workflow and bind exact SLA and assignment rule', async () => {
      // Same minimal executable graph as service/ticket_type_workflow_e2e_test.go.
      // No actor variables supplied: the worker reloads requester from the ticket.
      const bpmnXml = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="${workflowKey}" name="Ticket Type HTTP E2E" isExecutable="true">
    <bpmn:startEvent id="start" name="Start"/>
    <bpmn:userTask id="handle" name="Handle Ticket"/>
    <bpmn:endEvent id="end" name="End"/>
    <bpmn:sequenceFlow id="flow_start" sourceRef="start" targetRef="handle"/>
    <bpmn:sequenceFlow id="flow_end" sourceRef="handle" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`;
      const definition = await write<{ key: string }>(
        page,
        'POST',
        '/api/v1/bpmn/process-definitions',
        {
          key: workflowKey,
          name: `E2E ${suffix}`,
          bpmnXml,
          publish: true,
        }
      );
      expect(definition).toMatchObject({
        key: workflowKey,
        isActive: true,
        tenantId: admin.tenantId,
      });
      const sla = await write<{ id: number }>(page, 'POST', '/api/v1/sla/definitions', {
        name: `E2E SLA ${suffix}`,
        serviceType: 'incident',
        priority: 'high',
        responseTime: 37,
        resolutionTime: 247,
        businessHours: {},
        isActive: true,
      });
      expect(sla).toMatchObject({
        tenantId: admin.tenantId,
        responseTime: 37,
        resolutionTime: 247,
      });
      const assignment = await write<{ id: number }>(page, 'POST', `${TICKETS}/assignment-rules`, {
        name: `E2E Assignment ${suffix}`,
        priority: 100,
        conditions: [],
        actions: { type: 'user', value: assignee.id },
        isActive: true,
      });
      const bindings = {
        workflowDefinitionKey: workflowKey,
        slaEnabled: true,
        defaultSlaId: sla.id,
        autoAssignEnabled: true,
        assignmentRuleId: assignment.id,
      };
      expect(
        await write<TicketType>(page, 'PUT', `${TYPES}/${installed.id}`, bindings)
      ).toMatchObject(bindings);
      expect(await get<TicketType>(page, `${TYPES}/${installed.id}`)).toMatchObject(bindings);
      return { sla, assignment };
    });

  const ticket =
    await test.step('Create ticket without client-supplied requester, tenant or assignee', async () => {
      const created = await write<Ticket>(page, 'POST', TICKETS, {
        title: `E2E full chain ${suffix}`,
        description: 'Disposable HTTP/worker regression',
        priority: 'medium',
        type: 'incident',
        ticketTypeId: installed.id,
        formFields: { pacsNode: `node-${suffix}`, affectedDepartment: department.id },
      });
      expect(created).toMatchObject({
        id: expect.any(Number),
        tenantId: admin.tenantId,
        requesterId: admin.id,
        ticketTypeId: installed.id,
        ticketTypeCode: code,
        priority: 'high',
      });
      const detail = await get<Ticket>(page, `${TICKETS}/${created.id}`);
      expect(detail).toMatchObject({ assigneeId: assignee.id, requesterId: admin.id });
      assertSLA(detail);
      return detail;
    });
  const businessKey = `ticket:${ticket.id}`;

  const command =
    await test.step('Observe real workflow.start consumption (never invoke engine or replay)', async () => {
      await expect
        .poll(
          async () => {
            const commands = await workflowCommands(page, ticket.id);
            expect(commands.length).toBeLessThanOrEqual(1);
            return commands.map(item => item.status);
          },
          {
            timeout: 60_000,
            intervals: [500, 1_000, 2_000],
            message: `Worker must consume ${businessKey}`,
          }
        )
        .toEqual(['succeeded']);
      const result = single(await workflowCommands(page, ticket.id), 'one durable workflow start');
      expect(result).toMatchObject({
        tenantId: admin.tenantId,
        commandType: 'workflow.start',
        aggregateType: 'ticket',
        aggregateId: ticket.id,
        idempotencyKey: `ticket:${ticket.id}:workflow:start`,
        status: 'succeeded',
      });
      expect(result.attempt).toBeGreaterThanOrEqual(1);
      expect(result.fencingToken).toBeGreaterThanOrEqual(1);
      timestamp(result.completedAt);
      return result;
    });

  const instances = await get<PageDTO<Instance>>(page, `${INSTANCES}?page=1&pageSize=20`);
  assertPage(instances);
  const instance = single(
    instances.items.filter(item => item.businessKey === businessKey),
    'one workflow instance'
  );
  expect(instance).toMatchObject({
    businessKey,
    processDefinitionKey: workflowKey,
    currentActivityId: 'handle',
    tenantId: admin.tenantId,
    status: 'running',
  });
  expect(instance.instanceId).not.toBe('');
  const task = single(await ticketTasks(page, businessKey), 'one requester task');
  expect(task).toMatchObject({
    taskDefinitionKey: 'handle',
    processInstanceId: Number(instance.id),
    processInstanceKey: instance.instanceId,
    businessKey,
    assignee: String(admin.id),
    status: 'created',
  });

  await test.step('A separately logged-in tenant cannot read tenant A’s installed type', async () => {
    const tenantCode = `e2e-other-${suffix}`;
    const otherTenant = await write<{ id: number }>(page, 'POST', '/api/v1/tenants', {
      name: `E2E Other ${suffix}`,
      code: tenantCode,
      type: 'standard',
    });
    expect(otherTenant.id).not.toBe(admin.tenantId);
    // Match tenant-provisioning.spec.ts, but use only the authoritative DTO.
    await expect
      .poll(
        async () => {
          const state = await get<{
            ready: boolean;
            commandStatus: string;
            components: { verified: boolean }[];
          }>(page, `/api/v1/tenants/${otherTenant.id}/initialization`);
          return {
            ready: state.ready,
            commandStatus: state.commandStatus,
            verified: state.components.length > 0 && state.components.every(item => item.verified),
          };
        },
        { timeout: 60_000, intervals: [500, 1_000, 2_000] }
      )
      .toEqual({ ready: true, commandStatus: 'succeeded', verified: true });

    const otherContext = await browser.newContext({ baseURL });
    try {
      const otherPage = await otherContext.newPage();
      const username = `other${suffix}`;
      const otherPassword = `E2e!${randomBytes(16).toString('hex')}`;
      // P0-1 removed the tenantCode registration contract: self-registration
      // fail-closes with multiple active tenants. Tenant B's user is provisioned
      // by the super_admin switching into the tenant, then creating through the
      // standard own-tenant user API.
      await login(otherPage, 'admin', password);
      await write(otherPage, 'POST', '/api/v1/auth/switch-tenant', { tenantId: otherTenant.id });
      await write(otherPage, 'POST', '/api/v1/users', {
        username,
        email: `${username}@example.invalid`,
        password: otherPassword,
        name: 'E2E Other',
      });
      const other = await login(otherPage, username, otherPassword);
      expect(other.tenantId).toBe(otherTenant.id);
      expect(other.id).not.toBe(admin.id);
      // Positive route-permission control: a 403 from generic RBAC is not proof of isolation.
      const ownTypes = await get<PageDTO<TicketType>>(otherPage, `${TYPES}?page=1&pageSize=20`);
      assertPage(ownTypes);
      for (const item of ownTypes.items) expect(item.tenantId).toBe(other.tenantId);
      expect(ownTypes.items.some(item => item.id === installed.id)).toBe(false);
      const denied = await otherPage.request.get(`${TYPES}/${installed.id}`, { maxRedirects: 0 });
      expect(denied.status()).toBe(404);
      // common.Fail omits data; do not accept an alternative error envelope.
      expect(await denied.json()).toEqual({ code: 4004, message: '工单类型不存在' });
    } finally {
      await otherContext.close();
    }
    expect(await get<TicketType>(page, `${TYPES}/${installed.id}`)).toMatchObject({
      tenantId: admin.tenantId,
      defaultSlaId: sla.id,
      assignmentRuleId: assignment.id,
    });
  });

  await test.step('Complete task via HTTP and verify persisted instance, task and audits', async () => {
    const completion = await mutate(page, 'PUT', `${TASKS}/${task.id}/complete`, {
      variables: { resolution: 'done' },
    });
    // This route returns a message-only envelope, not a data DTO.
    expect(completion.status()).toBe(200);
    expect(await completion.json()).toEqual({ code: 0, message: '任务完成成功' });
    await expect
      .poll(async () => (await get<Instance>(page, `${INSTANCES}/${instance.instanceId}`)).status, {
        timeout: 15_000,
        intervals: [250, 500, 1_000],
      })
      .toBe('completed');
    const completed = await get<Instance>(page, `${INSTANCES}/${instance.instanceId}`);
    expect(completed).toMatchObject({
      id: instance.id,
      currentActivityId: 'end',
      tenantId: admin.tenantId,
    });
    expect(timestamp(completed.endTime)).toBeGreaterThanOrEqual(timestamp(completed.startTime));
    expect(
      single(await ticketTasks(page, businessKey), 'completed task remains queryable')
    ).toMatchObject({ id: task.id, status: 'completed', assignee: String(admin.id) });

    const timeline = await get<Timeline>(
      page,
      `/api/v1/bpmn/monitoring/instances/${instance.instanceId}/timeline`
    );
    expect(timeline.processInstanceId).toBe(instance.instanceId);
    expect(timeline.entries).toHaveLength(timeline.total);
    expect(timeline.entries).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          eventType: 'started',
          activityId: 'start',
          activityType: 'startEvent',
        }),
        // The wired completion path records the user task audit transactionally.
        // RecordProcessCompleted exists but is not invoked by completeProcess.
        expect.objectContaining({
          eventType: 'completed',
          activityId: 'handle',
          activityType: 'user_task',
          userId: admin.id,
        }),
      ])
    );
    for (const entry of timeline.entries) expect(entry.tenantId).toBe(admin.tenantId);

    for (const action of ['preset.install', 'binding.update']) {
      const audits = await get<{ items: Audit[]; total: number }>(
        page,
        `/api/v1/audit-logs?resource=ticket_type:${installed.id}&action=${action}&page=1&pageSize=20`
      );
      expect(audits.total).toBe(1);
      const audit = single(audits.items, action);
      expect(audit).toMatchObject({
        tenantId: admin.tenantId,
        userId: admin.id,
        resource: `ticket_type:${installed.id}`,
        action,
      });
      const details: unknown = JSON.parse(audit.requestBody);
      expect(details).toMatchObject(
        action === 'preset.install'
          ? { presetId: 'pacs-incident', code }
          : { workflowDefinitionKey: { after: workflowKey } }
      );
    }
    const finalTicket = await get<Ticket>(page, `${TICKETS}/${ticket.id}`);
    expect(finalTicket).toMatchObject({
      assigneeId: assignee.id,
      requesterId: admin.id,
      slaResponseDeadline: ticket.slaResponseDeadline,
      slaResolutionDeadline: ticket.slaResolutionDeadline,
    });
    assertSLA(finalTicket);
    expect(
      single(await workflowCommands(page, ticket.id), 'no duplicate workflow command')
    ).toMatchObject({ id: command.id, status: 'succeeded' });
    const finalInstances = await get<PageDTO<Instance>>(page, `${INSTANCES}?page=1&pageSize=20`);
    assertPage(finalInstances);
    expect(finalInstances.items.filter(item => item.businessKey === businessKey)).toHaveLength(1);
    await testInfo.attach('full-chain-identities', {
      body: JSON.stringify({
        tenantId: admin.tenantId,
        actorId: admin.id,
        ticketTypeId: installed.id,
        ticketId: ticket.id,
        commandId: command.id,
        instanceId: instance.instanceId,
        taskId: task.id,
      }),
      contentType: 'application/json',
    });
  });
});

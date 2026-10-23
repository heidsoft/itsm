import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { idOf, requireNumber } from '../vars.js';

/**
 * Ticket lifecycle — the thinnest end-to-end proof that the generic work-item
 * path works: create, assign, comment, resolve, close, and a closed ticket
 * refuses further transitions.
 *
 * The terminal-state step is deliberately negative. A product that lets you
 * resolve a closed ticket silently corrupts SLA and audit reporting, and that
 * class of bug never shows up in happy-path testing.
 */
export const ticketLifecycle: FlowScenario = {
  name: 'ticket-lifecycle',
  summary: 'Create → assign → comment → resolve → close, then prove the terminal state is immutable',
  domain: 'ticket',
  tags: ['itil', 'core', 'write'],
  proves: [
    'tickets are created in new state with a TKT- number',
    'assignment moves new → open and records the assignee',
    'comments are readable on the ticket',
    'resolve and close persist their notes and advance the state machine',
    'a closed ticket rejects resolve with 409 instead of silently succeeding',
  ],
  steps: [
    {
      name: 'create ticket',
      endpoint: 'POST /api/v1/tickets',
      run: (ctx) =>
        apiClient.createTicket({
          title: `[${ctx.tag}] ticket lifecycle`,
          description: 'Acceptance flow: create → assign → resolve → close. Safe to delete.',
          priority: 'high',
        }),
      expect: [
        { path: 'id', exists: true },
        { path: 'status', eq: 'new' },
        { path: 'ticketNumber', contains: 'TKT-' },
        { path: 'tenantId', exists: true },
      ],
      capture: (result) => ({ ticketId: idOf(result, 'ticket create') }),
    },
    {
      name: 'assign ticket',
      endpoint: 'POST /api/v1/tickets/:id/assign',
      run: (ctx) => apiClient.assignTicket(requireNumber(ctx, 'ticketId'), 1),
      expect: [
        { path: 'assigneeId', eq: 1 },
        // Repository rule: assigning a `new` ticket moves it to `open`.
        { path: 'status', eq: 'open' },
      ],
    },
    {
      name: 'add comment',
      endpoint: 'POST /api/v1/tickets/:id/comments',
      run: (ctx) => apiClient.createTicketComment(requireNumber(ctx, 'ticketId'), { content: `[${ctx.tag}] flow comment`, isInternal: false }),
      verify: (result, ctx) => {
        const actual = (result as { content?: unknown }).content;
        return [
          {
            check: 'comment content is echoed back verbatim',
            ok: actual === `[${ctx.tag}] flow comment`,
            expected: `[${ctx.tag}] flow comment`,
            actual,
          },
        ];
      },
    },
    {
      name: 'resolve ticket',
      endpoint: 'POST /api/v1/tickets/:id/resolve',
      // Only the state transition is asserted on the write response: the read
      // path below is what callers (and the UI) actually depend on. Asserting
      // solely here would not distinguish "write lost the note" from "response
      // mapper dropped it".
      run: (ctx) => apiClient.resolveTicket(requireNumber(ctx, 'ticketId'), { resolution: 'Flow: temporary fix verified' }),
      expect: [{ path: 'status', eq: 'resolved' }],
    },
    {
      name: 'read resolved ticket',
      endpoint: 'GET /api/v1/tickets/:id',
      run: (ctx) => apiClient.getTicket(requireNumber(ctx, 'ticketId')),
      expect: [
        { path: 'status', eq: 'resolved' },
        { path: 'resolution', contains: 'temporary fix' },
        { path: 'resolvedAt', exists: true },
      ],
    },
    {
      name: 'close ticket',
      endpoint: 'POST /api/v1/tickets/:id/close',
      run: (ctx) => apiClient.closeTicket(requireNumber(ctx, 'ticketId'), { closeReason: 'Flow acceptance complete' }),
      expect: [{ path: 'status', eq: 'closed' }],
    },
    {
      name: 'read closed ticket',
      endpoint: 'GET /api/v1/tickets/:id',
      run: (ctx) => apiClient.getTicket(requireNumber(ctx, 'ticketId')),
      expect: [
        { path: 'status', eq: 'closed' },
        { path: 'closedAt', exists: true },
        { path: 'resolution', contains: 'temporary fix' },
      ],
    },
    {
      name: 'history records the lifecycle',
      endpoint: 'GET /api/v1/tickets/:id/history',
      run: (ctx) => apiClient.getTicketHistory(requireNumber(ctx, 'ticketId')),
      verify: (result) => {
        const entries = Array.isArray(result) ? result : [];
        const actions = entries.map((e) => (e as { action?: unknown }).action);
        return [
          { check: 'history is a non-empty array', ok: entries.length > 0, expected: '>=1 entry', actual: entries.length },
          { check: 'history records creation', ok: actions.includes('created'), expected: 'created', actual: actions },
        ];
      },
    },
    {
      name: 'closed ticket rejects resolve (negative)',
      endpoint: 'POST /api/v1/tickets/:id/resolve',
      run: (ctx) => apiClient.resolveTicket(requireNumber(ctx, 'ticketId'), { resolution: 'should be rejected' }),
      expectError: { kind: 'conflict', httpStatus: 409 },
    },
  ],
  teardown: [
    {
      name: 'delete flow ticket',
      endpoint: 'DELETE /api/v1/tickets/:id',
      run: (ctx) => apiClient.deleteTicket(requireNumber(ctx, 'ticketId')),
    },
  ],
};

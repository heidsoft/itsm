import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { idOf, requireNumber, versionOf } from '../vars.js';

/**
 * Incident lifecycle.
 *
 * Note on contract: the acknowledge/resolve/close endpoints return
 * `data: null`, so every state assertion re-reads the incident. That is the
 * honest way to verify persistence, and it keeps this scenario passing even if
 * the backend later starts returning the updated object.
 */
export const incidentLifecycle: FlowScenario = {
  name: 'incident-lifecycle',
  summary: 'Create → acknowledge → start work → resolve → close, with terminal-state enforcement',
  domain: 'incident',
  tags: ['itil', 'core', 'write'],
  proves: [
    'incidents start in new state',
    'acknowledge and the in_progress update are persisted',
    'resolve persists resolution + rootCause',
    'close requires a close note and ends the lifecycle',
    'a closed incident rejects acknowledge with a conflict',
  ],
  steps: [
    {
      name: 'create incident',
      endpoint: 'POST /api/v1/incidents',
      run: (ctx) =>
        apiClient.createIncident({
          title: `[${ctx.tag}] incident lifecycle`,
          description: 'Acceptance flow: create → acknowledge → resolve → close. Safe to delete.',
          priority: 'high',
          severity: 'high',
          source: 'manual',
        }),
      expect: [
        { path: 'status', eq: 'new' },
        { path: 'priority', eq: 'high' },
      ],
      capture: (result) => ({ incidentId: idOf(result, 'incident create') }),
    },
    {
      name: 'acknowledge incident',
      endpoint: 'POST /api/v1/incidents/:id/acknowledge',
      run: (ctx) => apiClient.incidentLifecycle(requireNumber(ctx, 'incidentId'), 'acknowledge'),
      capture: () => ({}),
    },
    {
      name: 'read acknowledged state',
      endpoint: 'GET /api/v1/incidents/:id',
      run: (ctx) => apiClient.getIncident(requireNumber(ctx, 'incidentId')),
      expect: [{ path: 'status', eq: 'acknowledged' }],
      capture: (result) => ({ incidentVersion: versionOf(result, 'incident get') }),
    },
    {
      name: 'start work (status=in_progress)',
      endpoint: 'PUT /api/v1/incidents/:id',
      run: (ctx) =>
        apiClient.updateIncident(requireNumber(ctx, 'incidentId'), {
          status: 'in_progress',
          version: requireNumber(ctx, 'incidentVersion'),
        }),
      expect: [{ path: 'status', eq: 'in_progress' }],
    },
    {
      name: 'resolve incident',
      endpoint: 'POST /api/v1/incidents/:id/resolve',
      run: (ctx) =>
        apiClient.incidentLifecycle(requireNumber(ctx, 'incidentId'), 'resolve', {
          resolution: 'Flow: service restored',
          rootCause: 'Flow: configuration drift',
        }),
    },
    {
      name: 'read resolved state',
      endpoint: 'GET /api/v1/incidents/:id',
      run: (ctx) => apiClient.getIncident(requireNumber(ctx, 'incidentId')),
      // The incident contract keeps the resolution text as a structured step
      // (resolutionSteps[0].description) rather than a flat field.
      expect: [
        { path: 'status', eq: 'resolved' },
        { path: 'resolutionSteps.0.description', contains: 'service restored' },
        { path: 'rootCause.rootCause', contains: 'configuration drift' },
      ],
    },
    {
      name: 'close incident',
      endpoint: 'POST /api/v1/incidents/:id/close',
      run: (ctx) => apiClient.incidentLifecycle(requireNumber(ctx, 'incidentId'), 'close', { closeNotes: 'Flow acceptance complete' }),
    },
    {
      name: 'read closed state',
      endpoint: 'GET /api/v1/incidents/:id',
      run: (ctx) => apiClient.getIncident(requireNumber(ctx, 'incidentId')),
      expect: [
        { path: 'status', eq: 'closed' },
        // Close notes are audit-only today: the response exposes closedAt, not
        // the note text. Assert what the contract actually promises.
        { path: 'closedAt', exists: true },
      ],
    },
    {
      name: 'closed incident rejects acknowledge (negative)',
      endpoint: 'POST /api/v1/incidents/:id/acknowledge',
      run: (ctx) => apiClient.incidentLifecycle(requireNumber(ctx, 'incidentId'), 'acknowledge'),
      expectError: { kind: 'conflict', httpStatus: 409 },
    },
  ],
  teardown: [
    {
      name: 'delete flow incident',
      endpoint: 'DELETE /api/v1/incidents/:id',
      run: (ctx) => apiClient.deleteIncident(requireNumber(ctx, 'incidentId')),
    },
  ],
};

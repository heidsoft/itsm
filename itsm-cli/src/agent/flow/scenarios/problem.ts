import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { idOf, requireNumber } from '../vars.js';

/**
 * Problem management: investigate → root cause → workaround/solution → close.
 * This is the path that feeds the known-error database, so a silent state bug
 * here breaks knowledge reuse downstream.
 */
export const problemLifecycle: FlowScenario = {
  name: 'problem-lifecycle',
  summary: 'Create → investigate → record root cause + solution → close, then reject re-investigation',
  domain: 'problem',
  tags: ['itil', 'core', 'write'],
  proves: [
    'problems start in new state',
    'investigate advances to investigating',
    'root cause and solution are persisted on the problem',
    'close records the resolution and reaches the terminal state',
    'a closed problem refuses a new investigation',
  ],
  steps: [
    {
      name: 'create problem',
      endpoint: 'POST /api/v1/problems',
      run: (ctx) =>
        apiClient.createProblem({
          title: `[${ctx.tag}] problem lifecycle`,
          description: 'Acceptance flow: repeated incident root-cause analysis. Safe to delete.',
          priority: 'high',
          category: 'infrastructure',
        }),
      expect: [
        { path: 'status', in: ['open', 'new'] },
        { path: 'priority', eq: 'high' },
      ],
      capture: (result) => ({ problemId: idOf(result, 'problem create') }),
    },
    {
      name: 'start investigation',
      endpoint: 'POST /api/v1/problems/:id/investigate',
      run: (ctx) => apiClient.problemLifecycle(requireNumber(ctx, 'problemId'), 'investigate'),
      expect: [{ path: 'status', eq: 'investigating' }],
    },
    {
      name: 'record root cause',
      endpoint: 'PUT /api/v1/problems/:id/root-cause',
      run: (ctx) =>
        apiClient.updateProblemRootCause(requireNumber(ctx, 'problemId'), {
          rootCause: 'Connection pool exhausted by a leaked long-running query',
        }),
      expect: [{ path: 'rootCause', contains: 'Connection pool' }],
    },
    {
      name: 'record workaround + solution',
      endpoint: 'PUT /api/v1/problems/:id/solution',
      run: (ctx) =>
        apiClient.updateProblemSolution(requireNumber(ctx, 'problemId'), {
          workaround: 'Restart the affected worker and raise the pool ceiling',
          solution: 'Fix the query timeout and release connections in a defer block',
        }),
      expect: [
        { path: 'workaround', contains: 'Restart' },
        // The backend accepts `solution` as an alias but persists it in the
        // `resolution` field; the response only carries `resolution`.
        { path: 'resolution', contains: 'defer block' },
      ],
    },
    {
      name: 'resolve problem',
      endpoint: 'PUT /api/v1/problems/:id',
      // There is no dedicated /resolve action for problems; the documented path
      // is a status update (investigating → resolved) before close.
      run: (ctx) => apiClient.updateProblem(requireNumber(ctx, 'problemId'), { status: 'resolved' }),
      expect: [{ path: 'status', eq: 'resolved' }],
    },
    {
      name: 'close problem',
      endpoint: 'POST /api/v1/problems/:id/close',
      run: (ctx) => apiClient.problemLifecycle(requireNumber(ctx, 'problemId'), 'close', { resolution: 'Permanent fix deployed' }),
      expect: [
        { path: 'status', eq: 'closed' },
        { path: 'resolution', contains: 'Permanent fix' },
      ],
    },
    {
      name: 'closed problem rejects investigation (negative)',
      endpoint: 'POST /api/v1/problems/:id/investigate',
      run: (ctx) => apiClient.problemLifecycle(requireNumber(ctx, 'problemId'), 'investigate'),
      expectError: { kind: 'conflict', httpStatus: 409 },
    },
  ],
  teardown: [
    {
      name: 'delete flow problem',
      endpoint: 'DELETE /api/v1/problems/:id',
      run: (ctx) => apiClient.deleteProblem(requireNumber(ctx, 'problemId')),
    },
  ],
};

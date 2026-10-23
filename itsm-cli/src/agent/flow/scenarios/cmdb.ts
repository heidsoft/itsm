import { apiClient } from '../../../lib/api-client.js';
import type { FlowScenario } from '../types.js';
import { idOf, requireNumber, blocked } from '../vars.js';

/**
 * CMDB: two CIs, one directed dependency, then topology from the source and
 * impact analysis from the target.
 *
 * The pair matters. Topology and impact traverse in opposite directions; a CMDB
 * that only implements one direction looks fine in the UI until an outage
 * question ("what depends on this database?") gets the wrong answer.
 */
export const cmdbTopology: FlowScenario = {
  name: 'cmdb-topology',
  summary: 'CMDB: create two CIs + relationship, then verify topology and impact traversal',
  domain: 'cmdb',
  tags: ['cmdb', 'topology', 'write'],
  requires: ['at least one active CI type'],
  proves: [
    'CIs can be created against an existing CI type schema',
    'a relationship is persisted with source/target direction intact',
    'topology from the source contains the downstream CI',
    'impact analysis from the target walks back to the upstream CI',
    'a relationship to a nonexistent CI is rejected instead of dangling',
  ],
  steps: [
    {
      name: 'load CI types',
      endpoint: 'GET /api/v1/cmdb/ci-types',
      run: () => apiClient.listCITypes({ pageSize: 50 }),
      verify: (result) => {
        const items = ((result as { items?: unknown[] }).items ?? []) as Array<{ id?: unknown; name?: unknown }>;
        return [{ check: 'at least one CI type exists', ok: items.length > 0, expected: '>=1 CI type', actual: items.length }];
      },
      capture: (result) => {
        const items = ((result as { items?: Array<{ id?: unknown }> }).items ?? []) as Array<{ id?: unknown }>;
        const id = items.map((i) => i.id).find((v): v is number => typeof v === 'number');
        if (id === undefined) blocked('No usable CI type id; seed CMDB CI types before running this flow.');
        return { ciTypeId: id };
      },
    },
    {
      name: 'create upstream CI',
      endpoint: 'POST /api/v1/cmdb/cis',
      run: (ctx) =>
        apiClient.createCI({
          name: `[${ctx.tag}] flow app server`,
          ciTypeId: requireNumber(ctx, 'ciTypeId'),
          status: 'active',
          environment: 'test',
          criticality: 'medium',
          description: 'Acceptance flow artifact. Safe to delete.',
        }),
      expect: [
        { path: 'id', exists: true },
        { path: 'status', eq: 'active' },
      ],
      capture: (result) => ({ upstreamCiId: idOf(result, 'CI create (upstream)') }),
    },
    {
      name: 'create downstream CI',
      endpoint: 'POST /api/v1/cmdb/cis',
      run: (ctx) =>
        apiClient.createCI({
          name: `[${ctx.tag}] flow database`,
          ciTypeId: requireNumber(ctx, 'ciTypeId'),
          status: 'active',
          environment: 'test',
          criticality: 'high',
          description: 'Acceptance flow artifact. Safe to delete.',
        }),
      capture: (result) => ({ downstreamCiId: idOf(result, 'CI create (downstream)') }),
    },
    {
      name: 'relate upstream depends_on downstream',
      endpoint: 'POST /api/v1/cmdb/cis/relationships',
      run: (ctx) =>
        apiClient.createCIRelationship({
          sourceCiId: requireNumber(ctx, 'upstreamCiId'),
          targetCiId: requireNumber(ctx, 'downstreamCiId'),
          relationshipType: 'depends_on',
          strength: 'high',
          impactLevel: 'high',
          description: `[${ctx.tag}] flow dependency`,
        }),
      expect: [
        { path: 'sourceCiId', exists: true },
        { path: 'targetCiId', exists: true },
        { path: 'relationshipType', eq: 'depends_on' },
      ],
      capture: (result) => ({ relationshipId: idOf(result, 'CI relationship create') }),
    },
    {
      name: 'topology from upstream reaches downstream',
      endpoint: 'GET /api/v1/cmdb/cis/:id/topology',
      run: (ctx) => apiClient.getCITopology(requireNumber(ctx, 'upstreamCiId')),
      verify: (result, ctx) => {
        const downstream = ctx.vars['downstreamCiId'];
        const nodes = ((result as { nodes?: Array<{ id?: unknown }> }).nodes ?? []) as Array<{ id?: unknown }>;
        return [
          {
            check: 'topology includes the downstream CI',
            ok: nodes.some((n) => n.id === downstream),
            expected: `node ${String(downstream)}`,
            actual: nodes.map((n) => n.id),
          },
          {
            check: 'topology includes the relationship edge',
            ok: (((result as { edges?: unknown[] }).edges ?? []).length) > 0,
            expected: '>=1 edge',
            actual: ((result as { edges?: unknown[] }).edges ?? []).length,
          },
        ];
      },
    },
    {
      name: 'impact analysis from downstream reaches upstream',
      endpoint: 'GET /api/v1/cmdb/cis/:id/impact-analysis',
      run: (ctx) => apiClient.getCIImpactAnalysis(requireNumber(ctx, 'downstreamCiId')),
      verify: (result, ctx) => {
        const upstream = ctx.vars['upstreamCiId'];
        const graph = (result as { graph?: { nodes?: Array<{ id?: unknown }> } }).graph;
        const nodes = graph?.nodes ?? [];
        return [
          {
            check: 'impact graph includes the upstream dependent CI',
            ok: nodes.some((n) => n.id === upstream),
            expected: `node ${String(upstream)}`,
            actual: nodes.map((n) => n.id),
          },
        ];
      },
    },
    {
      name: 'relationship list exposes the edge',
      endpoint: 'GET /api/v1/cmdb/cis/relationships',
      run: (ctx) => apiClient.listCIRelationships({ pageSize: 100 }),
      verify: (result, ctx) => {
        const relationshipId = ctx.vars['relationshipId'];
        const items = ((result as { items?: Array<{ id?: unknown }> }).items ?? []);
        return [
          {
            check: 'created relationship is listed',
            ok: items.some((r) => r.id === relationshipId),
            expected: `relationship ${String(relationshipId)}`,
            actual: items.length,
          },
        ];
      },
    },
    {
      name: 'relationship to nonexistent CI is rejected (negative)',
      endpoint: 'POST /api/v1/cmdb/cis/relationships',
      run: (ctx) =>
        apiClient.createCIRelationship({
          sourceCiId: requireNumber(ctx, 'upstreamCiId'),
          targetCiId: 999999999,
          relationshipType: 'depends_on',
        }),
      expectError: { kind: 'not_found' },
    },
  ],
  teardown: [
    {
      name: 'delete flow relationship',
      endpoint: 'DELETE /api/v1/cmdb/relationships/:id',
      run: async (ctx) => {
        const id = ctx.vars['relationshipId'];
        if (typeof id === 'number') await apiClient.deleteCIRelationship(id);
      },
    },
    {
      name: 'delete downstream CI',
      endpoint: 'DELETE /api/v1/cmdb/cis/:id',
      run: async (ctx) => {
        const id = ctx.vars['downstreamCiId'];
        if (typeof id === 'number') await apiClient.deleteCI(id);
      },
    },
    {
      name: 'delete upstream CI',
      endpoint: 'DELETE /api/v1/cmdb/cis/:id',
      run: async (ctx) => {
        const id = ctx.vars['upstreamCiId'];
        if (typeof id === 'number') await apiClient.deleteCI(id);
      },
    },
  ],
};

import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, body } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';

/** CMDB: CI types, CI instances, relationships, topology, impact, history. */

export function cmdbCommands() {
  return [
    cmd('cmdb.list', 'List configuration items', 'itsm cmdb list [--type t] [--status s]', ['GET /api/v1/cmdb/cis'], async (ctx) =>
      apiClient.listCIs({
        ...withPagination(ctx.args),
        type: optionalString(ctx.args.flags, 'type'),
        status: optionalString(ctx.args.flags, 'status'),
      } as never)
    ),
    cmd('cmdb.get', 'CI detail', 'itsm cmdb get <id>', ['GET /api/v1/cmdb/cis/:id'], async (ctx) => apiClient.getCI(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.create', 'Create a CI (schema-validated by CI type)', 'itsm cmdb create --data <json>', ['POST /api/v1/cmdb/cis'], async (ctx) => apiClient.createCI(body(ctx.args) ?? {})),
    cmd('cmdb.update', 'Update CI attributes (versioned, audited)', 'itsm cmdb update <id> [--data <json>]', ['PUT /api/v1/cmdb/cis/:id'], async (ctx) => apiClient.updateCI(requirePositional(ctx.args, 0, 'ciId'), body(ctx.args) ?? {})),
    cmd('cmdb.delete', 'Delete a CI', 'itsm cmdb delete <id>', ['DELETE /api/v1/cmdb/cis/:id'], async (ctx) => apiClient.deleteCI(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.search', 'Search CIs (structured search body via --data; falls back to {q})', 'itsm cmdb search <q> | --data <json>', ['POST /api/v1/cmdb/cis/search'], async (ctx) => {
      const data = body(ctx.args);
      if (data) return apiClient.searchCIs('', data);
      return apiClient.searchCIs(requirePositional(ctx.args, 0, 'query'));
    }),
    cmd('cmdb.stats', 'CI statistics', 'itsm cmdb stats', ['GET /api/v1/cmdb/cis/stats'], async () => apiClient.getCIStats()),
    cmd('cmdb.types', 'List CI type schemas', 'itsm cmdb types', ['GET /api/v1/cmdb/ci-types'], async (ctx) => apiClient.listCITypes(withPagination(ctx.args))),
    cmd('cmdb.topology', 'CI relationship topology', 'itsm cmdb topology <id>', ['GET /api/v1/cmdb/cis/:id/topology'], async (ctx) => apiClient.getCITopology(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.impact', 'Impact analysis (bounded traversal from one CI)', 'itsm cmdb impact <id>', ['GET /api/v1/cmdb/cis/:id/impact-analysis'], async (ctx) => apiClient.getCIImpactAnalysis(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.changes-history', 'CI change history', 'itsm cmdb changes-history <id>', ['GET /api/v1/cmdb/cis/:id/history'], async (ctx) => apiClient.getCIHistory(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.lifecycle', 'Update lifecycle status (recover/audit only)', 'itsm cmdb lifecycle <id> --data <json>', ['PUT /api/v1/cmdb/cis/:id/lifecycle'], async (ctx) => apiClient.updateCILifecycle(requirePositional(ctx.args, 0, 'ciId'), body(ctx.args) ?? {})),
    cmd('cmdb.relationships.list', 'List CI relationships', 'itsm cmdb relationships list', ['GET /api/v1/cmdb/cis/relationships'], async (ctx) => apiClient.listCIRelationships(withPagination(ctx.args))),
    cmd('cmdb.relationships.create', 'Create a CI relationship', 'itsm cmdb relationships create --data <json>', ['POST /api/v1/cmdb/cis/relationships'], async (ctx) => apiClient.createCIRelationship(body(ctx.args) ?? {})),
  ];
}

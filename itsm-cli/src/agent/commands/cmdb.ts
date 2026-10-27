import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, numberFlag, numberListFlag, body } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';

/** CMDB: CI types, CI instances, relationships, topology, impact, history. */

export function cmdbCommands() {
  return [
    cmd('cmdb.list', 'List configuration items', 'itsm cmdb list [--type t] [--status s] [--page N] [--pageSize N]', ['GET /api/v1/cmdb/cis'], async (ctx) =>
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
    cmd('cmdb.search', 'Search CIs (keyword via <q>, structured body via --data)', 'itsm cmdb search <q> | --data <json> [--page N] [--pageSize N]', ['GET /api/v1/cmdb/cis', 'POST /api/v1/cmdb/cis/search'], async (ctx) => {
      const data = body(ctx.args);
      if (data) return apiClient.searchCIs(data);
      return apiClient.searchCIsByKeyword(requirePositional(ctx.args, 0, 'query'), withPagination(ctx.args));
    }),
    cmd('cmdb.stats', 'CI statistics', 'itsm cmdb stats', ['GET /api/v1/cmdb/cis/stats'], async () => apiClient.getCIStats()),
    cmd('cmdb.types', 'List CI type schemas', 'itsm cmdb types [--page N] [--pageSize N]', ['GET /api/v1/cmdb/ci-types'], async (ctx) => apiClient.listCITypes(withPagination(ctx.args))),
    cmd('cmdb.topology', 'CI relationship topology', 'itsm cmdb topology <id>', ['GET /api/v1/cmdb/cis/:id/topology'], async (ctx) => apiClient.getCITopology(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.impact', 'Impact analysis (bounded traversal from one CI)', 'itsm cmdb impact <id>', ['GET /api/v1/cmdb/cis/:id/impact-analysis'], async (ctx) => apiClient.getCIImpactAnalysis(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.changes-history', 'CI change history', 'itsm cmdb changes-history <id>', ['GET /api/v1/cmdb/cis/:id/history'], async (ctx) => apiClient.getCIHistory(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.lifecycle', 'Update lifecycle status (recover/audit only)', 'itsm cmdb lifecycle <id> --data <json>', ['PUT /api/v1/cmdb/cis/:id/lifecycle'], async (ctx) => apiClient.updateCILifecycle(requirePositional(ctx.args, 0, 'ciId'), body(ctx.args) ?? {})),
    cmd('cmdb.relationships.list', 'List CI relationships', 'itsm cmdb relationships list [--page N] [--pageSize N]', ['GET /api/v1/cmdb/cis/relationships'], async (ctx) => apiClient.listCIRelationships(withPagination(ctx.args))),
    cmd('cmdb.relationships.create', 'Create a CI relationship', 'itsm cmdb relationships create --data <json>', ['POST /api/v1/cmdb/cis/relationships'], async (ctx) => apiClient.createCIRelationship(body(ctx.args) ?? {})),
    cmd('cmdb.relationships.get', 'One relationship detail', 'itsm cmdb relationships get <id>', ['GET /api/v1/cmdb/relationships/:id'], async (ctx) => apiClient.getCIRelationship(requirePositional(ctx.args, 0, 'relationshipId'))),
    cmd('cmdb.relationships.update', 'Update a relationship', 'itsm cmdb relationships update <id> --data <json>', ['PUT /api/v1/cmdb/relationships/:id'], async (ctx) => apiClient.updateCIRelationship(requirePositional(ctx.args, 0, 'relationshipId'), body(ctx.args) ?? {})),
    cmd('cmdb.relationships.delete', 'Delete a relationship', 'itsm cmdb relationships delete <id>', ['DELETE /api/v1/cmdb/relationships/:id'], async (ctx) => apiClient.deleteCIRelationship(requirePositional(ctx.args, 0, 'relationshipId'))),
    cmd('cmdb.relationships.for-ci', 'Relationships of one CI (both directions)', 'itsm cmdb relationships for-ci <ciId>', ['GET /api/v1/cmdb/cis/:id/relationships'], async (ctx) => apiClient.listCIRelationshipsByCI(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.relationship-types', 'Relationship type dictionary (depends_on / affects …)', 'itsm cmdb relationship-types', ['GET /api/v1/cmdb/relationship-types'], async () => apiClient.listRelationshipTypes()),

    // Metadata: what this CMDB instance models and which capabilities are on.
    cmd('cmdb.ontology', 'CMDB ontology (CI types + attribute definitions, versioned)', 'itsm cmdb ontology', ['GET /api/v1/cmdb/ontology'], async () => apiClient.getCMDBOntology()),
    cmd('cmdb.capabilities', 'CMDB build capabilities (discovery, reconciliation, …)', 'itsm cmdb capabilities', ['GET /api/v1/cmdb/capabilities'], async () => apiClient.getCMDBCapabilities()),
    cmd('cmdb.reconciliation', 'Cloud resource reconciliation summary (aggregate, not paged)', 'itsm cmdb reconciliation', ['GET /api/v1/cmdb/reconciliation'], async () => apiClient.getCMDBReconciliation()),
    cmd('cmdb.governance.report', 'CMDB governance report (quality metrics + discovery diff buckets)', 'itsm cmdb governance report', ['GET /api/v1/cmdb/governance/report'], async () => apiClient.getCMDBGovernanceReport()),
    cmd('cmdb.type.get', 'One CI type schema', 'itsm cmdb type get <id>', ['GET /api/v1/cmdb/ci-types/:id'], async (ctx) => apiClient.getCIType(requirePositional(ctx.args, 0, 'ciTypeId'))),
    cmd('cmdb.type.attributes', 'Attribute definitions of one CI type', 'itsm cmdb type attributes <ciTypeId>', ['GET /api/v1/cmdb/ci-types/:id/attributes'], async (ctx) => apiClient.listCITypeAttributes(requirePositional(ctx.args, 0, 'ciTypeId'))),
    cmd('cmdb.type.create', 'Create a CI type', 'itsm cmdb type create --data <json>', ['POST /api/v1/cmdb/ci-types'], async (ctx) => apiClient.createCIType(body(ctx.args) ?? {})),
    cmd('cmdb.type.update', 'Update a CI type', 'itsm cmdb type update <id> --data <json>', ['PUT /api/v1/cmdb/ci-types/:id'], async (ctx) => apiClient.updateCIType(requirePositional(ctx.args, 0, 'ciTypeId'), body(ctx.args) ?? {})),
    cmd('cmdb.type.delete', 'Delete a CI type', 'itsm cmdb type delete <id>', ['DELETE /api/v1/cmdb/ci-types/:id'], async (ctx) => apiClient.deleteCIType(requirePositional(ctx.args, 0, 'ciTypeId'))),

    // Lifecycle trail and bulk edits.
    cmd('cmdb.lifecycle-history', 'Lifecycle status transitions of one CI', 'itsm cmdb lifecycle-history <ciId>', ['GET /api/v1/cmdb/cis/:id/lifecycle/history'], async (ctx) => apiClient.getCILifecycleHistory(requirePositional(ctx.args, 0, 'ciId'))),
    cmd('cmdb.revert', 'Revert a CI to a previous version', 'itsm cmdb revert <ciId> --version <n> [--remark t]', ['POST /api/v1/cmdb/cis/:id/revert'], async (ctx) => {
      const version = numberFlag(ctx.args.flags, 'version');
      if (version === undefined) throw usageError('Missing required --version <number>');
      return apiClient.revertCIVersion(requirePositional(ctx.args, 0, 'ciId'), version, optionalString(ctx.args.flags, 'remark'));
    }),
    cmd('cmdb.tags.add', 'Attach tags to a CI', 'itsm cmdb tags add <ciId> --tagIds 1,2,3', ['POST /api/v1/cmdb/cis/:id/tags'], async (ctx) => {
      const tagIds = numberListFlag(ctx.args.flags, 'tagIds');
      if (!tagIds?.length) throw usageError('Missing required --tagIds <id,id,...>');
      return apiClient.addCITags(requirePositional(ctx.args, 0, 'ciId'), tagIds);
    }),
    cmd('cmdb.tags.remove', 'Detach tags from a CI', 'itsm cmdb tags remove <ciId> --tagIds 1,2,3', ['DELETE /api/v1/cmdb/cis/:id/tags'], async (ctx) => {
      const tagIds = numberListFlag(ctx.args.flags, 'tagIds');
      if (!tagIds?.length) throw usageError('Missing required --tagIds <id,id,...>');
      return apiClient.removeCITags(requirePositional(ctx.args, 0, 'ciId'), tagIds);
    }),
    cmd('cmdb.batch-update', 'Bulk-update CI attributes', 'itsm cmdb batch-update --ids 1,2,3 --data <json>', ['PUT /api/v1/cmdb/cis/batch'], async (ctx) => {
      const ids = numberListFlag(ctx.args.flags, 'ids');
      if (!ids?.length) throw usageError('Missing required --ids <id,id,...>');
      return apiClient.batchUpdateCIs(ids, body(ctx.args) ?? {});
    }),
    cmd('cmdb.batch-lifecycle', 'Bulk lifecycle status change', 'itsm cmdb batch-lifecycle --ids 1,2,3 --data <json>', ['PUT /api/v1/cmdb/cis/batch/lifecycle'], async (ctx) => {
      const ids = numberListFlag(ctx.args.flags, 'ids');
      if (!ids?.length) throw usageError('Missing required --ids <id,id,...>');
      return apiClient.batchUpdateCILifecycle(ids, body(ctx.args) ?? {});
    }),
  ];
}

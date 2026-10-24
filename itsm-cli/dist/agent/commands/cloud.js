import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString, numberFlag, body } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
/**
 * Cloud accounts / services / resources and the discovery pipeline.
 *
 * These live under `/cmdb/cloud-*` and `/cmdb/discovery/*`, but they are a
 * separate domain from `cmdb.*` (which is CIs, types and relationships) for the
 * same reason they are separate in the backend: a cloud account is a credential
 * scope, not a configuration item.
 *
 * This whole family shipped with zero CLI reachability. The route-drift check
 * only fired when the CLI named a route the backend lacked — never the reverse —
 * so an entire product surface could exist in the backend and be undiscoverable
 * to every agent. The drift check now covers that direction too.
 */
/** Provider enum is bound by the backend DTO; repeating it makes it agent-visible. */
const PROVIDERS = 'aliyun|tencent|huawei|aws|azure|volcengine|k8s|onprem';
export function cloudCommands() {
    return [
        // --- Capability manifest -------------------------------------------------
        cmd('cloud.providers', 'Cloud discovery capability manifest (which providers can auto-discover)', 'itsm cloud providers', ['GET /api/v1/cmdb/discovery/providers'], async () => apiClient.listCloudProviders()),
        cmd('cloud.discovery.health', 'Discovery pipeline health', 'itsm cloud discovery health', ['GET /api/v1/cmdb/discovery/health'], async () => apiClient.getDiscoveryHealth()),
        // --- Cloud accounts ------------------------------------------------------
        // Accounts and services are *selector* endpoints: returned whole as the data
        // source for CI-form cascades, so there is no page/pageSize to accept.
        cmd('cloud.accounts.list', 'List cloud accounts (not paginated: selector source)', `itsm cloud accounts list [--provider ${PROVIDERS}]`, ['GET /api/v1/cmdb/cloud-accounts'], async (ctx) => apiClient.listCloudAccounts({ provider: optionalString(ctx.args.flags, 'provider') })),
        cmd('cloud.accounts.get', 'One cloud account', 'itsm cloud accounts get <id>', ['GET /api/v1/cmdb/cloud-accounts/:id'], async (ctx) => apiClient.getCloudAccount(requirePositional(ctx.args, 0, 'id'))),
        cmd('cloud.accounts.create', 'Create a cloud account', 'itsm cloud accounts create --data <json>', ['POST /api/v1/cmdb/cloud-accounts'], async (ctx) => apiClient.createCloudAccount(body(ctx.args) ?? {})),
        cmd('cloud.accounts.update', 'Update a cloud account', 'itsm cloud accounts update <id> [--data <json>]', ['PUT /api/v1/cmdb/cloud-accounts/:id'], async (ctx) => apiClient.updateCloudAccount(requirePositional(ctx.args, 0, 'id'), body(ctx.args) ?? {})),
        // Deleting an account orphans every resource discovered under it, so this is
        // gated behind --yes rather than treated as a routine write.
        cmd('cloud.accounts.delete', 'Delete a cloud account', 'itsm cloud accounts delete <id>', ['DELETE /api/v1/cmdb/cloud-accounts/:id'], async (ctx) => apiClient.deleteCloudAccount(requirePositional(ctx.args, 0, 'id'))),
        // --- Cloud services ------------------------------------------------------
        cmd('cloud.services.list', 'List cloud service types (not paginated: selector source)', `itsm cloud services list [--provider ${PROVIDERS}]`, ['GET /api/v1/cmdb/cloud-services'], async (ctx) => apiClient.listCloudServices({ provider: optionalString(ctx.args.flags, 'provider') })),
        cmd('cloud.services.get', 'One cloud service type', 'itsm cloud services get <id>', ['GET /api/v1/cmdb/cloud-services/:id'], async (ctx) => apiClient.getCloudService(requirePositional(ctx.args, 0, 'id'))),
        cmd('cloud.services.create', 'Create a cloud service type', 'itsm cloud services create --data <json>', ['POST /api/v1/cmdb/cloud-services'], async (ctx) => apiClient.createCloudService(body(ctx.args) ?? {})),
        cmd('cloud.services.update', 'Update a cloud service type', 'itsm cloud services update <id> [--data <json>]', ['PUT /api/v1/cmdb/cloud-services/:id'], async (ctx) => apiClient.updateCloudService(requirePositional(ctx.args, 0, 'id'), body(ctx.args) ?? {})),
        // Service types are referenced by cloud resources; removing one leaves those
        // resources pointing at a definition that no longer exists.
        cmd('cloud.services.delete', 'Delete a cloud service type', 'itsm cloud services delete <id>', ['DELETE /api/v1/cmdb/cloud-services/:id'], async (ctx) => apiClient.deleteCloudService(requirePositional(ctx.args, 0, 'id'))),
        // --- Cloud resources -----------------------------------------------------
        cmd('cloud.resources.list', 'List cloud resources (paged)', `itsm cloud resources list [--provider ${PROVIDERS}] [--cloudAccountId <n>] [--serviceId <n>] [--region <s>] [--status <s>] [--search <s>] [--page N] [--pageSize N]`, ['GET /api/v1/cmdb/cloud-resources'], async (ctx) => apiClient.listCloudResources({
            ...withPagination(ctx.args),
            provider: optionalString(ctx.args.flags, 'provider'),
            cloudAccountId: numberFlag(ctx.args.flags, 'cloudAccountId'),
            serviceId: numberFlag(ctx.args.flags, 'serviceId'),
            region: optionalString(ctx.args.flags, 'region'),
            status: optionalString(ctx.args.flags, 'status'),
            search: optionalString(ctx.args.flags, 'search'),
        })),
        cmd('cloud.resources.get', 'One cloud resource', 'itsm cloud resources get <id>', ['GET /api/v1/cmdb/cloud-resources/:id'], async (ctx) => apiClient.getCloudResource(requirePositional(ctx.args, 0, 'id'))),
        cmd('cloud.resources.create', 'Register a cloud resource by hand', 'itsm cloud resources create --data <json>', ['POST /api/v1/cmdb/cloud-resources'], async (ctx) => apiClient.createCloudResource(body(ctx.args) ?? {})),
        cmd('cloud.resources.update', 'Update a cloud resource', 'itsm cloud resources update <id> [--data <json>]', ['PUT /api/v1/cmdb/cloud-resources/:id'], async (ctx) => apiClient.updateCloudResource(requirePositional(ctx.args, 0, 'id'), body(ctx.args) ?? {})),
        cmd('cloud.resources.delete', 'Delete a cloud resource', 'itsm cloud resources delete <id>', ['DELETE /api/v1/cmdb/cloud-resources/:id'], async (ctx) => apiClient.deleteCloudResource(requirePositional(ctx.args, 0, 'id'))),
        // --- Discovery sources ---------------------------------------------------
        cmd('cloud.discovery.sources.list', 'List discovery sources', 'itsm cloud discovery sources list', ['GET /api/v1/cmdb/discovery/sources'], async () => apiClient.listDiscoverySources()),
        cmd('cloud.discovery.sources.create', 'Create a discovery source', 'itsm cloud discovery sources create --data <json>', ['POST /api/v1/cmdb/discovery/sources'], async (ctx) => apiClient.createDiscoverySource(body(ctx.args) ?? {})),
        // --- Discovery jobs ------------------------------------------------------
        cmd('cloud.discovery.jobs.list', 'List discovery jobs', 'itsm cloud discovery jobs list [--status <s>] [--page N] [--pageSize N]', ['GET /api/v1/cmdb/discovery/jobs'], async (ctx) => {
            const page = numberFlag(ctx.args.flags, 'page');
            const pageSize = numberFlag(ctx.args.flags, 'pageSize');
            return apiClient.listDiscoveryJobs({
                status: optionalString(ctx.args.flags, 'status'),
                ...(page !== undefined ? { page } : {}),
                ...(pageSize !== undefined ? { pageSize } : {}),
            });
        }),
        cmd('cloud.discovery.jobs.get', 'Discovery job detail (progress, error, scope snapshots)', 'itsm cloud discovery jobs get <id>', ['GET /api/v1/cmdb/discovery/jobs/:id'], async (ctx) => apiClient.getDiscoveryJob(requirePositional(ctx.args, 0, 'id'))),
        cmd('cloud.discovery.jobs.create', 'Start a discovery job', 'itsm cloud discovery jobs create --data <json>', ['POST /api/v1/cmdb/discovery/jobs'], async (ctx) => apiClient.createDiscoveryJob(body(ctx.args) ?? {})),
        // Cancelling is not reversible from the CLI side: scopes the job already
        // completed stand, so this demands --yes.
        cmd('cloud.discovery.jobs.cancel', 'Cancel a running discovery job', 'itsm cloud discovery jobs cancel <id>', ['POST /api/v1/cmdb/discovery/jobs/:id/cancel'], async (ctx) => apiClient.cancelDiscoveryJob(requirePositional(ctx.args, 0, 'id'))),
        cmd('cloud.discovery.jobs.retry', 'Retry a failed discovery job', 'itsm cloud discovery jobs retry <id>', ['POST /api/v1/cmdb/discovery/jobs/:id/retry'], async (ctx) => apiClient.retryDiscoveryJob(requirePositional(ctx.args, 0, 'id')), 'write'),
        // --- Discovery results ---------------------------------------------------
        cmd('cloud.discovery.results.list', 'List discovery results (diffs awaiting reconciliation)', 'itsm cloud discovery results list [--jobId <n>] [--page N] [--pageSize N]', ['GET /api/v1/cmdb/discovery/results'], async (ctx) => {
            const page = numberFlag(ctx.args.flags, 'page');
            const pageSize = numberFlag(ctx.args.flags, 'pageSize');
            return apiClient.listDiscoveryResults({
                jobId: numberFlag(ctx.args.flags, 'jobId'),
                ...(page !== undefined ? { page } : {}),
                ...(pageSize !== undefined ? { pageSize } : {}),
            });
        }),
        // Applying writes discovered state into CMDB (creating/updating/linking CIs),
        // so it is a write even though the verb is absent from the risk lexicon.
        cmd('cloud.discovery.results.apply', 'Apply a discovery result into CMDB', 'itsm cloud discovery results apply <id>', ['POST /api/v1/cmdb/discovery/results/:id/apply'], async (ctx) => apiClient.applyDiscoveryResult(requirePositional(ctx.args, 0, 'id')), 'write'),
        cmd('cloud.discovery.results.reject', 'Reject a discovery result', 'itsm cloud discovery results reject <id> [--reason <s>]', ['POST /api/v1/cmdb/discovery/results/:id/reject'], async (ctx) => apiClient.rejectDiscoveryResult(requirePositional(ctx.args, 0, 'id'), optionalString(ctx.args.flags, 'reason'))),
    ];
}

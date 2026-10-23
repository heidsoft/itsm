import { cmd } from '../spec.js';
import { requirePositional, optionalString, body } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/**
 * Connectors / IM marketplace. Connector credential provisioning is a
 * high-risk write. NOTE: the CLI never reads back connector secrets; the
 * backend returns masked metadata only.
 */
export function connectorCommands() {
    return [
        cmd('connector.list', 'List installed connectors (market)', 'itsm connector list', ['GET /api/v1/connectors'], async () => apiClient.listConnectors()),
        cmd('connector.health', 'Lifecycle/health of each integration', 'itsm connector health', ['GET /api/v1/connectors/health'], async () => apiClient.connectorHealth()),
        cmd('connector.configs', 'List configured integrations (credentials masked by backend)', 'itsm connector configs', ['GET /api/v1/connectors/configs'], async () => apiClient.listConnectorConfigs()),
        cmd('connector.provision', 'Configure a connector (credentials never echoed back; reuses existing secrets when omitted)', 'itsm connector provision --data <json>', ['POST /api/v1/connectors/configs'], async (ctx) => {
            const cfg = body(ctx.args);
            if (!cfg || typeof cfg.name !== 'string')
                throw usageError('Provide --data \'{"name":"feishu",...}\' (see docs)');
            return apiClient.provisionConnector(cfg);
        }),
        cmd('connector.revoke', 'Revoke a connector credential (provider from --provider or store default)', 'itsm connector revoke <name> [--provider p]', ['DELETE /api/v1/connectors/configs/:name'], async (ctx) => apiClient.revokeConnector(requirePositional(ctx.args, 0, 'connectorName'), optionalString(ctx.args.flags, 'provider'))),
        cmd('connector.send', 'Send a message through a connector (type text|markdown|post|card)', 'itsm connector send <name> --channel <ch> --message <text> | --data <json>', ['POST /api/v1/connectors/:name/send'], async (ctx) => {
            const name = requirePositional(ctx.args, 0, 'connectorName');
            const data = body(ctx.args);
            if (data)
                return apiClient.sendViaConnector(name, data);
            const channel = optionalString(ctx.args.flags, 'channel');
            const message = optionalString(ctx.args.flags, 'message');
            const title = optionalString(ctx.args.flags, 'title');
            if (!channel || !message)
                throw usageError('Usage: itsm connector send <name> --channel <ch> --message <text> | --data <json>');
            return apiClient.sendViaConnector(name, title ? { channel, type: 'text', title, content: message } : { channel, type: 'text', content: message });
        }),
        cmd('connector.test', 'Test integration reachability', 'itsm connector test <name>', ['POST /api/v1/connectors/:name/test'], async (ctx) => apiClient.testConnector(requirePositional(ctx.args, 0, 'connectorName'))),
    ];
}
/** Exported for docs: agents must know connectors never return secrets. */
export const CONNECTOR_SECRET_POLICY = { returnsSecrets: false, maskAppliedBy: 'backend' };

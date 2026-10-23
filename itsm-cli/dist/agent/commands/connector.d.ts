/**
 * Connectors / IM marketplace. Connector credential provisioning is a
 * high-risk write. NOTE: the CLI never reads back connector secrets; the
 * backend returns masked metadata only.
 */
export declare function connectorCommands(): import("../spec.js").CommandSpec[];
/** Exported for docs: agents must know connectors never return secrets. */
export declare const CONNECTOR_SECRET_POLICY: {
    returnsSecrets: boolean;
    maskAppliedBy: string;
};

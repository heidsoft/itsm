/**
 * Marketplace (tenant face). Covers the routes a tenant installs/uninstalls/
 * upgrades extensions through. The admin catalog CRUD face
 * (/marketplace/catalog/*) is deliberately not exposed — see api-client.ts.
 *
 * install/uninstall inherit high-risk-write from the verb (the /install/
 * substring in deriveRisk matches both), so they go through the confirmation
 * gate like every other destructive command.
 */
export declare function marketplaceCommands(): import("../spec.js").CommandSpec[];

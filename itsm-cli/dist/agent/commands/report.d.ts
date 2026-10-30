/**
 * Reports: the catalog plus every report it advertises.
 *
 * The catalog is the single source of truth for "which reports can I open":
 * `cmdb-quality` and `catalog-usage` disappear from it when the CMDB / service
 * catalog domain is not wired, so a command per entry keeps the CLI honest
 * instead of hard-coding a list that can drift from the backend routes.
 *
 * All endpoints require `report:read`; the domain-scoped reports additionally
 * require the domain read permission (`cmdb:read`, `service_catalog:read`).
 */
export declare function reportCommands(): import("../spec.js").CommandSpec[];

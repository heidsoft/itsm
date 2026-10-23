/**
 * Infra tier: the first commands an agent runs to establish trust.
 * These work without login for health/version/readyz (public routes).
 */
export declare function infraCommands(): import("../spec.js").CommandSpec[];

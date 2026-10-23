export declare enum ExitCode {
    Success = 0,
    /** Network/transport failure: backend unreachable, DNS, timeout, ECONNREFUSED. */
    Network = 1,
    /** Bad usage: unknown command, missing/invalid arguments. Safe to retry with fixed input. */
    Usage = 2,
    /** Not authenticated: 401 from backend or missing credentials. Run `itsm login` again. */
    NotAuthenticated = 3,
    /** Forbidden: 403 from backend (valid token, insufficient permission). */
    Forbidden = 4,
    /** Backend rejected the request: business code != 0 (validation, not-found 404, conflict 409, 5xx). */
    RequestRejected = 5,
    /** Unexpected local error (bug in CLI, unparseable response). */
    Internal = 6
}
export type ErrorKind = 'network' | 'usage' | 'not_authenticated' | 'forbidden' | 'not_found' | 'conflict' | 'validation' | 'unavailable' | 'confirmation_required'
/** A business-flow scenario ran but at least one assertion failed (see error.details). */
 | 'assertion_failed' | 'internal';
export interface AgentError {
    kind: ErrorKind;
    message: string;
    /** HTTP status of the failing request, when the request reached the backend or status is known. */
    httpStatus?: number;
    /** Business code from the backend envelope ({ code, message, data }), when present. */
    businessCode?: number;
    /** Retryable at the transport level (network errors and 5xx/429). */
    retryable: boolean;
    /** Structured next-step guidance (e.g. which permission is missing, what to run). */
    hint?: string;
    /** Extra machine-readable context (e.g. risk description for confirmation_required). */
    risk?: unknown;
    /**
     * Structured evidence for failures that carry a report (currently `flow run`).
     * Kept separate from `risk` so consumers never have to sniff the shape.
     */
    details?: unknown;
}
export declare class CliError extends Error {
    readonly agentError: AgentError;
    constructor(agentError: AgentError);
    get exitCode(): ExitCode;
}
export declare function kindToExitCode(kind: ErrorKind): ExitCode;
/** Wrap any thrown value into a CliError with a classified AgentError. */
export declare function classifyError(err: unknown): CliError;
export declare function httpStatusToAgentError(status: number, message: string): AgentError;
/** Provider/SQL raw errors must never leak into agent-visible output. */
export declare function sanitizeProviderMessage(message: string): string;
export declare function usageError(message: string): CliError;
export declare function notAuthenticatedError(message?: string): CliError;
/** Build the machine-readable envelope for `itsm --json` output. */
export interface AgentEnvelope<T> {
    ok: boolean;
    /** Present only when ok === false. */
    error?: AgentError;
    /** API payload (the backend envelope's `data` field), present only when ok === true. */
    data?: T;
    /** Shaping metadata when --fields/--limit changed the payload (returned vs total). */
    meta?: Record<string, unknown>;
}
export declare function successEnvelope<T>(data: T): AgentEnvelope<T>;
export declare function failureEnvelope(err: CliError): AgentEnvelope<never>;

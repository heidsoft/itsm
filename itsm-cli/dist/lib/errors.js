// Stable exit codes and error taxonomy for the headless agent layer.
// The agent surface must be deterministic: every failure maps to exactly one
// exit code, and every exit code carries machine-readable context in the JSON
// envelope (`kind`, `httpStatus`, `businessCode`, `message`).
export var ExitCode;
(function (ExitCode) {
    ExitCode[ExitCode["Success"] = 0] = "Success";
    /** Network/transport failure: backend unreachable, DNS, timeout, ECONNREFUSED. */
    ExitCode[ExitCode["Network"] = 1] = "Network";
    /** Bad usage: unknown command, missing/invalid arguments. Safe to retry with fixed input. */
    ExitCode[ExitCode["Usage"] = 2] = "Usage";
    /** Not authenticated: 401 from backend or missing credentials. Run `itsm login` again. */
    ExitCode[ExitCode["NotAuthenticated"] = 3] = "NotAuthenticated";
    /** Forbidden: 403 from backend (valid token, insufficient permission). */
    ExitCode[ExitCode["Forbidden"] = 4] = "Forbidden";
    /** Backend rejected the request: business code != 0 (validation, not-found 404, conflict 409, 5xx). */
    ExitCode[ExitCode["RequestRejected"] = 5] = "RequestRejected";
    /** Unexpected local error (bug in CLI, unparseable response). */
    ExitCode[ExitCode["Internal"] = 6] = "Internal";
})(ExitCode || (ExitCode = {}));
export class CliError extends Error {
    constructor(agentError) {
        super(agentError.message);
        this.name = 'CliError';
        this.agentError = agentError;
    }
    get exitCode() {
        return kindToExitCode(this.agentError.kind);
    }
}
export function kindToExitCode(kind) {
    switch (kind) {
        case 'network':
            return ExitCode.Network;
        case 'usage':
            return ExitCode.Usage;
        case 'not_authenticated':
            return ExitCode.NotAuthenticated;
        case 'forbidden':
            return ExitCode.Forbidden;
        case 'not_found':
        case 'conflict':
        case 'validation':
        case 'unavailable':
        case 'assertion_failed':
            return ExitCode.RequestRejected;
        case 'confirmation_required':
            // Normally short-circuited by the runner's exit-10 gate; kept for exhaustiveness.
            return 10;
        case 'internal':
            return ExitCode.Internal;
    }
}
/** Wrap any thrown value into a CliError with a classified AgentError. */
export function classifyError(err) {
    if (err instanceof CliError)
        return err;
    const message = err instanceof Error ? err.message : String(err);
    // fetch() throws TypeError on network failures ("fetch failed", ECONNREFUSED, DNS, timeout).
    if (err instanceof TypeError || /fetch failed|ECONNREFUSED|ENOTFOUND|EAI_AGAIN|ETIMEDOUT|socket hang up/i.test(message)) {
        return new CliError({ kind: 'network', message: `Backend unreachable: ${message}`, retryable: true });
    }
    // Our api-client throws `HTTP <status>: <text>` on !res.ok — try to recover the status.
    const httpMatch = message.match(/HTTP (\d{3})(?::\s*(.*))?/);
    if (httpMatch) {
        return new CliError(httpStatusToAgentError(Number(httpMatch[1]), httpMatch[2] || message));
    }
    return new CliError({ kind: 'internal', message: `Unexpected CLI error: ${message}`, retryable: false });
}
export function httpStatusToAgentError(status, message) {
    const safeMessage = sanitizeProviderMessage(message);
    if (status === 401) {
        return { kind: 'not_authenticated', message: safeMessage, httpStatus: status, retryable: false };
    }
    if (status === 403) {
        return { kind: 'forbidden', message: safeMessage, httpStatus: status, retryable: false };
    }
    if (status === 404) {
        return { kind: 'not_found', message: safeMessage, httpStatus: status, retryable: false };
    }
    if (status === 409 || status === 422) {
        return { kind: 'conflict', message: safeMessage, httpStatus: status, retryable: false };
    }
    if (status === 400) {
        return { kind: 'validation', message: safeMessage, httpStatus: status, retryable: false };
    }
    if (status === 429 || (status >= 500 && status < 600)) {
        return { kind: 'unavailable', message: safeMessage, httpStatus: status, retryable: true };
    }
    return { kind: 'internal', message: safeMessage, httpStatus: status, retryable: false };
}
/** Provider/SQL raw errors must never leak into agent-visible output. */
export function sanitizeProviderMessage(message) {
    const redacted = message
        .replace(/Bearer\s+[A-Za-z0-9\-._~+/]+=*/gi, 'Bearer***')
        .replace(/postgres|mysql|sqlite|pgx|driver error|sql[sS]tate/gi, 'storage');
    // Bound length to keep agent context clean; the backend correlation ID is enough for debugging.
    return redacted.length > 500 ? `${redacted.slice(0, 500)}…` : redacted;
}
export function usageError(message) {
    return new CliError({ kind: 'usage', message, retryable: false });
}
export function notAuthenticatedError(message = 'Not authenticated. Run `itsm login -u <user> -p <pass>` or set ITSM_TOKEN.') {
    return new CliError({ kind: 'not_authenticated', message, retryable: false });
}
export function successEnvelope(data) {
    return { ok: true, data };
}
export function failureEnvelope(err) {
    return { ok: false, error: err.agentError };
}

// Stable exit codes and error taxonomy for the headless agent layer.
// The agent surface must be deterministic: every failure maps to exactly one
// exit code, and every exit code carries machine-readable context in the JSON
// envelope (`kind`, `httpStatus`, `businessCode`, `message`).

export enum ExitCode {
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
  Internal = 6,
}

export type ErrorKind =
  | 'network'
  | 'usage'
  | 'not_authenticated'
  | 'forbidden'
  | 'not_found'
  | 'conflict'
  | 'validation'
  | 'unavailable'
  | 'confirmation_required'
  /** A business-flow scenario ran but at least one assertion failed (see error.details). */
  | 'assertion_failed'
  | 'internal';

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
   * Capability identifier when the backend refused the request with an explicit
   * capability state gate (HTTP 503 + `{data:{capability,state}}`, e.g. the MSP
   * routes in a private deployment). Carrying these through lets callers tell
   * "this deployment has the feature turned off" apart from "the resource does
   * not exist" or "the backend is broken", instead of just seeing kind=unavailable.
   */
  capability?: string;
  /** One of the backend capability states: disabled / unconfigured / unready. */
  capabilityState?: string;
  /**
   * Structured evidence for failures that carry a report (currently `flow run`).
   * Kept separate from `risk` so consumers never have to sniff the shape.
   */
  details?: unknown;
}

export class CliError extends Error {
  readonly agentError: AgentError;

  constructor(agentError: AgentError) {
    super(agentError.message);
    this.name = 'CliError';
    this.agentError = agentError;
  }

  get exitCode(): ExitCode {
    return kindToExitCode(this.agentError.kind);
  }
}

export function kindToExitCode(kind: ErrorKind): ExitCode {
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
      return 10 as ExitCode;
    case 'internal':
      return ExitCode.Internal;
  }
}

/** Wrap any thrown value into a CliError with a classified AgentError. */
export function classifyError(err: unknown): CliError {
  if (err instanceof CliError) return err;

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

export function httpStatusToAgentError(status: number, message: string): AgentError {
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
export function sanitizeProviderMessage(message: string): string {
  const redacted = message
    .replace(/Bearer\s+[A-Za-z0-9\-._~+/]+=*/gi, 'Bearer***')
    .replace(/postgres|mysql|sqlite|pgx|driver error|sql[sS]tate/gi, 'storage');
  // Bound length to keep agent context clean; the backend correlation ID is enough for debugging.
  return redacted.length > 500 ? `${redacted.slice(0, 500)}…` : redacted;
}

export function usageError(message: string): CliError {
  return new CliError({ kind: 'usage', message, retryable: false });
}

export function notAuthenticatedError(message = 'Not authenticated. Run `itsm login -u <user> -p <pass>` or set ITSM_TOKEN.'): CliError {
  return new CliError({ kind: 'not_authenticated', message, retryable: false });
}

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

export function successEnvelope<T>(data: T): AgentEnvelope<T> {
  return { ok: true, data };
}

export function failureEnvelope(err: CliError): AgentEnvelope<never> {
  return { ok: false, error: err.agentError };
}

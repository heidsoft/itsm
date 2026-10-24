/** Remove bearer tokens, JWTs and `password=`-style fields from free text. */
export declare function redactSecrets(input: string): string;
/** Recursively redact string values of a payload that is known to echo credentials. */
export declare function redactObject<T>(value: T): T;
/** Read one line/whole payload from stdin. Refuses to block on an interactive TTY. */
export declare function readStdinSecret(label: string): string;
export interface SecretSource {
    /** Literal flag value. Least safe: visible in argv. */
    literal?: string;
    /** When true, a literal value is rejected outright instead of used. */
    rejectLiteral?: boolean;
    /** Flag name quoted in the rejection message (defaults to `--${label}`). */
    literalFlag?: string;
    /** Read the secret from stdin (preferred for scripts). */
    stdin?: boolean;
    /** Read the secret from a file (mode checked when possible). */
    file?: string;
    /** Environment variable name that may carry a file path (e.g. ITSM_PASSWORD_FILE). */
    fileEnv?: string;
    /** Environment variable name that may carry the secret itself. */
    valueEnv?: string;
    /** Human label used in errors. */
    label: string;
    /** When false, whitespace-only input is still accepted (passwords can be spaces). */
    trim?: boolean;
}
/**
 * Resolve a secret from exactly one source. Priority is explicit-flag →
 * stdin → file → environment value, so an operator can always override a
 * long-lived env secret for one invocation.
 *
 * Multiple explicit sources are rejected: silently preferring one over the
 * other is how an agent ends up authenticating with a stale credential.
 */
export declare function resolveSecret(source: SecretSource): string | undefined;
/** Read a secret file, refusing group/world-readable files on POSIX. */
export declare function readSecretFile(filePath: string, label: string): string;

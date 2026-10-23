import * as fs from 'fs';
import { usageError } from './errors.js';
/**
 * Secret handling for the CLI.
 *
 * Two rules drive this module:
 *   1. A secret must be readable from a channel that is not the argv vector
 *      (`--password-stdin`, `ITSM_PASSWORD_FILE`), because argv leaks through
 *      the process table and shell history.
 *   2. A secret must never reach stdout, a log line, or an error message.
 *      `redactSecrets` is applied to every backend error body and to auth
 *      command output.
 */
const REDACTIONS = [
    // Authorization headers, with or without the header name.
    [/\b(?:authorization\s*[:=]\s*)?bearer\s+[A-Za-z0-9\-._~+/]+=*/gi, 'Bearer ***'],
    // JWT-shaped values (access/refresh tokens).
    [/\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\b/g, '***JWT***'],
    // JSON / query / log fields carrying secrets.
    [
        /\b(access[_-]?token|refresh[_-]?token|password|passwordConfirm|client[_-]?secret|api[_-]?key|secret|token)(\s*["']?\s*[:=]\s*["']?)[^\s"'&,}]+/gi,
        '$1$2***',
    ],
];
/** Remove bearer tokens, JWTs and `password=`-style fields from free text. */
export function redactSecrets(input) {
    let out = input;
    for (const [pattern, replacement] of REDACTIONS) {
        out = out.replace(pattern, replacement);
    }
    return out;
}
/** Recursively redact string values of a payload that is known to echo credentials. */
export function redactObject(value) {
    if (typeof value === 'string')
        return redactSecrets(value);
    if (Array.isArray(value))
        return value.map((item) => redactObject(item));
    if (value && typeof value === 'object') {
        const out = {};
        for (const [key, child] of Object.entries(value)) {
            if (/^(accessToken|refreshToken|password|passwordConfirm|clientSecret)$/i.test(key)) {
                out[key] = '***';
            }
            else {
                out[key] = redactObject(child);
            }
        }
        return out;
    }
    return value;
}
/** Read one line/whole payload from stdin. Refuses to block on an interactive TTY. */
export function readStdinSecret(label) {
    if (process.stdin.isTTY) {
        throw usageError(`${label} requested but stdin is a terminal. Pipe the secret in, e.g. \`read -rs PASSWORD && printf %s "$PASSWORD" | itsm auth login -u admin --password-stdin\`.`);
    }
    let raw;
    try {
        raw = fs.readFileSync(0, 'utf-8');
    }
    catch (err) {
        throw usageError(`Failed to read ${label} from stdin: ${err instanceof Error ? err.message : String(err)}`);
    }
    return raw.replace(/\r?\n$/, '');
}
/**
 * Resolve a secret from exactly one source. Priority is explicit-flag →
 * stdin → file → environment value, so an operator can always override a
 * long-lived env secret for one invocation.
 *
 * Multiple explicit sources are rejected: silently preferring one over the
 * other is how an agent ends up authenticating with a stale credential.
 */
export function resolveSecret(source) {
    const explicit = [source.literal !== undefined, Boolean(source.stdin), Boolean(source.file)].filter(Boolean).length;
    if (explicit > 1) {
        throw usageError(`Provide only one source for ${source.label} (--flag, --*-stdin, or --*-file).`);
    }
    let value;
    if (source.literal !== undefined) {
        value = source.literal;
    }
    else if (source.stdin) {
        value = readStdinSecret(source.label);
    }
    else if (source.file) {
        value = readSecretFile(source.file, source.label);
    }
    else if (source.fileEnv && process.env[source.fileEnv]) {
        value = readSecretFile(process.env[source.fileEnv], source.label);
    }
    else if (source.valueEnv && process.env[source.valueEnv]) {
        value = process.env[source.valueEnv];
    }
    if (value === undefined)
        return undefined;
    const normalized = source.trim === false ? value : value.trim();
    return normalized.length ? normalized : undefined;
}
/** Read a secret file, refusing group/world-readable files on POSIX. */
export function readSecretFile(filePath, label) {
    let stat;
    try {
        stat = fs.statSync(filePath);
    }
    catch {
        throw usageError(`${label} file does not exist: ${filePath}`);
    }
    if (!stat.isFile()) {
        throw usageError(`${label} file is not a regular file: ${filePath}`);
    }
    if (process.platform !== 'win32' && (stat.mode & 0o077) !== 0) {
        throw usageError(`${label} file ${filePath} is accessible by group/other (mode ${(stat.mode & 0o777).toString(8)}). Run \`chmod 600 ${filePath}\` first.`);
    }
    return fs.readFileSync(filePath, 'utf-8');
}

import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { randomBytes } from 'crypto';
import { CliError } from './errors.js';
const DEFAULT_DIR = path.join(os.homedir(), '.itsm');
export function credentialsPath() {
    const override = (process.env['ITSM_CREDENTIALS_FILE'] || '').trim();
    return override || path.join(DEFAULT_DIR, 'credentials');
}
function credentialsDir() {
    return path.dirname(credentialsPath());
}
/** Machine-readable store state for `auth status` / `doctor`. Never includes secrets. */
export function describeCredentialStore() {
    const file = credentialsPath();
    try {
        const stat = fs.lstatSync(file);
        const isSymlink = stat.isSymbolicLink();
        const mode = isSymlink ? null : (stat.mode & 0o777).toString(8);
        const ownerMatches = process.platform === 'win32' || isSymlink ? null : stat.uid === process.getuid?.();
        const secure = !isSymlink && stat.isFile() && mode === '600' && ownerMatches !== false;
        let reason;
        if (isSymlink)
            reason = 'credentials path is a symlink; refusing to trust it';
        else if (!stat.isFile())
            reason = 'credentials path is not a regular file';
        else if (mode !== '600')
            reason = `credentials file mode is ${mode}; expected 600`;
        else if (ownerMatches === false)
            reason = 'credentials file is owned by another user';
        return { path: file, exists: true, mode, ownerMatches, isSymlink, secure, ...(reason ? { reason } : {}) };
    }
    catch {
        return { path: file, exists: false, mode: null, ownerMatches: null, isSymlink: false, secure: true };
    }
}
function ensureSecureDir() {
    const dir = credentialsDir();
    if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
    }
    if (process.platform !== 'win32') {
        try {
            const stat = fs.lstatSync(dir);
            if (stat.isDirectory() && (stat.mode & 0o077) !== 0) {
                fs.chmodSync(dir, 0o700);
            }
        }
        catch {
            // Best effort: the file-level mode below is the security boundary that matters.
        }
    }
}
export function saveCredentials(cred) {
    ensureSecureDir();
    const target = credentialsPath();
    // A symlink here means someone can redirect the write to a file they read
    // afterwards. Fail closed instead of following it.
    let existing;
    try {
        existing = fs.lstatSync(target);
    }
    catch {
        existing = undefined;
    }
    if (existing?.isSymbolicLink()) {
        throw new CliError({
            kind: 'internal',
            message: `Refusing to write credentials through a symlink at ${target}`,
            retryable: false,
            hint: 'Remove the symlink or point ITSM_CREDENTIALS_FILE at a regular file you own.',
        });
    }
    const payload = {
        ...cred,
        version: 2,
        savedAt: new Date().toISOString(),
    };
    const temp = path.join(credentialsDir(), `.credentials.${process.pid}.${randomBytes(6).toString('hex')}.tmp`);
    let fd;
    try {
        fd = fs.openSync(temp, 'wx', 0o600);
        fs.writeFileSync(fd, JSON.stringify(payload, null, 2), { encoding: 'utf-8' });
        fs.fsyncSync(fd);
        fs.closeSync(fd);
        fd = undefined;
        fs.renameSync(temp, target);
    }
    catch (err) {
        if (fd !== undefined) {
            try {
                fs.closeSync(fd);
            }
            catch {
                // ignore
            }
        }
        try {
            fs.unlinkSync(temp);
        }
        catch {
            // Temp file may not exist; nothing to clean up.
        }
        throw err;
    }
    if (process.platform !== 'win32') {
        try {
            fs.chmodSync(target, 0o600);
        }
        catch {
            // Non-POSIX filesystem — best effort.
        }
    }
}
/** Merge a change into the stored credentials without dropping refresh token / identity. */
export function updateCredentials(patch) {
    const current = loadCredentials();
    if (!current)
        return null;
    const next = { ...current, ...patch };
    saveCredentials(next);
    return next;
}
export function loadCredentials() {
    const file = credentialsPath();
    try {
        const stat = fs.lstatSync(file);
        if (stat.isSymbolicLink() || !stat.isFile())
            return null;
        // A store owned by another user is attacker-controlled: do not use it.
        if (process.platform !== 'win32' && typeof process.getuid === 'function' && stat.uid !== process.getuid()) {
            return null;
        }
        if (process.platform !== 'win32' && (stat.mode & 0o077) !== 0) {
            try {
                fs.chmodSync(file, 0o600);
            }
            catch {
                // Keep reading: the token is still ours, we just could not tighten it.
            }
        }
        const parsed = JSON.parse(fs.readFileSync(file, 'utf-8'));
        if (typeof parsed !== 'object' || parsed === null)
            return null;
        if (typeof parsed.token !== 'string' && typeof parsed.refreshToken !== 'string')
            return null;
        return parsed;
    }
    catch {
        return null;
    }
}
/** Remove the store. Unlinks the link itself when the path is a symlink. */
export function clearCredentials() {
    try {
        if (fs.existsSync(credentialsPath())) {
            fs.unlinkSync(credentialsPath());
        }
    }
    catch {
        // ignore
    }
}
const EXPIRY_SKEW_MS = 30000;
/**
 * True when the access token is present and not about to expire.
 *
 * The 30s skew avoids the classic race where a token looks valid when the
 * request is built and expires while it is in flight.
 */
export function isLoggedIn() {
    const cred = loadCredentials();
    if (!cred?.token)
        return false;
    if (cred.expiresAt) {
        const expires = new Date(cred.expiresAt).getTime();
        if (Number.isFinite(expires))
            return expires - EXPIRY_SKEW_MS > Date.now();
    }
    return true;
}
/** True when the session can be renewed even though the access token expired. */
export function hasRefreshableSession() {
    return Boolean(loadCredentials()?.refreshToken);
}

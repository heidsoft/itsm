import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { randomBytes } from 'crypto';
import { CliError } from './errors.js';
import type { Credentials } from './types.js';

/**
 * On-disk credential store.
 *
 * Threat model for a CLI bearer token: another local user reading the file,
 * a symlink planted at the store path to redirect the write, a partially
 * written file after a crash, and credentials silently surviving `logout`.
 *
 * Mitigations implemented here:
 *   · directory 0700 + file 0600 (best effort on non-POSIX filesystems);
 *   · atomic write (same-directory temp file + rename), so a crash never
 *     leaves a truncated JSON blob that reads as "logged out";
 *   · refuse to write through a symlink and refuse to read a store whose
 *     owner is not the current user;
 *   · re-chmod a file left group/world-readable by an older CLI version and
 *     surface its state to `auth status` / `doctor`;
 *   · `ITSM_CREDENTIALS_FILE` override so CI can mount an ephemeral path
 *     instead of writing into the runner's home directory.
 */

export { type Credentials };

const DEFAULT_DIR = path.join(os.homedir(), '.itsm');

export function credentialsPath(): string {
  const override = (process.env['ITSM_CREDENTIALS_FILE'] || '').trim();
  return override || path.join(DEFAULT_DIR, 'credentials');
}

function credentialsDir(): string {
  return path.dirname(credentialsPath());
}

export interface CredentialStoreReport {
  path: string;
  exists: boolean;
  /** `null` when the file does not exist. */
  mode: string | null;
  ownerMatches: boolean | null;
  isSymlink: boolean;
  secure: boolean;
  /** Why the store is not secure, or why it was rejected on read. */
  reason?: string;
}

/** Machine-readable store state for `auth status` / `doctor`. Never includes secrets. */
export function describeCredentialStore(): CredentialStoreReport {
  const file = credentialsPath();
  try {
    const stat = fs.lstatSync(file);
    const isSymlink = stat.isSymbolicLink();
    const mode = isSymlink ? null : ((stat.mode & 0o777).toString(8) as string);
    const ownerMatches =
      process.platform === 'win32' || isSymlink ? null : stat.uid === process.getuid?.();
    const secure = !isSymlink && stat.isFile() && mode === '600' && ownerMatches !== false;
    let reason: string | undefined;
    if (isSymlink) reason = 'credentials path is a symlink; refusing to trust it';
    else if (!stat.isFile()) reason = 'credentials path is not a regular file';
    else if (mode !== '600') reason = `credentials file mode is ${mode}; expected 600`;
    else if (ownerMatches === false) reason = 'credentials file is owned by another user';
    return { path: file, exists: true, mode, ownerMatches, isSymlink, secure, ...(reason ? { reason } : {}) };
  } catch {
    return { path: file, exists: false, mode: null, ownerMatches: null, isSymlink: false, secure: true };
  }
}

function ensureSecureDir(): void {
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
    } catch {
      // Best effort: the file-level mode below is the security boundary that matters.
    }
  }
}

export function saveCredentials(cred: Credentials): void {
  ensureSecureDir();
  const target = credentialsPath();

  // A symlink here means someone can redirect the write to a file they read
  // afterwards. Fail closed instead of following it.
  let existing: fs.Stats | undefined;
  try {
    existing = fs.lstatSync(target);
  } catch {
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

  const payload: Credentials = {
    ...cred,
    version: 2,
    savedAt: new Date().toISOString(),
  };

  const temp = path.join(credentialsDir(), `.credentials.${process.pid}.${randomBytes(6).toString('hex')}.tmp`);
  let fd: number | undefined;
  try {
    fd = fs.openSync(temp, 'wx', 0o600);
    fs.writeFileSync(fd, JSON.stringify(payload, null, 2), { encoding: 'utf-8' });
    fs.fsyncSync(fd);
    fs.closeSync(fd);
    fd = undefined;
    fs.renameSync(temp, target);
  } catch (err) {
    if (fd !== undefined) {
      try {
        fs.closeSync(fd);
      } catch {
        // ignore
      }
    }
    try {
      fs.unlinkSync(temp);
    } catch {
      // Temp file may not exist; nothing to clean up.
    }
    throw err;
  }

  if (process.platform !== 'win32') {
    try {
      fs.chmodSync(target, 0o600);
    } catch {
      // Non-POSIX filesystem — best effort.
    }
  }
}

/** Merge a change into the stored credentials without dropping refresh token / identity. */
export function updateCredentials(patch: Partial<Credentials>): Credentials | null {
  const current = loadCredentials();
  if (!current) return null;
  const next: Credentials = { ...current, ...patch };
  saveCredentials(next);
  return next;
}

export function loadCredentials(): Credentials | null {
  const file = credentialsPath();
  try {
    const stat = fs.lstatSync(file);
    if (stat.isSymbolicLink() || !stat.isFile()) return null;
    // A store owned by another user is attacker-controlled: do not use it.
    if (process.platform !== 'win32' && typeof process.getuid === 'function' && stat.uid !== process.getuid()) {
      return null;
    }
    if (process.platform !== 'win32' && (stat.mode & 0o077) !== 0) {
      try {
        fs.chmodSync(file, 0o600);
      } catch {
        // Keep reading: the token is still ours, we just could not tighten it.
      }
    }
    const parsed = JSON.parse(fs.readFileSync(file, 'utf-8')) as Partial<Credentials>;
    if (typeof parsed !== 'object' || parsed === null) return null;
    if (typeof parsed.token !== 'string' && typeof parsed.refreshToken !== 'string') return null;
    return parsed as Credentials;
  } catch {
    return null;
  }
}

/** Remove the store. Unlinks the link itself when the path is a symlink. */
export function clearCredentials(): void {
  try {
    if (fs.existsSync(credentialsPath())) {
      fs.unlinkSync(credentialsPath());
    }
  } catch {
    // ignore
  }
}

const EXPIRY_SKEW_MS = 30_000;

/**
 * True when the access token is present and not about to expire.
 *
 * The 30s skew avoids the classic race where a token looks valid when the
 * request is built and expires while it is in flight.
 */
export function isLoggedIn(): boolean {
  const cred = loadCredentials();
  if (!cred?.token) return false;
  if (cred.expiresAt) {
    const expires = new Date(cred.expiresAt).getTime();
    if (Number.isFinite(expires)) return expires - EXPIRY_SKEW_MS > Date.now();
  }
  return true;
}

/** True when the session can be renewed even though the access token expired. */
export function hasRefreshableSession(): boolean {
  return Boolean(loadCredentials()?.refreshToken);
}

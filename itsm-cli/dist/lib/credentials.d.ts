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
export declare function credentialsPath(): string;
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
export declare function describeCredentialStore(): CredentialStoreReport;
export declare function saveCredentials(cred: Credentials): void;
/** Merge a change into the stored credentials without dropping refresh token / identity. */
export declare function updateCredentials(patch: Partial<Credentials>): Credentials | null;
export declare function loadCredentials(): Credentials | null;
/** Remove the store. Unlinks the link itself when the path is a symlink. */
export declare function clearCredentials(): void;
/**
 * True when the access token is present and not about to expire.
 *
 * The 30s skew avoids the classic race where a token looks valid when the
 * request is built and expires while it is in flight.
 */
export declare function isLoggedIn(): boolean;
/** True when the session can be renewed even though the access token expired. */
export declare function hasRefreshableSession(): boolean;

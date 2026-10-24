import React from 'react';
interface LoginProps {
    username: string;
    /** Password captured from argv. Never used: argv leaks through `ps` and shell history. */
    password?: string;
}
/**
 * Interactive login.
 *
 * The password is NEVER taken from argv. It is read either from a masked
 * prompt (TTY) or from stdin (`printf %s "$PASS" | itsm login -u admin`), so it
 * never appears in the process table or shell history. Headless callers should
 * use `itsm auth login --password-stdin`, which additionally stores a refresh
 * token and emits a JSON envelope.
 */
export declare const LoginCommand: React.FC<LoginProps>;
export {};

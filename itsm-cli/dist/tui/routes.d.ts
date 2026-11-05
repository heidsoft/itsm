/**
 * TUI route registry.
 *
 * The interactive surface used to be a hand-written `switch` in `index.tsx`,
 * separate from the agent command registry. That made drift structural: adding
 * a command to the registry never reached the TUI, and the two tables could
 * disagree about which verbs exist. Routes now live here as data, help is
 * generated from the same list, and `GenericCommand` (see `GenericCommand.tsx`)
 * executes anything the curated list does not cover through the *agent*
 * registry — so every one of the CLI's commands is reachable from the TUI,
 * with bespoke screens only where a table/detail view earns its keep.
 *
 * Rendering functions receive the parsed flags but must not mutate server
 * state directly; the components fetch through the shared `apiClient`.
 */
import React from 'react';
/** Flags the TUI understands. Unknown flags still reach `GenericCommand` through raw argv. */
export interface TuiFlags {
    status?: string;
    priority?: string;
    type?: string;
    category?: string;
    channel?: string;
    message?: string;
    title?: string;
    description?: string;
    outcome?: string;
    comment?: string;
    unread?: boolean;
    page?: number;
    q?: string;
    username?: string;
    password?: string;
    help?: boolean;
    version?: boolean;
    [key: string]: unknown;
}
export interface TuiRouteContext {
    input: string[];
    flags: TuiFlags;
}
export interface TuiRoute {
    id: string;
    /** Help group; routes are rendered in registry order, so keep a domain's entries adjacent. */
    group: string;
    usage: string;
    summary: string;
    match: (input: string[]) => boolean;
    render: (ctx: TuiRouteContext) => React.ReactNode;
}
export declare const TUI_ROUTES: TuiRoute[];
/** First matching route wins; specific routes must be listed before broader ones. */
export declare function resolveTuiRoute(input: string[]): TuiRoute | undefined;

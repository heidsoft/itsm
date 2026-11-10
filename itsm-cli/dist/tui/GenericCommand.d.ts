/**
 * Generic TUI command screen.
 *
 * Bespoke routes cover the hot paths with tables and detail layouts. Every
 * other CLI command — including commands added tomorrow — reaches the TUI
 * through this component, which parses the same raw argv as agent mode,
 * resolves the command in the *same* registry, runs the same flag/positional
 * validation, and honours the same confirmation gate. There is no second
 * dispatch table to keep in sync.
 *
 * The result is rendered generically: a list envelope becomes a table, an
 * object becomes key/value rows, a primitive is printed as-is.
 */
import React from 'react';
export declare const GenericCommand: React.FC<{
    argv: string[];
}>;

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
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { parseArgs } from '../agent/args.js';
import { findCommandByTokens, registry } from '../agent/registry.js';
import { needsConfirmation } from '../agent/spec.js';
import { validateFlags, validatePositionals } from '../agent/runner.js';
import { applyShaping } from '../agent/shape.js';
import { classifyError } from '../lib/errors.js';
import { configureBaseURL, configureTransport } from '../lib/api-client.js';
import { isRecord } from '../lib/records.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel, formatValue } from '../components/detail.js';
const MAX_GENERIC_COLUMNS = 6;
function scalarKeys(row) {
    return Object.entries(row)
        .filter(([, value]) => value === null || typeof value !== 'object')
        .map(([key]) => key);
}
function shapeRows(data) {
    if (Array.isArray(data)) {
        return data.filter(isRecord);
    }
    if (isRecord(data) && Array.isArray(data['items'])) {
        return data['items'].filter(isRecord);
    }
    return undefined;
}
function formatRows(rows, keys) {
    return rows.map((row) => {
        const out = {};
        for (const key of keys)
            out[key] = formatValue(row[key]);
        return out;
    });
}
const GenericResult = ({ spec, data, meta, skippedReason }) => {
    const rows = shapeRows(data);
    if (rows) {
        const keys = rows.length ? scalarKeys(rows[0]).slice(0, MAX_GENERIC_COLUMNS) : [];
        const total = typeof meta?.['total'] === 'number' ? meta['total'] : rows.length;
        return (React.createElement(Box, { flexDirection: "column" },
            React.createElement(Text, { bold: true, color: "cyan" }, spec.name),
            React.createElement(Text, { dimColor: true },
                "\u8FD4\u56DE ",
                rows.length,
                " \u6761",
                total !== rows.length ? ` / 共 ${total} 条` : ''),
            React.createElement(Newline, null),
            keys.length && rows.length ? (React.createElement(Table, { data: formatRows(rows, keys), columns: keys.map((key) => ({ key, header: key })) })) : (React.createElement(Text, { dimColor: true }, "\uFF08\u7A7A\u5217\u8868\uFF09")),
            React.createElement(Newline, null),
            React.createElement(Text, { dimColor: true },
                "\u673A\u5668\u53EF\u8BFB\uFF1Aitsm --json ",
                spec.name.replace(/\./g, ' '))));
    }
    if (isRecord(data)) {
        return (React.createElement(DetailPanel, { title: spec.name, rows: Object.entries(data).map(([key, value]) => [key, value]), sections: skippedReason ? [{ label: 'shaping', value: skippedReason }] : undefined }));
    }
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, spec.name),
        React.createElement(Newline, null),
        React.createElement(Text, { wrap: "wrap" }, formatValue(data))));
};
/**
 * Bare `itsm <domain>` has no command in the registry (commands are two tokens).
 * Rather than answering "unknown command", list that domain's commands from the
 * live registry — the same discovery an agent gets from `itsm commands --json`.
 */
const DomainHelp = ({ domain }) => {
    const commands = registry().filter((spec) => spec.name === domain || spec.name.startsWith(`${domain}.`));
    if (!commands.length) {
        return (React.createElement(Box, { flexDirection: "column", padding: 1 },
            React.createElement(Text, { color: "red" },
                "Unknown command: ",
                domain),
            React.createElement(Text, { dimColor: true }, "Run `itsm --help`, or `itsm --json commands` for the full catalog.")));
    }
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" },
            domain,
            " \u00B7 ",
            commands.length,
            " \u6761\u547D\u4EE4"),
        React.createElement(Newline, null),
        commands.slice(0, 40).map((spec) => (React.createElement(Box, { key: spec.name },
            React.createElement(Text, { color: "cyan" },
                spec.usage.padEnd(58),
                " "),
            React.createElement(Text, { dimColor: true }, spec.summary)))),
        commands.length > 40 ? React.createElement(Text, { dimColor: true },
            "\u2026 \u8FD8\u6709 ",
            commands.length - 40,
            " \u6761\uFF0C\u7528 `itsm --json commands` \u67E5\u770B") : null,
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "\u76F4\u63A5\u6267\u884C\u4EFB\u610F\u4E00\u6761\uFF1Aitsm <domain> <verb> [flags]\uFF1B\u673A\u5668\u53EF\u8BFB\u52A0 --json\u3002")));
};
export const GenericCommand = ({ argv }) => {
    const [phase, setPhase] = useState({ kind: 'loading', label: argv.join(' ') });
    const requestId = useRef(0);
    const argvKey = useMemo(() => argv.join('\u0000'), [argv]);
    useEffect(() => {
        const id = ++requestId.current;
        let cancelled = false;
        const fail = (error) => {
            const classified = classifyError(error);
            if (cancelled)
                return;
            setPhase({
                kind: 'error',
                message: classified.agentError.message,
                hint: classified.agentError.hint,
                errorKind: classified.agentError.kind,
            });
        };
        const execute = async () => {
            let parsed;
            try {
                parsed = parseArgs(argv);
            }
            catch (error) {
                fail(error);
                return;
            }
            configureBaseURL(parsed.baseUrl);
            configureTransport({
                ...(parsed.timeoutMs !== undefined ? { timeoutMs: parsed.timeoutMs } : {}),
                ...(parsed.retries !== undefined ? { retries: parsed.retries } : {}),
            });
            const spec = findCommandByTokens(parsed.positional);
            if (!spec) {
                if (cancelled)
                    return;
                if (parsed.positional.length === 1) {
                    setPhase({ kind: 'domain', domain: parsed.positional[0] ?? '' });
                    return;
                }
                setPhase({
                    kind: 'error',
                    message: `Unknown command '${parsed.positional.slice(0, 2).join(' ') || '(empty)'}'`,
                    hint: 'Run `itsm --help`, or `itsm --json commands` for the full catalog.',
                    errorKind: 'usage',
                });
                return;
            }
            const rest = parsed.positional.slice(spec.name.split('.').length);
            try {
                validateFlags(spec, parsed);
                validatePositionals(spec, rest);
            }
            catch (error) {
                fail(error);
                return;
            }
            if (parsed.dryRun) {
                if (!cancelled)
                    setPhase({ kind: 'dry-run', spec });
                return;
            }
            if (needsConfirmation(spec, rest[0]) && !parsed.yes) {
                if (!cancelled)
                    setPhase({ kind: 'confirm', spec });
                return;
            }
            try {
                const data = await spec.run({
                    args: { positional: rest, flags: parsed.flags, yes: parsed.yes, dryRun: parsed.dryRun },
                });
                if (cancelled || requestId.current !== id)
                    return;
                const shaped = applyShaping(data, {
                    ...(parsed.fields ? { fields: parsed.fields } : {}),
                    ...(parsed.limit !== undefined ? { limit: parsed.limit } : {}),
                });
                setPhase({
                    kind: 'done',
                    spec,
                    data: shaped.data,
                    ...(shaped.meta ? { meta: shaped.meta } : {}),
                    ...(shaped.skippedReason ? { skippedReason: shaped.skippedReason } : {}),
                });
            }
            catch (error) {
                if (requestId.current === id)
                    fail(error);
            }
        };
        void execute();
        return () => {
            cancelled = true;
        };
    }, [argv, argvKey]);
    switch (phase.kind) {
        case 'loading':
            return React.createElement(Spinner, { text: `Loading ${phase.label}...` });
        case 'domain':
            return React.createElement(DomainHelp, { domain: phase.domain });
        case 'confirm':
            return (React.createElement(Box, { flexDirection: "column", padding: 1 },
                React.createElement(Text, { bold: true, color: "yellow" }, "\u26A0 \u8BE5\u64CD\u4F5C\u9700\u8981\u786E\u8BA4\uFF0C\u5C1A\u672A\u53D1\u9001\u8BF7\u6C42"),
                React.createElement(Newline, null),
                React.createElement(Text, null,
                    "\u547D\u4EE4   ",
                    React.createElement(Text, { color: "cyan" }, phase.spec.name)),
                React.createElement(Text, null,
                    "\u98CE\u9669   ",
                    phase.spec.risk),
                React.createElement(Text, null,
                    "\u7AEF\u70B9   ",
                    phase.spec.endpoints.join(', ')),
                React.createElement(Text, null,
                    "\u7528\u6CD5   ",
                    phase.spec.usage),
                React.createElement(Newline, null),
                React.createElement(Text, null,
                    "\u786E\u8BA4\u540E\u8FFD\u52A0 ",
                    React.createElement(Text, { bold: true }, "--yes"),
                    " \u91CD\u8BD5\uFF1A"),
                React.createElement(Text, { color: "cyan" },
                    phase.spec.usage,
                    " --yes"),
                React.createElement(Text, { dimColor: true }, "\u9AD8\u98CE\u9669\u5199\u4E0D\u5F97\u81EA\u52A8\u91CD\u8BD5\uFF1B\u5148\u5411\u64CD\u4F5C\u8005\u8BF4\u660E\u5F71\u54CD\u9762\u3002")));
        case 'dry-run':
            return (React.createElement(Box, { flexDirection: "column", padding: 1 },
                React.createElement(Text, { bold: true, color: "cyan" },
                    "Dry run \u00B7 ",
                    phase.spec.name),
                React.createElement(Newline, null),
                React.createElement(Text, null,
                    "\u98CE\u9669   ",
                    phase.spec.risk),
                React.createElement(Text, null,
                    "\u7AEF\u70B9   ",
                    phase.spec.endpoints.join(', ')),
                React.createElement(Text, null,
                    "\u7528\u6CD5   ",
                    phase.spec.usage),
                React.createElement(Newline, null),
                React.createElement(Text, { dimColor: true }, "\u672A\u53D1\u9001\u8BF7\u6C42\u3002\u53BB\u6389 --dry-run \u6267\u884C\uFF1B\u9AD8\u98CE\u9669\u5199\u4ECD\u9700 --yes\u3002")));
        case 'error':
            return (React.createElement(Box, { flexDirection: "column", padding: 1 },
                React.createElement(Text, { color: "red" },
                    "\u2717 ",
                    phase.message),
                phase.errorKind ? React.createElement(Text, { dimColor: true },
                    "kind: ",
                    phase.errorKind) : null,
                phase.hint ? React.createElement(Text, { dimColor: true }, phase.hint) : null));
        case 'done':
            return React.createElement(GenericResult, { ...phase });
        default:
            return React.createElement(Text, { color: "red" },
                "Unknown command: ",
                argv.join(' '));
    }
};

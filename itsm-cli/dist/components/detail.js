import React from 'react';
import { Box, Text, Newline } from 'ink';
/** Render any cell value without leaking `[object Object]` or hiding `0`/`false`. */
export function formatValue(value) {
    if (value === null || value === undefined || value === '')
        return '—';
    if (typeof value === 'boolean')
        return value ? 'yes' : 'no';
    if (Array.isArray(value))
        return value.length ? value.map((entry) => formatValue(entry)).join(', ') : '—';
    if (typeof value === 'object')
        return JSON.stringify(value);
    return String(value);
}
/**
 * Key/value detail view shared by the domain screens.
 *
 * Rows are filtered only when the value is genuinely absent (undefined/null/'')
 * — `0`, `false` and empty arrays are real answers and must stay visible.
 */
export const DetailPanel = ({ title, rows, sections }) => (React.createElement(Box, { flexDirection: "column", padding: 1 },
    React.createElement(Text, { bold: true, color: "cyan" }, title),
    React.createElement(Newline, null),
    rows
        .filter(([, value]) => value !== undefined && value !== null && value !== '')
        .map(([label, value]) => (React.createElement(Box, { key: label },
        React.createElement(Text, { color: "gray" },
            label.padEnd(14),
            " "),
        React.createElement(Text, { wrap: "wrap" }, formatValue(value))))),
    (sections ?? [])
        .filter((section) => section.value !== undefined && section.value !== null && section.value !== '')
        .map((section) => (React.createElement(Box, { key: section.label, flexDirection: "column", marginTop: 1 },
        React.createElement(Text, { color: "gray" }, section.label),
        React.createElement(Text, { wrap: "wrap" }, formatValue(section.value)))))));

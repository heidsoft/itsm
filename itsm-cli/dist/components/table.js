import React from 'react';
import { Box, Text } from 'ink';
export function Table({ columns, data }) {
    if (data.length === 0) {
        return React.createElement(Text, { dimColor: true }, "No data");
    }
    const colWidths = columns.map(col => {
        if (col.width)
            return col.width;
        let max = col.header.length;
        for (const row of data) {
            const val = String(row[col.key] ?? '');
            if (val.length > max)
                max = val.length;
        }
        return Math.min(max, 30);
    });
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Box, null, columns.map((col, i) => (React.createElement(Box, { key: i, width: colWidths[i] + 2 },
            React.createElement(Text, { bold: true, color: "cyan" }, col.header.padEnd(colWidths[i])))))),
        React.createElement(Box, null, colWidths.map((w, i) => (React.createElement(Text, { key: i, color: "gray" }, '─'.repeat(w + 2))))),
        data.map((row, ri) => (React.createElement(Box, { key: ri }, columns.map((col, ci) => {
            const value = row[col.key] ?? '';
            const content = col.render
                ? col.render(row)
                : React.createElement(Text, null, String(value).substring(0, colWidths[ci]));
            return (React.createElement(Box, { key: ci, width: colWidths[ci] + 2 }, content));
        }))))));
}

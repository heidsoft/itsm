import React from 'react';
import { Box, Text, Newline } from 'ink';
import { TUI_ROUTES } from './routes.js';
/**
 * Help is generated from the same registry the router uses: a route that is
 * not listed here does not exist, and a route that exists cannot be missing
 * from `--help`.
 */
export const HelpScreen = () => {
    const groups = [];
    for (const route of TUI_ROUTES) {
        if (!groups.includes(route.group))
            groups.push(route.group);
    }
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "ITSM CLI v1.0.0"),
        React.createElement(Text, null, "AI-Native ITSM \u00B7 React for CLIs \u00B7 built with Ink"),
        React.createElement(Newline, null),
        groups.map((group) => {
            const routes = TUI_ROUTES.filter((route) => route.group === group);
            const width = Math.min(52, Math.max(...routes.map((route) => route.usage.length), 0) + 2);
            return (React.createElement(Box, { key: group, flexDirection: "column", marginBottom: 1 },
                React.createElement(Text, { bold: true }, group),
                routes.map((route) => (React.createElement(Box, { key: route.id },
                    React.createElement(Text, { dimColor: true }, `  ${route.usage.padEnd(width)}`),
                    React.createElement(Text, { dimColor: true }, route.summary))))));
        }),
        React.createElement(Text, { dimColor: true }, "\u901A\u7528\u515C\u5E95\uFF1A\u672A\u5355\u5217\u7684\u547D\u4EE4\u7528 `itsm <domain> <verb>` \u76F4\u63A5\u6267\u884C\uFF08\u4E0E --json \u540C\u4E00\u6CE8\u518C\u8868\uFF09\u3002"),
        React.createElement(Text, { dimColor: true }, "\u673A\u5668\u53EF\u8BFB\uFF1Aitsm --json <command>   \u00B7   MCP\uFF1Anode bin/itsm-mcp.js"),
        React.createElement(Text, { dimColor: true },
            "Backend: ",
            process.env['ITSM_BASE_URL'] ?? 'http://localhost:8090')));
};

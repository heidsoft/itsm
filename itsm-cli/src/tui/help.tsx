import React from 'react';
import { Box, Text, Newline } from 'ink';
import { TUI_ROUTES } from './routes.js';

/**
 * Help is generated from the same registry the router uses: a route that is
 * not listed here does not exist, and a route that exists cannot be missing
 * from `--help`.
 */
export const HelpScreen: React.FC = () => {
  const groups: string[] = [];
  for (const route of TUI_ROUTES) {
    if (!groups.includes(route.group)) groups.push(route.group);
  }

  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">ITSM CLI v1.0.0</Text>
      <Text>AI-Native ITSM · React for CLIs · built with Ink</Text>
      <Newline />
      {groups.map((group) => {
        const routes = TUI_ROUTES.filter((route) => route.group === group);
        const width = Math.min(
          52,
          Math.max(...routes.map((route) => route.usage.length), 0) + 2,
        );
        return (
          <Box key={group} flexDirection="column" marginBottom={1}>
            <Text bold>{group}</Text>
            {routes.map((route) => (
              <Box key={route.id}>
                <Text dimColor>{`  ${route.usage.padEnd(width)}`}</Text>
                <Text dimColor>{route.summary}</Text>
              </Box>
            ))}
          </Box>
        );
      })}
      <Text dimColor>通用兜底：未单列的命令用 `itsm &lt;domain&gt; &lt;verb&gt;` 直接执行（与 --json 同一注册表）。</Text>
      <Text dimColor>机器可读：itsm --json &lt;command&gt;   ·   MCP：node bin/itsm-mcp.js</Text>
      <Text dimColor>Backend: {process.env['ITSM_BASE_URL'] ?? 'http://localhost:8090'}</Text>
    </Box>
  );
};

import React from 'react';
import { Box, Text } from 'ink';
import { isLoggedIn } from './lib/credentials.js';
import { HelpScreen } from './tui/help.js';
import { resolveTuiRoute } from './tui/routes.js';
import type { TuiFlags } from './tui/routes.js';
import { GenericCommand } from './tui/GenericCommand.js';

export interface CliFlags extends TuiFlags {
  help: boolean;
  version: boolean;
}

/**
 * Interactive router.
 *
 * Curated routes come from `tui/routes.tsx`; anything else goes to
 * `GenericCommand`, which parses the raw argv and resolves the command in the
 * agent registry. The TUI therefore cannot fall behind the command surface.
 */
const Router: React.FC<{ input: string[]; flags: CliFlags; argv: string[] }> = ({ input, flags, argv }) => {
  const route = resolveTuiRoute(input);
  if (route) return <>{route.render({ input, flags })}</>;

  // `itsm --json` never reaches this branch (bin/itsm.js dispatches first), so
  // the raw argv is exactly what the generic executor needs.
  return <GenericCommand argv={argv} />;
};

export default function App({
  cli,
  argv,
}: {
  cli: { input: string[]; flags: CliFlags };
  argv: string[];
}) {
  const { input, flags } = cli;

  if (flags.help || flags.version || input.length === 0 || input[0] === 'help') {
    return <HelpScreen />;
  }

  const [cmd] = input;
  if (cmd !== 'login' && cmd !== 'logout' && !isLoggedIn()) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Not logged in</Text>
        <Text>Run: itsm login -u &lt;username&gt;   (password is prompted, never passed as a flag)</Text>
      </Box>
    );
  }

  return <Router input={input} flags={flags} argv={argv} />;
}

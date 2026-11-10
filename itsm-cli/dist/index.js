import React from 'react';
import { Box, Text } from 'ink';
import { isLoggedIn } from './lib/credentials.js';
import { HelpScreen } from './tui/help.js';
import { resolveTuiRoute } from './tui/routes.js';
import { GenericCommand } from './tui/GenericCommand.js';
/**
 * Interactive router.
 *
 * Curated routes come from `tui/routes.tsx`; anything else goes to
 * `GenericCommand`, which parses the raw argv and resolves the command in the
 * agent registry. The TUI therefore cannot fall behind the command surface.
 */
const Router = ({ input, flags, argv }) => {
    const route = resolveTuiRoute(input);
    if (route)
        return React.createElement(React.Fragment, null, route.render({ input, flags }));
    // `itsm --json` never reaches this branch (bin/itsm.js dispatches first), so
    // the raw argv is exactly what the generic executor needs.
    return React.createElement(GenericCommand, { argv: argv });
};
export default function App({ cli, argv, }) {
    const { input, flags } = cli;
    if (flags.help || flags.version || input.length === 0 || input[0] === 'help') {
        return React.createElement(HelpScreen, null);
    }
    const [cmd] = input;
    if (cmd !== 'login' && cmd !== 'logout' && !isLoggedIn()) {
        return (React.createElement(Box, { flexDirection: "column", padding: 1 },
            React.createElement(Text, { color: "red" }, "Not logged in"),
            React.createElement(Text, null, "Run: itsm login -u <username>   (password is prompted, never passed as a flag)")));
    }
    return React.createElement(Router, { input: input, flags: flags, argv: argv });
}

import React from 'react';
import type { TuiFlags } from './tui/routes.js';
export interface CliFlags extends TuiFlags {
    help: boolean;
    version: boolean;
}
export default function App({ cli, argv, }: {
    cli: {
        input: string[];
        flags: CliFlags;
    };
    argv: string[];
}): React.JSX.Element;

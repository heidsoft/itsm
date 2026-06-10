import React from 'react';
interface CliFlags {
    status?: string;
    priority?: string;
    page?: number;
    category?: string;
    type?: string;
    q?: string;
    unread?: boolean;
    channel?: string;
    comment?: string;
    outcome?: string;
    message?: string;
    title?: string;
    description?: string;
    username?: string;
    password?: string;
    help: boolean;
    version: boolean;
}
export default function App({ cli }: {
    cli: {
        input: string[];
        flags: CliFlags;
    };
}): React.JSX.Element;
export {};

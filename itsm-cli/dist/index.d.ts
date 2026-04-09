import React from 'react';
interface CliFlags {
    status?: string;
    priority?: string;
    page?: number;
    username?: string;
    password?: string;
    help: boolean;
    version: boolean;
}
interface AppProps {
    cli: {
        input: string[];
        flags: CliFlags;
    };
}
export default function App({ cli }: AppProps): React.JSX.Element;
export {};

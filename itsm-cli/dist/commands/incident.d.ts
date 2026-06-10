import React from 'react';
interface ListProps {
    status?: string;
    priority?: string;
    page?: number;
}
interface ViewProps {
    id: number;
}
export declare const IncidentListCommand: React.FC<ListProps>;
export declare const IncidentViewCommand: React.FC<ViewProps>;
export declare const IncidentTriageCommand: React.FC<{
    title: string;
    description: string;
}>;
export {};

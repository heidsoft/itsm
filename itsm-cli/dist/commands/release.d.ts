import React from 'react';
interface ListProps {
    status?: string;
    type?: string;
    page?: number;
}
interface ViewProps {
    id: number;
}
export declare const ReleaseListCommand: React.FC<ListProps>;
export declare const ReleaseViewCommand: React.FC<ViewProps>;
export {};

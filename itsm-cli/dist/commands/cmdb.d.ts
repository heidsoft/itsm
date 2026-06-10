import React from 'react';
interface ListProps {
    type?: string;
    status?: string;
    page?: number;
}
interface ViewProps {
    id: number;
}
export declare const CmdbListCommand: React.FC<ListProps>;
export declare const CmdbViewCommand: React.FC<ViewProps>;
export {};

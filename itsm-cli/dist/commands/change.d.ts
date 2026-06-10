import React from 'react';
interface ListProps {
    status?: string;
    page?: number;
}
interface ViewProps {
    id: number;
}
export declare const ChangeListCommand: React.FC<ListProps>;
export declare const ChangeViewCommand: React.FC<ViewProps>;
export {};

import React from 'react';
interface ListProps {
    page?: number;
}
interface ViewProps {
    id: number;
}
export declare const RequestListCommand: React.FC<ListProps>;
export declare const RequestViewCommand: React.FC<ViewProps>;
export {};

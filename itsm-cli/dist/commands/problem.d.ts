import React from 'react';
interface ListProps {
    status?: string;
    priority?: string;
    page?: number;
}
interface ViewProps {
    id: number;
}
export declare const ProblemListCommand: React.FC<ListProps>;
export declare const ProblemViewCommand: React.FC<ViewProps>;
export {};

import React from 'react';
interface Column<T> {
    key: string;
    header: string;
    width?: number;
    render?: (item: T) => React.ReactNode;
}
interface TableProps<T> {
    columns: Column<T>[];
    data: T[];
}
export declare function Table<T extends object>({ columns, data }: TableProps<T>): React.JSX.Element;
export {};

import React from 'react';
/** Render any cell value without leaking `[object Object]` or hiding `0`/`false`. */
export declare function formatValue(value: unknown): string;
export interface DetailSection {
    label: string;
    value: unknown;
}
interface DetailPanelProps {
    title: string;
    rows: Array<[string, unknown]>;
    sections?: DetailSection[];
}
/**
 * Key/value detail view shared by the domain screens.
 *
 * Rows are filtered only when the value is genuinely absent (undefined/null/'')
 * — `0`, `false` and empty arrays are real answers and must stay visible.
 */
export declare const DetailPanel: React.FC<DetailPanelProps>;
export {};

import React from 'react';
import { Box, Text, Newline } from 'ink';

/** Render any cell value without leaking `[object Object]` or hiding `0`/`false`. */
export function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'boolean') return value ? 'yes' : 'no';
  if (Array.isArray(value)) return value.length ? value.map((entry) => formatValue(entry)).join(', ') : '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

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
export const DetailPanel: React.FC<DetailPanelProps> = ({ title, rows, sections }) => (
  <Box flexDirection="column" padding={1}>
    <Text bold color="cyan">{title}</Text>
    <Newline />
    {rows
      .filter(([, value]) => value !== undefined && value !== null && value !== '')
      .map(([label, value]) => (
        <Box key={label}>
          <Text color="gray">{label.padEnd(14)} </Text>
          <Text wrap="wrap">{formatValue(value)}</Text>
        </Box>
      ))}
    {(sections ?? [])
      .filter((section) => section.value !== undefined && section.value !== null && section.value !== '')
      .map((section) => (
        <Box key={section.label} flexDirection="column" marginTop={1}>
          <Text color="gray">{section.label}</Text>
          <Text wrap="wrap">{formatValue(section.value)}</Text>
        </Box>
      ))}
  </Box>
);

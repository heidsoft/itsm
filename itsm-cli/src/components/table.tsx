import React from 'react';
import { Box, Text } from 'ink';

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

export function Table<T extends object>({ columns, data }: TableProps<T>) {
  if (data.length === 0) {
    return <Text dimColor>No data</Text>;
  }

  const colWidths = columns.map(col => {
    if (col.width) return col.width;
    let max = col.header.length;
    for (const row of data) {
      const val = String((row as Record<string, unknown>)[col.key] ?? '');
      if (val.length > max) max = val.length;
    }
    return Math.min(max, 30);
  });

  return (
    <Box flexDirection="column">
      <Box>
        {columns.map((col, i) => (
          <Box key={i} width={colWidths[i] + 2}>
            <Text bold color="cyan">
              {col.header.padEnd(colWidths[i])}
            </Text>
          </Box>
        ))}
      </Box>
      <Box>
        {colWidths.map((w, i) => (
          <Text key={i} color="gray">
            {'─'.repeat(w + 2)}
          </Text>
        ))}
      </Box>
      {data.map((row, ri) => (
        <Box key={ri}>
          {columns.map((col, ci) => {
            const value = (row as Record<string, unknown>)[col.key] ?? '';
            const content = col.render
              ? col.render(row)
              : <Text>{String(value).substring(0, colWidths[ci])}</Text>;
            return (
              <Box key={ci} width={colWidths[ci] + 2}>
                {content}
              </Box>
            );
          })}
        </Box>
      ))}
    </Box>
  );
}

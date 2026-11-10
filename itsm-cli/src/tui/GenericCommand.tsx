/**
 * Generic TUI command screen.
 *
 * Bespoke routes cover the hot paths with tables and detail layouts. Every
 * other CLI command — including commands added tomorrow — reaches the TUI
 * through this component, which parses the same raw argv as agent mode,
 * resolves the command in the *same* registry, runs the same flag/positional
 * validation, and honours the same confirmation gate. There is no second
 * dispatch table to keep in sync.
 *
 * The result is rendered generically: a list envelope becomes a table, an
 * object becomes key/value rows, a primitive is printed as-is.
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Box, Text, Newline } from 'ink';
import { parseArgs } from '../agent/args.js';
import type { ParsedArgs } from '../agent/args.js';
import { findCommandByTokens, registry } from '../agent/registry.js';
import { needsConfirmation } from '../agent/spec.js';
import type { CommandSpec } from '../agent/spec.js';
import { validateFlags, validatePositionals } from '../agent/runner.js';
import { applyShaping } from '../agent/shape.js';
import { classifyError } from '../lib/errors.js';
import { configureBaseURL, configureTransport } from '../lib/api-client.js';
import { isRecord, type JsonRecord } from '../lib/records.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
import { DetailPanel, formatValue } from '../components/detail.js';

type Phase =
  | { kind: 'loading'; label: string }
  | { kind: 'domain'; domain: string }
  | { kind: 'confirm'; spec: CommandSpec }
  | { kind: 'dry-run'; spec: CommandSpec }
  | {
      kind: 'done';
      spec: CommandSpec;
      data: unknown;
      meta?: Record<string, unknown>;
      skippedReason?: string;
    }
  | { kind: 'error'; message: string; hint?: string; errorKind?: string };

const MAX_GENERIC_COLUMNS = 6;

function scalarKeys(row: JsonRecord): string[] {
  return Object.entries(row)
    .filter(([, value]) => value === null || typeof value !== 'object')
    .map(([key]) => key);
}

function shapeRows(data: unknown): JsonRecord[] | undefined {
  if (Array.isArray(data)) {
    return data.filter(isRecord);
  }
  if (isRecord(data) && Array.isArray(data['items'])) {
    return data['items'].filter(isRecord);
  }
  return undefined;
}

function formatRows(rows: JsonRecord[], keys: string[]): JsonRecord[] {
  return rows.map((row) => {
    const out: JsonRecord = {};
    for (const key of keys) out[key] = formatValue(row[key]);
    return out;
  });
}

const GenericResult: React.FC<Extract<Phase, { kind: 'done' }>> = ({ spec, data, meta, skippedReason }) => {
  const rows = shapeRows(data);

  if (rows) {
    const keys = rows.length ? scalarKeys(rows[0]).slice(0, MAX_GENERIC_COLUMNS) : [];
    const total = typeof meta?.['total'] === 'number' ? meta['total'] : rows.length;
    return (
      <Box flexDirection="column">
        <Text bold color="cyan">{spec.name}</Text>
        <Text dimColor>
          返回 {rows.length} 条{total !== rows.length ? ` / 共 ${total} 条` : ''}
        </Text>
        <Newline />
        {keys.length && rows.length ? (
          <Table
            data={formatRows(rows, keys)}
            columns={keys.map((key) => ({ key, header: key }))}
          />
        ) : (
          <Text dimColor>（空列表）</Text>
        )}
        <Newline />
        <Text dimColor>机器可读：itsm --json {spec.name.replace(/\./g, ' ')}</Text>
      </Box>
    );
  }

  if (isRecord(data)) {
    return (
      <DetailPanel
        title={spec.name}
        rows={Object.entries(data).map(([key, value]) => [key, value])}
        sections={skippedReason ? [{ label: 'shaping', value: skippedReason }] : undefined}
      />
    );
  }

  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">{spec.name}</Text>
      <Newline />
      <Text wrap="wrap">{formatValue(data)}</Text>
    </Box>
  );
};

/**
 * Bare `itsm <domain>` has no command in the registry (commands are two tokens).
 * Rather than answering "unknown command", list that domain's commands from the
 * live registry — the same discovery an agent gets from `itsm commands --json`.
 */
const DomainHelp: React.FC<{ domain: string }> = ({ domain }) => {
  const commands = registry().filter((spec) => spec.name === domain || spec.name.startsWith(`${domain}.`));
  if (!commands.length) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Unknown command: {domain}</Text>
        <Text dimColor>Run `itsm --help`, or `itsm --json commands` for the full catalog.</Text>
      </Box>
    );
  }
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold color="cyan">{domain} · {commands.length} 条命令</Text>
      <Newline />
      {commands.slice(0, 40).map((spec) => (
        <Box key={spec.name}>
          <Text color="cyan">{spec.usage.padEnd(58)} </Text>
          <Text dimColor>{spec.summary}</Text>
        </Box>
      ))}
      {commands.length > 40 ? <Text dimColor>… 还有 {commands.length - 40} 条，用 `itsm --json commands` 查看</Text> : null}
      <Newline />
      <Text dimColor>直接执行任意一条：itsm &lt;domain&gt; &lt;verb&gt; [flags]；机器可读加 --json。</Text>
    </Box>
  );
};

export const GenericCommand: React.FC<{ argv: string[] }> = ({ argv }) => {
  const [phase, setPhase] = useState<Phase>({ kind: 'loading', label: argv.join(' ') });
  const requestId = useRef(0);
  const argvKey = useMemo(() => argv.join('\u0000'), [argv]);

  useEffect(() => {
    const id = ++requestId.current;
    let cancelled = false;

    const fail = (error: unknown): void => {
      const classified = classifyError(error);
      if (cancelled) return;
      setPhase({
        kind: 'error',
        message: classified.agentError.message,
        hint: classified.agentError.hint,
        errorKind: classified.agentError.kind,
      });
    };

    const execute = async (): Promise<void> => {
      let parsed: ParsedArgs;
      try {
        parsed = parseArgs(argv);
      } catch (error) {
        fail(error);
        return;
      }

      configureBaseURL(parsed.baseUrl);
      configureTransport({
        ...(parsed.timeoutMs !== undefined ? { timeoutMs: parsed.timeoutMs } : {}),
        ...(parsed.retries !== undefined ? { retries: parsed.retries } : {}),
      });

      const spec = findCommandByTokens(parsed.positional);
      if (!spec) {
        if (cancelled) return;
        if (parsed.positional.length === 1) {
          setPhase({ kind: 'domain', domain: parsed.positional[0] ?? '' });
          return;
        }
        setPhase({
          kind: 'error',
          message: `Unknown command '${parsed.positional.slice(0, 2).join(' ') || '(empty)'}'`,
          hint: 'Run `itsm --help`, or `itsm --json commands` for the full catalog.',
          errorKind: 'usage',
        });
        return;
      }

      const rest = parsed.positional.slice(spec.name.split('.').length);
      try {
        validateFlags(spec, parsed);
        validatePositionals(spec, rest);
      } catch (error) {
        fail(error);
        return;
      }

      if (parsed.dryRun) {
        if (!cancelled) setPhase({ kind: 'dry-run', spec });
        return;
      }

      if (needsConfirmation(spec, rest[0]) && !parsed.yes) {
        if (!cancelled) setPhase({ kind: 'confirm', spec });
        return;
      }

      try {
        const data = await spec.run({
          args: { positional: rest, flags: parsed.flags, yes: parsed.yes, dryRun: parsed.dryRun },
        });
        if (cancelled || requestId.current !== id) return;
        const shaped = applyShaping(data, {
          ...(parsed.fields ? { fields: parsed.fields } : {}),
          ...(parsed.limit !== undefined ? { limit: parsed.limit } : {}),
        });
        setPhase({
          kind: 'done',
          spec,
          data: shaped.data,
          ...(shaped.meta ? { meta: shaped.meta } : {}),
          ...(shaped.skippedReason ? { skippedReason: shaped.skippedReason } : {}),
        });
      } catch (error) {
        if (requestId.current === id) fail(error);
      }
    };

    void execute();
    return () => {
      cancelled = true;
    };
  }, [argv, argvKey]);

  switch (phase.kind) {
    case 'loading':
      return <Spinner text={`Loading ${phase.label}...`} />;

    case 'domain':
      return <DomainHelp domain={phase.domain} />;

    case 'confirm':
      return (
        <Box flexDirection="column" padding={1}>
          <Text bold color="yellow">⚠ 该操作需要确认，尚未发送请求</Text>
          <Newline />
          <Text>命令   <Text color="cyan">{phase.spec.name}</Text></Text>
          <Text>风险   {phase.spec.risk}</Text>
          <Text>端点   {phase.spec.endpoints.join(', ')}</Text>
          <Text>用法   {phase.spec.usage}</Text>
          <Newline />
          <Text>确认后追加 <Text bold>--yes</Text> 重试：</Text>
          <Text color="cyan">{phase.spec.usage} --yes</Text>
          <Text dimColor>高风险写不得自动重试；先向操作者说明影响面。</Text>
        </Box>
      );

    case 'dry-run':
      return (
        <Box flexDirection="column" padding={1}>
          <Text bold color="cyan">Dry run · {phase.spec.name}</Text>
          <Newline />
          <Text>风险   {phase.spec.risk}</Text>
          <Text>端点   {phase.spec.endpoints.join(', ')}</Text>
          <Text>用法   {phase.spec.usage}</Text>
          <Newline />
          <Text dimColor>未发送请求。去掉 --dry-run 执行；高风险写仍需 --yes。</Text>
        </Box>
      );

    case 'error':
      return (
        <Box flexDirection="column" padding={1}>
          <Text color="red">✗ {phase.message}</Text>
          {phase.errorKind ? <Text dimColor>kind: {phase.errorKind}</Text> : null}
          {phase.hint ? <Text dimColor>{phase.hint}</Text> : null}
        </Box>
      );

    case 'done':
      return <GenericResult {...phase} />;

    default:
      return <Text color="red">Unknown command: {argv.join(' ')}</Text>;
  }
};

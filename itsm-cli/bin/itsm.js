#!/usr/bin/env node
// itsm — dual-surface entrypoint.
//   itsm --json <domain verb> [...]        → headless agent protocol (see dist/agent/runner.js)
//   itsm <interactive commands>            → Ink TUI (human operators)
//
// Agent mode is checked BEFORE any TUI dependency is imported: Ink/React cost
// ~3.9s to load, which an agent paying it on every one of dozens of calls would
// experience as a broken tool. Keeping the agent branch dependency-free takes a
// call from ~4.7s to ~0.8s.
const argv = process.argv.slice(2);

if (argv.includes('--json')) {
  const { runAgent } = await import('../dist/agent/runner.js');
  const code = await runAgent(argv);
  // Do NOT plain process.exit(): stdout to a pipe is asynchronous in Node, and
  // exit() kills the process before libuv drains the buffer — a 26KB schema
  // envelope arrived truncated at exactly 8192 bytes (the first chunk) when
  // driven over a pipe, and the consumer saw "Unterminated string in JSON".
  // Do NOT fall through either (setting exitCode alone continues into the TUI
  // branch below, whose banner then pollutes stdout). Wait for the queued
  // writes to drain, then exit deterministically: no truncation, no stray
  // output, no reliance on the event loop being otherwise empty.
  await new Promise((resolve) => process.stdout.write('', () => resolve(undefined)));
  process.exit(code);
}

const { render } = await import('ink');
const React = (await import('react')).default;
const meow = (await import('meow')).default;
const { default: App } = await import('../dist/index.js');

const cli = meow(
  `
  ITSM CLI — AI-Native ITSM command line

  Interactive (TUI)
    $ itsm login -u <user> -p <pass>
    $ itsm tickets --status open                       List tickets
    $ itsm incident triage --title X --description Y   AI triage
    $ itsm sla stats                                   SLA dashboard
    $ itsm connector send feishu --channel ou_xxx --message "工单已派发"

  Agent mode (headless, exact-one JSON envelope + semantic exit codes)
    $ itsm --json doctor                               Reachability, identity, per-domain probes
    $ itsm --json schema ticket.create                 Required params for one command
    $ itsm --json schema --as tools --domain ticket    Function-calling definitions
    $ itsm --json commands                             Command catalog (capability discovery)
    $ itsm --json manifest                             Protocol manifest: exit codes, env vars
    $ itsm --json ticket list --limit 5 --fields id,title,status
    $ itsm --json infra api <METHOD> <path> [--data <json>]     Raw escape hatch (/api/v1 relative)
    $ itsm --json change delete CHG-1                  High-risk write → exit 10 + confirmation_required
    $ itsm --json change delete CHG-1 --yes            Execute after explicit consent

  Agent mode help
    $ itsm --json ticket --help                        Returns usage of all ticket verbs
  Options (TUI)
    --status / --priority / --type / --category / --channel / --message / --title /
    --description / --outcome / --comment / --unread / --page <n> / -h / -v
`,
  {
    importMeta: import.meta,
    flags: {
      status: { type: 'string' },
      priority: { type: 'string' },
      type: { type: 'string' },
      category: { type: 'string' },
      channel: { type: 'string' },
      message: { type: 'string' },
      title: { type: 'string' },
      description: { type: 'string' },
      outcome: { type: 'string' },
      comment: { type: 'string' },
      unread: { type: 'boolean' },
      page: { type: 'number' },
      username: { type: 'string' },
      password: { type: 'string' },
      assigneeId: { type: 'string' },
      requesterId: { type: 'string' },
      data: { type: 'string' },
      baseUrl: { type: 'string' },
      q: { type: 'string' },
    },
  }
);

render(React.createElement(App, { cli }));

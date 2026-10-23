#!/usr/bin/env node
// itsm-mcp — MCP server over the ITSM command registry (stdio transport).
//
// Point an MCP client at this file and it gets the same 200+ command surface as
// `itsm --json`, with derived JSON Schemas, risk-gated writes, bounded list
// payloads and product docs as resources. No npm install required: the compiled
// dist/ tree is committed, exactly like the headless CLI.
//
//   { "mcpServers": { "itsm": {
//       "command": "node",
//       "args": ["/path/to/itsm-cli/bin/itsm-mcp.js"],
//       "env": { "ITSM_BASE_URL": "http://localhost:8090" }
//   } } }

const argv = process.argv.slice(2);

if (argv.includes('--help') || argv.includes('-h')) {
  process.stderr.write(
    [
      'itsm-mcp — MCP server for the ITSM platform (stdio, JSON-RPC 2.0)',
      '',
      'Usage:',
      '  itsm-mcp                speak MCP on stdin/stdout',
      '  itsm-mcp --describe     print the tool surface and exit (debugging)',
      '',
      'Environment:',
      '  ITSM_BASE_URL          backend base URL (default http://localhost:8090, no /api/v1)',
      '  ITSM_TOKEN[_FILE]      bearer token; else ITSM_USERNAME/ITSM_PASSWORD[_FILE]; else ~/.itsm/credentials',
      '  ITSM_MCP_DEBUG         set to 1 to log protocol activity to stderr',
      '  ITSM_REPO_ROOT         repository root for itsm://docs/* resources (auto-detected)',
      '',
    ].join('\n')
  );
  process.exit(0);
}

const { serve } = await import('../dist/mcp/server.js');
const { buildTools } = await import('../dist/mcp/catalog.js');

if (argv.includes('--describe')) {
  const tools = buildTools();
  process.stdout.write(
    `${JSON.stringify(
      { tools: tools.map((t) => ({ name: t.name, command: t.command, title: t.title })) },
      null,
      2
    )}\n`
  );
  process.exit(0);
}

let exiting = false;

function shutdown(code) {
  if (exiting) return;
  exiting = true;
  // stdout to a pipe is asynchronous: writing before exit() prevents a truncated
  // final frame (the same failure mode the CLI agent branch hit at 8192 bytes).
  process.stdout.write('', () => process.exit(code));
}

const { transport } = serve({ onClose: () => shutdown(0) });

process.on('SIGINT', () => shutdown(0));
process.on('SIGTERM', () => shutdown(0));
process.stdout.on('error', (err) => {
  // A client that closes the pipe early is normal; don't die with EPIPE noise.
  if (err && err.code === 'EPIPE') shutdown(0);
});
process.stdin.on('error', () => shutdown(0));

// The transport already reported the reason; this keeps an in-flight write from
// resurrecting a half-closed session.
process.on('unhandledRejection', (reason) => {
  process.stderr.write(`[itsm-mcp] unhandled rejection: ${reason instanceof Error ? reason.stack : String(reason)}\n`);
});

void transport;

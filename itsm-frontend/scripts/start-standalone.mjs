import { spawn } from 'node:child_process';
import { stat } from 'node:fs/promises';
import path from 'node:path';

const server = path.join(process.cwd(), '.next', 'standalone', 'server.js');
await stat(server).catch(() => {
  throw new Error('Standalone server is missing. Run "npm run build" before "npm start".');
});

const child = spawn(process.execPath, [server], {
  env: process.env,
  stdio: 'inherit',
});

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => child.kill(signal));
}

child.on('error', error => {
  console.error(`Failed to start standalone server: ${error.message}`);
  process.exitCode = 1;
});

child.on('exit', (code, signal) => {
  process.exitCode = signal ? 1 : (code ?? 1);
});

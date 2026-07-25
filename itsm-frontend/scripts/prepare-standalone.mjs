import { cp, mkdir, stat } from 'node:fs/promises';
import path from 'node:path';

const root = process.cwd();
const standalone = path.join(root, '.next', 'standalone');
const server = path.join(standalone, 'server.js');

await stat(server).catch(() => {
  throw new Error('Next.js standalone output is missing. Ensure output: "standalone" is configured.');
});

await mkdir(path.join(standalone, '.next'), { recursive: true });
await cp(path.join(root, '.next', 'static'), path.join(standalone, '.next', 'static'), {
  recursive: true,
  force: true,
});

const publicDir = path.join(root, 'public');
if (await stat(publicDir).then(() => true).catch(() => false)) {
  await cp(publicDir, path.join(standalone, 'public'), { recursive: true, force: true });
}

console.log('Standalone runtime prepared.');

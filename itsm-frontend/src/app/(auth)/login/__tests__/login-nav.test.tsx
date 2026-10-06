import { readFileSync } from 'fs';
import { resolve } from 'path';

describe('login page router.push safety', () => {
  it('wraps router.push(redirectPath) in try/catch', () => {
    const pagePath = resolve(__dirname, '../page.tsx');
    const src = readFileSync(pagePath, 'utf-8');
    // The login page must wrap the post-login redirect in try/catch so a navigation
    // failure (e.g. dev-tools Offline) doesn't leave the user stuck on the login page.
    expect(src).toMatch(/try\s*{\s*router\.push\(redirectPath\)/);
  });
});

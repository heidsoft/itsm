import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import type { Credentials } from './types.js';

const CREDENTIALS_DIR = path.join(os.homedir(), '.itsm');
const CREDENTIALS_FILE = path.join(CREDENTIALS_DIR, 'credentials');

export { type Credentials };

export function saveCredentials(cred: Credentials): void {
  if (!fs.existsSync(CREDENTIALS_DIR)) {
    fs.mkdirSync(CREDENTIALS_DIR, { recursive: true });
  }
  fs.writeFileSync(CREDENTIALS_FILE, JSON.stringify(cred, null, 2), 'utf-8');
}

export function loadCredentials(): Credentials | null {
  try {
    if (!fs.existsSync(CREDENTIALS_FILE)) return null;
    const raw = fs.readFileSync(CREDENTIALS_FILE, 'utf-8');
    return JSON.parse(raw) as Credentials;
  } catch {
    return null;
  }
}

export function clearCredentials(): void {
  try {
    if (fs.existsSync(CREDENTIALS_FILE)) {
      fs.unlinkSync(CREDENTIALS_FILE);
    }
  } catch {
    // ignore
  }
}

export function isLoggedIn(): boolean {
  const cred = loadCredentials();
  if (!cred) return false;
  if (!cred.token) return false;
  if (cred.expiresAt) {
    return new Date(cred.expiresAt) > new Date();
  }
  return true;
}

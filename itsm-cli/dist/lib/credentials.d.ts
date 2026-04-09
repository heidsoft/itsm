import type { Credentials } from './types.js';
export { type Credentials };
export declare function saveCredentials(cred: Credentials): void;
export declare function loadCredentials(): Credentials | null;
export declare function clearCredentials(): void;
export declare function isLoggedIn(): boolean;

import type { CommandSpec } from './spec.js';
export declare function registry(): CommandSpec[];
export declare function findCommandByTokens(tokens: string[]): CommandSpec | undefined;
/** All domains with their verbs, for capability discovery output. */
export declare function domains(): Array<{
    domain: string;
    verbs: string[];
}>;

import type { FlowScenario } from '../types.js';
/**
 * Ticket lifecycle — the thinnest end-to-end proof that the generic work-item
 * path works: create, assign, comment, resolve, close, and a closed ticket
 * refuses further transitions.
 *
 * The terminal-state step is deliberately negative. A product that lets you
 * resolve a closed ticket silently corrupts SLA and audit reporting, and that
 * class of bug never shows up in happy-path testing.
 */
export declare const ticketLifecycle: FlowScenario;

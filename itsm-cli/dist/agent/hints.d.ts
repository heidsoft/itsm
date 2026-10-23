/**
 * Turning a classified failure into a next step.
 *
 * Shared by the headless runner (exit codes + JSON envelope) and the MCP server
 * (tool result with `isError: true`): a `forbidden` must not read the same as a
 * `not_found`, and neither should send the caller into a retry loop. The hint is
 * advisory text for the model, never a control-flow decision.
 */
import type { CommandSpec } from './spec.js';
import type { classifyError } from '../lib/errors.js';
type Classified = ReturnType<typeof classifyError>;
export declare function addFailureHint(err: Classified, spec: CommandSpec): void;
export {};

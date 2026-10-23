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

export function addFailureHint(err: Classified, spec: CommandSpec): void {
  const agentError = err.agentError;
  if (agentError.hint) return;
  switch (agentError.kind) {
    case 'network':
      agentError.hint =
        'Backend unreachable. Confirm ITSM_BASE_URL / --base-url and that the service is up (`itsm --json doctor`). Transport failures are retryable with backoff; writes are not auto-retried to avoid duplicates.';
      break;
    case 'forbidden':
      agentError.hint = `The token is valid but lacks the required permission for ${spec.name}. Ask an administrator for the matching resource:action grant; do not retry with the same identity.`;
      break;
    case 'not_found':
      agentError.hint = `The target does not exist or is outside the caller's tenant. Verify the id with a list command (e.g. \`itsm --json ${spec.name.split('.')[0]} list\`) before retrying.`;
      break;
    case 'conflict':
      agentError.hint = 'State or version conflict: re-read the resource, re-apply the change against the current version, then retry. Do not blindly repeat the same write.';
      break;
    case 'validation':
      agentError.hint = `The backend rejected the payload. Check required fields with \`itsm --json ${spec.name} --help\`.`;
      break;
    case 'unavailable':
      agentError.hint = 'Backend or a dependency is unavailable/degraded. Retry with backoff; surface it as an operational incident if it persists.';
      break;
    case 'not_authenticated':
      // auto-login specifics are filled in by the caller before this point
      break;
    default:
      break;
  }
}

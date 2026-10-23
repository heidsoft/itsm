/**
 * Business-flow scenario framework.
 *
 * A scenario is an executable acceptance test for one ITIL flow, written in
 * TypeScript so it reuses the same ApiClient (and therefore the same endpoint
 * contracts) that the CLI itself uses. An agent runs `itsm flow run <name>` and
 * gets a structured verdict: which step broke, what was expected, what the
 * backend actually returned, and how long each call took.
 *
 * Two deliberate design choices:
 *
 *   · Assertions are declarative (`expect: [{path:'status', eq:'resolved'}]`),
 *     not ad-hoc booleans. That is what makes evidence comparable across runs
 *     and lets the report say "expected X, got Y" without re-reading source.
 *   · Negative paths are first-class (`expectError`). A product test that only
 *     proves happy paths cannot detect the failures this repo keeps fixing —
 *     illegal transitions that silently succeed, RBAC holes, tenant leaks.
 */
export {};

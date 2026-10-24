import type { CommandSpec } from './spec.js';
import { deriveRisk } from './spec.js';
import { infraCommands } from './commands/infra.js';
import { authCommands } from './commands/auth.js';
import { ticketCommands } from './commands/ticket.js';
import { incidentCommands } from './commands/incident.js';
import { problemCommands } from './commands/problem.js';
import { changeCommands } from './commands/change.js';
import { releaseCommands } from './commands/release.js';
import { serviceCatalogCommands } from './commands/service-catalog.js';
import { cmdbCommands } from './commands/cmdb.js';
import { cloudCommands } from './commands/cloud.js';
import { knowledgeCommands } from './commands/knowledge.js';
import { slaCommands } from './commands/sla.js';
import { workflowCommands } from './commands/workflow.js';
import { notificationCommands } from './commands/notification.js';
import { connectorCommands } from './commands/connector.js';
import { adminCommands } from './commands/admin.js';
import { doctorCommands } from './commands/doctor.js';
import { flowCommands } from './commands/flow.js';

/**
 * Single source of truth for the agent command surface.
 * `itsm commands --json`, help text, and the route-drift check all derive
 * from this registry; duplicate names are treated as a build-time programming
 * error in dev and fail fast in the runner.
 */

function buildRegistry(): CommandSpec[] {
  const specs = [
    ...infraCommands(),
    ...authCommands(),
    ...ticketCommands(),
    ...incidentCommands(),
    ...problemCommands(),
    ...changeCommands(),
    ...releaseCommands(),
    ...serviceCatalogCommands(),
    ...cmdbCommands(),
    ...cloudCommands(),
    ...knowledgeCommands(),
    ...slaCommands(),
    ...workflowCommands(),
    ...notificationCommands(),
    ...connectorCommands(),
    ...adminCommands(),
    ...doctorCommands(),
    ...flowCommands(),
  ];

  const seen = new Set<string>();
  for (const spec of specs) {
    if (seen.has(spec.name)) {
      throw new Error(`Duplicate agent command: ${spec.name}`);
    }
    seen.add(spec.name);
    if (!spec.risk) spec.risk = deriveRisk(spec.name.split('.')[(spec.name.split('.').length - 1)]);
  }
  return specs;
}

let cached: CommandSpec[] | null = null;

export function registry(): CommandSpec[] {
  if (!cached) cached = buildRegistry();
  return cached;
}

export function findCommandByTokens(tokens: string[]): CommandSpec | undefined {
  // Longest prefix wins: positional aligns with the dotted command name
  // (`ticket comments add 42` consumes [ticket, comments, add]).
  const max = Math.min(tokens.length, 4);
  for (let take = max; take >= 2; take--) {
    const name = tokens.slice(0, take).join('.');
    const spec = registry().find((s) => s.name === name);
    if (spec) return spec;
  }
  return undefined;
}

/** All domains with their verbs, for capability discovery output. */
export function domains(): Array<{ domain: string; verbs: string[] }> {
  const out = new Map<string, string[]>();
  for (const spec of registry()) {
    const dot = spec.name.indexOf('.');
    const domain = spec.name.slice(0, dot);
    const verb = spec.name.slice(dot + 1);
    if (!out.has(domain)) out.set(domain, []);
    out.get(domain)!.push(verb);
  }
  return [...out.entries()].map(([domain, verbs]) => ({ domain, verbs: verbs.sort() }));
}

/**
 * The MCP tool surface.
 *
 * 205 CLI commands exist; advertising all of them costs more context than the
 * work they enable and makes tool selection worse (the model drowns in near
 * duplicates like `ticket.update` vs `ticket.subtasks.update`). So the surface
 * is two-tier:
 *
 *   Tier 1 — a curated set of hot-path tools with full JSON Schema, so the
 *            common requests (list/get/create/comment/transition, CMDB impact,
 *            knowledge search, SLA stats, my tasks) are one call away.
 *   Tier 2 — `itsm_discover` / `itsm_describe` / `itsm_call`, which reach every
 *            remaining command without bloating `tools/list`.
 *
 * Schemas are derived from the CLI's own command registry (`CommandSpec.input`)
 * rather than restated here, so a new command or a changed flag reaches MCP for
 * free and cannot drift from the CLI contract.
 */
import type { CommandSpec } from '../agent/spec.js';
export interface McpTool {
    name: string;
    title: string;
    description: string;
    inputSchema: Record<string, unknown>;
    annotations: Record<string, unknown>;
    /** Command executed by this tool, absent for the three meta tools. */
    command?: string;
    /** Arguments implied by the tool's purpose; caller arguments override them. */
    defaults?: Record<string, unknown>;
}
export declare function specIndex(): Map<string, CommandSpec>;
export declare function toolNameFor(command: string): string;
/**
 * Resolve a caller-supplied identifier to a command.
 * Accepts `ticket.list`, `ticket list`, `itsm_ticket_list`, `Ticket.List`.
 */
export declare function resolveCommand(input: string): CommandSpec | undefined;
/** Cheap suggestion for an unknown command: same domain, shared verb prefix. */
export declare function suggestCommand(input: string): string | undefined;
/** Full tool list: curated first (stable order), then the three meta tools. */
export declare function buildTools(): McpTool[];
/** `itsm_discover` payload: compact enough to stay inside a small context budget. */
export declare function discoverTools(query: string | undefined, domain: string | undefined, limit: number): Record<string, unknown>;
/** `itsm_describe` payload: the machine-readable contract for one command. */
export declare function describeTool(command: string): Record<string, unknown>;

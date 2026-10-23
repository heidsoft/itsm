/**
 * MCP resources — read-only context the model can pull without spending a tool
 * call on discovery.
 *
 * Two kinds:
 *   · Live capability map (`itsm://capabilities`), derived from the command
 *     registry: what this instance can do, and through which command.
 *   · Product documentation (`itsm://docs/*`), read from the repository. This is
 *     the "improve the product" half of the connector: an agent that can also
 *     read ROADMAP/CHANGELOG/API reference can propose a change that fits the
 *     existing contract instead of inventing a parallel one.
 *
 * Documentation files are optional at runtime (an installed CLI may sit outside
 * the repo). A missing file is reported as a normal resource error, never a
 * crash.
 */
export interface McpResource {
    uri: string;
    name: string;
    description: string;
    mimeType: string;
}
export interface ResourceContent {
    uri: string;
    mimeType: string;
    text: string;
}
export declare function listResources(): McpResource[];
export declare function readResource(uri: string): ResourceContent;

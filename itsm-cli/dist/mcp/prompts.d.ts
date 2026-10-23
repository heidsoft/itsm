/**
 * MCP prompts — reusable, parameterised operating procedures.
 *
 * A prompt is the counterpart of a tool: the tool performs one call, the prompt
 * describes the *sequence* an operator would follow, including where the data
 * lives and what must not be assumed. They exist because the most common ITSM
 * failures are procedural, not technical: closing a ticket without its SLA
 * clock, approving a change nobody assessed, writing a postmortem from memory
 * instead of from correlated events.
 */
export interface McpPrompt {
    name: string;
    title: string;
    description: string;
    arguments: Array<{
        name: string;
        description: string;
        required?: boolean;
    }>;
}
export interface PromptResult {
    description: string;
    messages: Array<{
        role: 'user';
        content: {
            type: 'text';
            text: string;
        };
    }>;
}
export declare const PROMPTS: McpPrompt[];
export declare function getPrompt(name: string, args: Record<string, string> | undefined): PromptResult;

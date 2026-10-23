import type { FlowScenario } from '../types.js';
/**
 * Incident lifecycle.
 *
 * Note on contract: the acknowledge/resolve/close endpoints return
 * `data: null`, so every state assertion re-reads the incident. That is the
 * honest way to verify persistence, and it keeps this scenario passing even if
 * the backend later starts returning the updated object.
 */
export declare const incidentLifecycle: FlowScenario;

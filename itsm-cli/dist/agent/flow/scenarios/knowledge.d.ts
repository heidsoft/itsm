import type { FlowScenario } from '../types.js';
/**
 * Knowledge base: draft → publish → retrievable by search → unpublish.
 *
 * Search is the step that matters. A drafted article leaking into retrieval is a
 * permission/visibility defect, and a published article failing to appear is a
 * knowledge-reuse outage. Both are invisible if you only test CRUD.
 */
export declare const knowledgeLifecycle: FlowScenario;

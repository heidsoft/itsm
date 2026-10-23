import type { FlowScenario } from '../types.js';
/**
 * CMDB: two CIs, one directed dependency, then topology from the source and
 * impact analysis from the target.
 *
 * The pair matters. Topology and impact traverse in opposite directions; a CMDB
 * that only implements one direction looks fine in the UI until an outage
 * question ("what depends on this database?") gets the wrong answer.
 */
export declare const cmdbTopology: FlowScenario;

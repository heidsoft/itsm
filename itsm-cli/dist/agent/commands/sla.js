import { cmd } from '../spec.js';
import { requirePositional, withPagination, optionalString } from '../helpers.js';
import { apiClient } from '../../lib/api-client.js';
import { usageError } from '../../lib/errors.js';
/** SLA: definitions, metrics, violations, monitoring, templates. */
export function slaCommands() {
    return [
        cmd('sla.stats', 'SLA dashboard totals', 'itsm sla stats', ['GET /api/v1/sla/stats'], async () => apiClient.getSLAStats()),
        cmd('sla.list', 'List SLA definitions', 'itsm sla list', ['GET /api/v1/sla/definitions'], async (ctx) => apiClient.listSLADefinitions(withPagination(ctx.args))),
        cmd('sla.get', 'SLA definition detail', 'itsm sla get <id>', ['GET /api/v1/sla/definitions/:id'], async (ctx) => apiClient.getSLADefinition(requirePositional(ctx.args, 0, 'definitionId'))),
        cmd('sla.metrics', 'SLA metrics', 'itsm sla metrics', ['GET /api/v1/sla/metrics'], async () => apiClient.getSLAMetrics()),
        cmd('sla.violations', 'SLA violations list (paged)', 'itsm sla violations', ['GET /api/v1/sla/violations'], async (ctx) => apiClient.getSLAViolations(withPagination(ctx.args))),
        cmd('sla.monitoring', 'Live SLA monitoring snapshot', 'itsm sla monitoring', ['POST /api/v1/sla/monitoring'], async () => apiClient.getSLAMonitoring()),
        cmd('sla.compliance-report', 'Window compliance report (startDate/endDate REQUIRED by the backend)', 'itsm sla compliance-report --startDate <d> --endDate <d>', ['GET /api/v1/sla/compliance-report'], async (ctx) => {
            // handlers/sla/handler.go:640 rejects a missing range with a 400 rather
            // than defaulting, so validate locally instead of burning a round trip.
            const startDate = optionalString(ctx.args.flags, 'startDate');
            const endDate = optionalString(ctx.args.flags, 'endDate');
            if (!startDate || !endDate)
                throw usageError('Missing --startDate and/or --endDate (both required, e.g. 2026-01-01T00:00:00Z)');
            return apiClient.getSLAComplianceReport({ startDate, endDate });
        }),
        cmd('sla.performance', 'SLA performance by dimension', 'itsm sla performance', ['GET /api/v1/sla/performance'], async () => apiClient.getSLAPerformance()),
        cmd('sla.alert-history', 'SLA alert history', 'itsm sla alert-history', ['GET /api/v1/sla/alert-history'], async () => apiClient.getSLAAlertHistory()),
        cmd('sla.check-compliance', 'Compliance check for one ticket', 'itsm sla check-compliance <ticketId>', ['POST /api/v1/sla/check-compliance/:ticketId'], async (ctx) => apiClient.checkSLACompliance(requirePositional(ctx.args, 0, 'ticketId'))),
        cmd('sla.templates', 'Out-of-the-box SLA template catalog', 'itsm sla templates', ['GET /api/v1/sla/templates'], async () => apiClient.listSLATemplates()),
        cmd('sla.install-template', 'Install one SLA template into the tenant', 'itsm sla install-template <key>', ['POST /api/v1/sla/templates/:key/install'], async (ctx) => apiClient.installSLATemplate(requirePositional(ctx.args, 0, 'templateKey'))),
    ];
}

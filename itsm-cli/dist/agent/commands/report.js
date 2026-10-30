import { cmd } from '../spec.js';
import { apiClient } from '../../lib/api-client.js';
/**
 * Reports: the catalog plus every report it advertises.
 *
 * The catalog is the single source of truth for "which reports can I open":
 * `cmdb-quality` and `catalog-usage` disappear from it when the CMDB / service
 * catalog domain is not wired, so a command per entry keeps the CLI honest
 * instead of hard-coding a list that can drift from the backend routes.
 *
 * All endpoints require `report:read`; the domain-scoped reports additionally
 * require the domain read permission (`cmdb:read`, `service_catalog:read`).
 */
export function reportCommands() {
    return [
        cmd('report.catalog', 'List reports available in this deployment', 'itsm report catalog', ['GET /api/v1/reports'], async () => apiClient.getReportCatalog()),
        cmd('report.service-quality', 'Aggregated management view (KPI/SLA/incidents/satisfaction/trends)', 'itsm report service-quality', ['GET /api/v1/reports/service-quality'], async () => apiClient.getServiceQualityReport()),
        cmd('report.tickets', 'Ticket statistics report', 'itsm report tickets', ['GET /api/v1/reports/tickets'], async () => apiClient.getTicketReport()),
        cmd('report.incidents', 'Incident distribution report', 'itsm report incidents', ['GET /api/v1/reports/incidents'], async () => apiClient.getIncidentReport()),
        cmd('report.sla', 'SLA report', 'itsm report sla', ['GET /api/v1/reports/sla'], async () => apiClient.getSLAReport()),
        cmd('report.problems', 'Problem trends report', 'itsm report problems', ['GET /api/v1/reports/problems'], async () => apiClient.getProblemReport()),
        cmd('report.changes', 'Change statistics report', 'itsm report changes', ['GET /api/v1/reports/changes'], async () => apiClient.getChangeReport()),
        cmd('report.cmdb-quality', 'CMDB governance/quality report (needs cmdb:read)', 'itsm report cmdb-quality', ['GET /api/v1/reports/cmdb-quality'], async () => apiClient.getCMDBQualityReport()),
        cmd('report.catalog-usage', 'Service catalog usage report (needs service_catalog:read)', 'itsm report catalog-usage', ['GET /api/v1/reports/catalog-usage'], async () => apiClient.getCatalogUsageReport()),
    ];
}

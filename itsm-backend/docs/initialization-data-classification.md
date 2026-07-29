# Production initialization data contract

This contract is normative for platform bootstrap and tenant provisioning.
Initializers may only write P0 or T0 records listed here. D0 and R0 writes are
release-blocking defects.

| Class | Ownership | Initialization behavior |
|---|---|---|
| P0 | Product/platform | Versioned platform manifest; managed fields are reconciled |
| T0 | Product template installed per tenant | Versioned tenant manifest; tenant overrides are preserved |
| D0 | Demo/test | Never reachable from production bootstrap; explicit demo command only |
| R0 | Runtime/customer | Never created, updated, or deleted by initialization |

## Model classification

| Domain | P0 | T0 | D0/R0 — forbidden in production initialization |
|---|---|---|---|
| Platform/tenant | permission definition, endpoint ACL, marketplace metadata, configuration key definition | tenant, tenant template version, tenant installation state | MSP allocation, billing activity, customer/demo tenant |
| Identity/organization | bootstrap-token definition | role, permission grant, menu, optional department/team/group skeleton | ordinary user, password reset token, engineer skill activity |
| Ticket | ticket type/category/tag/template definition | ticket category, tag, template, view, assignment/automation template | ticket, comment, attachment, CC, approval, notification, rating, workflow record |
| Incident/problem/change/release | lifecycle/type definitions | standard change, incident category/rule template | incident, alert/event/metric/escalation, problem, known error, change, CAB/PIR, release |
| Service catalog | catalog/form definition | enabled catalog template and approval template | service request, request approval, provisioning task |
| BPMN/workflow | versioned process definition/deployment manifest | process definition/deployment copy, binding, approval workflow | process/workflow instance, task, variable, execution history, approval decision, audit |
| SLA | SLA and escalation manifest | definition, policy, alert/escalation rule | metric, violation, alert history |
| CMDB | CI/relationship schema manifest | CI type, attribute definition, relationship type, saved-view/discovery-source template | CI, CI history, CI relationship, discovery/import/export result, cloud account/resource |
| Knowledge/AI | prompt/skill/help manifest | allowed prompt/help template | customer article/version/vector, conversation/message, tool invocation, AI audit |
| Connector/marketplace | official item metadata | explicit tenant installation metadata | connector secret/config, callback, sync record, Feishu ticket sync |
| Asset/vendor | product taxonomy only | optional asset/license/vendor/contract templates | asset, license allocation, vendor contract/customer data |
| Portfolio/cloud | type taxonomy only | optional project/application/microservice/cloud-service templates | project/application/customer topology, cloud account/resource/discovery result |
| Survey/notification/audit | notification/survey type definition | preference/survey template | survey response, notification, audit log |

Schemas not explicitly listed as P0/T0 are R0 by default. New Ent schemas must
update this file and declare a stable key before an initializer may write them.

## Stable keys and write allowlist

- Tenant: `code`
- Role: `(tenant_id, code)`
- Permission: `(tenant_id, code)` during compatibility; the target P0 catalog
  uses globally stable `code`
- Role permission: `(tenant_id, role_code, permission_code)`
- Menu: `(tenant_id, source_key)`; current compatibility key is `path`
- Process definition: `(tenant_id, key, version)`
- Process binding: `(tenant_id, business_type, business_sub_type, scenario, priority)`
- SLA definition: `(tenant_id, source_key)`; current compatibility key is `name`
- CI type: `(tenant_id, source_key)`; current compatibility key is `name`

Production initialization currently allows writes only to:

```text
tenants, users (bootstrap administrator only), roles, permissions,
role_permissions, menus, departments, teams, approval_workflows,
process_deployments, process_definitions, process_bindings, sla_definitions,
sla_policies, sla_alert_rules, service_catalogs, ticket_categories, tags,
ticket_views, ci_types, standard_changes, system_configs,
initialization_installations, initialization_runs,
initialization_component_attempts, initialization_managed_records
```

Every write uses a system context with reason. Runtime tables are asserted empty
by `TestSeedAllProductDefaultsDoNotCreateBusinessSamples`.

## Compatibility contract

- Supported database: PostgreSQL for production; SQLite is test-only.
- Active migration stream begins at `007`; historical `001`–`006` are
  documentation-only and are never replayed.
- Current tenant template version: `1.0.0`.
- Application compatibility range: exactly `1.0.x` until versioned manifests
  declare a wider range.
- Unknown schema fingerprints, dirty migrations, and checksum mismatches fail
  closed and require an audited repair; they are never auto-baselined.
- Platform readiness and tenant readiness are separate. A failed tenant is
  isolated; schema/platform-security failures make the whole instance Not Ready.


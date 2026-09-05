// Package incident implements incident management: lifecycle state
// machine (open → in_progress → resolved → closed), assignment,
// escalation and SLA-aware transitions guarded by CAS updates.
package incident

// Package ticket implements the unified ticket vertical slice:
// create/update/lifecycle transitions, templates, subtasks and SLA info.
// All persistence goes through the Repository interface which enforces
// tenant isolation and optimistic locking.
package ticket

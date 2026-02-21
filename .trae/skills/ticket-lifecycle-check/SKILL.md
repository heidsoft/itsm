---
name: "ticket-lifecycle-check"
description: "Verifies the complete lifecycle of tickets including creation, editing, status changes, and deletion. Invoke when user asks to test ticket management or verify CRUD operations."
---

# Ticket Lifecycle Check

This skill runs a comprehensive end-to-end test suite (`tests/e2e/test_tickets_full.py`) to verify the Ticket Management module.

## Scope

The test covers the following user flows:
1.  **Authentication**: Logs in as an administrator.
2.  **Navigation**: Accesses the Ticket Management dashboard.
3.  **Creation**: Creates a new ticket with specific priority and category.
4.  **Verification**: Confirms the ticket appears in the detail view with correct data.
5.  **Editing**: Modifies the ticket title and status.
6.  **Search**: Locates the modified ticket in the list view using search filters.
7.  **Deletion**: Deletes the ticket and verifies it is removed from the list.

## Prerequisites

- Frontend application running on `http://localhost:3000`.
- Backend API (mock or real) responding to ticket requests.
- Python 3 environment with `playwright` and `pytest` installed.

## Usage

The skill automatically:
1.  Checks/Activates the Python virtual environment (`tests/e2e/venv`).
2.  Installs necessary dependencies if missing.
3.  Executes the test script with verbose output.

## Command

```bash
# Manual execution reference
source tests/e2e/venv/bin/activate
pytest -s tests/e2e/test_tickets_full.py
```

## Troubleshooting

- **Login Failed**: Ensure the mock user `admin` / `admin123` is valid.
- **Element Not Found**: The frontend UI might have changed (e.g., class names, labels). Check `test_tickets_full.py` for selector updates.
- **Timeout**: Network latency or backend slowness. Increase timeouts in `expect()` calls.

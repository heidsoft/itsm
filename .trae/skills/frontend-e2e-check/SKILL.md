---
name: "frontend-e2e-check"
description: "Runs automated E2E tests using Playwright to verify login, navigation, and core modules (Tickets, Incidents, SLA). Invoke when user asks to test the UI, verify system health, or check navigation."
---

# Frontend E2E Check

This skill runs the end-to-end (E2E) test suite located in `tests/e2e/test_navigation.py`. It verifies that the frontend application is running, the user can log in, and all main menu items are accessible and load their respective pages correctly.

## Prerequisites

- The frontend application must be running (usually on `http://localhost:3000`).
- Python 3 must be installed.

## Usage

1.  **Environment Setup**:
    - Checks for `tests/e2e/venv`.
    - If missing, creates the virtual environment and installs dependencies: `pytest`, `playwright`, `pytest-playwright`.
    - Runs `playwright install chromium` to ensure the browser is available.

2.  **Run Tests**:
    - Executes `pytest -s tests/e2e/test_navigation.py`.
    - The `-s` flag allows seeing the console output (progress logs).

3.  **Result Analysis**:
    - Reports PASS/FAIL status.
    - If tests fail, checks for screenshots (e.g., `login_failed.png`, `error_*.png`) and informs the user.

## Command

```bash
# Ensure frontend is running (if not already)
# npm run dev --cwd itsm-frontend &

# Run the tests
cd tests/e2e
python3 -m venv venv
source venv/bin/activate
pip install pytest playwright pytest-playwright
playwright install chromium
pytest -s test_navigation.py
```

## Scenarios

- **Regression Testing**: Verify that recent changes didn't break core navigation.
- **Health Check**: Confirm the system is operational after startup.
- **Smoke Test**: Quick verification of login and main modules.

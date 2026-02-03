# Frontend API Integration Rules

## Core Principle
**Strictly prohibit using mock data in the frontend code. All data must be fetched from the real backend API.**

## Guidelines

1.  **No Mock Data Fallback**:
    - Do not include mock data objects or arrays in the source code as a fallback mechanism.
    - If an API call fails, do not catch the error and render mock data.
    - Instead, handle the error gracefully:
        - Show an error message (toast/notification).
        - Display an empty state or a specific error UI component.
        - Log the error for debugging.

2.  **API Endpoint Accuracy**:
    - Ensure all API calls point to the correct backend endpoints (typically `/api/v1/...`).
    - Verify endpoint paths, HTTP methods (GET, POST, PUT, DELETE), and request/response payloads against the backend API definition.

3.  **Backend API Alignment (404 Handling)**:
    - If a frontend API call returns a **404 Not Found**:
        1.  **Verify Frontend Request**: Check if the URL path, HTTP method, and parameters in the frontend code match the expected API design.
        2.  **Verify Backend Implementation**:
            -   Check `router.go` or route registration files to ensure the route exists and is registered correctly.
            -   Verify that the controller and handler functions are implemented and linked.
        3.  **Action**:
            -   If the backend is missing the endpoint, **implement the endpoint in the backend** to match the frontend requirement. Do NOT revert the frontend to use mock data.
            -   If the path is mismatched, correct the frontend path or alias the backend route to match.

4.  **Data Initialization**:
    - Initialize component state with empty values (e.g., `[]` for arrays, `null` or default objects for details) rather than mock data.
    - Use loading states (`isLoading`) to indicate data fetching progress.

5.  **Scope**:
    - This rule applies to all functional modules including but not limited to:
        - Ticket Management
        - SLA Monitoring
        - Workflow/BPMN
        - Analytics & Reporting
        - User & Role Management
        - Service Catalog

## Implementation Checklist
- [ ] Remove any existing `mock-data.ts` or similar files.
- [ ] Search for and remove `mockData`, `DEMO_DATA`, etc. in component files.
- [ ] Review `catch` blocks in API service calls to ensure they don't load mock data.
- [ ] Verify that `useQuery` or `useEffect` hooks fetch data from the configured API client.
- [ ] If 404 occurs, verify backend `router.go` and implement missing endpoints instead of using mocks.

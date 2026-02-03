---
name: "frontend-audit-fix"
description: "Audits frontend for 'No Mock' rule, fixes API consistency (camel/snake case), and resolves Jest/Antd test issues. Invoke for frontend QA tasks or build validation."
---

# Frontend Audit & Fix Workflow

Use this workflow to validate and fix frontend projects, specifically Next.js + Ant Design.

## 1. Enforce "No Mock Data" Rule
- **Check**: Verify `http-client.ts` uses real API endpoints.
- **Action**: Remove any `mockData` objects in components. Ensure 404s are handled by implementing backend routes, not falling back to mocks.

## 2. API Consistency Check
- **Check**: Ensure frontend interfaces (camelCase) match backend API responses (often snake_case) via the HTTP client interceptors.
- **Action**:
    - If `http-client` converts snake_case -> camelCase, ensure TypeScript interfaces use **camelCase**.
    - If data loading fails, check if the component expects camelCase but receives snakeCase (or vice versa) and if the conversion logic is active.

## 3. Test Health (Jest + Antd)
- **Common Issues**:
    - `TypeError: window.matchMedia is not a function`: Mock in `jest.setup.js`.
    - `TypeError: Form.useForm is not a function`: Mock `antd` in test files or globally.
    - `Element type is invalid`: Check if `antd` mock returns all used components (Row, Col, Card, etc.).
- **Fixing Strategies**:
    - Use `screen.getAllByTestId` if multiple elements exist (e.g., desktop/mobile layouts).
    - Wrap state updates in `waitFor(() => expect(...))`.
    - Mock icons (`lucide-react` or `@ant-design/icons`) if they cause rendering issues.

## 4. Verification
- Run `npm run test:ci` (or `npm test`) to verify logic.
- Run `npm run build` to verify types and production build.

---
name: "frontend-maintenance"
description: "Diagnoses and fixes common frontend issues including Ant Design deprecations and API failures. Invoke when user reports console warnings, UI crashes, or requests project maintenance."
---

# Frontend Maintenance & Resilience Skill

This skill encapsulates strategies for maintaining a healthy frontend codebase, focusing on library upgrades (specifically Ant Design) and API error handling.

## Capabilities

### 1. Ant Design Deprecation Fixes

Detects and repairs deprecated component props to ensure compatibility with Ant Design v5/v6.

**Common Replacements:**

- **Statistic**: `valueStyle` -> `styles={{ content: { ... } }}`
  - *Caution*: When batch replacing, ensure closing brackets are balanced: `}}` -> `}}}`.
- **Collapse**: `children` (nested Panels) -> `items` (prop with array of objects).
- **Alert**: `message` -> `description` (when used for detailed content).
  - *Note*: Use `message` only for the title, and `description` for the main content.
- **Static Methods**: `message`/`notification`/`modal` cannot consume Context (Theme/I18n).
  - *Fix*: Use `App.useApp()` hook: `const { message } = App.useApp();`.
- **Space**: `direction` -> `orientation`
- **Button**: `iconPosition` -> `iconPlacement`
- **Card**: `bordered={false}` -> `variant="borderless"`, `bordered={true}` -> `variant="outlined"`
  - *Note*: Requires handling type definitions (e.g., `Omit<CardProps, 'variant'>`) if extending props.
- **Drawer**: `width` -> `size` (Use `size="large"` or `style={{ width: ... }}` for custom widths).
- **Dependencies**: Remove `@ant-design/v5-patch-for-react-19` if on Antd v6+.

### 2. API Resilience & Fallback

Ensures the application remains usable even when backend services fail.

**Pattern:**

- Wrap API calls in `try-catch` blocks.
- **Never** let a 500 error crash the page.
- **Fallback Strategy**:
  1. Return a structured "empty" or "safe" object matches the expected type.
  2. Log the error to console/monitoring with a clear prefix.
  3. Optionally use cached or mock data if critical for UX.
  4. **Auth Loop Prevention**: Check `window.location.pathname` before redirecting to `/login` on 401.

**Example Implementation:**

```typescript
async getData() {
  try {
    return await api.fetchData();
  } catch (error) {
    console.error('API Failed, using fallback:', error);
    return { 
      items: [], 
      total: 0, 
      status: 'fallback' 
    }; // Returns safe default matching interface
  }
}
```

### 3. Diagnostic Toolkit (Grep Patterns)

Use these commands to quickly identify issues:

- **Find Static Method Misuse**:

  ```bash
  grep -r "import {.*message.*} from 'antd'" src/
  grep -r "import {.*Modal.*} from 'antd'" src/
  ```

- **Find Deprecated Statistic Styles**:

  ```bash
  grep -r "valueStyle" src/
  ```

- **Find Deprecated Alert Props**:

  ```bash
  grep -r "Alert" src/ | grep "message="
  ```

### 4. Workflow

1. **Audit**: Check browser console for Red (Errors) and Yellow (Warnings) logs.
2. **Port Check**: If `net::ERR_CONNECTION_REFUSED`, check ports: `lsof -i :3000` and kill zombies.
3. **Search**: Use Diagnostic Toolkit to locate affected files.
4. **Fix**: Apply code changes (prop renaming, try-catch wrappers, useApp hook).
5. **Verify**: Restart dev server (`npm run dev`) and check console again.

## When to Apply

- When the console is flooded with "Warning: [antd: Component] prop is deprecated".
- When API endpoints are unstable (500/502) causing white screens.
- During routine code quality audits.

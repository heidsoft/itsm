---
name: component-standardization
description: Refactors UI components to use Ant Design + Tailwind CSS standards. Enforces specific styling patterns for Cards, Typography, and Layouts to ensure consistency. Invoke when standardizing UI or refactoring legacy styles.
---

# Component Standardization Skill

This skill guides the process of replacing custom, "reinvented" UI components and legacy styles with standard library implementations (Ant Design + Tailwind CSS) to ensure consistency and maintainability.

## Goal
To eliminate redundant custom styles and components by leveraging a standardized design system based on Ant Design components styled with Tailwind CSS utility classes.

## Standardization Rules (The "Gold Standard")

### 1. Card Component
**Pattern**: Use Ant Design `Card` with specific Tailwind classes for borders and shadows.
- **DO**: `<Card className="rounded-lg shadow-sm border border-gray-200" bordered={false}>`
- **DON'T**: `<Card bordered={false} style={{...}}>` or custom `div` wrappers.
- **Rationale**: Ensures consistent `gray-200` borders and `shadow-sm` depth across the app.

### 2. Layout & Spacing
**Pattern**: Use Tailwind utility classes for layout and spacing.
- **DO**: `<div className="p-6 mb-6">`, `<div className="flex items-center gap-4">`
- **DON'T**: `style={{ padding: 24, marginBottom: 16 }}`, `<Space style={{...}}>`
- **Rationale**: Reduces runtime JS overhead and enforces the design system's spacing scale.

### 3. Typography
**Pattern**: Use native HTML tags with Tailwind classes.
- **DO**: `<h2 className="text-2xl font-bold text-gray-900">`, `<p className="text-gray-500">`
- **DON'T**: `<Typography.Title level={2}>`, `<Typography.Text type="secondary">` (unless strictly necessary for complex text features).
- **Rationale**: Native tags are semantically correct and Tailwind classes offer more flexible styling control without component overhead.

### 4. Icons
**Pattern**: Use `lucide-react` icons (preferred) or Ant Icons styled with Tailwind.
- **DO**: `<Bell className="w-4 h-4 text-gray-500" />`
- **DON'T**: `<BellOutlined style={{ fontSize: 16, color: '#666' }} />`
- **Rationale**: Consistent sizing and coloring via utility classes.

### 5. Theme & Colors
**Pattern**: Use Tailwind color palette.
- **DO**: `text-green-500`, `bg-blue-50`, `border-red-200`
- **DON'T**: `theme.useToken()` (e.g., `token.colorSuccess`) for static elements.
- **Rationale**: Decouples components from runtime theme engine for static styles, improving performance and readability.

### 6. Inputs & Controls
**Pattern**: Use Ant Design form controls with Tailwind width classes.
- **DO**: `<Input className="w-64" />`, `<Select className="w-32" />`
- **DON'T**: `<Input style={{ width: 200 }} />`

## Workflow

### 1. Audit & Identify
- specific file or component to refactor.
- Look for `style={{...}}` props, `token.*` usage, and legacy CSS classes.

### 2. Refactor
- **Replace Container**: Wrap content in the standard `Card` pattern.
- **Replace Layouts**: Convert `style` props to Tailwind classes (padding, margin, flex, grid).
- **Replace Text**: Convert `Typography` components to HTML tags + Tailwind.
- **Replace Colors**: Map hex codes or tokens to the nearest Tailwind color (e.g., `#1890ff` -> `blue-600`).

#### For Extended Wrappers (e.g., `AuthButton`)
- **Action**: Rewrite as a lightweight wrapper.
- **Implementation**:
  - Import the standard component.
  - Define an interface that extends the standard props (using `Omit` if necessary).
  - Map custom props (e.g., `variant="outline"`) to standard props (e.g., `type="default"`, `className="border-primary"`).
  - **Ant Design v5 Note**: Map `bodyStyle` / `headStyle` to `styles={{ body: ..., header: ... }}`.
  - Pass through all other props.
  - **Example**:
    ```typescript
    import { Button, type ButtonProps } from 'antd';
    // ... logic to map custom props to Antd props ...
    return <Button {...mappedProps} {...rest} />;
    ```

### 4. Cleanup
- **Delete**: Remove the source files of the original custom implementations.
- **Update Exports**: Clean up `index.ts` files to remove dead exports.
- **Grep Check**: Search the codebase for lingering references to deleted components (e.g., `grep -r "EnhancedCard" src/`).

### 5. Verification
- **Type Check**: Run `npx tsc --noEmit` to catch broken references immediately.
- **Build**: Run `npm run build` to ensure production build stability.
- **Visual Check**: Verify key pages (e.g., Login, Dashboard) to ensure styles remain consistent.

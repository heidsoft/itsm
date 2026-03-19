---
name: "route-and-menu-config"
description: "Guide for ITSM route registration and menu configuration. Covers backend controller routing, frontend sidebar navigation, database menu data, and common routing issues. Invoke when adding new modules, fixing 404 errors, or configuring menu entries."
---

# Route Registration and Menu Configuration Guide

This skill encapsulates the complete workflow for adding new modules to the ITSM system, covering backend routing, frontend menu configuration, and database menu data management.

## Common Issues & Solutions

| Issue | Symptom | Root Cause | Solution |
|-------|---------|------------|----------|
| **Controller exists but API not accessible** | 404 on `/api/v1/xxx` | Controller created but not registered in `router.go` | Add route registration in `router/router.go` |
| **Sidebar menu item not visible** | Menu entry missing | Using dynamic menu from DB but entry not in DB | Add menu to `seed_data.sql` + execute SQL |
| **Menu click leads to 404** | Route mismatch | Sidebar path doesn't match actual page path | Verify actual page location and update Sidebar |
| **Icon not showing** | Missing icon | Icon not imported or not in icon map | Add import + add to `iconMap` in Sidebar |

## Development Workflow

### 1. Backend Route Registration

When adding a new module, follow this checklist:

1. **Create Controller** (if not exists):
   - Location: `controller/<module>_controller.go`
   - Pattern: Use factory function `New<Module>Controller(client)`

2. **Add Route to Router** (`router/router.go`):
   ```go
   // 1. Add controller field to RouterConfig struct
   ProjectController *controller.ProjectController

   // 2. Register routes in SetupRoutes function
   if config.ProjectController != nil {
       projects := tenant.(*gin.RouterGroup).Group("/projects")
       {
           projects.GET("", config.ProjectController.ListProjects)
           projects.POST("", config.ProjectController.CreateProject)
           projects.GET("/:id", config.ProjectController.GetProject)
           projects.PUT("/:id", config.ProjectController.UpdateProject)
           projects.DELETE("/:id", config.ProjectController.DeleteProject)
       }
   }
   ```

3. **Verify Service Methods Exist**: Check if service layer has required methods (GetProject, ListProjects, etc.)

### 2. Frontend Menu Configuration

The ITSM system uses **dynamic menus from database** with a **static fallback** in Sidebar.

#### Option A: Database Menu (Recommended for production)

1. **Add to seed SQL** (`config/seed/seed_data.sql`):
   ```sql
   -- 主菜单
   INSERT INTO menus (name, path, icon, permission_code, sort_order, tenant_id, is_visible, is_enabled, description)
   VALUES ('AI助手', '/ai/chat', 'Bot', 'ai:view', 14, 1, true, true, 'AI智能助手')
   ON CONFLICT DO NOTHING;
   ```

2. **Execute SQL**:
   ```bash
   psql -h localhost -U dev -d itsm -f config/seed/seed_data.sql
   # Or manually:
   psql -h localhost -U dev -d itsm -c "INSERT INTO menus ..."
   ```

3. **Menu Table Schema**:
   ```
   menus:
   - id (bigint, auto-generated)
   - name (varchar, required)
   - path (varchar, required)
   - icon (varchar, optional)
   - permission_code (varchar, optional)
   - sort_order (bigint, required)
   - tenant_id (bigint, required)
   - is_visible (boolean, default true)
   - is_enabled (boolean, default true)
   - description (varchar, optional)
   - parent_id (bigint, optional, for submenus)
   ```

#### Option B: Static Fallback (Sidebar.tsx)

For development/testing or as fallback:

1. **Add Icon Import**:
   ```tsx
   import { Bot, LayoutDashboard, ... } from 'lucide-react';
   ```

2. **Add Menu Item**:
   ```tsx
   // In getMenuConfig() function
   {
     key: '/ai/chat',
     icon: <Bot style={iconStyle} />,
     label: 'AI助手',
     path: '/ai/chat',
     permission: 'ai:view',
     description: 'AI智能助手',
   },
   ```

3. **Add to Icon Map** (for dynamic menu):
   ```tsx
   const iconMap = {
     // ... existing icons
     Bot: <Bot style={iconStyle} />,
   };
   ```

### 3. Route Path Matching

**Critical**: Sidebar path must match actual page location.

```tsx
// Sidebar.tsx - WRONG
{ key: '/admin/departments', path: '/admin/departments' }

// Correct - matches actual page at /app/(main)/enterprise/departments/page.tsx
{ key: '/enterprise/departments', path: '/enterprise/departments' }
```

**Finding actual page path**:
```bash
# Find page location
find . -path "*/app/*" -name "page.tsx" | xargs grep -l "departments"
# Output: src/app/(main)/enterprise/departments/page.tsx
```

## Debugging Commands

### Backend
```bash
# Test API endpoint
curl http://localhost:8080/api/v1/projects

# Check if route is registered (look for route in router.go)
grep -n "projects" router/router.go
```

### Frontend
```bash
# Check current routes
grep -r "path:" src/app/\(main\)/ --include="*.tsx" | head -20

# Find page file by route
find src/app -name "page.tsx" | xargs grep -l "department" 2>/dev/null
```

### Database
```bash
# List all menus
psql -h localhost -U dev -d itsm -c "SELECT id, name, path, icon FROM menus ORDER BY id;"

# Check specific menu
psql -h localhost -U dev -d itsm -c "SELECT * FROM menus WHERE path LIKE '%ai%';"

# Add new menu
psql -h localhost -U dev -d itsm -c "INSERT INTO menus (name, path, icon, permission_code, sort_order, tenant_id, is_visible, is_enabled, description) VALUES ('AI助手', '/ai/chat', 'Bot', 'ai:view', 26, 1, true, true, 'AI智能助手');"

# Update menu path
psql -h localhost -U dev -d itsm -c "UPDATE menus SET path = '/enterprise/departments' WHERE path = '/admin/departments';"
```

## Complete Example: Adding AI Chat Module

### Step 1: Backend
- Verify `controller/project_controller.go` exists (or create)
- Check service has required methods
- Add routes in `router/router.go`

### Step 2: Frontend Page
- Verify page exists at `src/app/(main)/ai/chat/page.tsx`

### Step 3: Database Menu
```sql
-- Add menu
INSERT INTO menus (name, path, icon, permission_code, sort_order, tenant_id, is_visible, is_enabled, description)
SELECT 'AI助手', '/ai/chat', 'Bot', 'ai:view', 14, id, true, true, 'AI智能助手'
FROM tenants WHERE code = 'default'
ON CONFLICT DO NOTHING;
```

### Step 4: Verify
```bash
# Backend
curl http://localhost:8080/api/v1/projects

# Frontend - restart and check sidebar
npm run dev
```

## Key Files Reference

| File | Purpose |
|------|---------|
| `router/router.go` | Route registration |
| `controller/*_controller.go` | HTTP handlers |
| `service/*_service.go` | Business logic |
| `src/components/layout/Sidebar.tsx` | Menu configuration |
| `config/seed/seed_data.sql` | Database seed data |
| `ent/schema/menu.go` | Menu entity schema |

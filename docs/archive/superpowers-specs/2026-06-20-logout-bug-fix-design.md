# Logout Bug Fix Design

**Date:** 2026-06-20
**Scope:** Fix — Header logout not clearing httpOnly cookies

## Problem

`Header.tsx:143-146` defines `handleLogout` as:

```ts
const handleLogout = () => {
  logout(); // only clears Zustand store (local state)
  router.push('/login');
};
```

This clears the frontend store but **never calls the backend** `POST /api/v1/auth/logout`, so the httpOnly `access_token` and `refresh_token` cookies remain valid on the server. A user who clicks "logout" is redirected to `/login` but their session cookies are still active.

## Current Architecture

| Layer | File | Behavior |
|-------|------|----------|
| UI trigger | `UserMenuDropdown.tsx` | Calls `onLogout` prop on "退出登录" click |
| Header handler | `Header.tsx` | Calls `logout()` from store only — **the bug** |
| Store | `auth-store.ts` | `logout()` clears local state + localStorage |
| Auth service | `auth-service.ts` | `AuthService.logout()` calls backend + store — **correct but unused by Header** |
| Backend | `auth_controller.go` | Revokes tokens, clears httpOnly cookies |

## Fix

**Single change in `Header.tsx`:**

1. Import `AuthService` from `@/lib/services/auth-service`
2. Change `handleLogout` to call `AuthService.logout()` which:
   - Calls `POST /api/v1/auth/logout` (backend clears httpOnly cookies)
   - Calls `useAuthStore.getState().logout()` (clears local state)
3. Keep `router.push('/login')` after the logout call

### Fixed code:

```ts
import { AuthService } from '@/lib/services/auth-service';

const handleLogout = async () => {
  AuthService.logout(); // calls backend + clears store (fire-and-forget)
  router.push('/login');
};
```

Note: `AuthService.logout()` uses fire-and-forget for the backend call (`.catch(() => {})`), so it doesn't need to be awaited — the redirect happens immediately regardless of network outcome.

## Files Changed

| File | Change |
|------|--------|
| `src/components/layout/header/Header.tsx` | Add `AuthService` import, update `handleLogout` |

## Verification

1. Click user menu → "退出登录"
2. Confirm redirect to `/login`
3. Verify `access_token` and `refresh_token` cookies are cleared (DevTools → Application → Cookies)
4. Attempt to access a protected route — should redirect to login

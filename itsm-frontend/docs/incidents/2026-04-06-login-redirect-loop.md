# Incident: Login Redirect Loop Bug

**Date:** 2026-04-06
**Severity:** Critical
**Status:** Resolved

## Summary

Users could successfully log in (backend returned JWT with 200 OK), but upon redirect to `/dashboard`, the app detected the user as "not authenticated" and redirected back to `/login`, creating an infinite redirect loop.

## Root Cause

**File:** `src/lib/auth/token-storage.ts`
**Function:** `getAccessToken()`

The function had incorrect priority order — it checked the httpOnly cookie BEFORE localStorage:

```typescript
// BEFORE (buggy):
export function getAccessToken(): string | null {
  if (hasTokenCookie()) {
    return 'httpOnly';  // ← NOT a valid JWT!
  }
  return null;
}
```

When localStorage was empty but the httpOnly cookie existed (set by backend), it returned the literal string `'httpOnly'`. Since this is not a valid JWT (`'httpOnly'.split('.')` = 1 part, not 3), JWT validation in `AuthService.isAuthenticated()` failed, causing `isAuthenticated()` to return `false`.

## Login Flow (Before Fix)

1. User submits login form → `AuthService.login()` called
2. Backend returns JWT → stored in localStorage as `'access_token'`
3. `router.push('/dashboard')` executed
4. `MainLayout` calls `AuthService.isAuthenticated()`
5. `AuthService.isAuthenticated()` calls `getAccessToken()`
6. `getAccessToken()` returns `'httpOnly'` (from cookie fallback)
7. JWT validation fails: `parts.length !== 3` → returns `false`
8. `MainLayout` redirects to `/login` → loop!

## The Fix

**File:** `src/lib/auth/token-storage.ts`

Changed `getAccessToken()` to check localStorage FIRST (where `AuthService.login()` stores the actual JWT):

```typescript
// AFTER (fixed):
export function getAccessToken(): string | null {
  migrateLegacyAuthStorage();
  // Check localStorage FIRST - that's where the actual token is stored
  const localToken = safeGet(STORAGE_KEYS.ACCESS_TOKEN);
  if (localToken) {
    return localToken;  // Returns the actual JWT
  }
  // Fallback to cookie check
  if (hasTokenCookie()) {
    return 'httpOnly';
  }
  return null;
}
```

## Auth Flow After Fix

1. User submits login form → `AuthService.login()` called
2. Backend returns JWT → stored in localStorage as `'access_token'`
3. `router.push('/dashboard')` executed
4. `MainLayout` calls `AuthService.isAuthenticated()`
5. `AuthService.isAuthenticated()` calls `getAccessToken()`
6. `getAccessToken()` returns actual JWT from localStorage
7. JWT validation succeeds (3 parts) → returns `true`
8. Dashboard renders correctly!

## Related Files

| File | Role |
|------|------|
| `src/lib/auth/token-storage.ts` | **Bug location** - `getAccessToken()` with wrong priority |
| `src/lib/services/auth-service.ts` | Login flow, `AuthService.isAuthenticated()` uses localStorage token |
| `src/app/(main)/layout.tsx` | Main layout auth check using `AuthService.isAuthenticated()` |
| `src/middleware.ts` | Server-side route protection (uses cookies, NOT affected) |
| `src/components/auth/AuthGuard.tsx` | Client-side auth guard using `isAuthenticated()` from token-storage |
| `src/lib/api/http-client.ts` | HTTP client token retrieval (checks localStorage first, worked correctly) |

## Related Commit

- `eaabe03c fix: add permission code alias mapping for menu permissions` (most recent)
- Previous: `f5671a89 fix: change SetCookie to non-httpOnly so frontend JS can read token`

## Verification

To verify the fix:

1. Start backend: `cd itsm-backend && go run main.go`
2. Start frontend: `cd itsm-frontend && npm run dev`
3. Navigate to `http://localhost:3000`
4. Login with `admin` / `admin123`
5. Verify dashboard loads without redirect loop
6. Check browser console for any `SyntaxError: Unexpected token` errors

## Lessons Learned

1. **Token storage priority matters**: localStorage (where frontend stores) must be checked before httpOnly cookie (backend fallback)
2. **Avoid literal string placeholders**: Returning `'httpOnly'` as a token is wrong — it should return `null` or the actual token
3. **Test auth flows end-to-end**: Unit tests for `getAccessToken()` and `isAuthenticated()` would have caught this

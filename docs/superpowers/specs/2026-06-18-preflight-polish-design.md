# Pre-TestFlight Polish — Design Spec

**Date:** 2026-06-18
**Scope:** Seven small fixes required before TestFlight distribution.

---

## 1. Backend — Email Normalization

Apply `strings.ToLower(strings.TrimSpace(email))` to the incoming email in both `Register` and `Login` handlers before any lookup or storage operation. Prevents `User@Example.com` and `user@example.com` being treated as different accounts.

**Files:**
- `netme-backend/internal/handlers/auth.go` — normalize in `Register` (before `GetUserByEmail` call) and `Login` (before `GetUserByEmail` call)

---

## 2. Backend — Register TOCTOU Fix

After `CreateUser` returns an error, inspect the error for Postgres unique constraint violation code `23505` (via `pq.Error`). If matched → return 409 `user_exists`. All other errors → return 500. Eliminates the race window between the pre-check `GetUserByEmail` and the `INSERT`.

**Files:**
- `netme-backend/internal/handlers/auth.go` — wrap `CreateUser` error check in `Register`
- `netme-backend/go.mod` — `github.com/lib/pq` already present (no new dep)

---

## 3. Mobile — Stale Type Cleanup

Remove fields that no longer exist on the backend `User` model:
- `display_name?: string`
- `picture_url?: string`
- `last_login_at?: string`

from both the `AuthResponse.user` type in `authService.ts` and the `User` interface in `AuthContext.tsx`.

**Files:**
- `netme-mobile/src/services/authService.ts`
- `netme-mobile/src/context/AuthContext.tsx`

---

## 4. Mobile — Remove `logoutAllDevices`

Remove the `logoutAllDevices` method from `authService.ts`. It calls `POST /auth/logout-all-devices` which does not exist on the backend. Dead code.

**Files:**
- `netme-mobile/src/services/authService.ts`

---

## 5. Mobile — Remove Hardcoded IP Fallback

Replace the hardcoded local-machine IP fallback `'http://192.168.1.158:8080/v1'` with `'http://localhost:8080/v1'`. The previous value only works on one developer's machine.

**Files:**
- `netme-mobile/src/services/authService.ts`

---

## 6. Mobile — `bootstrapAsync` Token Expiry Check

After restoring tokens from secure storage, decode the JWT access token locally (parse the base64url middle segment, JSON parse, read `exp` claim — no network call). If the token expires within 60 seconds or is already expired:
1. Call `authService.refresh(savedRefreshToken)`
2. On success: use the new tokens and persist them to secure storage
3. On failure: call `clearAuth()`, present unauthenticated state

This eliminates the flash of authenticated UI followed by a 401 on the first API call after the access token expires.

**Files:**
- `netme-mobile/src/context/AuthContext.tsx` — update `bootstrapAsync`

---

## 7. Mobile — `clearAll` Only on Auth Errors

In the Axios 401 interceptor's refresh catch block, only call `secureStorage.clearAll()` when the refresh request itself returned an HTTP error response (i.e. `error.response` exists with a 4xx/5xx status). If it is a network error (`error.response` is undefined — timeout, no connectivity), rethrow without clearing storage so the user remains logged in and the next attempt succeeds when connectivity returns.

**Files:**
- `netme-mobile/src/services/authService.ts` — update the catch block in `setupInterceptors`

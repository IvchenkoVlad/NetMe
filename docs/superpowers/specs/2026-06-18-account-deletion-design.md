# Account Deletion — Design Spec

**Date:** 2026-06-18
**Scope:** Implement hard-delete account deletion required for App Store submission (Apple guideline 5.1.1).

---

## Approach

Hard delete: user row is deleted immediately and permanently. `refresh_tokens` are cascade-deleted via the existing `ON DELETE CASCADE` foreign key. No migration needed. No grace period.

---

## Backend

### Repository

Add `DeleteUser(userID string) error` to the `UserRepo` interface and implement it in `UserRepository`:

```sql
DELETE FROM users WHERE id = $1
```

The existing `ON DELETE CASCADE` on `refresh_tokens.user_id` handles token cleanup automatically.

**Files:**
- `netme-backend/internal/repositories/interfaces.go` — add `DeleteUser` to `UserRepo`
- `netme-backend/internal/repositories/user.go` — implement `DeleteUser`

### Handler

`DELETE /v1/me` (already registered in `app.go`, currently returns 501) calls `h.userRepo.DeleteUser(userID)` and returns:
- `204 No Content` on success
- `500 ErrorResponse` on DB error

**Files:**
- `netme-backend/internal/handlers/users.go` — replace 501 stub with real implementation

---

## Mobile

### Settings Screen

Add a "Delete Account" button to `SettingsScreen` (currently a stub). On tap:
1. Show a native confirmation alert: title "Delete Account", message "This will permanently delete your account and all your data. This cannot be undone.", buttons "Cancel" and "Delete" (destructive style)
2. On confirm: call `DELETE /v1/me` via a new `authService.deleteAccount(accessToken)` method
3. On success: call `clearAuth()`, navigate to the login screen
4. On error: show an alert "Failed to delete account. Please try again."

**Files:**
- `netme-mobile/src/services/authService.ts` — add `deleteAccount()` method
- `netme-mobile/src/screens/SettingsScreen.tsx` — add delete account button and flow (create file if it does not exist)
- `netme-mobile/src/context/AuthContext.tsx` — expose `deleteAccount` or call `authService` directly from the screen

---

## Out of Scope

- Revoking Plaid access tokens (no Plaid integration yet)
- Deleting future financial data (accounts, transactions, budgets) — cascade will be added as those tables are created with `ON DELETE CASCADE`
- Email confirmation before deletion
- Soft delete / grace period / recovery

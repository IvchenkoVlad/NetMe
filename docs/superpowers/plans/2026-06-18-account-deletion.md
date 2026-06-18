# Account Deletion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement hard-delete account deletion (`DELETE /v1/me`) required for Apple App Store submission.

**Architecture:** Backend adds `DeleteUser` to `UserRepo` interface and `UserRepository`, then replaces the 501 stub in `UsersHandler.DeleteMe`. The existing `ON DELETE CASCADE` on `refresh_tokens` cleans up tokens automatically — no migration needed. Mobile adds a SettingsScreen with a confirmation alert and wires it into navigation.

**Tech Stack:** Go 1.25+, Gin, PostgreSQL, React Native 0.81.5, Expo, React Navigation

## Global Constraints

- Working directory for Go commands: `netme-backend/`
- Working directory for TypeScript check: `netme-mobile/`
- `DELETE /v1/me` returns `204 No Content` on success, `500 ErrorResponse` on DB error
- No response body on 204
- Account deletion is permanent and immediate (hard delete)
- Mobile shows native confirmation alert before calling the API

---

## File Map

**Modify (backend):**
- `netme-backend/internal/repositories/interfaces.go` — add `DeleteUser(userID string) error` to `UserRepo`
- `netme-backend/internal/repositories/user.go` — implement `DeleteUser`
- `netme-backend/internal/handlers/users.go` — replace 501 stub with real `DeleteMe`
- `netme-backend/internal/handlers/auth_test.go` — `DeleteUser` already added to mock in preflight-polish plan Task 1

**Create (backend):**
- `netme-backend/internal/handlers/users_test.go` — `TestDeleteMeSuccess`, `TestDeleteMeUnauthenticated`

**Modify (mobile):**
- `netme-mobile/src/navigation/RootNavigator.tsx` — add SettingsScreen to app stack
- `netme-mobile/src/services/authService.ts` — `deleteAccount()` already added in preflight-polish plan Task 2

**Create (mobile):**
- `netme-mobile/src/screens/SettingsScreen.tsx` — delete account button with confirmation

---

### Task 1: Backend — `DeleteUser` + `DeleteMe` handler

**Files:**
- Modify: `netme-backend/internal/repositories/interfaces.go`
- Modify: `netme-backend/internal/repositories/user.go`
- Modify: `netme-backend/internal/handlers/users.go`
- Create: `netme-backend/internal/handlers/users_test.go`

**Interfaces:**
- Produces: `UserRepo.DeleteUser(userID string) error`
- Produces: `DELETE /v1/me` → `204` on success, `500 ErrorResponse` on error

- [ ] **Step 1: Write failing tests**

Create `netme-backend/internal/handlers/users_test.go`:

```go
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/models"
)

func newUsersRouter(userRepo *mockUserRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := handlers.NewUsersHandler(userRepo)
	r := gin.New()
	// Simulate AuthMiddleware setting user_id
	r.GET("/v1/me", func(c *gin.Context) {
		c.Set("user_id", "user-delete-123")
		h.GetMe(c)
	})
	r.DELETE("/v1/me", func(c *gin.Context) {
		c.Set("user_id", "user-delete-123")
		h.DeleteMe(c)
	})
	return r
}

func TestDeleteMeSuccess(t *testing.T) {
	userRepo := newMockUserRepo()
	userRepo.users["delete@example.com"] = &models.User{
		ID:           "user-delete-123",
		Email:        "delete@example.com",
		AuthProvider: "local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	r := newUsersRouter(userRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/v1/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify user is removed from the store
	if _, exists := userRepo.users["delete@example.com"]; exists {
		t.Error("expected user to be deleted from store")
	}
}

func TestDeleteMeUserNotFound(t *testing.T) {
	userRepo := newMockUserRepo() // empty — no user with that ID

	r := newUsersRouter(userRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/v1/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != "delete_error" {
		t.Errorf("expected error 'delete_error', got %q", resp.Error)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd netme-backend && go test ./internal/handlers/... -v -run TestDeleteMe
```

Expected: compilation error — `DeleteUser` not in `UserRepo` interface yet.

- [ ] **Step 3: Add `DeleteUser` to `UserRepo` interface**

In `netme-backend/internal/repositories/interfaces.go`, add to `UserRepo`:

```go
DeleteUser(userID string) error
```

Full updated `UserRepo`:

```go
type UserRepo interface {
	CreateUser(email, passwordHash string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateLastLogin(userID string) error
	FindOrCreateGoogleUser(googleID, email string) (*models.User, error)
	DeleteUser(userID string) error
}
```

- [ ] **Step 4: Implement `DeleteUser` in `user.go`**

Add to `netme-backend/internal/repositories/user.go`:

```go
func (r *UserRepository) DeleteUser(userID string) error {
	result, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}
```

- [ ] **Step 5: Replace 501 stub in `users.go`**

Replace `DeleteMe` in `netme-backend/internal/handlers/users.go`:

```go
func (h *UsersHandler) DeleteMe(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Not authenticated",
		})
		return
	}

	if err := h.userRepo.DeleteUser(userIDVal.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "delete_error",
			Message: "Failed to delete account",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 6: Run all backend tests**

```bash
cd netme-backend && go test ./... -v
```

Expected: all tests PASS. `TestDeleteMeSuccess` and `TestDeleteMeUserNotFound` now pass.

- [ ] **Step 7: Commit**

```bash
git add internal/repositories/interfaces.go internal/repositories/user.go \
        internal/handlers/users.go internal/handlers/users_test.go
git commit -m "feat: implement DELETE /v1/me — hard delete user account and cascade tokens"
```

---

### Task 2: Mobile — SettingsScreen + navigation wiring

**Files:**
- Create: `netme-mobile/src/screens/SettingsScreen.tsx`
- Modify: `netme-mobile/src/navigation/RootNavigator.tsx`

**Interfaces:**
- Consumes: `authService.deleteAccount()` from preflight-polish Task 2
- Consumes: `useAuth().logout()` and `useAuth().clearAuth()` from `AuthContext`
- Produces: Settings tab/screen with delete account confirmation flow

- [ ] **Step 1: Create `SettingsScreen.tsx`**

Create `netme-mobile/src/screens/SettingsScreen.tsx`:

```typescript
import React, { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  Alert,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { useAuth } from '../context/AuthContext';
import { authService } from '../services/authService';

export default function SettingsScreen() {
  const { user, clearAuth } = useAuth();
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDeleteAccount = () => {
    Alert.alert(
      'Delete Account',
      'This will permanently delete your account and all your data. This cannot be undone.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: confirmDelete,
        },
      ]
    );
  };

  const confirmDelete = async () => {
    try {
      setIsDeleting(true);
      await authService.deleteAccount();
      await clearAuth();
    } catch {
      Alert.alert('Error', 'Failed to delete account. Please try again.');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.email}>{user?.email}</Text>

      {isDeleting ? (
        <ActivityIndicator style={styles.button} />
      ) : (
        <TouchableOpacity style={styles.deleteButton} onPress={handleDeleteAccount}>
          <Text style={styles.deleteButtonText}>Delete Account</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 24,
    backgroundColor: '#fff',
  },
  email: {
    fontSize: 16,
    color: '#333',
    marginBottom: 32,
  },
  button: {
    marginTop: 16,
  },
  deleteButton: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#e53e3e',
    borderRadius: 8,
    padding: 16,
    alignItems: 'center',
    marginTop: 16,
  },
  deleteButtonText: {
    color: '#e53e3e',
    fontSize: 16,
    fontWeight: '600',
  },
});
```

- [ ] **Step 2: Add SettingsScreen to navigation**

Read `netme-mobile/src/navigation/RootNavigator.tsx` first, then add SettingsScreen to the authenticated app stack. Replace the existing app stack navigator content to include both the current screen(s) and Settings:

```typescript
import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { View, ActivityIndicator } from 'react-native';
import { useAuth } from '../context/AuthContext';
import LoginScreen from '../screens/LoginScreen';
import RegisterScreen from '../screens/RegisterScreen';
import ProfileScreen from '../screens/ProfileScreen';
import SettingsScreen from '../screens/SettingsScreen';

const Stack = createNativeStackNavigator();

export default function RootNavigator() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
        <ActivityIndicator size="large" />
      </View>
    );
  }

  return (
    <NavigationContainer>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {isAuthenticated ? (
          <>
            <Stack.Screen name="Profile" component={ProfileScreen} />
            <Stack.Screen name="Settings" component={SettingsScreen} />
          </>
        ) : (
          <>
            <Stack.Screen name="Login" component={LoginScreen} />
            <Stack.Screen name="Register" component={RegisterScreen} />
          </>
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}
```

Note: if `RootNavigator.tsx` has a different structure, preserve the existing auth stack logic and simply add `SettingsScreen` as an additional screen in the authenticated stack.

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd netme-mobile && npx tsc --noEmit 2>&1 | grep -v "customConditions"
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/screens/SettingsScreen.tsx src/navigation/RootNavigator.tsx
git commit -m "feat: add SettingsScreen with delete account confirmation flow"
```

# Pre-TestFlight Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix seven small auth and session issues required before TestFlight distribution.

**Architecture:** Backend fixes go in `auth.go` (email normalization + TOCTOU). Mobile changes are split into static cleanup (type removals, dead code, IP fallback) and session management (bootstrap expiry check, network-vs-auth clearAll). No new dependencies on the backend; no new dependencies on mobile.

**Tech Stack:** Go 1.25+, Gin, lib/pq, React Native 0.81.5, Expo, axios, expo-secure-store

## Global Constraints

- Working directory for all Go commands: `netme-backend/`
- Working directory for TypeScript check: `netme-mobile/`
- Email must be normalized to lowercase + trimmed before any lookup or storage
- `clearAll` only fires on HTTP auth errors (`error.response` exists), not network errors
- JWT expiry decoded locally — no network call during bootstrap
- Proactive refresh threshold: 60 seconds before expiry

---

## File Map

**Modify (backend):**
- `netme-backend/internal/handlers/auth.go` — email normalization in Register + Login; pq duplicate-key 409 in Register
- `netme-backend/internal/handlers/auth_test.go` — add `TestRegisterNormalizesEmail`, `TestLoginNormalizesEmail`, `TestRegisterConcurrentDuplicate`

**Modify (mobile):**
- `netme-mobile/src/services/authService.ts` — remove stale fields, remove `logoutAllDevices`, fix IP fallback, fix `clearAll` on network errors
- `netme-mobile/src/context/AuthContext.tsx` — remove stale fields, add `getJWTExpiry` helper, update `bootstrapAsync`

---

### Task 1: Backend email normalization + TOCTOU fix

**Files:**
- Modify: `netme-backend/internal/handlers/auth.go`
- Modify: `netme-backend/internal/handlers/auth_test.go`

**Interfaces:**
- Produces: `Register` normalizes email before all operations; returns 409 on pq unique constraint violation from `CreateUser`
- Produces: `Login` normalizes email before `GetUserByEmail`

- [ ] **Step 1: Add `DeleteUser` to mock so test file compiles for later tasks**

Add this method to `mockUserRepo` in `netme-backend/internal/handlers/auth_test.go` (needed now so all tasks compile together):

```go
func (m *mockUserRepo) DeleteUser(userID string) error {
	for email, u := range m.users {
		if u.ID == userID {
			delete(m.users, email)
			return nil
		}
	}
	return errors.New("user not found")
}
```

- [ ] **Step 2: Write failing tests for normalization and TOCTOU**

Add to `netme-backend/internal/handlers/auth_test.go`:

```go
func TestRegisterNormalizesEmail(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "  USER@EXAMPLE.COM  ", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.User.Email != "user@example.com" {
		t.Errorf("expected normalized email 'user@example.com', got %q", resp.User.Email)
	}
}

func TestLoginNormalizesEmail(t *testing.T) {
	r, userRepo, _ := newTestAuthRouter()

	// Register with lowercase first
	regReq, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "mixed@example.com", "password": "password123"}))
	regReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), regReq)
	_ = userRepo

	// Login with mixed case
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login",
		jsonBody(t, map[string]string{"email": "MIXED@EXAMPLE.COM", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for normalized login, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterConcurrentDuplicate(t *testing.T) {
	// Simulate TOCTOU: GetUserByEmail passes but CreateUser returns pq unique violation
	userRepo := &concurrentMockUserRepo{}
	tokenRepo := newMockTokenRepo()
	jwtSvc := services.NewJWTService(testAuthSecret)
	h := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc, &mockGoogleVerifier{})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v1/auth/register", h.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "race@example.com", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for concurrent duplicate, got %d: %s", w.Code, w.Body.String())
	}
}
```

Add the `concurrentMockUserRepo` helper (simulates TOCTOU: GetUserByEmail returns nil but CreateUser returns pq error):

```go
// concurrentMockUserRepo simulates a TOCTOU race: pre-check passes but INSERT hits unique constraint.
type concurrentMockUserRepo struct{}

func (m *concurrentMockUserRepo) CreateUser(email, passwordHash string) (*models.User, error) {
	return nil, &pq.Error{Code: "23505"}
}
func (m *concurrentMockUserRepo) GetUserByEmail(email string) (*models.User, error) {
	return nil, errors.New("user not found")
}
func (m *concurrentMockUserRepo) GetUserByID(id string) (*models.User, error) {
	return nil, errors.New("user not found")
}
func (m *concurrentMockUserRepo) UpdateLastLogin(userID string) error { return nil }
func (m *concurrentMockUserRepo) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *concurrentMockUserRepo) DeleteUser(userID string) error { return nil }
```

Add `"github.com/lib/pq"` to imports in `auth_test.go`.

- [ ] **Step 3: Run tests to confirm they fail**

```bash
cd netme-backend && go test ./internal/handlers/... -v -run "TestRegisterNormalizes|TestLoginNormalizes|TestRegisterConcurrent"
```

Expected: compilation or test failures (normalization not yet applied, pq check not yet in handler).

- [ ] **Step 4: Apply email normalization and TOCTOU fix to auth.go**

Add `"strings"` and `"github.com/lib/pq"` to imports in `netme-backend/internal/handlers/auth.go`.

In `Register`, immediately after `c.ShouldBindJSON(&req)` succeeds, add:

```go
req.Email = strings.ToLower(strings.TrimSpace(req.Email))
```

Replace the `CreateUser` error block in `Register`:

```go
user, err := h.userRepo.CreateUser(req.Email, passwordHash)
if err != nil {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "user_exists",
			Message: "User with this email already exists",
		})
		return
	}
	c.JSON(http.StatusInternalServerError, models.ErrorResponse{
		Error:   "creation_error",
		Message: "Failed to create user",
	})
	return
}
```

In `Login`, immediately after `c.ShouldBindJSON(&req)` succeeds, add:

```go
req.Email = strings.ToLower(strings.TrimSpace(req.Email))
```

- [ ] **Step 5: Run all backend tests**

```bash
cd netme-backend && go test ./... -v
```

Expected: all tests PASS including the 3 new ones (total 20).

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/auth.go internal/handlers/auth_test.go
git commit -m "fix: normalize email to lowercase on register/login; catch pq unique violation as 409"
```

---

### Task 2: Mobile static cleanup

**Files:**
- Modify: `netme-mobile/src/services/authService.ts`
- Modify: `netme-mobile/src/context/AuthContext.tsx`

**Interfaces:**
- Produces: `AuthResponse.user` type without `display_name`, `picture_url`
- Produces: `User` interface without `display_name`, `picture_url`, `last_login_at`
- Produces: `authService` without `logoutAllDevices`
- Produces: API base URL fallback is `http://localhost:8080/v1`

- [ ] **Step 1: Clean up `authService.ts`**

Full new content of `netme-mobile/src/services/authService.ts`:

```typescript
import axios, { AxiosInstance } from 'axios';
import { secureStorage } from './secureStorage';

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    auth_provider: string;
    auth_provider_user_id?: string;
    created_at: string;
    updated_at: string;
  };
}

class AuthService {
  private api: AxiosInstance;
  private isRefreshing = false;
  private refreshSubscribers: ((token: string) => void)[] = [];

  constructor() {
    const apiUrl = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/v1';

    this.api = axios.create({
      baseURL: apiUrl,
      timeout: 30000,
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    this.api.interceptors.request.use(
      async (config) => {
        const token = await secureStorage.getAccessToken();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    this.api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          originalRequest.url !== '/auth/refresh'
        ) {
          originalRequest._retry = true;

          if (!this.isRefreshing) {
            this.isRefreshing = true;
            try {
              const refreshToken = await secureStorage.getRefreshToken();
              if (refreshToken) {
                const response = await this.refresh(refreshToken);
                const { access_token } = response;

                originalRequest.headers.Authorization = `Bearer ${access_token}`;

                this.isRefreshing = false;
                this.onRefreshed(access_token);

                return this.api(originalRequest);
              } else {
                this.isRefreshing = false;
                throw new Error('No refresh token available');
              }
            } catch (refreshError) {
              this.isRefreshing = false;
              // Only clear auth on HTTP auth failures, not network errors
              if (axios.isAxiosError(refreshError) && refreshError.response) {
                await secureStorage.clearAll();
              }
              throw refreshError;
            }
          } else {
            return new Promise((resolve) => {
              this.refreshSubscribers.push((token) => {
                originalRequest.headers.Authorization = `Bearer ${token}`;
                resolve(this.api(originalRequest));
              });
            });
          }
        }

        return Promise.reject(error);
      }
    );
  }

  private onRefreshed(token: string) {
    this.refreshSubscribers.forEach((callback) => callback(token));
    this.refreshSubscribers = [];
  }

  async register(email: string, password: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/register', { email, password });
    return response.data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/login', { email, password });
    return response.data;
  }

  async loginWithGoogle(googleIDToken: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/google', {
      id_token: googleIDToken,
    });
    return response.data;
  }

  async refresh(refreshToken: string): Promise<AuthResponse> {
    const response = await this.api.post<AuthResponse>('/auth/refresh', {
      refresh_token: refreshToken,
    });
    return response.data;
  }

  async logout(refreshToken: string, accessToken: string): Promise<void> {
    try {
      await this.api.post(
        '/auth/logout',
        { refresh_token: refreshToken },
        { headers: { Authorization: `Bearer ${accessToken}` } }
      );
    } catch (error) {
      console.error('Logout API call failed:', error);
    }
  }

  async deleteAccount(): Promise<void> {
    await this.api.delete('/me');
  }
}

export const authService = new AuthService();
```

Note: `deleteAccount` is added here so it's ready for the account deletion feature in the next plan.

- [ ] **Step 2: Clean up `AuthContext.tsx` — remove stale User fields**

Update the `User` interface in `netme-mobile/src/context/AuthContext.tsx`:

```typescript
export interface User {
  id: string;
  email: string;
  auth_provider: string;
  auth_provider_user_id?: string;
  created_at: string;
  updated_at: string;
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd netme-mobile && npx tsc --noEmit 2>&1 | grep -v "customConditions"
```

Expected: no errors (the `customConditions` tsconfig warning is pre-existing and ignored).

- [ ] **Step 4: Commit**

```bash
git add src/services/authService.ts src/context/AuthContext.tsx
git commit -m "fix: remove stale user fields, logoutAllDevices dead code, hardcoded IP; only clearAll on auth errors"
```

---

### Task 3: Mobile session management — bootstrap token expiry

**Files:**
- Modify: `netme-mobile/src/context/AuthContext.tsx`

**Interfaces:**
- Consumes: `authService.refresh(refreshToken)` from Task 2
- Produces: `bootstrapAsync` proactively refreshes tokens expiring within 60 seconds

- [ ] **Step 1: Add `getJWTExpiry` helper and update `bootstrapAsync`**

In `netme-mobile/src/context/AuthContext.tsx`, add the helper function before the `AuthProvider` component:

```typescript
function getJWTExpiry(token: string): number | null {
  try {
    const base64Url = token.split('.')[1];
    if (!base64Url) return null;
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '==='.slice((base64.length + 3) % 4);
    const payload = JSON.parse(atob(padded));
    return typeof payload.exp === 'number' ? payload.exp : null;
  } catch {
    return null;
  }
}
```

Replace `bootstrapAsync` with:

```typescript
const bootstrapAsync = async () => {
  try {
    const savedAccessToken = await secureStorage.getAccessToken();
    const savedRefreshToken = await secureStorage.getRefreshToken();
    const savedUser = await secureStorage.getUser();

    if (!savedAccessToken || !savedRefreshToken || !savedUser) {
      return;
    }

    const expiry = getJWTExpiry(savedAccessToken);
    const nowSeconds = Math.floor(Date.now() / 1000);
    const isExpiredOrExpiringSoon = expiry === null || expiry - nowSeconds < 60;

    if (isExpiredOrExpiringSoon) {
      try {
        const response = await authService.refresh(savedRefreshToken);
        setAccessToken(response.access_token);
        setRefreshToken(response.refresh_token);
        setUser(response.user);
        await secureStorage.saveAccessToken(response.access_token);
        await secureStorage.saveRefreshToken(response.refresh_token);
        await secureStorage.saveUser(JSON.stringify(response.user));
      } catch {
        await secureStorage.clearAll();
      }
    } else {
      setAccessToken(savedAccessToken);
      setRefreshToken(savedRefreshToken);
      setUser(JSON.parse(savedUser));
    }
  } catch (error) {
    console.error('Failed to restore session:', error);
  } finally {
    setIsLoading(false);
  }
};
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd netme-mobile && npx tsc --noEmit 2>&1 | grep -v "customConditions"
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/context/AuthContext.tsx
git commit -m "fix: proactively refresh expired tokens during bootstrap; add getJWTExpiry helper"
```

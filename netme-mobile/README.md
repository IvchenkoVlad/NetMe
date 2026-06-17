# NetMe Mobile

React Native + Expo mobile app for personal finance.

## Stack

- **Runtime:** React Native 0.73
- **Framework:** Expo
- **Language:** TypeScript
- **Navigation:** React Navigation
- **State:** Zustand
- **Data fetching:** TanStack Query
- **Forms:** React Hook Form + Zod
- **Styling:** React Native built-in (no external library yet)

## Structure

```
app/
  screens/           ← Screens (Home, Transactions, Budgets, Accounts, Settings)
  components/        ← Reusable UI components (buttons, cards, lists, etc.)
  hooks/             ← Custom React hooks
  utils/             ← Helpers (API client, formatters, validators)
  types/             ← TypeScript type definitions
  stores/            ← Zustand state management
  navigation/        ← Navigation configuration
  App.tsx            ← Root app component (navigation setup)

App.json             ← Expo configuration
package.json         ← Dependencies
tsconfig.json        ← TypeScript config
```

## Quick Start

### Prerequisites

- Node 18+
- npm or yarn
- Expo CLI: `npm install -g expo-cli`
- iOS Simulator or physical device + Expo Go app

### Local Development

```bash
# Install dependencies
npm install

# Start Expo dev server (port 8081)
npm start

# Then in another terminal:
# iOS Simulator:
npm run ios

# Physical device:
# Scan QR code with Expo Go app (iOS) or with camera (Android)

# Web (for testing):
npm run web
```

### Development Commands

```bash
make mobile              # Start Expo dev server
make mobile-test         # Run tests
make mobile-build-ios-testflight   # Build for TestFlight
make mobile-build-ios-release      # Build for App Store
```

## Navigation Structure

Suggested tab-based navigation:

```
App
├── Home             ← Dashboard, budget overview
├── Transactions     ← Transaction list, search, filter
├── Budgets          ← Category budgets, editing
├── Accounts         ← Linked accounts, institution management
└── Settings         ← Profile, categories, account deletion
```

## App Flow

### Onboarding (First Time)
```
Welcome → Sign Up/Login → Data Explanation → Connect Bank → Sync → Budget Setup → Dashboard
```

### Daily Usage
```
Dashboard → Review Spending → Correct Categories → Check Budgets → Done
```

## Key Screens

### Home Screen
- Month-to-date spending
- Budget progress
- Top categories
- Categories over budget
- Transactions needing review
- Recent transactions
- Sync status

### Transactions Screen
- Infinite list of transactions
- Search, filter by account/category
- Month selector
- Pending indicator
- Swipe to edit (or tap detail)

### Transaction Detail
- Full transaction info
- Category selector
- Exclude toggle
- Create merchant rule
- Apply rule to past transactions

### Budgets Screen
- Current month selector
- Category rows with spent/remaining
- Progress bars
- Edit budget amount

### Accounts Screen
- Institution list
- Account cards with balance
- Connection status
- Hide/reconnect/remove actions

### Settings Screen
- Profile
- Manage categories
- Privacy/data explanation
- Delete account
- Log out

## State Management (Zustand)

Example store structure:

```typescript
// stores/authStore.ts
import { create } from 'zustand';

interface AuthStore {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  token: null,
  login: async (email, password) => { /* ... */ },
  logout: () => set({ user: null, token: null }),
}));
```

Use throughout app:

```typescript
const { user, logout } = useAuthStore();
```

## API Client

Create in `utils/api.ts`:

```typescript
import axios from 'axios';

export const api = axios.create({
  baseURL: process.env.MOBILE_API_URL || 'http://localhost:8080/api/v1',
});

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = /* get from store */;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

Use in hooks:

```typescript
export const useTransactions = () => {
  return useQuery({
    queryKey: ['transactions'],
    queryFn: () => api.get('/transactions').then((r) => r.data),
  });
};
```

## Forms

Use React Hook Form + Zod for validation:

```typescript
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
});

export function LoginForm() {
  const { control, handleSubmit } = useForm({
    resolver: zodResolver(schema),
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      {/* ... */}
    </form>
  );
}
```

## Testing

```bash
npm test               # Run all tests
npm test -- --watch   # Watch mode
```

## Building for iOS

### Development Build (TestFlight)

```bash
eas build --platform ios --profile testflight
```

Configure in `eas.json`:

```json
{
  "build": {
    "preview": {
      "ios": {
        "buildType": "simulator"
      }
    },
    "testflight": {
      "ios": {
        "buildType": "app-store"
      },
      "autoIncrement": true
    },
    "release": {
      "ios": {
        "buildType": "app-store"
      },
      "autoIncrement": true
    }
  },
  "submit": {
    "testflight": {
      "ios": {
        "testerId": "your-test-group-id"
      }
    }
  }
}
```

### App Store Release

```bash
eas build --platform ios --profile release
eas submit --platform ios
```

## Environment Variables

In `app.json` or `.env`:

```json
{
  "extra": {
    "apiUrl": "http://localhost:8080/api/v1",
    "plaidClientId": "...",
    "sentry": "..."
  }
}
```

Access in code:

```typescript
import Constants from 'expo-constants';

const apiUrl = Constants.expoConfig?.extra?.apiUrl;
```

## Debugging

### Expo DevTools
- Press `i` or `a` in the terminal running `npm start`
- Opens in Expo Go app on device/simulator

### React DevTools
- Install: `npm install --save-dev @react-devtools/core`
- See React component tree

### Network Debugging
- Use Reactotron or native browser dev tools
- Monitor API calls

## Next Steps

1. Create screen components and navigation
2. Implement auth flow (login/signup)
3. Build transaction list and detail screens
4. Connect to backend API
5. Add state management for user data
6. Implement categories, budgets, rules
7. Build dashboard
8. Test on real devices
9. TestFlight beta
10. App Store submission

## Resources

- **React Native:** https://reactnative.dev
- **Expo:** https://expo.dev
- **React Navigation:** https://reactnavigation.org
- **Zustand:** https://github.com/pmndrs/zustand
- **TanStack Query:** https://tanstack.com/query
- **MVP Plan:** `docs/MVP_PLAN.md` (workspace root)

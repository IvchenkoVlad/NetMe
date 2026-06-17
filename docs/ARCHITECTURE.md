# NetMe Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                  React Native Mobile App                    │
│         (Expo, TypeScript, Zustand, TanStack Query)         │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTPS
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway / Load Balancer              │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┴─────────────┐
         ↓                           ↓
    ┌─────────────┐          ┌──────────────┐
    │  Go Backend │          │   Webhooks   │
    │   (Gin)     │◄────────►│  (Plaid)     │
    └──────┬──────┘          └──────────────┘
           │
    ┌──────┴──────────┬─────────────────┐
    ↓                 ↓                 ↓
┌────────┐      ┌─────────┐      ┌──────────┐
│  Redis │      │PostgreSQL     │  Asynq   │
│(Cache) │      │(Data Store)   │(Jobs)    │
└────────┘      └─────────────┘  └──────────┘
                      │
         ┌────────────┴──────────────┐
         ↓                           ↓
    ┌──────────┐              ┌────────────┐
    │Plaid API │              │ OpenAI API │
    │(Banking) │              │ (AI)       │
    └──────────┘              └────────────┘
```

## Component Responsibilities

### Mobile App (React Native + Expo)
- **Screens:** Authentication, Dashboard, Accounts, Transactions, Insights, Settings
- **State:** Zustand stores for auth, accounts, transactions, filters
- **API Client:** Axios with TanStack Query for caching & sync
- **Forms:** React Hook Form + Zod validation

### Backend (Go + Gin)
- **Auth Handler:** JWT generation, password hashing (bcrypt), refresh logic
- **Account Handler:** List accounts, balance aggregation
- **Transaction Handler:** List, search, filter transactions
- **Analytics Handler:** Spending trends, category breakdown, net worth calculations
- **Plaid Handler:** Link token generation, token exchange, account sync
- **Background Jobs:** Daily net worth snapshots, transaction sync, AI insights

### Database (PostgreSQL)
- **users** — User accounts, credentials
- **plaid_items** — Plaid connections (encrypted access tokens)
- **institutions** — Bank metadata (name, Plaid ID)
- **accounts** — Connected bank accounts (routing to institutions)
- **transactions** — All synced transactions
- **net_worth_snapshots** — Daily snapshots for trend analysis

### Cache (Redis)
- **Sessions:** JWT-based sessions (optional, for logout support)
- **Rate Limiting:** Per-user rate limits on API endpoints
- **Caching:** Frequently accessed data (account list, spending summaries)

### Background Jobs (Asynq)
- **PlaidSync:** Daily or on-demand sync of transactions from Plaid
- **NetWorthSnapshot:** Daily calculation of total assets - liabilities
- **AIInsights:** Generate spending summaries & recommendations (future)
- **Notifications:** Send mobile push notifications (future)

## Data Flow

### Registration & Login
```
Mobile → Register/Login → Backend Auth Handler → PostgreSQL (store user, hash password)
                           ← JWT Token ← Mobile (store in secure storage)
```

### Connect Bank Account
```
Mobile → Create Link Token Request → Backend → Plaid API → Mobile (open Plaid Link)
         ← Link Token Response ←
         → Public Token Exchange → Backend → Plaid API (exchange for access token)
                                ← Access Token (encrypted & stored) ←
         Trigger Account Sync (background job)
```

### Sync Transactions
```
Background Job → Plaid API (fetch new/updated transactions)
              → Normalize & store in PostgreSQL
              → Update account balances
              → Calculate net worth snapshot
```

### View Spending
```
Mobile → Request Spending Analytics → Backend
         ← Query transactions by category, date range ←
         ← Return aggregated spending by category ←
         Display chart/breakdown
```

## Error Handling

- **Auth Errors:** 401 Unauthorized for invalid/expired tokens
- **Validation:** 400 Bad Request for invalid input
- **Not Found:** 404 for missing resources
- **Rate Limit:** 429 Too Many Requests
- **Server Error:** 500 + Sentry logging

## Security

- **Passwords:** bcrypt hashing (bcryptjs in mobile for registration validation)
- **Tokens:** JWT with RS256 (or HS256 for simplicity)
- **Sensitive Data:** Plaid access tokens encrypted with AWS KMS (production) or AES-256 (local)
- **API:** HTTPS only (enforced in production)
- **CORS:** Controlled origin list
- **Rate Limiting:** Redis-based per-user/IP limits

## Scalability Considerations

**Current (MVP):**
- Single backend instance with connection pooling
- PostgreSQL + Redis on managed services
- Asynq for background jobs

**Future (Post-MVP):**
- Kubernetes for backend scaling
- RDS read replicas for query scaling
- ElastiCache cluster mode for Redis HA
- API Gateway rate limiting
- CDN for static mobile assets
- Event streaming (Kafka) for audit logs

## Deployment

**Development:** Docker Compose (local PostgreSQL + Redis)

**Staging:** AWS ECS Fargate + RDS + ElastiCache + Sentry

**Production:** Multi-AZ Fargate, RDS backups, CloudWatch monitoring, auto-scaling

---

See [API.md](./API.md) for endpoint details and [DATABASE.md](./DATABASE.md) for schema.

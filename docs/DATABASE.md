# NetMe Database Schema

**Database:** PostgreSQL 16

**Connection:** `postgres://netme:devpassword@localhost:5432/netme_dev` (local dev)

---

## Tables

### users

Stores user accounts and authentication credentials.

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Unique user ID |
| email | VARCHAR(255) | UNIQUE NOT NULL | Email address |
| password_hash | VARCHAR(255) | NOT NULL | bcrypt hash of password |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Account creation time |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update time |

**Indexes:**
```sql
CREATE UNIQUE INDEX idx_users_email ON users(email);
```

---

### bank_connections

Stores bank API connections (one per provider/link for each user). Supports multiple providers (Plaid, Yodlee, Open Banking APIs, etc.).

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY | Unique connection ID |
| user_id | UUID | FK → users.id, NOT NULL | Owner of the connection |
| provider | VARCHAR(50) | NOT NULL | Connection provider (e.g., `plaid`, `yodlee`, `open-banking`) |
| external_item_id | VARCHAR(255) | NOT NULL | Provider's item ID (e.g., Plaid item_id) |
| access_token | TEXT | NOT NULL | **Encrypted** access token from provider |
| external_institution_id | VARCHAR(255) | | Provider's institution ID (e.g., Plaid institution_id) |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Connection time |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last sync time |

**Constraints:**
```sql
UNIQUE(user_id, provider, external_item_id)
```

**Notes:**
- `access_token` must be encrypted at rest (AES-256 or AWS KMS)
- One row per provider per user (user can connect to Plaid + Yodlee separately)
- Easy to add new providers without schema changes

---

### institutions

Stores bank/financial institution metadata from any provider.

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY | Unique institution ID |
| provider | VARCHAR(50) | NOT NULL | Data source (e.g., `plaid`, `yodlee`) |
| external_provider_id | VARCHAR(255) | NOT NULL | Provider's institution ID (e.g., Plaid `ins_123456`) |
| name | VARCHAR(255) | NOT NULL | Institution name (e.g., "Chase Bank") |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When record was added |

**Constraints:**
```sql
UNIQUE(provider, external_provider_id)
```

**Notes:**
- Same institution can have records from different providers
- De-duplication happens at application layer

---

### accounts

Stores user's connected bank accounts (checking, savings, credit cards, etc).

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY | Unique account ID |
| user_id | UUID | FK → users.id, NOT NULL | Account owner |
| institution_id | UUID | FK → institutions.id | Which bank |
| external_account_id | VARCHAR(255) | NOT NULL | Provider's account ID (e.g., Plaid account ID) |
| account_name | VARCHAR(255) | NOT NULL | User-friendly name (e.g., "Chase Checking") |
| type | VARCHAR(50) | | `depository`, `credit`, `investment`, `loan` |
| subtype | VARCHAR(50) | | `checking`, `savings`, `credit_card`, etc. |
| current_balance | DECIMAL(19, 2) | | Account balance in USD |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When account was connected |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last balance update |

**Constraints:**
```sql
UNIQUE(user_id, external_account_id)
```

**Notes:**
- `external_account_id` is provider-specific (Plaid account ID, Yodlee account ID, etc.)
- Easy to support multiple providers for same user

---

### transactions

Stores all financial transactions synced from any provider.

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY | Unique transaction ID |
| account_id | UUID | FK → accounts.id, NOT NULL | Which account |
| external_transaction_id | VARCHAR(255) | | Provider's transaction ID (for dedup) |
| amount | DECIMAL(19, 2) | NOT NULL | Transaction amount (positive for debits) |
| merchant_name | VARCHAR(255) | | Merchant/payee name |
| category | VARCHAR(100) | | Category (e.g., `FOOD_AND_DRINK_RESTAURANTS`) |
| transaction_date | DATE | NOT NULL | When transaction occurred |
| pending | BOOLEAN | DEFAULT FALSE | True if pending (not yet settled) |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When synced |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update |

**Constraints:**
```sql
UNIQUE(account_id, external_transaction_id)
```

**Notes:**
- `external_transaction_id` is provider-specific (prevents duplicate syncs)
- Full-text search on `merchant_name` for search feature
- Supports multiple data sources seamlessly

---

### net_worth_snapshots

Daily snapshots of total assets and liabilities for trend analysis.

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY | Unique snapshot ID |
| user_id | UUID | FK → users.id, NOT NULL | Snapshot owner |
| assets | DECIMAL(19, 2) | NOT NULL | Sum of all account balances (checking, savings, investments) |
| liabilities | DECIMAL(19, 2) | NOT NULL | Sum of all liabilities (loans, credit cards) |
| net_worth | DECIMAL(19, 2) | NOT NULL | assets - liabilities |
| snapshot_date | DATE | NOT NULL | Date of snapshot (typically today) |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When created |

**Indexes:**
```sql
CREATE INDEX idx_net_worth_user_id ON net_worth_snapshots(user_id);
CREATE UNIQUE INDEX idx_net_worth_user_date ON net_worth_snapshots(user_id, snapshot_date);
CREATE INDEX idx_net_worth_snapshot_date ON net_worth_snapshots(snapshot_date);
```

**Notes:**
- One snapshot per user per day
- Created automatically by a background job (Asynq) at midnight UTC

---

## Key Relationships

```
users (1) ──→ (∞) bank_connections (supports multiple providers)
users (1) ──→ (∞) accounts (from all providers)
users (1) ──→ (∞) transactions (indirectly via accounts)
users (1) ──→ (∞) net_worth_snapshots

institutions (1) ──→ (∞) accounts
accounts (1) ──→ (∞) transactions
```

**Multi-provider support:**
- User can connect via Plaid AND Yodlee simultaneously
- Each provider has separate bank_connections row
- Accounts aggregated across all providers
- Same institution (e.g., Chase) can appear multiple times from different providers

---

## Sample Queries

### Get user's total net worth
```sql
SELECT 
  COALESCE(SUM(CASE WHEN type IN ('depository') THEN current_balance ELSE 0 END), 0) as assets,
  COALESCE(SUM(CASE WHEN type IN ('credit', 'loan') THEN ABS(current_balance) ELSE 0 END), 0) as liabilities
FROM accounts
WHERE user_id = $1;
```

### Get monthly spending by category
```sql
SELECT 
  category,
  SUM(amount) as total,
  COUNT(*) as transaction_count
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE a.user_id = $1
  AND DATE_TRUNC('month', t.transaction_date) = DATE_TRUNC('month', CURRENT_DATE)
  AND amount > 0  -- Outflows only
GROUP BY category
ORDER BY total DESC;
```

### Get net worth history (last 30 days)
```sql
SELECT 
  snapshot_date,
  assets,
  liabilities,
  net_worth
FROM net_worth_snapshots
WHERE user_id = $1
  AND snapshot_date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY snapshot_date ASC;
```

### Search transactions by merchant
```sql
SELECT *
FROM transactions
WHERE account_id IN (SELECT id FROM accounts WHERE user_id = $1)
  AND to_tsvector('english', merchant_name) @@ plainto_tsquery('english', $2)
ORDER BY transaction_date DESC
LIMIT 50;
```

### Get pending transactions
```sql
SELECT *
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE a.user_id = $1 AND t.pending = TRUE
ORDER BY t.transaction_date DESC;
```

---

## Migration Strategy

Migrations use separate `.up.sql` and `.down.sql` files organized by type, with automatic version tracking.

**Migration structure:**
```
backend/internal/db/migrations/
├── tables/          # DDL for tables
│   ├── 001_users.up.sql / .down.sql
│   ├── 002_bank_connections.up.sql / .down.sql
│   └── ...
├── indices/         # Index definitions
└── functions/       # Stored procedures (future)
```

**To apply migrations:**
```bash
make migrate-up      # Apply all pending migrations
make migrate-down    # Rollback last migration
make migrate-down STEPS=3  # Rollback last 3
```

**Migrations run:**
- Automatically on backend startup (checks `schema_migrations` table)
- Tracked in `schema_migrations` table (version, direction, applied_at)
- Idempotent (safe to run multiple times)

**To add a new migration:**
1. Create `backend/internal/db/migrations/tables/NNN_name.up.sql` with CREATE/ALTER
2. Create `backend/internal/db/migrations/tables/NNN_name.down.sql` with DROP/rollback
3. Run `make migrate-up`

**Notes:**
- Version numbers are auto-sorted (001, 002, 003, etc.)
- Must provide both .up and .down SQL files
- Down migrations must be exact inverses of up migrations

---

## Backup & Recovery

**Local Dev:**
```bash
# Backup
docker-compose exec postgres pg_dump -U netme netme_dev > backup.sql

# Restore
cat backup.sql | docker-compose exec -T postgres psql -U netme netme_dev
```

**Production:**
- RDS automatic backups (daily)
- Retention: 7 days minimum (configurable)
- Point-in-time recovery available

---

## Performance Considerations

### Indexing Strategy
- **High cardinality:** user_id, account_id, transaction_date
- **Full-text search:** merchant_name (GIN index)
- **Uniqueness:** plaid_transaction_id (prevents duplicate syncs)

### Query Optimization
- Always filter by `user_id` before transactions for data isolation
- Use `DATE_TRUNC` for grouping by month/year
- Paginate transaction lists (limit 50-100)

### Connection Pooling
- Backend uses connection pool (min 5, max 25 connections)
- Redis caching for frequently accessed data (accounts, recent transactions)

---

See [ARCHITECTURE.md](./ARCHITECTURE.md) for system design and [API.md](./API.md) for endpoint details.

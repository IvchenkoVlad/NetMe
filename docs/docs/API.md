# NetMe API Reference

**Base URL:** `http://localhost:8080/api/v1` (local) | `https://api.netme.com/api/v1` (production)

**Authentication:** All endpoints except `/auth/register` and `/auth/login` require `Authorization: Bearer <jwt_token>` header.

---

## Authentication

### Register
```
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password"
}

Response 201:
{
  "id": "uuid",
  "email": "user@example.com",
  "created_at": "2025-06-13T10:00:00Z"
}

Response 400: { "error": "invalid email" } or { "error": "password too short" }
```

### Login
```
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password"
}

Response 200:
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600
}

Response 401: { "error": "invalid credentials" }
```

### Logout
```
POST /auth/logout
Authorization: Bearer <access_token>

Response 200: { "success": true }
```

### Refresh Token
```
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}

Response 200:
{
  "access_token": "eyJhbGc...",
  "expires_in": 3600
}

Response 401: { "error": "invalid refresh token" }
```

---

## Plaid Integration

### Create Link Token
```
POST /plaid/create-link-token
Authorization: Bearer <access_token>

Response 200:
{
  "link_token": "link-sandbox-...",
  "expiration": "2025-06-13T11:00:00Z"
}
```

### Exchange Public Token
```
POST /plaid/exchange-public-token
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "public_token": "public-sandbox-...",
  "metadata": {
    "institution": {
      "name": "Chase",
      "institution_id": "ins_12345"
    },
    "accounts": [
      {
        "id": "account-id-...",
        "name": "Checking",
        "type": "depository",
        "subtype": "checking"
      }
    ]
  }
}

Response 200:
{
  "item_id": "item-id-...",
  "accounts": [
    {
      "id": "uuid",
      "plaid_account_id": "account-id-...",
      "name": "Checking",
      "type": "depository",
      "subtype": "checking",
      "balance": 1500.00,
      "institution": {
        "id": "uuid",
        "plaid_id": "ins_12345",
        "name": "Chase"
      }
    }
  ]
}
```

### Sync Transactions (Manual Trigger)
```
POST /plaid/sync
Authorization: Bearer <access_token>

Response 200:
{
  "synced_accounts": 2,
  "new_transactions": 15,
  "updated_transactions": 3
}
```

### Plaid Webhook (Inbound)
```
POST /plaid/webhook
Content-Type: application/json

{
  "webhook_code": "TRANSACTIONS_UPDATE",
  "item_id": "item-id-...",
  "new_transactions": 5,
  "webhook_verification_request": false
}

Response 200: (empty)
```

---

## Accounts

### List Accounts
```
GET /accounts
Authorization: Bearer <access_token>
Query Params: ?institution_id=uuid (optional)

Response 200:
{
  "accounts": [
    {
      "id": "uuid",
      "name": "Checking",
      "type": "depository",
      "subtype": "checking",
      "balance": 1500.00,
      "currency": "USD",
      "institution": {
        "id": "uuid",
        "plaid_id": "ins_12345",
        "name": "Chase"
      },
      "connected_at": "2025-06-01T10:00:00Z",
      "last_synced_at": "2025-06-13T10:00:00Z"
    }
  ],
  "total_assets": 15000.00,
  "total_liabilities": 5000.00
}
```

### Get Account Details
```
GET /accounts/:id
Authorization: Bearer <access_token>

Response 200:
{
  "id": "uuid",
  "name": "Checking",
  "type": "depository",
  "subtype": "checking",
  "balance": 1500.00,
  "currency": "USD",
  "institution": {
    "id": "uuid",
    "plaid_id": "ins_12345",
    "name": "Chase"
  },
  "transactions_count": 42,
  "connected_at": "2025-06-01T10:00:00Z",
  "last_synced_at": "2025-06-13T10:00:00Z"
}

Response 404: { "error": "account not found" }
```

---

## Transactions

### List Transactions
```
GET /transactions
Authorization: Bearer <access_token>
Query Params:
  ?account_id=uuid (optional)
  ?category=groceries (optional)
  ?start_date=2025-06-01 (optional)
  ?end_date=2025-06-13 (optional)
  ?limit=50 (default 50)
  ?offset=0 (default 0)

Response 200:
{
  "transactions": [
    {
      "id": "uuid",
      "account_id": "uuid",
      "amount": 25.50,
      "merchant": "Whole Foods",
      "category": "groceries",
      "date": "2025-06-13",
      "pending": false,
      "synced_at": "2025-06-13T10:00:00Z"
    }
  ],
  "total": 250,
  "limit": 50,
  "offset": 0
}
```

### Get Transaction Details
```
GET /transactions/:id
Authorization: Bearer <access_token>

Response 200:
{
  "id": "uuid",
  "account_id": "uuid",
  "amount": 25.50,
  "merchant": "Whole Foods",
  "category": "groceries",
  "date": "2025-06-13",
  "pending": false,
  "plaid_id": "txn-id-...",
  "synced_at": "2025-06-13T10:00:00Z"
}

Response 404: { "error": "transaction not found" }
```

### Search Transactions
```
GET /transactions/search
Authorization: Bearer <access_token>
Query Params:
  ?q=whole foods
  ?category=groceries (optional)
  ?limit=50 (default 50)

Response 200:
{
  "transactions": [
    {
      "id": "uuid",
      "merchant": "Whole Foods Market",
      "amount": 25.50,
      "date": "2025-06-13",
      "category": "groceries"
    }
  ],
  "total": 5
}
```

---

## Analytics

### Spending Analytics
```
GET /analytics/spending
Authorization: Bearer <access_token>
Query Params:
  ?month=2025-06 (YYYY-MM, default current month)

Response 200:
{
  "month": "2025-06",
  "total_spending": 1250.00,
  "by_category": {
    "groceries": 300.00,
    "dining": 450.00,
    "utilities": 200.00,
    "other": 300.00
  },
  "by_account": {
    "uuid": 1250.00
  },
  "daily_breakdown": [
    {
      "date": "2025-06-01",
      "amount": 50.00,
      "transaction_count": 2
    }
  ]
}
```

### Net Worth Analytics
```
GET /analytics/net-worth
Authorization: Bearer <access_token>
Query Params:
  ?months=6 (default 6, max 24)

Response 200:
{
  "current": {
    "assets": 50000.00,
    "liabilities": 10000.00,
    "net_worth": 40000.00,
    "date": "2025-06-13"
  },
  "history": [
    {
      "date": "2025-06-13",
      "assets": 50000.00,
      "liabilities": 10000.00,
      "net_worth": 40000.00
    },
    {
      "date": "2025-06-12",
      "assets": 49500.00,
      "liabilities": 10000.00,
      "net_worth": 39500.00
    }
  ],
  "trend": {
    "net_worth_change": 500.00,
    "percentage_change": 1.27
  }
}
```

### Category Analytics
```
GET /analytics/categories
Authorization: Bearer <access_token>
Query Params:
  ?months=3 (default 3)

Response 200:
{
  "categories": [
    {
      "name": "groceries",
      "total": 900.00,
      "average": 300.00,
      "transaction_count": 15,
      "trend": 5.2
    },
    {
      "name": "dining",
      "total": 1200.00,
      "average": 400.00,
      "transaction_count": 12,
      "trend": -3.1
    }
  ],
  "period": "last 3 months"
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "timestamp": "2025-06-13T10:00:00Z"
}
```

**Common Codes:**
- `INVALID_REQUEST` — 400 Bad Request
- `UNAUTHORIZED` — 401 Unauthorized
- `FORBIDDEN` — 403 Forbidden
- `NOT_FOUND` — 404 Not Found
- `CONFLICT` — 409 Conflict
- `RATE_LIMITED` — 429 Too Many Requests
- `INTERNAL_ERROR` — 500 Internal Server Error
- `SERVICE_UNAVAILABLE` — 503 Service Unavailable

---

## Rate Limiting

- **Limit:** 100 requests per minute per user
- **Response Headers:** `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- **Exceeded:** 429 status with `Retry-After` header

---

## Pagination

List endpoints support standard pagination:
- `?limit=50` (max 100)
- `?offset=0`
- Response includes `total`, `limit`, `offset`

---

See [ARCHITECTURE.md](./ARCHITECTURE.md) for system overview and [DATABASE.md](./DATABASE.md) for schema details.

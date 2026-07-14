# Dashboard (Home Tab) Design

## Goal

Add a Home tab as the default landing screen showing the user's financial snapshot: net position for the current month, total account balances, over-budget alerts, and recent transactions.

## Architecture

No new backend endpoint. All data already exists across three endpoints fetched in parallel:

- `GET /v1/budget/summary?month=YYYY-MM` — spending, income, per-category totals with budget limits
- `GET /v1/accounts` — accounts with `current_balance`
- `GET /v1/transactions?limit=5` — 5 most recent transactions

`HomeScreen.tsx` calls all three in a single `Promise.all` on mount. No new service file — reuses `budgetService` and `plaidService`.

## Mobile: HomeScreen

**File:** `netme-mobile/src/screens/HomeScreen.tsx`

**Sections (top to bottom):**

1. **Net Position card** — "This month" header, Income / Spending row (same layout as BudgetScreen hero), and a "Saved" or "Over by" line derived as `income - spending`. Green if positive, red if negative.

2. **Total Balance card** — Sum of `current_balance` across all accounts. Single large number. Label "Total balance". If no accounts linked, show an empty state with a prompt to connect.

3. **Over-budget alerts** (only rendered if any category has `spent > budget_limit > 0`) — compact list of categories in a "Budgets" card, each row showing the category icon + name + "over by $X" in red. No tap action (navigation to Budget tab is out of scope for v1).

4. **Recent Transactions card** — Last 5 transactions from `plaidService.getTransactions(5)`. Each row: merchant name (fallback to name), amount (positive = expense in white, negative = income in green), date. Tapping any row navigates to `TransactionDetail` (already wired in the navigator). If no transactions, show "No transactions yet".

**Refresh:** Pull-to-refresh re-fetches all three in parallel.

**Loading state:** Single `ActivityIndicator` centered while any of the three requests are in flight.

**Error state:** Silent `console.error` only — no error UI; if data fails to load the relevant section renders empty.

## MainScreen integration

Add `HomeScreen` as the first entry in `TABS` in `MainScreen.tsx`:

```ts
const TABS = [
  { label: 'Home', screen: <HomeScreen /> },
  { label: 'Accounts', screen: <AccountsScreen /> },
  { label: 'Budget', screen: <BudgetScreen /> },
];
```

The pager handles the rest — Home becomes the default landing tab.

## Visual style

Match existing app conventions exactly:
- `#0f172a` background, `#2dd4a7` accent
- GLASS card style: `rgba(255,255,255,0.06)` fill, `rgba(255,255,255,0.1)` border, `borderRadius: 16`
- Card title style: 12px uppercase, `rgba(255,255,255,0.4)`
- Section gap: 14px, horizontal padding: 16px
- Income values: `#4ade80`; over-budget / negative: `#ef4444` or `#fca5a5`

## Testing

No new backend code — no new backend tests.

Mobile: manual verification — `HomeScreen` renders with mock data, pull-to-refresh works, tapping a transaction navigates to `TransactionDetail`.

## Out of scope (v1)

- Syncing from the home tab
- Navigating to the Budget tab from over-budget alerts
- Sparkline / mini chart on the home tab
- Week-over-week comparison

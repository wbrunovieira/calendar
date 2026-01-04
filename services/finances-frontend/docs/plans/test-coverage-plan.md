# Test Coverage Plan - Finances Frontend

## Overview

This document outlines the test coverage plan for the finances-frontend application.
Mark items as completed by changing `[ ]` to `[x]`.

**Target Coverage:** 80%+
**Framework:** Vitest + React Testing Library
**Last Updated:** 2026-01-04

---

## 1. Setup & Configuration

- [ ] Install testing dependencies (vitest, @testing-library/react, @testing-library/user-event, jsdom)
- [ ] Configure vitest.config.ts
- [ ] Create test setup file with providers and mocks
- [ ] Configure MSW (Mock Service Worker) for API mocking
- [ ] Add test scripts to package.json

---

## 2. Unit Tests - Domain Logic

### 2.1 Currency Formatting
- [ ] Test Brazilian Real (BRL) formatting
- [ ] Test negative values display
- [ ] Test zero values
- [ ] Test large numbers formatting

### 2.2 Date Calculations
- [ ] Test date range calculations (30/90 days)
- [ ] Test month period generation
- [ ] Test date formatting (pt-BR locale)

### 2.3 Forecast Calculations (plan/page.tsx)
- [ ] Test recurring transaction expansion (DAILY frequency)
- [ ] Test recurring transaction expansion (WEEKLY frequency)
- [ ] Test recurring transaction expansion (MONTHLY frequency)
- [ ] Test BYMONTHDAY rule parsing
- [ ] Test date clamping with startOn/endOn
- [ ] Test income vs expense totals calculation
- [ ] Test balance projection

### 2.4 Account Filtering
- [ ] Test filtering accounts by profile
- [ ] Test separating regular accounts from investments
- [ ] Test total balance calculation
- [ ] Test investment balance calculation

---

## 3. Component Tests

### 3.1 AppLayout
- [ ] Test renders header with title
- [ ] Test navigation links render correctly
- [ ] Test "Voltar ao Calendar" link
- [ ] Test children content rendering

### 3.2 QuickExpense
- [ ] Test renders all form fields
- [ ] Test profile selection changes categories
- [ ] Test account dropdown filters by profile
- [ ] Test category dropdown shows only EXPENSE type
- [ ] Test form validation (empty fields)
- [ ] Test form validation (amount <= 0)
- [ ] Test successful submission
- [ ] Test form reset after submission
- [ ] Test loading state during submission

### 3.3 BankAccountModal
- [ ] Test renders in create mode
- [ ] Test renders in edit mode with pre-filled data
- [ ] Test account type selection
- [ ] Test investment type fields appear for INVESTMENT accounts
- [ ] Test yield type selection
- [ ] Test quota input modes (total vs price)
- [ ] Test automatic quota price calculation
- [ ] Test automatic total calculation
- [ ] Test linked account selection for CREDIT_CARD
- [ ] Test linked account selection for INVESTMENT
- [ ] Test form validation
- [ ] Test successful creation
- [ ] Test successful update
- [ ] Test modal close on cancel

### 3.4 InvestmentAccountInfo
- [ ] Test renders investment type label
- [ ] Test renders yield type info
- [ ] Test renders maturity date when present
- [ ] Test renders quota information
- [ ] Test renders linked account name
- [ ] Test handles missing optional fields

### 3.5 TransactionList
- [ ] Test renders empty state
- [ ] Test renders transaction items
- [ ] Test expense styling (negative, rose color)
- [ ] Test income styling (positive, emerald color)
- [ ] Test date formatting
- [ ] Test category display

### 3.6 BudgetProgressCard
- [ ] Test renders budget name
- [ ] Test renders progress bar
- [ ] Test progress percentage calculation
- [ ] Test over-budget styling
- [ ] Test remaining amount display

### 3.7 BalanceCard
- [ ] Test renders account name
- [ ] Test renders current balance
- [ ] Test positive balance styling
- [ ] Test negative balance styling

### 3.8 CategoryBadge
- [ ] Test renders category name
- [ ] Test renders with correct color/style

### 3.9 MonthSelector
- [ ] Test renders current month
- [ ] Test previous month navigation
- [ ] Test next month navigation
- [ ] Test month change callback

---

## 4. Page Tests (Integration)

### 4.1 Home Page (/)
- [ ] Test initial loading state
- [ ] Test profile list rendering
- [ ] Test profile selection
- [ ] Test accounts card rendering
- [ ] Test investments card rendering (separate from accounts)
- [ ] Test QuickExpense integration
- [ ] Test recent transactions display
- [ ] Test error state handling
- [ ] Test empty state (no accounts)

### 4.2 Budgets Page (/budgets)
- [ ] Test initial loading state
- [ ] Test profile selection
- [ ] Test budget summary display
- [ ] Test progress bars accuracy
- [ ] Test month selector functionality
- [ ] Test error state handling
- [ ] Test empty state (no budgets)

### 4.3 Plan Page (/plan)
- [ ] Test initial loading state
- [ ] Test profile selection
- [ ] Test 30-day range selection
- [ ] Test 90-day range selection
- [ ] Test recurring items list
- [ ] Test period summary totals
- [ ] Test budget summary sidebar
- [ ] Test error state handling
- [ ] Test empty state (no recurring)

### 4.4 Recurring Page (/recurring)
- [ ] Test initial loading state
- [ ] Test profile selection
- [ ] Test recurring transactions table
- [ ] Test recurrence rule display
- [ ] Test next occurrence date
- [ ] Test error state handling
- [ ] Test empty state (no items)

---

## 5. API Integration Tests

### 5.1 Profile API
- [ ] Test GET /profiles success
- [ ] Test GET /profiles error handling
- [ ] Test profile data transformation

### 5.2 Bank Accounts API
- [ ] Test GET /bank-accounts success
- [ ] Test POST /bank-accounts success
- [ ] Test PUT /bank-accounts/:id success
- [ ] Test DELETE /bank-accounts/:id success
- [ ] Test error handling for all endpoints

### 5.3 Transactions API
- [ ] Test GET /transactions with filters
- [ ] Test POST /transactions success
- [ ] Test error handling

### 5.4 Recurring Transactions API
- [ ] Test GET /recurring-transactions success
- [ ] Test POST /recurring-transactions success
- [ ] Test error handling

### 5.5 Categories API
- [ ] Test GET /categories with profileId filter
- [ ] Test GET /categories with type filter
- [ ] Test error handling

### 5.6 Budgets API
- [ ] Test GET /budgets/summary success
- [ ] Test budget period parameter
- [ ] Test error handling

---

## 6. User Flow Tests (E2E-like)

### 6.1 Quick Expense Flow
- [ ] Test complete expense entry flow
- [ ] Test profile switch updates accounts and categories
- [ ] Test successful submission refreshes data

### 6.2 Account Management Flow
- [ ] Test create checking account
- [ ] Test create savings account
- [ ] Test create credit card with linked account
- [ ] Test create investment with quotas
- [ ] Test edit account
- [ ] Test delete account

### 6.3 Planning Flow
- [ ] Test view 30-day forecast
- [ ] Test switch to 90-day forecast
- [ ] Test forecast updates on profile change

---

## 7. Accessibility Tests

- [ ] Test keyboard navigation on forms
- [ ] Test focus management in modals
- [ ] Test ARIA labels on interactive elements
- [ ] Test color contrast compliance
- [ ] Test screen reader compatibility

---

## 8. Edge Cases & Error Handling

- [ ] Test network failure recovery
- [ ] Test empty API responses
- [ ] Test malformed API responses
- [ ] Test very long text truncation
- [ ] Test special characters in descriptions
- [ ] Test extreme numeric values
- [ ] Test timezone handling

---

## Progress Summary

| Category | Total | Completed | Progress |
|----------|-------|-----------|----------|
| Setup | 5 | 0 | 0% |
| Unit Tests | 16 | 0 | 0% |
| Component Tests | 43 | 0 | 0% |
| Page Tests | 28 | 0 | 0% |
| API Tests | 16 | 0 | 0% |
| User Flow Tests | 10 | 0 | 0% |
| Accessibility | 5 | 0 | 0% |
| Edge Cases | 8 | 0 | 0% |
| **Total** | **131** | **0** | **0%** |

---

## Priority Order

1. **High Priority** (Core functionality)
   - Setup & Configuration
   - QuickExpense component
   - Home Page integration
   - Forecast calculations
   - API integration tests

2. **Medium Priority** (Important features)
   - BankAccountModal component
   - Other page tests
   - User flow tests

3. **Lower Priority** (Polish)
   - Accessibility tests
   - Edge cases
   - Remaining component tests

---

## Notes

- Use MSW to mock API calls consistently
- Create shared test fixtures for profiles, accounts, categories
- Run tests in CI pipeline before merge
- Update this document as tests are implemented

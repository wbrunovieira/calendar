export type ProfileType = 'PERSONAL' | 'BUSINESS';

export interface Profile {
  id: string;
  calendarId: string;
  name: string;
  type: ProfileType;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export type AccountType =
  | 'CHECKING'
  | 'SAVINGS'
  | 'INVESTMENT'
  | 'CREDIT_CARD'
  | 'CASH'
  | 'OTHER';

export type InvestmentType =
  | 'SAVINGS_BOX'  // Caixinha (Nubank, etc.)
  | 'CDB'          // Certificado de Depósito Bancário
  | 'LCI'          // Letra de Crédito Imobiliário
  | 'LCA'          // Letra de Crédito do Agronegócio
  | 'STOCKS'       // Ações
  | 'FUNDS'        // Fundos de investimento
  | 'FII'          // Fundos Imobiliários
  | 'CRYPTO'       // Criptomoedas
  | 'TREASURY'     // Tesouro Direto
  | 'OTHER';       // Outros

export type YieldType =
  | 'FIXED'          // Taxa fixa (ex: 12% a.a.)
  | 'CDI_PERCENTAGE' // Percentual do CDI (ex: 100% CDI)
  | 'IPCA_PLUS'      // IPCA + taxa (ex: IPCA + 5%)
  | 'VARIABLE';      // Taxa variável (ações, fundos, crypto)

export interface BankAccount {
  id: string;
  profileId: string;
  name: string;
  type: AccountType;
  initialBalance: number;
  currentBalance: number;
  currency: string;
  isActive: boolean;
  bankName?: string;
  bankCode?: string;
  agency?: string;
  accountNumber?: string;
  accountDigit?: string;
  color?: string;
  icon?: string;
  description?: string;
  creditLimit?: number;
  dueDay?: number;
  closingDay?: number;
  linkedAccountId?: string;
  // Investment-specific fields
  investmentType?: InvestmentType;
  yieldType?: YieldType;
  yieldRate?: number;
  maturityDate?: string;
  broker?: string;
  numberOfQuotas?: number;
  quotaPrice?: number;
  createdAt: string;
  updatedAt: string;
}

export type CategoryType = 'INCOME' | 'EXPENSE' | 'TRANSFER';

export interface Category {
  id: string;
  profileId: string;
  name: string;
  type: CategoryType;
  color?: string;
  icon?: string;
  parentId?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export type TransactionType = 'INCOME' | 'EXPENSE' | 'TRANSFER';
export type TransactionStatus = 'PLANNED' | 'CONFIRMED' | 'CANCELLED';

export interface TransactionSplit {
  id: string;
  categoryId?: string;
  amount: number;
  memo?: string;
  createdAt: string;
}

export interface Transaction {
  id: string;
  profileId: string;
  bankAccountId: string;
  destinationAccountId?: string;
  categoryId?: string;
  type: TransactionType;
  status: TransactionStatus;
  amount: number;
  currency: string;
  description: string;
  notes?: string;
  costCenter?: string;
  occurredOn: string;
  dueOn?: string;
  recurrenceRule?: string;
  installmentNumber?: number;
  installmentTotal?: number;
  externalId?: string;
  tags?: string[];
  splits?: TransactionSplit[];
  createdAt: string;
  updatedAt: string;
}

export interface TransactionFormData {
  profileId: string;
  bankAccountId: string;
  destinationAccountId?: string;
  categoryId?: string;
  type: TransactionType;
  status?: TransactionStatus;
  amount: number;
  currency: string;
  description: string;
  notes?: string;
  costCenter?: string;
  occurredOn: string;
  dueOn?: string;
  recurrenceRule?: string;
  installmentNumber?: number;
  installmentTotal?: number;
  externalId?: string;
  tags: string[];
}

export interface TransactionFilters {
  bankAccountId: string | null;
  type: TransactionType | 'ALL';
  status: TransactionStatus | 'ALL';
  from?: string;
  to?: string;
}

export type RecurringStatus = 'ACTIVE' | 'PAUSED' | 'CANCELLED';

export interface RecurringTransaction {
  id: string;
  profileId: string;
  bankAccountId?: string;
  categoryId?: string;
  type: TransactionType;
  amount: number;
  currency: string;
  description: string;
  recurrenceRule: string;
  startOn: string;
  endOn?: string;
  nextOccurrence: string;
  status: RecurringStatus;
  reviewOn?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RecurringTransactionForm {
  profileId: string;
  bankAccountId?: string;
  categoryId?: string;
  type: TransactionType;
  amount: number;
  currency: string;
  description: string;
  frequency: 'DAILY' | 'WEEKLY' | 'MONTHLY';
  startOn: string;
  endOn?: string;
  nextOccurrence: string;
  notes?: string;
}

export interface BudgetTarget {
  id: string;
  profileId: string;
  categoryId: string;
  periodStart: string;
  amount: number;
  notes?: string;
  isRecurring: boolean;
  effectiveUntil?: string;
  createdAt: string;
  updatedAt: string;
}

export interface BudgetTargetForm {
  profileId: string;
  categoryId: string;
  period: string; // YYYY-MM
  amount: number;
  notes?: string;
  isRecurring?: boolean;
}

export interface BudgetSummaryItem {
  target: BudgetTarget;
  spent: number;
  remaining: number;
}

export type InvoiceStatus = 'OPEN' | 'CLOSED' | 'PAID';

export interface Invoice {
  id: string;
  bankAccountId: string;
  referenceDate: string;
  openingDate: string;
  closingDate: string;
  dueDate: string;
  amount: number;
  status: InvoiceStatus;
  paidAt?: string;
  paidAmount?: number;
  createdAt: string;
  updatedAt: string;
}

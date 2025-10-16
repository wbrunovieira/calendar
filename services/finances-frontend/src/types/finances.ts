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

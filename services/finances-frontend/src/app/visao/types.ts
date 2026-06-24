export type Trend = 'up' | 'down' | 'flat';

export interface CategoryTrend {
  categoryId: string;
  name: string;
  byMonth: number[];
  total: number;
  average: number;
  deltaVsAvg: number;
  trend: Trend;
  children?: CategoryTrend[];
}

export interface Mover {
  name: string;
  delta: number;
}

export interface ExpenseAnalysis {
  periods: string[];
  totalByMonth: number[];
  average: number;
  fixedByMonth: number[];
  variableByMonth: number[];
  fixedAverage: number;
  variableAverage: number;
  categories: CategoryTrend[];
  topUp: Mover[];
  topDown: Mover[];
}

export interface DRELine {
  classification: string;
  label: string;
  kind: 'revenue' | 'expense';
  byMonth: number[];
  total: number;
}

export interface FinancialSummary {
  periods: string[];
  revenueByMonth: number[];
  financialIncomeByMonth: number[];
  expenseByMonth: number[];
  resultByMonth: number[];
  marginByMonth: number[];
  totalRevenue: number;
  totalFinancialIncome: number;
  totalExpense: number;
  totalResult: number;
  avgMargin: number;
  expenseCategories: CategoryTrend[];
  revenueCategories: CategoryTrend[];
  dre: DRELine[];
}

export interface CapitalSummary {
  totalContributed: number;
  totalWithdrawn: number;
  totalLoaned: number;
  totalReturned: number;
  outstandingDebt: number;
}

export type ViewMode = 3 | 6 | 12;

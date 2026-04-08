export type ProfileType = 'PERSONAL' | 'BUSINESS';
export type LegalEntityType = 'MEI' | 'ME' | 'EPP' | 'LTDA' | 'SA';
export type TaxRegime = 'SIMPLES' | 'LUCRO_PRESUMIDO' | 'LUCRO_REAL';

export interface Profile {
  id: string;
  calendarId: string;
  name: string;
  type: ProfileType;
  isActive: boolean;
  // PJ-specific (nullable)
  legalEntityType?: LegalEntityType;
  companyName?: string;
  cnpj?: string;
  simplesNacional?: boolean;
  taxRegime?: TaxRegime;
  dasAliquota?: number;
  openingDate?: string;
  createdAt: string;
  updatedAt: string;
}

export type AccountType =
  | 'CHECKING'
  | 'SAVINGS'
  | 'INVESTMENT'
  | 'CREDIT_CARD'
  | 'CASH'
  | 'EXCHANGE'
  | 'WALLET'
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
  displayOrder?: number;
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
export type ClassificationDRE =
  | 'REVENUE'
  | 'TAX'
  | 'FIXED_COST'
  | 'VARIABLE_COST'
  | 'PROLABORE'
  | 'MARKETING'
  | 'FINANCIAL'
  | 'ASSET'
  | 'CAPITAL';

export interface Category {
  id: string;
  profileId: string;
  name: string;
  type: CategoryType;
  color?: string;
  icon?: string;
  parentId?: string;
  classificationDRE?: ClassificationDRE;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export type CostCenterType = 'CLIENT' | 'PROJECT' | 'DEPARTMENT';

export interface CostCenter {
  id: string;
  profileId: string;
  name: string;
  type: CostCenterType;
  color?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export type CampaignPlatform = 'META_ADS' | 'GOOGLE_ADS' | 'INSTAGRAM' | 'LINKEDIN' | 'OTHER';

export interface MarketingCampaign {
  id: string;
  profileId: string;
  name: string;
  platform: CampaignPlatform;
  startDate: string;
  endDate?: string;
  budget: number;
  revenueAttributed: number;
  leads: number;
  conversions: number;
  notes?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface MarketingCampaignMetrics extends MarketingCampaign {
  totalSpent: number;
  roi: number;
  costPerLead: number;
  costPerConversion: number;
}

export type ContributionType = 'CONTRIBUTION' | 'WITHDRAWAL' | 'LOAN';

export interface CapitalContribution {
  id: string;
  profileId: string;
  type: ContributionType;
  amount: number;
  date: string;
  description: string;
  sourceAccountId?: string;
  notes?: string;
  isReturned: boolean;
  returnedAmount: number;
  returnedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CapitalContributionSummary {
  totalContributed: number;
  totalWithdrawn: number;
  totalLoaned: number;
  totalReturned: number;
  outstandingDebt: number;
}

export type AssetCategory = 'HARDWARE' | 'SOFTWARE' | 'FURNITURE' | 'VEHICLE' | 'REAL_ESTATE' | 'OTHER';

export interface CompanyAsset {
  id: string;
  profileId: string;
  name: string;
  category: AssetCategory;
  purchaseDate: string;
  purchaseAmount: number;
  currentValue: number;
  depreciationRate: number;
  linkedTransactionId?: string;
  notes?: string;
  isActive: boolean;
  disposalDate?: string;
  disposalAmount?: number;
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
  reminderOn?: string;
  recurrenceRule?: string;
  installmentNumber?: number;
  installmentTotal?: number;
  externalId?: string;
  linkedTransactionId?: string;
  isPersonalReimbursement?: boolean;
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
  destinationCategoryId?: string;
  type: TransactionType;
  status?: TransactionStatus;
  amount: number;
  currency: string;
  description: string;
  notes?: string;
  costCenter?: string;
  isPersonalReimbursement?: boolean;
  occurredOn: string;
  dueOn?: string;
  reminderOn?: string;
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
  confirmedAmount: number;
  plannedAmount: number;
  status: InvoiceStatus;
  paidAt?: string;
  paidAmount?: number;
  createdAt: string;
  updatedAt: string;
}

export type GoalPriority = 'HIGH' | 'MEDIUM' | 'LOW';
export type GoalStatus = 'ACTIVE' | 'COMPLETED' | 'CANCELLED';
export type GoalType =
  | 'PERSONAL_SAVINGS'
  | 'OPERATIONAL_RESERVE'
  | 'TAX_FUND'
  | 'INVESTMENT_FUND'
  | 'REVENUE_TARGET';

export interface Goal {
  id: string;
  profileId: string;
  categoryId?: string;
  name: string;
  description: string;
  targetAmount: number;
  currentAmount: number;
  priority: GoalPriority;
  targetDate?: string;
  status: GoalStatus;
  link?: string;
  goalType: GoalType;
  displayOrder?: number;
  createdAt: string;
  updatedAt: string;
}

export interface CryptoPurchaseWithGains {
  id: string;
  asset: string;
  quantity: number;
  priceUsd: number;
  exchangeRate: number;
  investedBrl: number;
  investedUsd: number;
  occurredOn: string;
  currentPriceUsd: number;
  currentExchangeRate: number;
  currentValueUsd: number;
  currentValueBrl: number;
  gainCryptoUsd: number;
  gainCryptoPercent: number;
  gainExchangeBrl: number;
  gainExchangePercent: number;
  gainTotalBrl: number;
  gainTotalPercent: number;
}

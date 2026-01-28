'use client';

import { BankAccount, InvestmentType, YieldType } from '@/types/finances';

interface InvestmentAccountInfoProps {
  account: BankAccount;
  linkedAccount?: BankAccount;
  onEdit?: () => void;
}

const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

const formatPercent = (value: number) => {
  if (value === Math.floor(value)) {
    return `${value}%`;
  }
  return `${value.toFixed(2)}%`;
};

const getInvestmentTypeLabel = (type: InvestmentType): { label: string; icon: string } => {
  switch (type) {
    case 'SAVINGS_BOX':
      return { label: 'Caixinha', icon: '🐷' };
    case 'CDB':
      return { label: 'CDB', icon: '📜' };
    case 'LCI':
      return { label: 'LCI', icon: '🏠' };
    case 'LCA':
      return { label: 'LCA', icon: '🌾' };
    case 'STOCKS':
      return { label: 'Ações', icon: '📊' };
    case 'FUNDS':
      return { label: 'Fundos', icon: '📦' };
    case 'FII':
      return { label: 'FII', icon: '🏢' };
    case 'CRYPTO':
      return { label: 'Crypto', icon: '₿' };
    case 'TREASURY':
      return { label: 'Tesouro', icon: '🏛️' };
    default:
      return { label: 'Outro', icon: '📈' };
  }
};

const formatQuotas = (value: number): string => {
  if (value === Math.floor(value)) {
    return value.toString();
  }
  return value.toFixed(6).replace(/\.?0+$/, '');
};

const getYieldDescription = (yieldType?: YieldType, yieldRate?: number): string => {
  if (!yieldType) return '';

  const rate = yieldRate || 0;
  switch (yieldType) {
    case 'FIXED':
      return `${formatPercent(rate)} a.a.`;
    case 'CDI_PERCENTAGE':
      return `${formatPercent(rate)} do CDI`;
    case 'IPCA_PLUS':
      return `IPCA + ${formatPercent(rate)}`;
    case 'VARIABLE':
      return 'Variável';
    default:
      return '';
  }
};

const formatDate = (dateStr?: string) => {
  if (!dateStr) return null;
  const date = new Date(dateStr);
  return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
};

const getDaysToMaturity = (dateStr?: string): number | null => {
  if (!dateStr) return null;
  const maturity = new Date(dateStr);
  const today = new Date();
  const diff = maturity.getTime() - today.getTime();
  if (diff < 0) return 0;
  return Math.ceil(diff / (1000 * 60 * 60 * 24));
};

export default function InvestmentAccountInfo({
  account,
  linkedAccount,
  onEdit,
}: InvestmentAccountInfoProps) {
  const investmentInfo = account.investmentType
    ? getInvestmentTypeLabel(account.investmentType)
    : { label: 'Investimento', icon: '📈' };

  const yieldDescription = getYieldDescription(account.yieldType, account.yieldRate);
  const daysToMaturity = getDaysToMaturity(account.maturityDate);
  const maturityDate = formatDate(account.maturityDate);

  // Calculate return
  const returnAmount = account.currentBalance - account.initialBalance;
  const returnPercent = account.initialBalance > 0
    ? ((returnAmount / account.initialBalance) * 100)
    : 0;
  const isPositiveReturn = returnAmount >= 0;

  return (
    <div className="border border-white/10 rounded-xl overflow-hidden bg-gradient-to-br from-white/5 to-white/[0.02]">
      {/* Header */}
      <div
        className="p-4 flex items-center justify-between cursor-pointer hover:bg-white/5 transition-colors"
        onClick={onEdit}
        style={{ borderLeft: `4px solid ${account.color || '#8b5cf6'}` }}
      >
        <div className="flex items-center gap-3">
          <span className="text-2xl">{account.icon || investmentInfo.icon}</span>
          <div>
            <p className="text-white font-semibold">{account.name}</p>
            <div className="flex items-center gap-2 text-white/50 text-xs">
              <span className="px-2 py-0.5 rounded-full bg-purple-500/20 text-purple-300">
                {investmentInfo.label}
              </span>
              {account.broker && <span>{account.broker}</span>}
            </div>
          </div>
        </div>
        <div className="text-right">
          <p className="text-white font-bold text-lg">{formatCurrency(account.currentBalance)}</p>
          {yieldDescription && (
            <p className="text-white/50 text-xs">{yieldDescription}</p>
          )}
        </div>
      </div>

      {/* Details */}
      <div className="px-4 pb-4 space-y-3">
        {/* Quota Info - for stocks, FII, funds, crypto */}
        {account.numberOfQuotas && account.numberOfQuotas > 0 && (
          <div className="grid grid-cols-3 gap-3">
            <div className="bg-white/5 rounded-lg p-3">
              <p className="text-white/50 text-xs mb-1">Cotas</p>
              <p className="text-white font-semibold">{formatQuotas(account.numberOfQuotas)}</p>
            </div>
            <div className="bg-white/5 rounded-lg p-3">
              <p className="text-white/50 text-xs mb-1">Preço/Cota</p>
              <p className="text-white font-semibold">
                {account.quotaPrice ? formatCurrency(account.quotaPrice) : '-'}
              </p>
            </div>
            <div className="bg-white/5 rounded-lg p-3">
              <p className="text-white/50 text-xs mb-1">Total</p>
              <p className="text-white font-semibold">{formatCurrency(account.currentBalance)}</p>
            </div>
          </div>
        )}

        {/* Return Info */}
        <div className="grid grid-cols-2 gap-3">
          <div className="bg-white/5 rounded-lg p-3">
            <p className="text-white/50 text-xs mb-1">Aplicado</p>
            <p className="text-white font-semibold">{formatCurrency(account.initialBalance)}</p>
          </div>
          <div className="bg-white/5 rounded-lg p-3">
            <p className="text-white/50 text-xs mb-1">Rendimento</p>
            <p className={`font-bold ${isPositiveReturn ? 'text-emerald-400' : 'text-red-400'}`}>
              {isPositiveReturn ? '+' : ''}{formatCurrency(returnAmount)}
            </p>
            <p className={`text-xs ${isPositiveReturn ? 'text-emerald-400/70' : 'text-red-400/70'}`}>
              {isPositiveReturn ? '+' : ''}{formatPercent(returnPercent)}
            </p>
          </div>
        </div>

        {/* Maturity Info */}
        {maturityDate && (
          <div className="bg-white/5 rounded-lg p-3 flex items-center justify-between">
            <div>
              <p className="text-white/50 text-xs">Vencimento</p>
              <p className="text-white font-semibold">{maturityDate}</p>
            </div>
            {daysToMaturity !== null && (
              <div className="text-right">
                {daysToMaturity === 0 ? (
                  <span className="px-3 py-1 rounded-full bg-amber-500/20 text-amber-300 text-sm">
                    Vencido
                  </span>
                ) : (
                  <span className="text-white/70 text-sm">
                    {daysToMaturity} {daysToMaturity === 1 ? 'dia' : 'dias'}
                  </span>
                )}
              </div>
            )}
          </div>
        )}

        {/* Linked Account Info */}
        {linkedAccount && (
          <div className="bg-white/5 rounded-lg p-3 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="text-lg">{linkedAccount.icon || '🏦'}</span>
              <div>
                <p className="text-white/50 text-xs">Conta de origem</p>
                <p className="text-white font-semibold text-sm">{linkedAccount.name}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-white/50 text-xs">{linkedAccount.bankName || linkedAccount.type}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

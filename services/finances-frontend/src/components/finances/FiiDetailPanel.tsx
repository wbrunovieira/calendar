'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { formatCurrency, formatDate } from '@/utils/format';
import type { BankAccount, Transaction } from '@/types/finances';

const formatMonthYear = (dateStr: string) => {
  const raw = dateStr.includes('T') ? dateStr : dateStr + 'T12:00:00';
  const d = new Date(raw);
  return d.toLocaleDateString('pt-BR', { month: 'short', year: '2-digit' });
};

interface FiiDetailPanelProps {
  account: BankAccount;
  clearAccountId: string;
  profileId: string;
}

export default function FiiDetailPanel({ account, clearAccountId, profileId }: FiiDetailPanelProps) {
  const [purchases, setPurchases] = useState<Transaction[]>([]);
  const [dividends, setDividends] = useState<Transaction[]>([]);
  const [benchmarks, setBenchmarks] = useState<{ cdi: number; ibovespa: number | null; poupanca: number; ifix: number | null; days: number } | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const params = new URLSearchParams({
          profileId,
          bankAccountId: clearAccountId,
          includeAsDestination: 'true',
        });
        let firstDate: string | null = null;

        {
          const data = await api.get<{ data: Transaction[] }>(`/transactions?${params}`);
          const allTx: Transaction[] = data.data || [];

          const fiiPurchases = allTx
            .filter((tx) => tx.type === 'TRANSFER' && tx.destinationAccountId === account.id)
            .sort((a, b) => a.occurredOn.localeCompare(b.occurredOn));
          setPurchases(fiiPurchases);

          const fiiDividends = allTx
            .filter((tx) => tx.type === 'INCOME' && tx.description.includes(`Dividendo ${account.name}`))
            .sort((a, b) => a.occurredOn.localeCompare(b.occurredOn));
          setDividends(fiiDividends);

          if (fiiPurchases.length > 0) {
            firstDate = fiiPurchases[0].occurredOn.split('T')[0];
          }
        }

        if (firstDate) {
          try {
            const benchData = await api.get<{ data: { cdi: number; ibovespa: number | null; poupanca: number; ifix: number | null; days: number } }>(`/benchmarks/returns?from=${firstDate}`);
            setBenchmarks(benchData.data);
          } catch {
            // Benchmark data is optional
          }
        }
      } catch (e) {
        console.warn('Error loading FII details:', e);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [account.id, clearAccountId, profileId, account.name]);

  if (loading) {
    return <div className="py-3 text-center text-white/40 text-xs">Carregando detalhes...</div>;
  }

  const totalInvested = account.initialBalance;
  const currentValue = account.currentBalance;
  const priceGain = currentValue - totalInvested;
  const priceGainPct = totalInvested > 0 ? (priceGain / totalInvested) * 100 : 0;

  const totalDividends = dividends.reduce((sum, d) => sum + d.amount, 0);
  const dividendMonths = dividends.length > 0
    ? Math.max(1, Math.ceil(
        (new Date(dividends[dividends.length - 1].occurredOn).getTime() -
          new Date(dividends[0].occurredOn).getTime()) / (30.44 * 24 * 60 * 60 * 1000)
      ) + 1)
    : 0;
  const monthlyAvgDividend = dividendMonths > 0 ? totalDividends / dividendMonths : 0;
  const annualDividendYield = totalInvested > 0 ? (monthlyAvgDividend * 12 / totalInvested) * 100 : 0;

  const totalReturn = priceGain + totalDividends;
  const totalReturnPct = totalInvested > 0 ? (totalReturn / totalInvested) * 100 : 0;

  const firstPurchaseDate = purchases.length > 0 ? new Date(purchases[0].occurredOn) : null;
  const daysInvested = firstPurchaseDate ? Math.ceil((Date.now() - firstPurchaseDate.getTime()) / (24 * 60 * 60 * 1000)) : 0;
  const annualizedReturn = daysInvested > 30 && totalInvested > 0
    ? (Math.pow(1 + totalReturn / totalInvested, 365 / daysInvested) - 1) * 100
    : totalReturnPct;

  const currentPricePerQuota = account.numberOfQuotas && account.numberOfQuotas > 0
    ? currentValue / account.numberOfQuotas
    : 0;

  const parsePurchase = (tx: Transaction) => {
    const match = tx.description.match(/(\d+)\s*cotas/);
    const cotas = match ? parseInt(match[1]) : 0;
    const pricePerCota = cotas > 0 ? tx.amount / cotas : 0;
    const currentVal = cotas * currentPricePerQuota;
    const gain = currentVal - tx.amount;
    const gainPct = tx.amount > 0 ? (gain / tx.amount) * 100 : 0;
    return { cotas, pricePerCota, currentVal, gain, gainPct };
  };

  const gainColor = (v: number) => v >= 0 ? 'text-emerald-400' : 'text-red-400';
  const gainSign = (v: number) => v >= 0 ? '+' : '';

  const recentDividends = [...dividends].reverse().slice(0, 6);

  return (
    <div className="mt-2 space-y-3 px-1">
      {purchases.length > 0 && (
        <div className="bg-white/[0.03] rounded-lg p-3">
          <p className="text-white/50 text-xs font-semibold uppercase tracking-wider mb-2">Compras</p>
          <div className="space-y-2">
            {purchases.map((tx) => {
              const p = parsePurchase(tx);
              return (
                <div key={tx.id} className="flex items-center justify-between text-xs">
                  <div className="flex items-center gap-3">
                    <span className="text-white/40">{formatDate(tx.occurredOn)}</span>
                    <span className="text-white/70">{p.cotas} cotas × {formatCurrency(p.pricePerCota)}</span>
                    <span className="text-white/40">= {formatCurrency(tx.amount)}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-white/50">Atual: {formatCurrency(p.currentVal)}</span>
                    <span className={`font-medium ${gainColor(p.gain)}`}>
                      {gainSign(p.gain)}{p.gainPct.toFixed(1)}%
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {dividends.length > 0 && (
        <div className="bg-white/[0.03] rounded-lg p-3">
          <p className="text-white/50 text-xs font-semibold uppercase tracking-wider mb-2">Dividendos</p>
          <div className="grid grid-cols-3 gap-3 mb-3">
            <div>
              <p className="text-white/30 text-[10px]">Total recebido</p>
              <p className="text-emerald-400 text-sm font-medium">{formatCurrency(totalDividends)}</p>
            </div>
            <div>
              <p className="text-white/30 text-[10px]">Media mensal</p>
              <p className="text-white/80 text-sm font-medium">{formatCurrency(monthlyAvgDividend)}</p>
            </div>
            <div>
              <p className="text-white/30 text-[10px]">Yield anual</p>
              <p className="text-amber-400 text-sm font-medium">{annualDividendYield.toFixed(2)}% a.a.</p>
            </div>
          </div>
          <div className="flex flex-wrap gap-x-4 gap-y-1">
            {recentDividends.map((d) => (
              <div key={d.id} className="flex items-center gap-2 text-xs">
                <span className="text-white/30">{formatMonthYear(d.occurredOn)}</span>
                <span className="text-emerald-400/80">{formatCurrency(d.amount)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="bg-white/[0.03] rounded-lg p-3">
        <p className="text-white/50 text-xs font-semibold uppercase tracking-wider mb-2">Retorno Total</p>
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between">
            <span className="text-white/50">Valorizacao</span>
            <span className={gainColor(priceGain)}>
              {gainSign(priceGain)}{formatCurrency(priceGain)} ({gainSign(priceGainPct)}{priceGainPct.toFixed(2)}%)
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-white/50">Dividendos</span>
            <span className="text-emerald-400">
              +{formatCurrency(totalDividends)} (+{totalInvested > 0 ? ((totalDividends / totalInvested) * 100).toFixed(2) : '0'}%)
            </span>
          </div>
          <div className="border-t border-white/10 pt-1.5 flex justify-between font-medium">
            <span className="text-white/70">Total</span>
            <span className={gainColor(totalReturn)}>
              {gainSign(totalReturn)}{formatCurrency(totalReturn)} ({gainSign(totalReturnPct)}{totalReturnPct.toFixed(2)}%)
            </span>
          </div>
          {daysInvested > 30 && (
            <div className="flex justify-between">
              <span className="text-white/40">Anualizado ({Math.floor(daysInvested / 30.44)}m)</span>
              <span className={`font-medium ${gainColor(annualizedReturn)}`}>
                {gainSign(annualizedReturn)}{annualizedReturn.toFixed(2)}% a.a.
              </span>
            </div>
          )}
        </div>

        {benchmarks && (
          <div className="mt-3 pt-2 border-t border-white/10 space-y-1.5 text-xs">
            <p className="text-white/40 text-[10px] uppercase tracking-wider mb-1">Comparativo no periodo</p>
            {[
              { label: 'CDI', value: benchmarks.cdi },
              { label: 'Poupanca', value: benchmarks.poupanca },
              ...(benchmarks.ibovespa != null ? [{ label: 'Ibovespa', value: benchmarks.ibovespa }] : []),
              ...(benchmarks.ifix != null ? [{ label: 'IFIX', value: benchmarks.ifix }] : []),
            ].map(({ label, value }) => {
              const diff = totalReturnPct - value;
              const isAbove = diff >= 0;
              return (
                <div key={label} className="flex justify-between items-center">
                  <span className="text-white/50">{label} ({value.toFixed(1)}%)</span>
                  <span className={`font-medium ${isAbove ? 'text-emerald-400' : 'text-red-400'}`}>
                    {isAbove ? '▲' : '▼'} {isAbove ? 'acima' : 'abaixo'} ({gainSign(diff)}{diff.toFixed(1)}pp)
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

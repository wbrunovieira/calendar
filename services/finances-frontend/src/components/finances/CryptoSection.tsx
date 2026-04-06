'use client';

import { useState } from 'react';
import {
  DndContext,
  closestCenter,
  type DragEndEvent,
} from '@dnd-kit/core';
import type { SensorDescriptor, SensorOptions } from '@dnd-kit/core';
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { formatCurrency } from '@/utils/format';
import { DragHandle, SortableItem } from '@/components/finances/SortableHelpers';
import ExpandedTransactionPanel from '@/components/finances/ExpandedTransactionPanel';
import type { BankAccount, Category, CryptoPurchaseWithGains, Transaction } from '@/types/finances';

type BotTab = 'all' | 'MACross1' | 'MemeCoin1';

interface CryptoSectionProps {
  cryptoAccounts: BankAccount[];
  cryptoPrices: { usdBrl: number; prices: Record<string, { priceUsd: number; priceBrl: number }> } | null;
  cryptoPurchases: Record<string, CryptoPurchaseWithGains[]>;
  expandedAccountId: string | null;
  selectedProfileId: string | null;
  categories: Category[];
  sensors: SensorDescriptor<SensorOptions>[];
  getSubAssets: (parentId: string) => BankAccount[];
  onToggleExpand: (accountId: string) => void;
  onDragEnd: (event: DragEndEvent, accountList: BankAccount[]) => void;
  onEditAccount: (account: BankAccount) => void;
  onAddTransaction: (accountId: string) => void;
  onEditTransaction: (tx: Transaction) => void;
  onDeleteTransaction: (tx: Transaction) => void;
  onConfirmTransaction?: (tx: Transaction) => Promise<void> | void;
}

export default function CryptoSection({
  cryptoAccounts,
  cryptoPrices,
  cryptoPurchases,
  expandedAccountId,
  selectedProfileId,
  categories,
  sensors,
  getSubAssets,
  onToggleExpand,
  onDragEnd,
  onEditAccount,
  onAddTransaction,
  onEditTransaction,
  onDeleteTransaction,
  onConfirmTransaction,
}: CryptoSectionProps) {
  const [activeBot, setActiveBot] = useState<BotTab>('all');

  if (cryptoAccounts.length === 0) return null;

  const botTabs: { key: BotTab; label: string }[] = [
    { key: 'all', label: 'Todas' },
    { key: 'MACross1', label: 'MACross1 (BRL)' },
    { key: 'MemeCoin1', label: 'MemeCoin1 (USD)' },
  ];

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xl">🪙</span>
          <h3 className="text-lg font-semibold text-white">Cripto</h3>
        </div>
        <div className="flex items-center gap-3">
          {cryptoPrices && (
            <span className="text-white/40 text-xs">
              USD/BRL: {cryptoPrices.usdBrl.toFixed(2)}
            </span>
          )}
          <span className="text-white/50 text-xs">
            {cryptoAccounts.length} {cryptoAccounts.length === 1 ? 'conta' : 'contas'}
          </span>
        </div>
      </div>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={(event) => onDragEnd(event, cryptoAccounts)}
      >
        <SortableContext items={cryptoAccounts.map((a) => a.id)} strategy={verticalListSortingStrategy}>
          {cryptoAccounts.map((account) => {
            const isExpanded = expandedAccountId === account.id;
            const typeLabel = account.type === 'EXCHANGE' ? 'Corretora' : 'Carteira';
            const subAssets = getSubAssets(account.id);
            const subAssetsTotalBrl = subAssets.reduce((sum, sub) => {
              const sym = sub.name.match(/\(([A-Z]+)\)/)?.[1];
              const pi = sym && cryptoPrices?.prices[sym];
              if (pi && sub.numberOfQuotas != null) return sum + sub.numberOfQuotas * pi.priceBrl;
              return sum + (sub.currentBalance || 0);
            }, 0);
            const totalBrl = account.currentBalance + subAssetsTotalBrl;
            const totalUsd = cryptoPrices?.usdBrl ? totalBrl / cryptoPrices.usdBrl : null;
            return (
              <SortableItem key={account.id} id={account.id}>
                {({ listeners, attributes }) => (
                  <div>
                    <div className="flex items-center gap-1">
                      <DragHandle listeners={listeners} attributes={attributes} />
                      <div
                        className="cursor-pointer flex-1 rounded-xl bg-gradient-to-r from-white/[0.06] to-white/[0.02] border border-white/10 p-4 transition-all duration-300 hover:border-white/20"
                        style={isExpanded ? {
                          boxShadow: '0 0 20px rgba(52, 211, 153, 0.3), 0 0 0 2px #34d399',
                        } : undefined}
                        onClick={() => onToggleExpand(account.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div>
                            <div className="flex items-center gap-2">
                              <p className="text-white font-semibold">{account.name}</p>
                              <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-400">
                                {typeLabel}
                              </span>
                              {subAssets.length > 0 && (
                                <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-white/10 text-white/50">
                                  {subAssets.length} {subAssets.length === 1 ? 'ativo' : 'ativos'}
                                </span>
                              )}
                            </div>
                            {account.broker && (
                              <p className="text-white/40 text-xs">{account.broker}</p>
                            )}
                          </div>
                          <div className="text-right">
                            <p className="text-white font-bold">{formatCurrency(totalBrl)}</p>
                            {totalUsd != null && (
                              <p className="text-white/40 text-xs">{formatCurrency(totalUsd, 'USD')}</p>
                            )}
                            <button
                              onClick={(e) => { e.stopPropagation(); onEditAccount(account); }}
                              className="text-white/30 hover:text-white/70 text-xs transition-colors"
                            >
                              Editar
                            </button>
                          </div>
                        </div>
                        {(subAssets.length > 0 || account.currentBalance > 0) && (
                          <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
                            {account.currentBalance > 0 && (
                              <div className="px-2 py-2 rounded-lg bg-white/[0.04]">
                                <div className="flex items-center justify-between">
                                  <div className="flex items-center gap-2">
                                    <span className="text-emerald-400 text-xs">●</span>
                                    <span className="text-white/80 text-sm font-medium">Saldo em Reais</span>
                                    <span className="text-white/40 text-xs">BRL</span>
                                  </div>
                                  <div className="text-right">
                                    <span className="text-white/80 text-sm font-medium">{formatCurrency(account.currentBalance)}</span>
                                    {cryptoPrices?.usdBrl && (
                                      <p className="text-white/40 text-xs">{formatCurrency(account.currentBalance / cryptoPrices.usdBrl, 'USD')}</p>
                                    )}
                                  </div>
                                </div>
                              </div>
                            )}
                            {subAssets.map((sub) => {
                              const symbolMatch = sub.name.match(/\(([A-Z]+)\)/);
                              const symbol = symbolMatch ? symbolMatch[1] : '';
                              const priceInfo = symbol && cryptoPrices?.prices[symbol];
                              const purchases = symbol ? (cryptoPurchases[symbol] || []) : [];
                              const totalInvestedBrl = purchases.reduce((sum, p) => sum + p.investedBrl, 0);
                              const totalGainBrl = purchases.reduce((sum, p) => sum + p.gainTotalBrl, 0);
                              const totalGainPercent = totalInvestedBrl > 0 ? (totalGainBrl / totalInvestedBrl) * 100 : 0;
                              const gainCryptoPercent = purchases.length > 0 ? purchases[0].gainCryptoPercent : null;
                              const gainExchangePercent = purchases.length > 0 ? purchases[0].gainExchangePercent : null;
                              return (
                                <div key={sub.id} className="px-2 py-2 rounded-lg bg-white/[0.04]">
                                  <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                      <span className="text-amber-400 text-xs">●</span>
                                      <span className="text-white/80 text-sm font-medium">{sub.name.replace(account.name + ' - ', '')}</span>
                                      {sub.numberOfQuotas != null && (
                                        <span className="text-white/40 text-xs">{sub.numberOfQuotas} quotas</span>
                                      )}
                                      {priceInfo && (
                                        <span className="text-white/30 text-xs">@ {formatCurrency(priceInfo.priceUsd, 'USD')}</span>
                                      )}
                                    </div>
                                    <div className="flex items-center gap-3">
                                      <div className="text-right">
                                        {priceInfo && sub.numberOfQuotas != null ? (
                                          <>
                                            <span className="text-white/80 text-sm font-medium">
                                              {formatCurrency(sub.numberOfQuotas * priceInfo.priceUsd, 'USD')}
                                            </span>
                                            <p className="text-white/40 text-xs">
                                              {formatCurrency(sub.numberOfQuotas * priceInfo.priceBrl)}
                                            </p>
                                          </>
                                        ) : (
                                          <span className="text-white/80 text-sm font-medium">{formatCurrency(sub.currentBalance, sub.currency)}</span>
                                        )}
                                      </div>
                                      <button
                                        onClick={(e) => { e.stopPropagation(); onEditAccount(sub); }}
                                        className="text-white/30 hover:text-white/70 text-xs transition-colors"
                                      >
                                        Editar
                                      </button>
                                    </div>
                                  </div>
                                  {purchases.length > 0 && (
                                    <div className="mt-2 pt-2 border-t border-white/[0.06] grid grid-cols-3 gap-2 text-[10px]">
                                      <div>
                                        <p className="text-white/30 uppercase">Cripto</p>
                                        <p className={gainCryptoPercent != null && gainCryptoPercent >= 0 ? 'text-emerald-400' : 'text-red-400'}>
                                          {gainCryptoPercent != null ? `${gainCryptoPercent >= 0 ? '+' : ''}${gainCryptoPercent.toFixed(1)}%` : '-'}
                                        </p>
                                        <p className="text-white/25">${purchases[0].priceUsd.toFixed(2)} → ${purchases[0].currentPriceUsd.toFixed(2)}</p>
                                      </div>
                                      <div>
                                        <p className="text-white/30 uppercase">Cambio</p>
                                        <p className={gainExchangePercent != null && gainExchangePercent >= 0 ? 'text-emerald-400' : 'text-red-400'}>
                                          {gainExchangePercent != null ? `${gainExchangePercent >= 0 ? '+' : ''}${gainExchangePercent.toFixed(1)}%` : '-'}
                                        </p>
                                        <p className="text-white/25">R${purchases[0].exchangeRate.toFixed(2)} → R${purchases[0].currentExchangeRate.toFixed(2)}</p>
                                      </div>
                                      <div>
                                        <p className="text-white/30 uppercase">Total BRL</p>
                                        <p className={totalGainBrl >= 0 ? 'text-emerald-400 font-semibold' : 'text-red-400 font-semibold'}>
                                          {totalGainBrl >= 0 ? '+' : ''}{formatCurrency(totalGainBrl)}
                                        </p>
                                        <p className={totalGainPercent >= 0 ? 'text-emerald-400/60' : 'text-red-400/60'}>
                                          {totalGainPercent >= 0 ? '+' : ''}{totalGainPercent.toFixed(1)}%
                                        </p>
                                      </div>
                                    </div>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    </div>
                    {isExpanded && selectedProfileId && (
                      <div className="ml-7 bg-white/[0.03] border border-white/10 border-t-0 rounded-b-xl p-4">
                        <div className="flex items-center gap-2 mb-3">
                          {botTabs.map((tab) => (
                            <button
                              key={tab.key}
                              onClick={(e) => { e.stopPropagation(); setActiveBot(tab.key); }}
                              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                                activeBot === tab.key
                                  ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                                  : 'bg-white/5 text-white/50 border border-white/10 hover:bg-white/10'
                              }`}
                            >
                              {tab.label}
                            </button>
                          ))}
                          {activeBot === 'MemeCoin1' && cryptoPrices?.usdBrl && (
                            <span className="ml-auto text-xs text-white/40">
                              Saldo conta: {formatCurrency(account.currentBalance / cryptoPrices.usdBrl, 'USD')}
                            </span>
                          )}
                        </div>
                        <ExpandedTransactionPanel
                          accountId={account.id}
                          profileId={selectedProfileId}
                          categories={categories}
                          accountCurrency={activeBot === 'MemeCoin1' ? 'USD' : account.currency}
                          includeAsDestination
                          botFilter={activeBot === 'all' ? undefined : activeBot}
                          onAddTransaction={onAddTransaction}
                          onEdit={onEditTransaction}
                          onDelete={onDeleteTransaction}
                          onConfirm={onConfirmTransaction}
                        />
                      </div>
                    )}
                  </div>
                )}
              </SortableItem>
            );
          })}
        </SortableContext>
      </DndContext>
    </div>
  );
}

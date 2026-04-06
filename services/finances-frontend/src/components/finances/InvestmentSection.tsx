'use client';

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
import InvestmentAccountInfo from '@/components/finances/InvestmentAccountInfo';
import ExpandedTransactionPanel from '@/components/finances/ExpandedTransactionPanel';
import type { BankAccount, Category, Transaction } from '@/types/finances';

interface InvestmentSectionProps {
  investmentAccounts: BankAccount[];
  investedByCurrency: Record<string, number>;
  allAccounts: BankAccount[];
  expandedAccountId: string | null;
  selectedProfileId: string | null;
  categories: Category[];
  sensors: SensorDescriptor<SensorOptions>[];
  onToggleExpand: (accountId: string) => void;
  onDragEnd: (event: DragEndEvent, accountList: BankAccount[]) => void;
  onEditAccount: (account: BankAccount) => void;
  onAddTransaction: (accountId: string) => void;
  onEditTransaction: (tx: Transaction) => void;
  onDeleteTransaction: (tx: Transaction) => void;
  onConfirmTransaction?: (tx: Transaction) => Promise<void> | void;
}

export default function InvestmentSection({
  investmentAccounts,
  investedByCurrency,
  allAccounts,
  expandedAccountId,
  selectedProfileId,
  categories,
  sensors,
  onToggleExpand,
  onDragEnd,
  onEditAccount,
  onAddTransaction,
  onEditTransaction,
  onDeleteTransaction,
  onConfirmTransaction,
}: InvestmentSectionProps) {
  if (investmentAccounts.length === 0) return null;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xl">📈</span>
          <h3 className="text-lg font-semibold text-white">Investimentos</h3>
        </div>
        <div className="text-right">
          {Object.entries(investedByCurrency).map(([cur, total]) => (
            <p key={cur} className="text-purple-300 text-sm font-semibold">
              {formatCurrency(total, cur)}
            </p>
          ))}
          <span className="text-white/50 text-xs">
            {investmentAccounts.length} {investmentAccounts.length === 1 ? 'investimento' : 'investimentos'}
          </span>
        </div>
      </div>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={(event) => onDragEnd(event, investmentAccounts)}
      >
        <SortableContext items={investmentAccounts.map((a) => a.id)} strategy={verticalListSortingStrategy}>
          {investmentAccounts.map((account) => {
            const isExpanded = expandedAccountId === account.id;
            return (
              <SortableItem key={account.id} id={account.id}>
                {({ listeners, attributes }) => (
                  <div>
                    <div className="flex items-center gap-1">
                      <DragHandle listeners={listeners} attributes={attributes} />
                      <div
                        className="cursor-pointer flex-1 rounded-xl transition-all duration-300"
                        style={isExpanded ? {
                          boxShadow: '0 0 20px rgba(52, 211, 153, 0.3), 0 0 0 2px #34d399',
                        } : undefined}
                        onClick={() => onToggleExpand(account.id)}
                      >
                        <InvestmentAccountInfo
                          account={account}
                          linkedAccount={account.linkedAccountId
                            ? allAccounts.find((a) => a.id === account.linkedAccountId)
                            : undefined
                          }
                          onEdit={() => onEditAccount(account)}
                        />
                      </div>
                    </div>
                    {isExpanded && selectedProfileId && (
                      <ExpandedTransactionPanel
                        accountId={account.id}
                        profileId={selectedProfileId}
                        categories={categories}
                        accountCurrency={account.currency}
                        onAddTransaction={onAddTransaction}
                        onEdit={onEditTransaction}
                        onDelete={onDeleteTransaction}
                        onConfirm={onConfirmTransaction}
                        className="ml-7"
                      />
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

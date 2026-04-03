'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { api } from '@/lib/api';
import { parseLocalDate } from '@/utils/format';
import type { Profile, Category, Transaction, BankAccount } from '@/types/finances';

const MONTHS_SHORT = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

export function useDashboardData() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const data = await api.get<{ data: Profile[] }>('/profiles');
        const list: Profile[] = data.data || [];
        setProfiles(list);
        if (list.length > 0) setSelectedProfileId(list[0].id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  const loadData = useCallback(async () => {
    if (!selectedProfileId) return;
    setLoading(true);
    try {
      const startDate = `${selectedYear}-01-01`;
      const endDate = `${selectedYear}-12-31`;

      const [catData, txData, accData] = await Promise.all([
        api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}`),
        api.get<{ data: Transaction[] }>(`/transactions?profileId=${selectedProfileId}&from=${startDate}&to=${endDate}`),
        api.get<{ data: BankAccount[] }>(`/bank-accounts?profileId=${selectedProfileId}`),
      ]);

      setCategories(catData.data || []);
      setTransactions(txData.data || []);
      setAccounts(accData.data || []);
    } catch (e) {
      console.warn('Erro ao carregar dados', e);
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId, selectedYear]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const filteredTransactions = useMemo(() => {
    if (selectedMonth === null) return transactions;
    return transactions.filter((tx) => {
      const txMonth = parseLocalDate(tx.occurredOn).getMonth();
      return txMonth === selectedMonth;
    });
  }, [transactions, selectedMonth]);

  const monthlyData = useMemo(() => {
    return MONTHS_SHORT.map((month, index) => {
      const monthTx = transactions.filter((tx) => {
        const txMonth = parseLocalDate(tx.occurredOn).getMonth();
        return txMonth === index && tx.status === 'CONFIRMED';
      });

      const income = monthTx
        .filter((tx) => tx.type === 'INCOME')
        .reduce((sum, tx) => sum + tx.amount, 0);

      const expense = monthTx
        .filter((tx) => tx.type === 'EXPENSE')
        .reduce((sum, tx) => sum + tx.amount, 0);

      return { month, monthIndex: index, income, expense, balance: income - expense };
    });
  }, [transactions]);

  const categoryData = useMemo(() => {
    const expenseTx = filteredTransactions.filter(
      (tx) => tx.type === 'EXPENSE' && tx.status === 'CONFIRMED'
    );

    const byParent: Record<string, { total: number; subcategories: Record<string, number> }> = {};

    expenseTx.forEach((tx) => {
      const category = categories.find((c) => c.id === tx.categoryId);
      const parentId = category?.parentId || category?.id || 'sem-categoria';
      if (!byParent[parentId]) {
        byParent[parentId] = { total: 0, subcategories: {} };
      }
      byParent[parentId].total += tx.amount;

      if (category?.parentId) {
        const subName = category.name;
        byParent[parentId].subcategories[subName] = (byParent[parentId].subcategories[subName] || 0) + tx.amount;
      }
    });

    return Object.entries(byParent)
      .map(([categoryId, data]) => {
        const category = categories.find((c) => c.id === categoryId);
        return {
          id: categoryId,
          name: category?.name || 'Sem categoria',
          value: data.total,
          color: category?.color || '#64748b',
          subcategories: Object.entries(data.subcategories)
            .map(([name, value]) => ({ name, value }))
            .sort((a, b) => b.value - a.value),
        };
      })
      .sort((a, b) => b.value - a.value);
  }, [filteredTransactions, categories]);

  const incomeCategoryData = useMemo(() => {
    const incomeTx = filteredTransactions.filter(
      (tx) => tx.type === 'INCOME' && tx.status === 'CONFIRMED'
    );

    const byCategory: Record<string, number> = {};
    incomeTx.forEach((tx) => {
      const catId = tx.categoryId || 'sem-categoria';
      byCategory[catId] = (byCategory[catId] || 0) + tx.amount;
    });

    return Object.entries(byCategory)
      .map(([categoryId, amount]) => {
        const category = categories.find((c) => c.id === categoryId);
        return {
          name: category?.name || 'Sem categoria',
          value: amount,
          color: category?.color || '#64748b',
        };
      })
      .sort((a, b) => b.value - a.value);
  }, [filteredTransactions, categories]);

  const totals = useMemo(() => {
    const confirmed = filteredTransactions.filter((tx) => tx.status === 'CONFIRMED');
    const income = confirmed
      .filter((tx) => tx.type === 'INCOME')
      .reduce((sum, tx) => sum + tx.amount, 0);
    const expense = confirmed
      .filter((tx) => tx.type === 'EXPENSE')
      .reduce((sum, tx) => sum + tx.amount, 0);
    return { income, expense, balance: income - expense };
  }, [filteredTransactions]);

  const accountTotals = useMemo(() => {
    const profileAccounts = accounts.filter((acc) => acc.profileId === selectedProfileId);
    const available = profileAccounts
      .filter((acc) => acc.type !== 'CREDIT_CARD' && acc.type !== 'INVESTMENT')
      .reduce((sum, acc) => sum + acc.currentBalance, 0);
    const investments = profileAccounts
      .filter((acc) => acc.type === 'INVESTMENT')
      .reduce((sum, acc) => sum + acc.currentBalance, 0);
    return { available, investments, total: available + investments };
  }, [accounts, selectedProfileId]);

  const cumulativeData = useMemo(() => {
    let cumulative = 0;
    return monthlyData.map((m) => {
      cumulative += m.balance;
      return { ...m, cumulative };
    });
  }, [monthlyData]);

  const topExpenses = useMemo(() => {
    return filteredTransactions
      .filter((tx) => tx.type === 'EXPENSE' && tx.status === 'CONFIRMED')
      .sort((a, b) => b.amount - a.amount)
      .slice(0, 5)
      .map((tx) => ({
        ...tx,
        categoryName: categories.find((c) => c.id === tx.categoryId)?.name || 'Sem categoria',
      }));
  }, [filteredTransactions, categories]);

  return {
    profiles,
    selectedProfileId,
    setSelectedProfileId,
    selectedYear,
    setSelectedYear,
    selectedMonth,
    setSelectedMonth,
    loading,
    filteredTransactions,
    monthlyData,
    categoryData,
    incomeCategoryData,
    totals,
    accountTotals,
    cumulativeData,
    topExpenses,
  };
}

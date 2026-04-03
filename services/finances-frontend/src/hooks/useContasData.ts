'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { api } from '@/lib/api';
import type { BankAccount, Invoice, Category, CryptoPurchaseWithGains } from '@/types/finances';

export function useContasData(selectedProfileId: string | null) {
  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [invoicesByAccount, setInvoicesByAccount] = useState<Record<string, Invoice[]>>({});
  const [currentInvoices, setCurrentInvoices] = useState<Record<string, Invoice>>({});
  const [cryptoPrices, setCryptoPrices] = useState<{ usdBrl: number; prices: Record<string, { priceUsd: number; priceBrl: number }> } | null>(null);
  const [cryptoPurchases, setCryptoPurchases] = useState<Record<string, CryptoPurchaseWithGains[]>>({});

  const filteredAccounts = useMemo(
    () => bankAccounts
      .filter((account) => account.profileId === selectedProfileId)
      .sort((a, b) => (a.displayOrder ?? 0) - (b.displayOrder ?? 0)),
    [bankAccounts, selectedProfileId],
  );

  const fetchBankAccounts = useCallback(async () => {
    try {
      const data = await api.get<{ data: BankAccount[] }>('/bank-accounts');
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar contas bancarias:', error);
    }
  }, []);

  const fetchCategories = useCallback(async (profileId: string) => {
    try {
      const data = await api.get<{ data: Category[] }>(`/categories?profileId=${profileId}`);
      setCategories(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar categorias:', error);
    }
  }, []);

  const syncCryptoPrices = useCallback(async (profileId: string) => {
    try {
      const data = await api.post<{ data: { usdBrl: number; prices: { symbol: string; priceUsd: number; priceBrl: number }[] } }>(`/crypto/sync?profileId=${profileId}`, {});
      const syncData = data.data;
      const pricesMap: Record<string, { priceUsd: number; priceBrl: number }> = {};
      if (syncData.prices) {
        for (const p of syncData.prices) {
          pricesMap[p.symbol] = { priceUsd: p.priceUsd, priceBrl: p.priceBrl };
        }
      }
      setCryptoPrices({ usdBrl: syncData.usdBrl, prices: pricesMap });
      await fetchBankAccounts();

      const pData = await api.get<{ data: CryptoPurchaseWithGains[] }>(`/crypto/purchases?profileId=${profileId}`);
      const byAsset: Record<string, CryptoPurchaseWithGains[]> = {};
      for (const p of (pData.data || [])) {
        if (!byAsset[p.asset]) byAsset[p.asset] = [];
        byAsset[p.asset].push(p);
      }
      setCryptoPurchases(byAsset);
    } catch (error) {
      console.warn('Erro ao sincronizar precos crypto:', error);
    }
  }, [fetchBankAccounts]);

  const fetchInvoicesForCreditCards = useCallback(async (accounts: BankAccount[]) => {
    const creditCards = accounts.filter((acc) => acc.type === 'CREDIT_CARD');
    if (creditCards.length === 0) return;

    const invoicesMap: Record<string, Invoice[]> = {};
    const currentMap: Record<string, Invoice> = {};

    await Promise.all(
      creditCards.map(async (card) => {
        try {
          const data = await api.get<{ data: Invoice[] }>(`/invoices?bankAccountId=${card.id}`);
          invoicesMap[card.id] = data.data || [];

          const currentData = await api.get<{ data: Invoice }>(`/invoices/current?bankAccountId=${card.id}`);
          if (currentData.data) {
            currentMap[card.id] = currentData.data;
          }
        } catch (error) {
          console.warn(`Erro ao carregar faturas do cartao ${card.name}:`, error);
        }
      }),
    );

    setInvoicesByAccount(invoicesMap);
    setCurrentInvoices(currentMap);
  }, []);

  // Initial fetch
  useEffect(() => {
    fetchBankAccounts();
  }, [fetchBankAccounts]);

  // Profile-dependent fetches
  useEffect(() => {
    if (selectedProfileId) {
      fetchCategories(selectedProfileId);
      syncCryptoPrices(selectedProfileId);
    }
  }, [selectedProfileId, fetchCategories, syncCryptoPrices]);

  // Invoice fetching when accounts change
  useEffect(() => {
    if (filteredAccounts.length > 0) {
      fetchInvoicesForCreditCards(filteredAccounts);
    }
  }, [filteredAccounts, fetchInvoicesForCreditCards]);

  return {
    bankAccounts,
    setBankAccounts,
    categories,
    filteredAccounts,
    invoicesByAccount,
    currentInvoices,
    cryptoPrices,
    cryptoPurchases,
    fetchBankAccounts,
    fetchInvoicesForCreditCards,
  };
}

'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { BankAccount } from '@/types/finances';

export function useBankAccounts(profileId?: string | null) {
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchAccounts = useCallback(async () => {
    setLoading(true);
    try {
      const path = profileId
        ? `/bank-accounts?profileId=${profileId}`
        : '/bank-accounts';
      const data = await api.get<{ data: BankAccount[] }>(path);
      setAccounts(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar contas:', error);
    } finally {
      setLoading(false);
    }
  }, [profileId]);

  useEffect(() => {
    fetchAccounts();
  }, [fetchAccounts]);

  return { accounts, setAccounts, loading, refetch: fetchAccounts };
}

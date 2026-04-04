'use client';

import { useCallback } from 'react';
import { api } from '@/lib/api';
import { useToast } from '@/components/ui/Toast';
import type { BankAccount, Invoice, TransactionFormData } from '@/types/finances';

interface UseContasActionsParams {
  fetchBankAccounts: () => Promise<void>;
  fetchInvoicesForCreditCards: (accounts: BankAccount[]) => Promise<void>;
  filteredAccounts: BankAccount[];
  expandedAccountId: string | null;
  setExpandedAccountId: (id: string | null) => void;
  setIsTransactionFormOpen: (open: boolean) => void;
  setEditingTransaction: (tx: null) => void;
  setEditingBankAccount: (account: BankAccount | null) => void;
}

export function useContasActions({
  fetchBankAccounts,
  fetchInvoicesForCreditCards,
  filteredAccounts,
  expandedAccountId,
  setExpandedAccountId,
  setIsTransactionFormOpen,
  setEditingTransaction,
  setEditingBankAccount,
}: UseContasActionsParams) {
  const { toast } = useToast();

  const refreshExpandedAccount = useCallback(() => {
    const current = expandedAccountId;
    setExpandedAccountId(null);
    setTimeout(() => setExpandedAccountId(current), 50);
  }, [expandedAccountId, setExpandedAccountId]);

  const handleCreateBankAccount = useCallback(async (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'> & { initialInvoiceAmount?: number },
  ) => {
    try {
      const { initialInvoiceAmount, ...bankAccountData } = accountData;
      const createdAccount = await api.post<{ data: BankAccount }>('/bank-accounts', { ...bankAccountData, isActive: true });

      if (
        accountData.type === 'CREDIT_CARD' &&
        initialInvoiceAmount &&
        initialInvoiceAmount > 0 &&
        createdAccount.data?.id
      ) {
        const currentDate = new Date();
        const referenceDate = `${currentDate.getFullYear()}-${String(currentDate.getMonth() + 1).padStart(2, '0')}`;
        try {
          const invoiceData = await api.post<{ data: Invoice }>('/invoices', { bankAccountId: createdAccount.data.id, referenceDate });
          if (invoiceData.data?.id) {
            await api.post(`/invoices/${invoiceData.data.id}/add-amount`, { amount: initialInvoiceAmount });
          }
        } catch (invoiceError) {
          console.warn('Erro ao criar fatura inicial:', invoiceError);
        }
      }
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao criar conta bancaria:', error);
      toast('Erro ao criar conta bancaria', 'error');
    }
  }, [fetchBankAccounts, toast]);

  const handleUpdateBankAccount = useCallback(async (
    editingBankAccount: BankAccount,
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    try {
      await api.put(`/bank-accounts/${editingBankAccount.id}`, { ...accountData, isActive: editingBankAccount.isActive });
      await fetchBankAccounts();
      setEditingBankAccount(null);
    } catch (error) {
      console.error('Erro ao atualizar conta bancaria:', error);
      toast('Erro ao atualizar conta bancaria', 'error');
    }
  }, [fetchBankAccounts, setEditingBankAccount, toast]);

  const handlePayInvoice = useCallback(async (invoiceId: string, amount: number) => {
    try {
      const today = new Date().toISOString().split('T')[0];
      await api.post(`/invoices/${invoiceId}/pay`, { paidAmount: amount, paidAt: today });
      await fetchInvoicesForCreditCards(filteredAccounts);
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao pagar fatura:', error);
      toast('Erro ao pagar fatura', 'error');
    }
  }, [fetchBankAccounts, fetchInvoicesForCreditCards, filteredAccounts, toast]);

  const handleUpdateInvoice = useCallback(async (invoiceId: string, data: { closingDate?: string; dueDate?: string }) => {
    try {
      await api.put(`/invoices/${invoiceId}`, data);
      await fetchInvoicesForCreditCards(filteredAccounts);
    } catch (error) {
      console.error('Erro ao atualizar fatura:', error);
      toast('Erro ao atualizar fatura', 'error');
    }
  }, [fetchInvoicesForCreditCards, filteredAccounts, toast]);

  const handleSaveTransaction = useCallback(async (payload: TransactionFormData) => {
    try {
      await api.post('/transactions', payload);
      await fetchBankAccounts();
      refreshExpandedAccount();
    } catch (error) {
      console.error('Erro ao criar transacao:', error);
      toast('Erro ao criar transacao', 'error');
    }
  }, [fetchBankAccounts, refreshExpandedAccount, toast]);

  const handleUpdateTransaction = useCallback(async (id: string, payload: TransactionFormData) => {
    try {
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      await api.put(`/transactions/${id}`, cleanPayload);
      setIsTransactionFormOpen(false);
      setEditingTransaction(null);
      await fetchBankAccounts();
      refreshExpandedAccount();
    } catch (error) {
      console.error('Erro ao atualizar transacao:', error);
      toast('Erro ao atualizar transacao', 'error');
    }
  }, [fetchBankAccounts, refreshExpandedAccount, setIsTransactionFormOpen, setEditingTransaction, toast]);

  const handleDeleteTransaction = useCallback(async (tx: { id: string }) => {
    try {
      await api.delete(`/transactions/${tx.id}`);
      await fetchBankAccounts();
      refreshExpandedAccount();
      toast('Transacao excluida com sucesso');
    } catch (error) {
      console.error('Erro ao excluir transacao:', error);
      toast('Erro ao excluir transacao', 'error');
    }
  }, [fetchBankAccounts, refreshExpandedAccount, toast]);

  return {
    handleCreateBankAccount,
    handleUpdateBankAccount,
    handlePayInvoice,
    handleUpdateInvoice,
    handleSaveTransaction,
    handleUpdateTransaction,
    handleDeleteTransaction,
  };
}

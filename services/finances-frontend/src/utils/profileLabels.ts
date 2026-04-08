import type { Profile } from '@/types/finances';

interface ProfileLabels {
  income: string;
  expense: string;
  balance: string;
  transactions: string;
  goals: string;
  summary: string;
  newTransaction: string;
}

const personalLabels: ProfileLabels = {
  income: 'Entradas',
  expense: 'Saídas',
  balance: 'Disponível',
  transactions: 'Lançamentos',
  goals: 'Metas',
  summary: 'Resumo',
  newTransaction: 'Novo lançamento',
};

const businessLabels: ProfileLabels = {
  income: 'Faturamento',
  expense: 'Despesas Operacionais',
  balance: 'Caixa Disponível',
  transactions: 'Movimentações',
  goals: 'Reservas e Objetivos',
  summary: 'Resultado do Período',
  newTransaction: 'Nova movimentação',
};

export function getProfileLabels(profile: Profile | null | undefined): ProfileLabels {
  if (profile?.type === 'BUSINESS' && profile.legalEntityType) {
    return businessLabels;
  }
  return personalLabels;
}

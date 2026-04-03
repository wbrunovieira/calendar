'use client';

import { useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import StocksTab from '@/components/finances/StocksTab';
import FiisTab from '@/components/finances/FiisTab';

type Tab = 'acoes' | 'fiis';

export default function MercadoPage() {
  const [activeTab, setActiveTab] = useState<Tab>('acoes');

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-white">Analise de Mercado</h1>
          <p className="text-white/60 text-sm">
            Acoes e Fundos Imobiliarios — dados e indicadores para analise
          </p>
        </div>

        <div className="flex gap-2">
          <button
            onClick={() => setActiveTab('acoes')}
            className={`px-5 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              activeTab === 'acoes'
                ? 'bg-emerald-500/20 border border-emerald-400/50 text-emerald-200'
                : 'bg-white/5 border border-white/10 text-white/60 hover:bg-white/10'
            }`}
          >
            Acoes (Magic Formula)
          </button>
          <button
            onClick={() => setActiveTab('fiis')}
            className={`px-5 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              activeTab === 'fiis'
                ? 'bg-blue-500/20 border border-blue-400/50 text-blue-200'
                : 'bg-white/5 border border-white/10 text-white/60 hover:bg-white/10'
            }`}
          >
            Fundos Imobiliarios (FIIs)
          </button>
        </div>

        {activeTab === 'acoes' && <StocksTab />}
        {activeTab === 'fiis' && <FiisTab />}
      </div>
    </AppLayout>
  );
}

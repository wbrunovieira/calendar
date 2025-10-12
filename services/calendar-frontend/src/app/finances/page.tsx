'use client';

import { useState } from 'react';
import AppLayout from '@/components/navigation/AppLayout';

type TabType = 'dashboard' | 'settings';

export default function FinancesPage() {
  const [activeTab, setActiveTab] = useState<TabType>('dashboard');

  return (
    <AppLayout>
      <div className="flex-1 w-full py-8 relative">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-extrabold text-white drop-shadow-lg mb-2">
            💰 Finanças
          </h1>
          <p className="text-white/70 text-lg">
            Gerencie suas contas, transações e investimentos
          </p>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setActiveTab('dashboard')}
            className={`px-6 py-3 rounded-xl font-semibold transition-all duration-300 ${
              activeTab === 'dashboard'
                ? 'bg-white/20 text-white shadow-lg'
                : 'bg-white/5 text-white/70 hover:bg-white/10'
            }`}
          >
            Dashboard
          </button>
          <button
            onClick={() => setActiveTab('settings')}
            className={`px-6 py-3 rounded-xl font-semibold transition-all duration-300 ${
              activeTab === 'settings'
                ? 'bg-white/20 text-white shadow-lg'
                : 'bg-white/5 text-white/70 hover:bg-white/10'
            }`}
          >
            Configurações
          </button>
        </div>

        {/* Dashboard Tab */}
        {activeTab === 'dashboard' && (
          <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-12 border border-white/10 shadow-2xl text-center">
          <div className="max-w-2xl mx-auto">
            <div className="text-6xl mb-6">🚧</div>
            <h2 className="text-3xl font-bold text-white mb-4">
              Em Construção
            </h2>
            <p className="text-white/70 text-lg mb-8">
              O módulo financeiro está sendo desenvolvido e em breve estará disponível com as seguintes funcionalidades:
            </p>

            <div className="grid md:grid-cols-2 gap-6 text-left">
              {/* Contas */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">🏦</div>
                  <h3 className="text-xl font-bold text-white">Contas</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Contas pessoais</li>
                  <li>• Conta empresarial</li>
                  <li>• Controle de saldos</li>
                  <li>• Múltiplas moedas</li>
                </ul>
              </div>

              {/* Transações */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">💸</div>
                  <h3 className="text-xl font-bold text-white">Transações</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Receitas e despesas</li>
                  <li>• Categorização automática</li>
                  <li>• Contas recorrentes</li>
                  <li>• Tags personalizadas</li>
                </ul>
              </div>

              {/* Investimentos */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">📈</div>
                  <h3 className="text-xl font-bold text-white">Investimentos</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Ações e FIIs</li>
                  <li>• Tesouro Direto</li>
                  <li>• CDB, LCI, LCA</li>
                  <li>• Rentabilidade</li>
                </ul>
              </div>

              {/* Relatórios */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">📊</div>
                  <h3 className="text-xl font-bold text-white">Relatórios</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Fluxo de caixa</li>
                  <li>• Previsões com IA</li>
                  <li>• Análise de gastos</li>
                  <li>• Metas financeiras</li>
                </ul>
              </div>
            </div>

            {/* API Status */}
            <div className="mt-8 p-4 bg-green-500/20 rounded-lg border border-green-500/30">
              <div className="flex items-center justify-center gap-2 text-green-400">
                <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                <span className="font-semibold">API Calendar Finances rodando em http://localhost:3335</span>
              </div>
            </div>
          </div>
        </div>
        )}

        {/* Settings Tab */}
        {activeTab === 'settings' && (
          <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-8 border border-white/10 shadow-2xl">
            <div className="max-w-6xl mx-auto">
              <div className="mb-8">
                <h2 className="text-3xl font-bold text-white mb-2">Configurações Financeiras</h2>
                <p className="text-white/70">
                  Gerencie perfis financeiros e suas configurações
                </p>
              </div>

              {/* Profiles Section */}
              <div className="mb-8">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-2xl font-bold text-white">Perfis Financeiros</h3>
                  <button className="flex items-center gap-2 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105 border border-white/20">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    <span>Novo Perfil</span>
                  </button>
                </div>

                <div className="grid gap-4">
                  {/* Profile Card Placeholder */}
                  <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/10">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-2xl">
                          💼
                        </div>
                        <div>
                          <h4 className="text-xl font-bold text-white">Aguardando configuração</h4>
                          <p className="text-white/60 text-sm">Crie perfis financeiros para começar</p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Info Box */}
              <div className="p-6 bg-blue-500/10 rounded-xl border border-blue-500/20">
                <div className="flex items-start gap-3">
                  <svg className="w-6 h-6 text-blue-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div>
                    <h4 className="text-white font-semibold mb-1">Sobre os Perfis Financeiros</h4>
                    <p className="text-white/70 text-sm">
                      Cada perfil financeiro está vinculado a um calendário. Você pode ter perfis do tipo PESSOAL ou EMPRESARIAL.
                      Os perfis controlam contas, transações e investimentos de forma independente.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}

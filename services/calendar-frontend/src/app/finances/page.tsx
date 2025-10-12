'use client';

import AppLayout from '@/components/navigation/AppLayout';

export default function FinancesPage() {
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

        {/* Coming Soon Message */}
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
      </div>
    </AppLayout>
  );
}

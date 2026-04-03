'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Bar,
  ComposedChart,
  ReferenceLine,
} from 'recharts';
import { api } from '@/lib/api';
import { formatCurrency } from '@/utils/format';

interface FIIMonthlyDiv {
  month: string;
  amount: number;
  yield: number;
}

interface FIIMarketItem {
  ticker: string;
  segment: string;
  currentPrice: number;
  priceChange12M: number;
  dividendYield: number;
  dividends12M: number;
  lastDividend: number;
  lastDividendDate: string;
  priceToBook: number | null;
  bookValue: number | null;
  totalReturn12M: number;
  monthlyDividends: FIIMonthlyDiv[];
}

interface FIIMarketResponse {
  data: FIIMarketItem[];
  cdiRate: number;
  cdiYield: number;
}

const SEGMENT_COLORS: Record<string, string> = {
  Logistica: '#10b981',
  Papel: '#3b82f6',
  Shopping: '#f59e0b',
  'Lajes Corporativas': '#8b5cf6',
  Hibrido: '#ec4899',
  Agro: '#84cc16',
  'Fundo de Fundos': '#06b6d4',
  Residencial: '#f97316',
  Educacional: '#6366f1',
};

export default function FiisTab() {
  const [fiis, setFiis] = useState<FIIMarketItem[]>([]);
  const [cdiRate, setCdiRate] = useState(0);
  const [cdiYield, setCdiYield] = useState(0);
  const [loadingFiis, setLoadingFiis] = useState(true);
  const [fiiSegmentFilter, setFiiSegmentFilter] = useState<string>('all');
  const [fiiSortBy, setFiiSortBy] = useState<'dividendYield' | 'priceChange12M' | 'currentPrice' | 'totalReturn12M' | 'priceToBook'>('dividendYield');
  const [expandedFii, setExpandedFii] = useState<string | null>(null);

  const loadFIIs = useCallback(async () => {
    setLoadingFiis(true);
    try {
      const json = await api.get<FIIMarketResponse>('/fiis/market');
      setFiis(json.data || []);
      setCdiRate(json.cdiRate || 0);
      setCdiYield(json.cdiYield || 0);
    } catch (e) {
      console.warn('Erro FIIs:', e);
    } finally {
      setLoadingFiis(false);
    }
  }, []);

  useEffect(() => {
    loadFIIs();
  }, [loadFIIs]);

  const fiiSegments = [...new Set(fiis.map((f) => f.segment))].sort();

  const filteredFiis = fiis
    .filter((f) => fiiSegmentFilter === 'all' || f.segment === fiiSegmentFilter)
    .sort((a, b) => {
      if (fiiSortBy === 'dividendYield') return b.dividendYield - a.dividendYield;
      if (fiiSortBy === 'priceChange12M') return b.priceChange12M - a.priceChange12M;
      if (fiiSortBy === 'totalReturn12M') return b.totalReturn12M - a.totalReturn12M;
      if (fiiSortBy === 'priceToBook') {
        const aVal = a.priceToBook ?? 999;
        const bVal = b.priceToBook ?? 999;
        return aVal - bVal;
      }
      return a.currentPrice - b.currentPrice;
    });

  const fiiDYChartData = filteredFiis.map((f) => ({
    ticker: f.ticker,
    dy: f.dividendYield,
    segment: f.segment,
    cdi: cdiYield,
  }));

  if (loadingFiis) {
    return <div className="text-center py-12 text-white/60">Carregando dados de FIIs do Yahoo Finance...</div>;
  }

  return (
    <>
      {/* Filters */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <div className="flex flex-wrap items-center gap-6">
          <div className="flex items-center gap-2">
            <span className="text-white/70 text-sm">Segmento:</span>
            <select
              value={fiiSegmentFilter}
              onChange={(e) => setFiiSegmentFilter(e.target.value)}
              className="bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm"
            >
              <option value="all" className="bg-gray-800">Todos</option>
              {fiiSegments.map((seg) => (
                <option key={seg} value={seg} className="bg-gray-800">{seg}</option>
              ))}
            </select>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-white/70 text-sm">Ordenar:</span>
            <select
              value={fiiSortBy}
              onChange={(e) => setFiiSortBy(e.target.value as typeof fiiSortBy)}
              className="bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm"
            >
              <option value="dividendYield" className="bg-gray-800">Dividend Yield</option>
              <option value="totalReturn12M" className="bg-gray-800">Retorno Total 12M</option>
              <option value="priceChange12M" className="bg-gray-800">Valorizacao 12M</option>
              <option value="priceToBook" className="bg-gray-800">P/VP (menor)</option>
              <option value="currentPrice" className="bg-gray-800">Preco</option>
            </select>
          </div>
          <span className="text-white/50 text-sm">{filteredFiis.length} FIIs</span>
          {cdiRate > 0 && (
            <span className="text-amber-300/80 text-xs bg-amber-500/10 border border-amber-500/20 px-2 py-1 rounded-lg">
              CDI (Selic): {cdiRate}% a.a. | Liquido: {cdiYield}% (IR 15%)
            </span>
          )}
        </div>
      </div>

      {/* DY Chart */}
      {fiiDYChartData.length > 0 && (
        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-2">Dividend Yield 12 Meses (%)</h3>
          <p className="text-white/50 text-xs mb-4">Rendimento dos ultimos 12 meses baseado em dividendos pagos</p>
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart data={fiiDYChartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                <XAxis dataKey="ticker" tick={{ fill: '#94a3b8', fontSize: 10 }} angle={-45} textAnchor="end" height={60} />
                <YAxis tick={{ fill: '#94a3b8', fontSize: 12 }} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  content={({ active, payload }) => {
                    if (active && payload && payload.length) {
                      const d = payload[0].payload;
                      const aboveCdi = d.dy > cdiYield;
                      return (
                        <div className="bg-gray-900/95 border border-white/20 rounded-lg p-3 shadow-xl">
                          <p className="text-white font-semibold">{d.ticker}</p>
                          <p className="text-white/60 text-xs">{d.segment}</p>
                          <p className="text-blue-400 text-sm mt-1">DY: {d.dy.toFixed(2)}%</p>
                          {cdiYield > 0 && (
                            <p className={`text-xs mt-1 ${aboveCdi ? 'text-emerald-400' : 'text-rose-400'}`}>
                              {aboveCdi ? 'Acima' : 'Abaixo'} do CDI liquido ({cdiYield}%)
                            </p>
                          )}
                        </div>
                      );
                    }
                    return null;
                  }}
                />
                <Bar dataKey="dy" radius={[4, 4, 0, 0]} fill="#3b82f6" />
                {cdiYield > 0 && (
                  <ReferenceLine
                    y={cdiYield}
                    stroke="#f59e0b"
                    strokeWidth={2}
                    strokeDasharray="6 3"
                    label={{ value: `CDI ${cdiYield}%`, position: 'right', fill: '#f59e0b', fontSize: 11 }}
                  />
                )}
              </ComposedChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* FII Table */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">FIIs do Mercado</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-white/10">
                <th className="text-left text-white/60 pb-3 pr-3">Ticker</th>
                <th className="text-left text-white/60 pb-3 pr-3">Seg.</th>
                <th className="text-right text-white/60 pb-3 pr-3">Preco</th>
                <th className="text-right text-white/60 pb-3 pr-3">DY 12M</th>
                <th className="text-right text-white/60 pb-3 pr-3">P/VP</th>
                <th className="text-right text-white/60 pb-3 pr-3">Retorno Total</th>
                <th className="text-right text-white/60 pb-3 pr-3">Div. 12M</th>
                <th className="text-right text-white/60 pb-3 pr-3">Ult. Div.</th>
                <th className="text-right text-white/60 pb-3">Var. 12M</th>
              </tr>
            </thead>
            <tbody>
              {filteredFiis.map((fii) => (
                <>
                  <tr
                    key={fii.ticker}
                    className="border-b border-white/5 hover:bg-white/5 transition-colors cursor-pointer"
                    onClick={() => setExpandedFii(expandedFii === fii.ticker ? null : fii.ticker)}
                  >
                    <td className="py-3 pr-3">
                      <div className="flex items-center gap-1">
                        <span className="text-white/40 text-xs">{expandedFii === fii.ticker ? '▼' : '▶'}</span>
                        <span className="text-white font-semibold">{fii.ticker}</span>
                      </div>
                    </td>
                    <td className="py-3 pr-3">
                      <span
                        className="px-2 py-0.5 rounded-full text-xs font-medium"
                        style={{
                          backgroundColor: `${SEGMENT_COLORS[fii.segment] || '#64748b'}20`,
                          color: SEGMENT_COLORS[fii.segment] || '#94a3b8',
                          border: `1px solid ${SEGMENT_COLORS[fii.segment] || '#64748b'}40`,
                        }}
                      >
                        {fii.segment}
                      </span>
                    </td>
                    <td className="py-3 pr-3 text-right text-white/80">{formatCurrency(fii.currentPrice)}</td>
                    <td className="py-3 pr-3 text-right">
                      <span className={`font-semibold ${
                        fii.dividendYield >= cdiYield ? 'text-emerald-400' :
                        fii.dividendYield >= cdiYield * 0.8 ? 'text-blue-400' :
                        'text-white/60'
                      }`}>{fii.dividendYield.toFixed(2)}%</span>
                    </td>
                    <td className="py-3 pr-3 text-right">
                      {fii.priceToBook != null ? (
                        <span className={`font-medium ${
                          fii.priceToBook < 0.95 ? 'text-emerald-400' :
                          fii.priceToBook <= 1.05 ? 'text-blue-400' :
                          'text-rose-400'
                        }`}>{fii.priceToBook.toFixed(2)}</span>
                      ) : (
                        <span className="text-white/30">-</span>
                      )}
                    </td>
                    <td className="py-3 pr-3 text-right">
                      <span className={`font-semibold ${fii.totalReturn12M >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                        {fii.totalReturn12M >= 0 ? '+' : ''}{fii.totalReturn12M.toFixed(2)}%
                      </span>
                    </td>
                    <td className="py-3 pr-3 text-right text-white/70">{formatCurrency(fii.dividends12M)}</td>
                    <td className="py-3 pr-3 text-right text-white/60 text-xs">
                      {formatCurrency(fii.lastDividend)}
                      {fii.lastDividendDate && <span className="block text-white/40">{fii.lastDividendDate}</span>}
                    </td>
                    <td className="py-3 text-right">
                      <span className={`font-medium ${fii.priceChange12M >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                        {fii.priceChange12M >= 0 ? '+' : ''}{fii.priceChange12M.toFixed(2)}%
                      </span>
                    </td>
                  </tr>
                  {expandedFii === fii.ticker && fii.monthlyDividends && fii.monthlyDividends.length > 0 && (
                    <tr key={`${fii.ticker}-detail`} className="border-b border-white/5">
                      <td colSpan={9} className="py-3 px-6">
                        <div className="bg-white/5 rounded-xl p-4">
                          <div className="flex items-center justify-between mb-3">
                            <p className="text-white/70 text-xs font-medium">Dividendos Mensais</p>
                            {fii.bookValue != null && (
                              <p className="text-white/50 text-xs">
                                VP: {formatCurrency(fii.bookValue)} | P/VP: {fii.priceToBook?.toFixed(2) ?? '-'}
                              </p>
                            )}
                          </div>
                          <div className="grid grid-cols-6 md:grid-cols-12 gap-2">
                            {fii.monthlyDividends.map((md) => (
                              <div key={md.month} className="bg-white/5 rounded-lg p-2 text-center">
                                <p className="text-white/40 text-[10px]">{md.month}</p>
                                <p className="text-white font-semibold text-xs">{formatCurrency(md.amount)}</p>
                                <p className="text-blue-400 text-[10px]">{md.yield.toFixed(2)}%</p>
                              </div>
                            ))}
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
        <div className="mt-4 flex flex-wrap gap-4 text-xs text-white/50">
          <span><span className="text-emerald-400 font-medium">P/VP &lt; 0.95</span> = desconto sobre VP</span>
          <span><span className="text-blue-400 font-medium">P/VP 0.95-1.05</span> = proximo ao VP</span>
          <span><span className="text-rose-400 font-medium">P/VP &gt; 1.05</span> = premio sobre VP</span>
          <span><span className="text-emerald-400 font-medium">DY verde</span> = acima CDI liquido</span>
        </div>
      </div>

      {/* FII Info */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
        <h3 className="text-lg font-semibold text-white mb-3">Sobre Fundos Imobiliarios</h3>
        <div className="text-white/60 text-sm space-y-2">
          <p>
            <span className="text-white/80 font-medium">FIIs</span> sao fundos que investem em imoveis ou
            titulos imobiliarios e distribuem rendimentos mensais (isentos de IR para pessoa fisica).
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4 mt-3">
            <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-4">
              <p className="text-blue-300 font-semibold mb-1">Dividend Yield (DY)</p>
              <p className="text-white/50 text-xs">
                Rendimento anual em dividendos em relacao ao preco da cota.
                Compare com o CDI liquido para avaliar se o FII rende mais que a renda fixa.
              </p>
            </div>
            <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl p-4">
              <p className="text-emerald-300 font-semibold mb-1">P/VP (Preco / Valor Patrimonial)</p>
              <p className="text-white/50 text-xs">
                Indica se a cota esta barata (&lt;1.0 = desconto) ou cara (&gt;1.0 = premio)
                em relacao ao valor patrimonial do fundo.
              </p>
            </div>
            <div className="bg-amber-500/10 border border-amber-500/20 rounded-xl p-4">
              <p className="text-amber-300 font-semibold mb-1">CDI como Referencia</p>
              <p className="text-white/50 text-xs">
                FIIs sao isentos de IR, entao compare o DY com o CDI liquido
                (CDI - 15% IR) para uma comparacao justa.
              </p>
            </div>
            <div className="bg-purple-500/10 border border-purple-500/20 rounded-xl p-4">
              <p className="text-purple-300 font-semibold mb-1">Retorno Total</p>
              <p className="text-white/50 text-xs">
                Soma do DY + valorizacao/desvalorizacao da cota nos ultimos 12 meses.
                Clique em um FII para ver os dividendos mensais.
              </p>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

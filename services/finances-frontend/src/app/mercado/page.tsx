'use client';

import { useCallback, useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  AreaChart,
  Area,
  Bar,
  ComposedChart,
  ReferenceLine,
} from 'recharts';
import { API_BASE } from '@/lib/api';

const PROXY_BASE = '/api/magic-formula';

type Tab = 'acoes' | 'fiis';

interface Stock {
  rank: number;
  ticker: string;
  price: number;
  ebit_ev: number;
  roic: number;
  ranking_ebit_ev: number;
  ranking_roic: number;
  volume: number;
}

interface BacktestData {
  dates: string[];
  magic_formula: number[];
  ibovespa: number[];
}

interface RankingHistory {
  ticker: string;
  history: Array<{
    date: string;
    rank: number;
    ebit_ev: number;
    roic: number;
    price: number;
    in_top10: boolean;
  }>;
}

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

const formatCurrency = (value: number) =>
  new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(value);

const formatPercent = (value: number) => `${value.toFixed(2)}%`;

const formatVolume = (value: number) =>
  new Intl.NumberFormat('pt-BR', {
    notation: 'compact',
    compactDisplay: 'short',
  }).format(value);

export default function MercadoPage() {
  const [activeTab, setActiveTab] = useState<Tab>('acoes');
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [recommendationDate, setRecommendationDate] = useState('');
  const [backtest, setBacktest] = useState<BacktestData | null>(null);
  const [rankingHistory, setRankingHistory] = useState<RankingHistory | null>(null);
  const [, setSelectedTicker] = useState('');
  const [searchTicker, setSearchTicker] = useState('');
  const [topN, setTopN] = useState(10);
  const [loading, setLoading] = useState(true);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [error, setError] = useState('');

  // FII state
  const [fiis, setFiis] = useState<FIIMarketItem[]>([]);
  const [cdiRate, setCdiRate] = useState(0);
  const [cdiYield, setCdiYield] = useState(0);
  const [loadingFiis, setLoadingFiis] = useState(false);
  const [fiiSegmentFilter, setFiiSegmentFilter] = useState<string>('all');
  const [fiiSortBy, setFiiSortBy] = useState<'dividendYield' | 'priceChange12M' | 'currentPrice' | 'totalReturn12M' | 'priceToBook'>('dividendYield');
  const [expandedFii, setExpandedFii] = useState<string | null>(null);
  const [minVolume, setMinVolume] = useState(1_000_000);

  const loadRecommendations = useCallback(async () => {
    try {
      const res = await fetch(`${PROXY_BASE}?endpoint=recommendations&top=${topN}`);
      if (!res.ok) throw new Error('Falha ao carregar recomendacoes');
      const json = await res.json();
      setStocks(json.data.stocks);
      setRecommendationDate(json.data.date);
    } catch (e) {
      console.warn('Erro recommendations:', e);
      setError('Erro ao carregar recomendacoes');
    }
  }, [topN]);

  const loadBacktest = useCallback(async () => {
    try {
      const res = await fetch(`${PROXY_BASE}?endpoint=backtest`);
      if (!res.ok) throw new Error('Falha ao carregar backtest');
      const json = await res.json();
      setBacktest(json.data);
    } catch (e) {
      console.warn('Erro backtest:', e);
    }
  }, []);

  const loadRankingHistory = useCallback(async (ticker: string) => {
    if (!ticker) return;
    setLoadingHistory(true);
    try {
      const res = await fetch(`${PROXY_BASE}?endpoint=ranking-history&ticker=${ticker}`);
      if (!res.ok) {
        if (res.status === 404) {
          setRankingHistory(null);
          setError(`Ticker ${ticker.toUpperCase()} nao encontrado`);
        }
        return;
      }
      const json = await res.json();
      setRankingHistory(json.data);
      setError('');
    } catch (e) {
      console.warn('Erro ranking history:', e);
    } finally {
      setLoadingHistory(false);
    }
  }, []);

  const loadFIIs = useCallback(async () => {
    setLoadingFiis(true);
    try {
      const res = await fetch(`${API_BASE}/fiis/market`);
      if (!res.ok) throw new Error('Falha ao carregar FIIs');
      const json: FIIMarketResponse = await res.json();
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
    (async () => {
      setLoading(true);
      await Promise.all([loadRecommendations(), loadBacktest()]);
      setLoading(false);
    })();
  }, [loadRecommendations, loadBacktest]);

  useEffect(() => {
    if (activeTab === 'fiis' && fiis.length === 0 && !loadingFiis) {
      loadFIIs();
    }
  }, [activeTab, fiis.length, loadingFiis, loadFIIs]);

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

  const filteredStocks = stocks.filter((s) => s.volume >= minVolume);

  const fiiDYChartData = filteredFiis.map((f) => ({
    ticker: f.ticker,
    dy: f.dividendYield,
    segment: f.segment,
    cdi: cdiYield,
  }));

  const backtestChartData = backtest
    ? backtest.dates.map((date, i) => ({
        date,
        magic_formula: backtest.magic_formula[i],
        ibovespa: backtest.ibovespa[i],
      }))
    : [];

  const handleSearch = () => {
    if (searchTicker.trim()) {
      setSelectedTicker(searchTicker.trim().toUpperCase());
      loadRankingHistory(searchTicker.trim());
    }
  };

  const CustomTooltip = ({
    active,
    payload,
    label,
  }: {
    active?: boolean;
    payload?: Array<{ name: string; value: number; color: string }>;
    label?: string;
  }) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-gray-900/95 border border-white/20 rounded-lg p-3 shadow-xl">
          <p className="text-white/80 text-sm font-medium mb-1">{label}</p>
          {payload.map((entry, index) => (
            <p key={index} className="text-sm" style={{ color: entry.color }}>
              {entry.name}: {formatPercent(entry.value)}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold text-white">Analise de Mercado</h1>
          <p className="text-white/60 text-sm">
            Acoes e Fundos Imobiliarios — dados e indicadores para analise
          </p>
        </div>

        {/* Tabs */}
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

        {/* ═══ ACOES TAB ═══ */}
        {activeTab === 'acoes' && (loading ? (
          <div className="text-center py-12 text-white/60">Carregando dados do mercado...</div>
        ) : error && stocks.length === 0 ? (
          <div className="text-center py-12 text-rose-400">{error}</div>
        ) : (
          <>
            {/* Data staleness warning */}
            {recommendationDate && (
              <div className="bg-amber-500/10 border border-amber-500/30 rounded-2xl p-4 flex items-center gap-3">
                <span className="text-amber-400 text-lg">&#9888;</span>
                <div>
                  <p className="text-amber-200 text-sm font-medium">Dados historicos</p>
                  <p className="text-amber-200/60 text-xs">
                    Os dados de acoes sao baseados na planilha de {recommendationDate}. Precos e indicadores podem estar defasados em relacao ao mercado atual.
                  </p>
                </div>
              </div>
            )}

            {/* Top N selector + Volume filter */}
            <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
              <div className="flex flex-wrap items-center gap-6">
                <div className="flex items-center gap-2">
                  <span className="text-white/70 text-sm">Top:</span>
                  <select
                    value={topN}
                    onChange={(e) => setTopN(Number(e.target.value))}
                    className="bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm"
                  >
                    {[5, 10, 15, 20, 30].map((n) => (
                      <option key={n} value={n} className="bg-gray-800">
                        {n} acoes
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-white/70 text-sm">Volume min:</span>
                  <select
                    value={minVolume}
                    onChange={(e) => setMinVolume(Number(e.target.value))}
                    className="bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm"
                  >
                    <option value={0} className="bg-gray-800">Sem filtro</option>
                    <option value={500000} className="bg-gray-800">500K+</option>
                    <option value={1000000} className="bg-gray-800">1M+</option>
                    <option value={5000000} className="bg-gray-800">5M+</option>
                    <option value={10000000} className="bg-gray-800">10M+</option>
                  </select>
                </div>
                {recommendationDate && (
                  <span className="text-white/50 text-sm">
                    Dados de: {recommendationDate}
                  </span>
                )}
                <span className="text-white/50 text-sm">
                  {filteredStocks.length} de {stocks.length} acoes
                </span>
              </div>
            </div>

            {/* Recommendations Table */}
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-lg font-semibold text-white mb-4">
                Top {topN} Acoes - Magic Formula
              </h3>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-white/10">
                      <th className="text-left text-white/60 pb-3 pr-4">#</th>
                      <th className="text-left text-white/60 pb-3 pr-4">Ticker</th>
                      <th className="text-right text-white/60 pb-3 pr-4">Preco</th>
                      <th className="text-right text-white/60 pb-3 pr-4">EBIT/EV</th>
                      <th className="text-right text-white/60 pb-3 pr-4">ROIC</th>
                      <th className="text-right text-white/60 pb-3 pr-4">Rank EBIT/EV</th>
                      <th className="text-right text-white/60 pb-3 pr-4">Rank ROIC</th>
                      <th className="text-right text-white/60 pb-3">Volume</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredStocks.map((stock) => (
                      <tr
                        key={stock.ticker}
                        className="border-b border-white/5 hover:bg-white/5 cursor-pointer transition-colors"
                        onClick={() => {
                          setSearchTicker(stock.ticker);
                          setSelectedTicker(stock.ticker);
                          loadRankingHistory(stock.ticker);
                        }}
                      >
                        <td className="py-3 pr-4">
                          <span className={`inline-flex items-center justify-center w-7 h-7 rounded-full text-xs font-bold ${
                            stock.rank <= 3
                              ? 'bg-amber-500/20 text-amber-300'
                              : stock.rank <= 5
                              ? 'bg-emerald-500/20 text-emerald-300'
                              : 'bg-white/10 text-white/60'
                          }`}>
                            {stock.rank}
                          </span>
                        </td>
                        <td className="py-3 pr-4 text-white font-semibold">{stock.ticker}</td>
                        <td className="py-3 pr-4 text-right text-white/80">
                          {formatCurrency(stock.price)}
                        </td>
                        <td className="py-3 pr-4 text-right text-emerald-400">
                          {(stock.ebit_ev * 100).toFixed(1)}%
                        </td>
                        <td className="py-3 pr-4 text-right text-blue-400">
                          {(stock.roic * 100).toFixed(1)}%
                        </td>
                        <td className="py-3 pr-4 text-right text-white/60">
                          {stock.ranking_ebit_ev}
                        </td>
                        <td className="py-3 pr-4 text-right text-white/60">
                          {stock.ranking_roic}
                        </td>
                        <td className="py-3 text-right text-white/50">
                          {formatVolume(stock.volume)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Backtest Chart */}
            {backtestChartData.length > 0 && (
              <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
                <h3 className="text-lg font-semibold text-white mb-2">
                  Backtest: Magic Formula vs Ibovespa
                </h3>
                <p className="text-white/50 text-xs mb-4">
                  Retorno acumulado (%) ao longo do tempo
                </p>
                <div className="h-80">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={backtestChartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                      <defs>
                        <linearGradient id="colorMF" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#10b981" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                        </linearGradient>
                        <linearGradient id="colorIbov" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                      <XAxis
                        dataKey="date"
                        tick={{ fill: '#94a3b8', fontSize: 10 }}
                        interval={Math.floor(backtestChartData.length / 8)}
                      />
                      <YAxis
                        tick={{ fill: '#94a3b8', fontSize: 12 }}
                        tickFormatter={(v) => `${v}%`}
                      />
                      <Tooltip content={<CustomTooltip />} />
                      <Legend />
                      <Area
                        type="monotone"
                        dataKey="magic_formula"
                        name="Magic Formula"
                        stroke="#10b981"
                        fill="url(#colorMF)"
                        strokeWidth={2}
                      />
                      <Area
                        type="monotone"
                        dataKey="ibovespa"
                        name="Ibovespa"
                        stroke="#3b82f6"
                        fill="url(#colorIbov)"
                        strokeWidth={2}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
                {backtest && (
                  <div className="mt-4 flex gap-6">
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 rounded-full bg-emerald-500" />
                      <span className="text-white/70 text-sm">
                        Magic Formula:{' '}
                        <span className={`font-semibold ${
                          backtest.magic_formula[backtest.magic_formula.length - 1] >= 0
                            ? 'text-emerald-400'
                            : 'text-rose-400'
                        }`}>
                          {formatPercent(backtest.magic_formula[backtest.magic_formula.length - 1])}
                        </span>
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 rounded-full bg-blue-500" />
                      <span className="text-white/70 text-sm">
                        Ibovespa:{' '}
                        <span className={`font-semibold ${
                          backtest.ibovespa[backtest.ibovespa.length - 1] >= 0
                            ? 'text-blue-400'
                            : 'text-rose-400'
                        }`}>
                          {formatPercent(backtest.ibovespa[backtest.ibovespa.length - 1])}
                        </span>
                      </span>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Ticker Search */}
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-lg font-semibold text-white mb-4">Historico de Ranking</h3>
              <div className="flex items-center gap-3 mb-4">
                <input
                  type="text"
                  value={searchTicker}
                  onChange={(e) => setSearchTicker(e.target.value.toUpperCase())}
                  onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                  placeholder="Digite o ticker (ex: PETR4)"
                  className="bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white text-sm placeholder-white/40 w-48"
                />
                <button
                  onClick={handleSearch}
                  className="px-4 py-2 bg-emerald-500/20 border border-emerald-400/40 text-emerald-200 rounded-lg text-sm hover:bg-emerald-500/30 transition-colors"
                >
                  Buscar
                </button>
                {error && rankingHistory === null && (
                  <span className="text-rose-400 text-sm">{error}</span>
                )}
              </div>

              {loadingHistory && (
                <div className="text-white/60 text-sm py-4">Carregando historico...</div>
              )}

              {rankingHistory && !loadingHistory && (
                <div>
                  <div className="flex items-center gap-3 mb-4">
                    <h4 className="text-white font-semibold">{rankingHistory.ticker}</h4>
                    <span className="text-white/50 text-xs">
                      {rankingHistory.history.length} registros
                    </span>
                  </div>

                  {/* Ranking History Chart */}
                  <div className="h-64 mb-6">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart
                        data={rankingHistory.history}
                        margin={{ top: 5, right: 20, left: 0, bottom: 5 }}
                      >
                        <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                        <XAxis
                          dataKey="date"
                          tick={{ fill: '#94a3b8', fontSize: 10 }}
                          interval={Math.floor(rankingHistory.history.length / 6)}
                        />
                        <YAxis
                          reversed
                          tick={{ fill: '#94a3b8', fontSize: 12 }}
                          domain={[1, 'auto']}
                          label={{
                            value: 'Ranking',
                            angle: -90,
                            position: 'insideLeft',
                            fill: '#94a3b8',
                            fontSize: 12,
                          }}
                        />
                        <Tooltip
                          content={({ active, payload, label }) => {
                            if (active && payload && payload.length) {
                              const d = payload[0].payload;
                              return (
                                <div className="bg-gray-900/95 border border-white/20 rounded-lg p-3 shadow-xl">
                                  <p className="text-white/80 text-sm font-medium mb-1">{label}</p>
                                  <p className="text-sm text-amber-300">
                                    Ranking: #{d.rank}
                                  </p>
                                  <p className="text-sm text-emerald-400">
                                    EBIT/EV: {(d.ebit_ev * 100).toFixed(1)}%
                                  </p>
                                  <p className="text-sm text-blue-400">
                                    ROIC: {(d.roic * 100).toFixed(1)}%
                                  </p>
                                  <p className="text-sm text-white/70">
                                    Preco: {formatCurrency(d.price)}
                                  </p>
                                  {d.in_top10 && (
                                    <p className="text-sm text-emerald-300 font-semibold mt-1">
                                      Top 10
                                    </p>
                                  )}
                                </div>
                              );
                            }
                            return null;
                          }}
                        />
                        <Line
                          type="monotone"
                          dataKey="rank"
                          stroke="#f59e0b"
                          strokeWidth={2}
                          dot={(props) => {
                            const { cx, cy, payload } = props as { cx?: number; cy?: number; payload?: { in_top10?: boolean } };
                            if (cx == null || cy == null) return <circle r={0} />;
                            const inTop10 = payload?.in_top10 ?? false;
                            return (
                              <circle
                                key={`${cx}-${cy}`}
                                cx={cx}
                                cy={cy}
                                r={inTop10 ? 5 : 3}
                                fill={inTop10 ? '#10b981' : '#f59e0b'}
                                stroke={inTop10 ? '#10b981' : '#f59e0b'}
                                strokeWidth={1}
                              />
                            );
                          }}
                        />
                        {/* Reference line at rank 10 */}
                        <Line
                          type="monotone"
                          dataKey={() => 10}
                          stroke="#10b981"
                          strokeWidth={1}
                          strokeDasharray="5 5"
                          dot={false}
                          name="Top 10"
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>

                  {/* Price History */}
                  <div className="h-48">
                    <p className="text-white/60 text-xs mb-2">Evolucao do Preco</p>
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart
                        data={rankingHistory.history}
                        margin={{ top: 5, right: 20, left: 0, bottom: 5 }}
                      >
                        <defs>
                          <linearGradient id="colorPrice" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3} />
                            <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                        <XAxis
                          dataKey="date"
                          tick={{ fill: '#94a3b8', fontSize: 10 }}
                          interval={Math.floor(rankingHistory.history.length / 6)}
                        />
                        <YAxis
                          tick={{ fill: '#94a3b8', fontSize: 12 }}
                          tickFormatter={(v) => formatCurrency(v)}
                        />
                        <Tooltip
                          formatter={(value) => [formatCurrency(Number(value)), 'Preco']}
                          labelStyle={{ color: '#94a3b8' }}
                          contentStyle={{
                            backgroundColor: '#1e293b',
                            border: '1px solid rgba(255,255,255,0.2)',
                            borderRadius: '8px',
                          }}
                        />
                        <Area
                          type="monotone"
                          dataKey="price"
                          stroke="#8b5cf6"
                          fill="url(#colorPrice)"
                          strokeWidth={2}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                </div>
              )}
            </div>

            {/* Strategy Explanation */}
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-lg font-semibold text-white mb-3">Sobre a Magic Formula</h3>
              <div className="text-white/60 text-sm space-y-2">
                <p>
                  A <span className="text-white/80 font-medium">Magic Formula</span> e uma estrategia
                  de investimento criada por Joel Greenblatt que combina dois indicadores fundamentalistas:
                </p>
                <div className="grid md:grid-cols-2 gap-4 mt-3">
                  <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl p-4">
                    <p className="text-emerald-300 font-semibold mb-1">EBIT/EV (Earnings Yield)</p>
                    <p className="text-white/50 text-xs">
                      Mede o retorno dos lucros em relacao ao valor da empresa.
                      Quanto maior, mais barata a acao esta em relacao aos seus lucros.
                    </p>
                  </div>
                  <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-4">
                    <p className="text-blue-300 font-semibold mb-1">ROIC (Return on Invested Capital)</p>
                    <p className="text-white/50 text-xs">
                      Mede a eficiencia da empresa em gerar lucros com o capital investido.
                      Quanto maior, melhor a empresa utiliza seus recursos.
                    </p>
                  </div>
                </div>
                <p className="mt-3 text-white/50 text-xs">
                  O ranking final combina ambos os indicadores. Acoes com ranking mais baixo (1, 2, 3...)
                  sao consideradas as melhores oportunidades — empresas baratas E de qualidade.
                </p>
              </div>
            </div>
          </>
        ))}

        {/* ═══ FIIs TAB ═══ */}
        {activeTab === 'fiis' && (loadingFiis ? (
          <div className="text-center py-12 text-white/60">Carregando dados de FIIs do Yahoo Finance...</div>
        ) : (
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
                      <Bar
                        dataKey="dy"
                        radius={[4, 4, 0, 0]}
                        fill="#3b82f6"
                      />
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
                          <td className="py-3 pr-3 text-right text-white/80">
                            {formatCurrency(fii.currentPrice)}
                          </td>
                          <td className="py-3 pr-3 text-right">
                            <span className={`font-semibold ${
                              fii.dividendYield >= cdiYield ? 'text-emerald-400' :
                              fii.dividendYield >= cdiYield * 0.8 ? 'text-blue-400' :
                              'text-white/60'
                            }`}>
                              {fii.dividendYield.toFixed(2)}%
                            </span>
                          </td>
                          <td className="py-3 pr-3 text-right">
                            {fii.priceToBook != null ? (
                              <span className={`font-medium ${
                                fii.priceToBook < 0.95 ? 'text-emerald-400' :
                                fii.priceToBook <= 1.05 ? 'text-blue-400' :
                                'text-rose-400'
                              }`}>
                                {fii.priceToBook.toFixed(2)}
                              </span>
                            ) : (
                              <span className="text-white/30">-</span>
                            )}
                          </td>
                          <td className="py-3 pr-3 text-right">
                            <span className={`font-semibold ${
                              fii.totalReturn12M >= 0 ? 'text-emerald-400' : 'text-rose-400'
                            }`}>
                              {fii.totalReturn12M >= 0 ? '+' : ''}{fii.totalReturn12M.toFixed(2)}%
                            </span>
                          </td>
                          <td className="py-3 pr-3 text-right text-white/70">
                            {formatCurrency(fii.dividends12M)}
                          </td>
                          <td className="py-3 pr-3 text-right text-white/60 text-xs">
                            {formatCurrency(fii.lastDividend)}
                            {fii.lastDividendDate && (
                              <span className="block text-white/40">{fii.lastDividendDate}</span>
                            )}
                          </td>
                          <td className="py-3 text-right">
                            <span className={`font-medium ${
                              fii.priceChange12M >= 0 ? 'text-emerald-400' : 'text-rose-400'
                            }`}>
                              {fii.priceChange12M >= 0 ? '+' : ''}{fii.priceChange12M.toFixed(2)}%
                            </span>
                          </td>
                        </tr>
                        {/* Expanded: monthly dividends */}
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
              {/* P/VP Legend */}
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
        ))}
      </div>
    </AppLayout>
  );
}

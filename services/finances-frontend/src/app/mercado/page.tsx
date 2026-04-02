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
} from 'recharts';

const PROXY_BASE = '/api/magic-formula';

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
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [recommendationDate, setRecommendationDate] = useState('');
  const [backtest, setBacktest] = useState<BacktestData | null>(null);
  const [rankingHistory, setRankingHistory] = useState<RankingHistory | null>(null);
  const [selectedTicker, setSelectedTicker] = useState('');
  const [searchTicker, setSearchTicker] = useState('');
  const [topN, setTopN] = useState(10);
  const [loading, setLoading] = useState(true);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [error, setError] = useState('');

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

  useEffect(() => {
    (async () => {
      setLoading(true);
      await Promise.all([loadRecommendations(), loadBacktest()]);
      setLoading(false);
    })();
  }, [loadRecommendations, loadBacktest]);

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
            Recomendacoes baseadas na estrategia Magic Formula (Joel Greenblatt)
          </p>
        </div>

        {loading ? (
          <div className="text-center py-12 text-white/60">Carregando dados do mercado...</div>
        ) : error && stocks.length === 0 ? (
          <div className="text-center py-12 text-rose-400">{error}</div>
        ) : (
          <>
            {/* Top N selector */}
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
                {recommendationDate && (
                  <span className="text-white/50 text-sm">
                    Dados de: {recommendationDate}
                  </span>
                )}
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
                    {stocks.map((stock) => (
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
                          dot={(props: { cx: number; cy: number; payload: { in_top10: boolean } }) => {
                            const { cx, cy, payload } = props;
                            return (
                              <circle
                                key={`${cx}-${cy}`}
                                cx={cx}
                                cy={cy}
                                r={payload.in_top10 ? 5 : 3}
                                fill={payload.in_top10 ? '#10b981' : '#f59e0b'}
                                stroke={payload.in_top10 ? '#10b981' : '#f59e0b'}
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
                          formatter={(value: number) => [formatCurrency(value), 'Preco']}
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
        )}
      </div>
    </AppLayout>
  );
}

'use client';

import { formatCurrency } from '@/utils/format';

interface DreData {
  grossRevenue: number;
  tax: number;
  netRevenue: number;
  fixedCost: number;
  variableCost: number;
  prolabore: number;
  marketing: number;
  financial: number;
  operationalResult: number;
  finalResult: number;
}

interface DreSummaryProps {
  dreData: DreData;
  period: string; // ex: "Abril 2026" or "2026"
}

interface DreRow {
  label: string;
  value: number;
  indent?: boolean;
  bold?: boolean;
  separator?: boolean;
  isResult?: boolean;
  positiveIsGood?: boolean;
}

function DreLineRow({ row }: { row: DreRow }) {
  const isPositive = row.value >= 0;
  const valueColor = row.isResult
    ? isPositive ? 'text-emerald-300' : 'text-rose-300'
    : row.value < 0 ? 'text-rose-300' : 'text-white/80';

  return (
    <div className={`flex items-center justify-between py-2 ${row.separator ? 'border-t border-white/10 mt-1' : ''}`}>
      <span className={`text-sm ${row.indent ? 'pl-4 text-white/60' : ''} ${row.bold ? 'font-semibold text-white' : 'text-white/80'}`}>
        {row.label}
      </span>
      <span className={`text-sm font-mono ${row.bold ? 'font-bold' : ''} ${valueColor}`}>
        {row.value < 0 ? `(${formatCurrency(Math.abs(row.value))})` : formatCurrency(row.value)}
      </span>
    </div>
  );
}

export default function DreSummary({ dreData, period }: DreSummaryProps) {
  const hasData = dreData.grossRevenue !== 0 || dreData.tax !== 0 || dreData.fixedCost !== 0 ||
    dreData.variableCost !== 0 || dreData.prolabore !== 0 || dreData.marketing !== 0;

  const rows: DreRow[] = [
    { label: '(+) Receita Bruta', value: dreData.grossRevenue },
    { label: '(-) Impostos / Deduções', value: dreData.tax, indent: true },
    { label: '(=) Receita Líquida', value: dreData.netRevenue, bold: true, separator: true, isResult: true },
    { label: '(-) Custos Fixos', value: dreData.fixedCost, indent: true },
    { label: '(-) Custos Variáveis', value: dreData.variableCost, indent: true },
    { label: '(-) Pró-labore', value: dreData.prolabore, indent: true },
    { label: '(=) Resultado Operacional', value: dreData.operationalResult, bold: true, separator: true, isResult: true },
    { label: '(-) Marketing / Anúncios', value: dreData.marketing, indent: true },
    { label: '(+/-) Financeiro', value: dreData.financial, indent: true },
    { label: '(=) Resultado Final', value: dreData.finalResult, bold: true, separator: true, isResult: true },
  ];

  return (
    <div className="bg-white/5 border border-amber-400/20 rounded-2xl p-5">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-white font-bold text-lg">DRE — Demonstração de Resultado</h3>
          <p className="text-white/50 text-xs mt-0.5">{period} · baseado nas classificações das categorias</p>
        </div>
        <span className={`text-lg font-bold px-3 py-1 rounded-xl border ${
          dreData.finalResult >= 0
            ? 'bg-emerald-500/15 text-emerald-300 border-emerald-400/30'
            : 'bg-rose-500/15 text-rose-300 border-rose-400/30'
        }`}>
          {dreData.finalResult >= 0 ? 'Lucro' : 'Prejuízo'}
        </span>
      </div>

      {!hasData ? (
        <div className="text-center py-6 text-white/40 text-sm">
          Nenhuma categoria com classificação DRE encontrada no período.
          <br />
          <span className="text-xs">Acesse <strong>Categorias</strong> e classifique suas categorias para ver a DRE.</span>
        </div>
      ) : (
        <div className="divide-y divide-white/5">
          {rows.map((row, i) => (
            <DreLineRow key={i} row={row} />
          ))}
        </div>
      )}
    </div>
  );
}

'use client';

import { useState } from 'react';
import type { CategoryTrend } from './types';
import { fmtShort, monthLabel, trendIcon, trendColor } from './format';

export function CategoryTable({
  title,
  categories,
  periods,
  invertTrend = false,
}: {
  title: string;
  categories: CategoryTrend[];
  periods: string[];
  invertTrend?: boolean;
}) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const toggle = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  return (
    <section className="bg-white/5 border border-white/10 rounded-2xl p-5 overflow-x-auto">
      <h2 className="text-white font-semibold mb-3">{title}</h2>
      <table className="w-full text-sm min-w-[640px]">
        <thead>
          <tr className="text-white/50 text-xs">
            <th className="text-left font-medium pb-2">Categoria</th>
            {periods.map((p, i) => (
              <th key={i} className="text-right font-medium pb-2 px-2">{monthLabel(p)}</th>
            ))}
            <th className="text-right font-medium pb-2 px-2">média</th>
            <th className="text-center font-medium pb-2 pl-2">tend.</th>
          </tr>
        </thead>
        <tbody>
          {(categories ?? []).map((cat) => (
            <CategoryRows key={cat.categoryId || cat.name} cat={cat} expanded={expanded} toggle={toggle} invertTrend={invertTrend} />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function CategoryRows({
  cat,
  expanded,
  toggle,
  invertTrend,
}: {
  cat: CategoryTrend;
  expanded: Set<string>;
  toggle: (id: string) => void;
  invertTrend: boolean;
}) {
  const id = cat.categoryId || cat.name;
  const hasChildren = (cat.children?.length ?? 0) > 0;
  const isOpen = expanded.has(id);
  return (
    <>
      <tr className="border-t border-white/5 hover:bg-white/5">
        <td className="py-2 pr-2">
          <button
            onClick={() => hasChildren && toggle(id)}
            className={`flex items-center gap-1.5 text-left ${hasChildren ? 'text-white hover:text-purple-300' : 'text-white/90'}`}
          >
            {hasChildren ? <span className="text-white/40 text-xs w-3">{isOpen ? '▾' : '▸'}</span> : <span className="w-3" />}
            <span className="truncate">{cat.name}</span>
          </button>
        </td>
        {cat.byMonth.map((v, i) => (
          <td key={i} className="text-right px-2 py-2 text-white/70 tabular-nums">{v ? fmtShort(v) : '·'}</td>
        ))}
        <td className="text-right px-2 py-2 text-white/90 font-medium tabular-nums">{fmtShort(cat.average)}</td>
        <td className={`text-center pl-2 py-2 ${trendColor(cat.trend, invertTrend)}`}>{trendIcon(cat.trend)}</td>
      </tr>
      {isOpen &&
        cat.children?.map((child) => (
          <tr key={child.categoryId || child.name} className="bg-black/20">
            <td className="py-1.5 pr-2 pl-8 text-white/60 truncate">{child.name}</td>
            {child.byMonth.map((v, i) => (
              <td key={i} className="text-right px-2 py-1.5 text-white/50 tabular-nums">{v ? fmtShort(v) : '·'}</td>
            ))}
            <td className="text-right px-2 py-1.5 text-white/60 tabular-nums">{fmtShort(child.average)}</td>
            <td className={`text-center pl-2 py-1.5 ${trendColor(child.trend, invertTrend)}`}>{trendIcon(child.trend)}</td>
          </tr>
        ))}
    </>
  );
}

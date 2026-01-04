'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import type { Profile, RecurringTransaction } from '@/types/finances';

const API_BASE = 'http://localhost:3335/api/v1';

export default function RecurringPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [items, setItems] = useState<RecurringTransaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedProfile = useMemo(
    () => profiles.find((p) => p.id === selectedProfileId) || null,
    [profiles, selectedProfileId],
  );

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/profiles`);
        const data = await res.json();
        const list: Profile[] = data.data || [];
        setProfiles(list);
        if (list.length > 0) setSelectedProfileId(list[0].id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  useEffect(() => {
    if (!selectedProfileId) return;
    (async () => {
      try {
        setLoading(true);
        setError(null);
        const res = await fetch(`${API_BASE}/recurring-transactions?profileId=${selectedProfileId}`);
        if (!res.ok) throw new Error(`status ${res.status}`);
        const data = await res.json();
        setItems(data.data || []);
      } catch (e) {
        console.warn('Erro ao carregar recorrentes', e);
        setItems([]);
        setError('Não foi possível carregar as transações recorrentes.');
      } finally {
        setLoading(false);
      }
    })();
  }, [selectedProfileId]);

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Despesas Fixas (Recorrentes)</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-white/70 text-sm">Visualizando dados de:</p>
            <div className="flex flex-wrap gap-2">
              {profiles.map((profile) => {
                const isSelected = selectedProfileId === profile.id;
                return (
                  <button
                    key={profile.id}
                    onClick={() => setSelectedProfileId(profile.id)}
                    className={`px-3 py-1.5 rounded-xl border transition-colors ${
                      isSelected ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'
                    }`}
                  >
                    {profile.name}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          {!selectedProfile && (
            <p className="text-white/70">Selecione um perfil para visualizar os lançamentos recorrentes.</p>
          )}
          {selectedProfile && (
            <>
              {loading && <p className="text-white/70">Carregando...</p>}
              {error && (
                <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3 mb-3">
                  {error}
                </div>
              )}
              {items.length === 0 && !loading ? (
                <p className="text-white/70">Nenhuma transação recorrente encontrada.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead>
                      <tr className="text-white/60">
                        <th className="text-left py-2 pr-4">Descrição</th>
                        <th className="text-left py-2 pr-4">Tipo</th>
                        <th className="text-left py-2 pr-4">Valor</th>
                        <th className="text-left py-2 pr-4">Recorrência</th>
                        <th className="text-left py-2">Próxima</th>
                      </tr>
                    </thead>
                    <tbody>
                      {items.map((it) => (
                        <tr key={it.id} className="text-white/90 border-t border-white/10">
                          <td className="py-2 pr-4">{it.description}</td>
                          <td className="py-2 pr-4">{it.type}</td>
                          <td className="py-2 pr-4">
                            {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: it.currency }).format(it.amount)}
                          </td>
                          <td className="py-2 pr-4">{it.recurrenceRule}</td>
                          <td className="py-2">{new Date(it.nextOccurrence).toLocaleDateString('pt-BR')}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </AppLayout>
  );
}


'use client';

import { useEffect, useState, useCallback } from 'react';
import { useSearchParams } from 'next/navigation';
import AppLayout from '@/components/navigation/AppLayout';
import { Calendar } from '@/types/calendar';
import { api } from '@/lib/api';

type SyncResult = { created: number; updated: number; deleted: number };
type SyncState = Record<string, 'idle' | 'syncing' | 'done' | 'error'>;

export default function IntegrationsPage() {
  const searchParams = useSearchParams();
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncStates, setSyncStates] = useState<SyncState>({});
  const [syncResults, setSyncResults] = useState<Record<string, SyncResult>>({});
  const [toast, setToast] = useState<string | null>(null);

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 4000);
  };

  const fetchCalendars = useCallback(async () => {
    try {
      const data = await api.calendars.list();
      setCalendars(data);
    } catch (error) {
      console.error('Error fetching calendars:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCalendars();
  }, [fetchCalendars]);

  useEffect(() => {
    if (searchParams.get('connected') === 'true') {
      showToast('Google Calendar conectado com sucesso!');
      fetchCalendars();
    }
  }, [searchParams, fetchCalendars]);

  const handleConnect = (calendarId: string) => {
    window.location.href = api.googleCalendar.getConnectUrl(calendarId);
  };

  const handleDisconnect = async (calendarId: string) => {
    if (!confirm('Deseja desconectar o Google Calendar deste perfil?')) return;
    try {
      await api.googleCalendar.disconnect(calendarId);
      setCalendars((prev) =>
        prev.map((c) =>
          c.id === calendarId
            ? { ...c, googleCalendarId: null, googleSyncToken: null, lastSyncAt: null }
            : c,
        ),
      );
      showToast('Google Calendar desconectado.');
    } catch (error) {
      console.error('Error disconnecting:', error);
      showToast('Erro ao desconectar. Tente novamente.');
    }
  };

  const handleSync = async (calendarId: string) => {
    setSyncStates((prev) => ({ ...prev, [calendarId]: 'syncing' }));
    try {
      const res = await api.googleCalendar.pullSync(calendarId);
      const result = res.data;
      setSyncResults((prev) => ({ ...prev, [calendarId]: result }));
      setSyncStates((prev) => ({ ...prev, [calendarId]: 'done' }));
      fetchCalendars();
    } catch (error) {
      console.error('Error syncing:', error);
      setSyncStates((prev) => ({ ...prev, [calendarId]: 'error' }));
    }
  };

  const formatLastSync = (lastSyncAt: string | null | undefined) => {
    if (!lastSyncAt) return 'Nunca sincronizado';
    const d = new Date(lastSyncAt);
    return d.toLocaleString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <AppLayout>
      <div className="flex-1 w-full py-8 relative">
        {/* Toast */}
        {toast && (
          <div className="fixed top-6 right-6 z-50 bg-white/20 backdrop-blur-sm text-white px-6 py-3 rounded-xl shadow-lg border border-white/20 transition-all">
            {toast}
          </div>
        )}

        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-extrabold text-white drop-shadow-lg mb-2">
            Integrações
          </h1>
          <p className="text-white/70 text-lg">
            Conecte seus perfis ao Google Calendar para sincronização bidirecional
          </p>
        </div>

        {/* Google Calendar Section */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-white/10 flex items-center justify-center">
              <svg viewBox="0 0 24 24" className="w-6 h-6" fill="none">
                <rect x="3" y="3" width="18" height="18" rx="2" fill="white" fillOpacity="0.15" stroke="white" strokeOpacity="0.4" strokeWidth="1.5" />
                <path d="M3 9h18" stroke="white" strokeOpacity="0.4" strokeWidth="1.5" />
                <path d="M8 3v6M16 3v6" stroke="white" strokeOpacity="0.4" strokeWidth="1.5" strokeLinecap="round" />
                <circle cx="12" cy="15" r="2" fill="white" fillOpacity="0.6" />
              </svg>
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Google Calendar</h2>
              <p className="text-white/60 text-sm">
                Eventos criados aqui aparecem no Google, e eventos do Google aparecem aqui
              </p>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-16">
              <div className="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {calendars.map((calendar) => {
                const isConnected = !!calendar.googleCalendarId;
                const syncState = syncStates[calendar.id] || 'idle';
                const syncResult = syncResults[calendar.id];

                return (
                  <div
                    key={calendar.id}
                    className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20 shadow-lg"
                  >
                    {/* Calendar header */}
                    <div className="flex items-center gap-3 mb-4">
                      <div
                        className="w-10 h-10 rounded-xl flex-shrink-0"
                        style={{ backgroundColor: calendar.color }}
                      />
                      <div className="flex-1 min-w-0">
                        <h3 className="text-white font-bold text-lg leading-tight truncate">
                          {calendar.name}
                        </h3>
                        <p className="text-white/50 text-xs truncate">{calendar.email}</p>
                      </div>
                      <span
                        className={`px-2 py-1 rounded-lg text-xs font-semibold flex-shrink-0 ${
                          isConnected
                            ? 'bg-green-500/20 text-green-300 border border-green-500/30'
                            : 'bg-white/10 text-white/50 border border-white/10'
                        }`}
                      >
                        {isConnected ? 'Conectado' : 'Desconectado'}
                      </span>
                    </div>

                    {/* Connected state info */}
                    {isConnected && (
                      <div className="mb-4 bg-white/5 rounded-xl p-3 border border-white/10">
                        <div className="flex items-center gap-2 text-white/60 text-sm mb-1">
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          <span>Última sincronização</span>
                        </div>
                        <p className="text-white/80 text-sm font-medium pl-6">
                          {formatLastSync(calendar.lastSyncAt)}
                        </p>

                        {syncState === 'done' && syncResult && (
                          <div className="mt-2 pl-6 flex gap-3 text-xs text-white/60">
                            <span className="text-green-400">+{syncResult.created} criados</span>
                            <span className="text-blue-400">~{syncResult.updated} atualizados</span>
                            <span className="text-red-400">-{syncResult.deleted} deletados</span>
                          </div>
                        )}
                        {syncState === 'error' && (
                          <p className="mt-2 pl-6 text-xs text-red-400">
                            Erro ao sincronizar. Tente novamente.
                          </p>
                        )}
                      </div>
                    )}

                    {/* Actions */}
                    <div className="flex gap-2">
                      {isConnected ? (
                        <>
                          <button
                            onClick={() => handleSync(calendar.id)}
                            disabled={syncState === 'syncing'}
                            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-white/15 hover:bg-white/25 text-white text-sm font-semibold transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {syncState === 'syncing' ? (
                              <>
                                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                                Sincronizando...
                              </>
                            ) : (
                              <>
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                                </svg>
                                Sincronizar agora
                              </>
                            )}
                          </button>
                          <button
                            onClick={() => handleDisconnect(calendar.id)}
                            className="px-4 py-2.5 rounded-xl bg-red-500/15 hover:bg-red-500/25 text-red-300 text-sm font-semibold transition-all duration-200 border border-red-500/20"
                          >
                            Desconectar
                          </button>
                        </>
                      ) : (
                        <button
                          onClick={() => handleConnect(calendar.id)}
                          className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-white/20 hover:bg-white/30 text-white text-sm font-semibold transition-all duration-200 border border-white/20"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                          </svg>
                          Conectar Google Calendar
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* How it works */}
        <div className="bg-white/5 rounded-2xl p-6 border border-white/10">
          <h3 className="text-white font-bold mb-3">Como funciona</h3>
          <ul className="space-y-2 text-white/60 text-sm">
            <li className="flex items-start gap-2">
              <span className="text-green-400 mt-0.5">→</span>
              <span>Eventos criados ou editados aqui são automaticamente sincronizados para o Google Calendar</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-blue-400 mt-0.5">→</span>
              <span>Eventos criados no Google Calendar aparecem aqui automaticamente a cada 5 minutos</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-purple-400 mt-0.5">→</span>
              <span>Use "Sincronizar agora" para buscar mudanças imediatamente</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-yellow-400 mt-0.5">→</span>
              <span>Cada perfil conecta a uma conta Google diferente (pessoal e profissional)</span>
            </li>
          </ul>
        </div>
      </div>
    </AppLayout>
  );
}

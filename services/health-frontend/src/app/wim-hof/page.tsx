'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import { useWimHofSession, Phase } from '@/hooks/useWimHofSession';
import { profilesApi, activitiesApi } from '@/lib/api';
import { Profile } from '@/types/health';

function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

function BreathingCircle({ phase, breathCount, totalBreaths, holdTime, recoveryTime }: {
  phase: Phase;
  breathCount: number;
  totalBreaths: number;
  holdTime: number;
  recoveryTime: number;
}) {
  const getCircleStyle = () => {
    switch (phase) {
      case 'BREATHING':
        return 'border-cyan-400/60 shadow-[0_0_30px_rgba(34,211,238,0.2)] animate-[breathe-pulse_2s_ease-in-out_infinite]';
      case 'HOLD':
        return 'border-red-400/60 shadow-[0_0_30px_rgba(239,68,68,0.3)] animate-[hold-glow_2s_ease-in-out_infinite] scale-100';
      case 'RECOVERY':
        return 'border-emerald-400/60 shadow-[0_0_30px_rgba(52,211,153,0.2)] scale-90';
      default:
        return 'border-white/20';
    }
  };

  const getCenterContent = () => {
    switch (phase) {
      case 'BREATHING':
        return (
          <>
            <span className="text-5xl font-bold text-cyan-300">{breathCount}</span>
            <span className="text-lg text-white/50">/ {totalBreaths}</span>
          </>
        );
      case 'HOLD':
        return (
          <span className="text-5xl font-bold text-red-300 font-mono">{formatTime(holdTime)}</span>
        );
      case 'RECOVERY':
        return (
          <span className="text-5xl font-bold text-emerald-300 font-mono">{recoveryTime}s</span>
        );
      default:
        return null;
    }
  };

  return (
    <div className={`w-56 h-56 rounded-full border-4 flex flex-col items-center justify-center transition-all duration-300 ${getCircleStyle()}`}>
      {getCenterContent()}
    </div>
  );
}

export default function WimHofPage() {
  const router = useRouter();
  const session = useWimHofSession();
  const { state } = session;

  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState('');
  const [showProfileModal, setShowProfileModal] = useState(false);
  const [profileLoading, setProfileLoading] = useState(true);
  const [rating, setRating] = useState<number>(0);
  const [notes, setNotes] = useState('');
  const [saving, setSaving] = useState(false);
  const [pushUpInput, setPushUpInput] = useState('');

  const PROFILE_STORAGE_KEY = 'health-selected-profile';

  useEffect(() => {
    profilesApi.list().then(data => {
      setProfiles(data);
      const savedId = localStorage.getItem(PROFILE_STORAGE_KEY);
      const savedExists = savedId && data.some(p => p.id === savedId);
      if (savedExists) {
        setSelectedProfile(savedId!);
      } else if (data.length === 1) {
        setSelectedProfile(data[0].id);
        localStorage.setItem(PROFILE_STORAGE_KEY, data[0].id);
      } else if (data.length > 1) {
        setShowProfileModal(true);
      }
    }).finally(() => setProfileLoading(false));
  }, []);

  const selectProfile = (id: string) => {
    setSelectedProfile(id);
    localStorage.setItem(PROFILE_STORAGE_KEY, id);
    setShowProfileModal(false);
  };

  const handleSave = useCallback(async () => {
    if (!selectedProfile) return;
    setSaving(true);
    try {
      const data = session.getSessionData(selectedProfile, rating || undefined, notes || undefined);
      await activitiesApi.create(data);
      router.push('/');
    } catch (error) {
      console.error('Error saving session:', error);
      alert('Erro ao salvar sessao');
    } finally {
      setSaving(false);
    }
  }, [selectedProfile, rating, notes, session, router]);

  const isActivePhase = ['BREATHING', 'HOLD', 'RECOVERY', 'PUSHUPS'].includes(state.phase);

  // Active session overlay
  if (isActivePhase) {
    return (
      <div className="fixed inset-0 z-50 bg-gradient-to-b from-gray-950 via-gray-900 to-gray-950 flex flex-col">
        {/* Top bar */}
        <div className="flex items-center justify-between px-6 py-4">
          <button
            onClick={() => { session.reset(); }}
            className="p-2 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
            title="Sair"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
          <div className="flex items-center gap-3">
            <span className="text-white/60 text-sm">Round</span>
            <span className="text-white font-bold text-lg">{state.currentRound}/{state.totalRounds}</span>
          </div>
          <div className="flex items-center gap-2">
            {state.phase === 'BREATHING' && (
              <span className="px-3 py-1 bg-cyan-500/20 text-cyan-300 rounded-full text-sm">Respiracao</span>
            )}
            {state.phase === 'HOLD' && (
              <span className="px-3 py-1 bg-red-500/20 text-red-300 rounded-full text-sm">Retencao</span>
            )}
            {state.phase === 'RECOVERY' && (
              <span className="px-3 py-1 bg-emerald-500/20 text-emerald-300 rounded-full text-sm">Recuperacao</span>
            )}
            {state.phase === 'PUSHUPS' && (
              <span className="px-3 py-1 bg-amber-500/20 text-amber-300 rounded-full text-sm">Flexoes</span>
            )}
          </div>
        </div>

        {/* Round progress dots */}
        <div className="flex justify-center gap-2 px-6 pb-4">
          {Array.from({ length: state.totalRounds }, (_, i) => (
            <div
              key={i}
              className={`h-1.5 rounded-full transition-all ${
                i < state.currentRound - 1
                  ? 'w-8 bg-cyan-400'
                  : i === state.currentRound - 1
                    ? 'w-8 bg-white'
                    : 'w-8 bg-white/20'
              }`}
            />
          ))}
        </div>

        {/* Main content */}
        <div className="flex-1 flex flex-col items-center justify-center gap-8">
          {state.phase === 'PUSHUPS' ? (
            // Push-ups input
            <>
              <div className="text-6xl">💪</div>
              <h2 className="text-2xl font-bold text-white">Flexoes</h2>
              <p className="text-white/60">Faca o maximo de flexoes que conseguir!</p>
              <div className="flex items-center gap-4">
                <input
                  type="number"
                  inputMode="numeric"
                  value={pushUpInput}
                  onChange={(e) => {
                    setPushUpInput(e.target.value);
                    const n = parseInt(e.target.value);
                    if (!isNaN(n)) session.setPushUpCount(n);
                  }}
                  placeholder="0"
                  className="w-28 text-center text-4xl font-bold bg-white/10 border border-white/20 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-amber-400/50"
                  autoFocus
                />
              </div>
              <div className="flex gap-3 mt-4">
                <button
                  onClick={() => session.confirmPushUps()}
                  className="px-8 py-3 bg-amber-600 hover:bg-amber-500 text-white rounded-xl text-lg font-medium transition-colors"
                >
                  Registrar
                </button>
                <button
                  onClick={() => session.skipPushUps()}
                  className="px-8 py-3 bg-white/10 hover:bg-white/20 text-white rounded-xl text-lg transition-colors"
                >
                  Pular
                </button>
              </div>
            </>
          ) : (
            // Breathing / Hold / Recovery
            <>
              <BreathingCircle
                phase={state.phase}
                breathCount={state.currentBreath}
                totalBreaths={state.breathsPerRound}
                holdTime={state.holdTimeSeconds}
                recoveryTime={state.recoveryTimeSeconds}
              />

              {state.phase === 'BREATHING' && (
                <>
                  <p className="text-white/50 text-sm">Respire forte pelo nariz, solte pela boca</p>
                  <p className="text-cyan-300/60 text-xs">Acompanhe o ritmo do pulso</p>
                  {state.currentBreath > 0 && (
                    <button
                      onClick={() => session.skipToHold()}
                      className="px-8 py-3 bg-red-600/20 border border-red-400/30 hover:bg-red-600/30 active:scale-95 rounded-xl transition-all"
                    >
                      <span className="text-red-300 text-base font-medium">Segurar agora</span>
                    </button>
                  )}
                </>
              )}

              {state.phase === 'HOLD' && (
                <>
                  <button
                    onClick={() => session.endHold()}
                    className="px-10 py-4 bg-red-600/20 border-2 border-red-400/40 hover:bg-red-600/30 active:scale-95 rounded-xl transition-all"
                  >
                    <span className="text-red-200 text-lg font-medium">Respirei</span>
                  </button>
                  <p className="text-white/40 text-sm">Segure a respiracao o maximo que conseguir</p>
                </>
              )}

              {state.phase === 'RECOVERY' && (
                <p className="text-emerald-300/80 text-lg">Inspire fundo e segure...</p>
              )}
            </>
          )}
        </div>

        {/* Retention times from previous rounds */}
        {state.retentionTimes.length > 0 && (
          <div className="px-6 pb-6">
            <div className="flex justify-center gap-4">
              {state.retentionTimes.map((time, i) => (
                <div key={i} className="text-center">
                  <p className="text-white/40 text-xs">R{i + 1}</p>
                  <p className="text-white/80 text-sm font-mono">{formatTime(time)}</p>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    );
  }

  // SUMMARY phase
  if (state.phase === 'SUMMARY') {
    const totalDuration = state.sessionStartTime && state.sessionEndTime
      ? Math.round((state.sessionEndTime.getTime() - state.sessionStartTime.getTime()) / 1000)
      : 0;
    const bestRetention = Math.max(...(state.retentionTimes.length > 0 ? state.retentionTimes : [0]));

    return (
      <AppLayout>
        <div className="max-w-lg mx-auto space-y-6">
          <div className="text-center mb-8">
            <div className="text-5xl mb-4">🧊</div>
            <h2 className="text-2xl font-bold text-white">Sessao Completa!</h2>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-3 gap-3">
            <div className="bg-white/5 border border-white/10 rounded-xl p-4 text-center">
              <p className="text-white/50 text-xs">Rounds</p>
              <p className="text-2xl font-bold text-white">{state.totalRounds}</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-xl p-4 text-center">
              <p className="text-white/50 text-xs">Tempo Total</p>
              <p className="text-2xl font-bold text-white">{formatTime(totalDuration)}</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-xl p-4 text-center">
              <p className="text-white/50 text-xs">Melhor Retencao</p>
              <p className="text-2xl font-bold text-cyan-300">{formatTime(bestRetention)}</p>
            </div>
          </div>

          {/* Round details */}
          <div className="bg-white/5 border border-white/10 rounded-xl p-4">
            <h3 className="text-sm font-semibold text-white/60 mb-3">Detalhes por Round</h3>
            <div className="space-y-2">
              {state.retentionTimes.map((time, i) => (
                <div key={i} className="flex items-center justify-between bg-white/5 rounded-lg px-4 py-2">
                  <span className="text-white/60 text-sm">Round {i + 1}</span>
                  <div className="flex items-center gap-4">
                    <span className="text-white/40 text-xs">{state.roundBreaths[i] || state.breathsPerRound} resp</span>
                    <span className="text-white font-mono font-medium">{formatTime(time)}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Push-ups */}
          {state.pushUpCount !== null && (
            <div className="bg-white/5 border border-white/10 rounded-xl p-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="text-2xl">💪</span>
                <span className="text-white">Flexoes</span>
              </div>
              <span className="text-2xl font-bold text-amber-300">{state.pushUpCount}</span>
            </div>
          )}

          {/* Rating */}
          <div className="bg-white/5 border border-white/10 rounded-xl p-4">
            <h3 className="text-sm font-semibold text-white/60 mb-3">Avaliacao</h3>
            <div className="flex gap-2">
              {[1, 2, 3, 4, 5].map(star => (
                <button
                  key={star}
                  onClick={() => setRating(star)}
                  className={`text-2xl transition-transform hover:scale-110 ${star <= rating ? 'grayscale-0' : 'grayscale opacity-30'}`}
                >
                  ⭐
                </button>
              ))}
            </div>
          </div>

          {/* Notes */}
          <div className="bg-white/5 border border-white/10 rounded-xl p-4">
            <h3 className="text-sm font-semibold text-white/60 mb-3">Notas</h3>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Como foi a sessao?"
              rows={2}
              className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white text-sm placeholder:text-white/30 focus:outline-none focus:border-cyan-400/50"
            />
          </div>

          {/* Actions */}
          <div className="flex gap-3">
            <button
              onClick={handleSave}
              disabled={saving || !selectedProfile}
              className="flex-1 px-6 py-3 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-xl font-medium transition-colors"
            >
              {saving ? 'Salvando...' : 'Salvar'}
            </button>
            <button
              onClick={() => { session.reset(); router.push('/'); }}
              className="px-6 py-3 bg-white/10 hover:bg-white/20 text-white rounded-xl transition-colors"
            >
              Descartar
            </button>
          </div>
        </div>
      </AppLayout>
    );
  }

  // SETUP phase
  return (
    <AppLayout>
      <div className="max-w-lg mx-auto space-y-6">
        <Link href="/" className="text-violet-400 text-sm hover:underline inline-flex items-center gap-1">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Dashboard
        </Link>

        <div className="bg-white/5 border border-white/10 rounded-xl p-8 text-center space-y-6">
          <div className="text-6xl">🧊</div>
          <div>
            <h2 className="text-2xl font-bold text-white">Wim Hof Method</h2>
            <p className="text-white/60 mt-1">Respiracao guiada + retencao + flexoes</p>
          </div>

          {/* Settings */}
          <div className="grid grid-cols-2 gap-4 text-left">
            <div>
              <label className="block text-white/60 text-sm mb-1">Rounds de respiracao</label>
              <input
                type="number"
                inputMode="numeric"
                value={state.totalRounds}
                onChange={(e) => session.setTotalRounds(parseInt(e.target.value) || 4)}
                min={1}
                max={10}
                className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-center focus:outline-none focus:border-cyan-400/50"
              />
              <p className="text-white/30 text-xs mt-1">Cada round: respirar + segurar + recuperar</p>
            </div>
            <div>
              <label className="block text-white/60 text-sm mb-1">Respiracoes por round</label>
              <input
                type="number"
                inputMode="numeric"
                value={state.breathsPerRound}
                onChange={(e) => session.setBreathsPerRound(parseInt(e.target.value) || 30)}
                min={10}
                max={60}
                className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-center focus:outline-none focus:border-cyan-400/50"
              />
              <p className="text-white/30 text-xs mt-1">Alvo — pode interromper antes</p>
            </div>
          </div>

          {/* Selected profile indicator */}
          {selectedProfile && profiles.length > 0 && (
            <div className="flex items-center justify-center gap-2 text-white/40 text-sm">
              <span>Perfil: {profiles.find(p => p.id === selectedProfile)?.name}</span>
              {profiles.length > 1 && (
                <button onClick={() => setShowProfileModal(true)} className="text-violet-400 hover:underline">trocar</button>
              )}
            </div>
          )}

          {!profileLoading && profiles.length === 0 ? (
            <div className="text-center space-y-3">
              <p className="text-white/60">Nenhum perfil encontrado.</p>
              <Link href="/configuracoes" className="text-violet-400 hover:underline text-sm">
                Criar perfil em Configuracoes
              </Link>
            </div>
          ) : (
            <button
              onClick={() => session.startSession()}
              disabled={profileLoading || !selectedProfile}
              className="w-full px-6 py-4 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-xl text-lg font-semibold transition-colors"
            >
              Iniciar Sessao
            </button>
          )}
        </div>
      </div>

      {/* Profile selection modal */}
      {showProfileModal && (
        <div className="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4">
          <div className="bg-gray-900 border border-white/10 rounded-xl p-6 max-w-sm w-full space-y-4">
            <h3 className="text-lg font-semibold text-white">Selecione seu perfil</h3>
            <div className="space-y-2">
              {profiles.map(p => (
                <button
                  key={p.id}
                  onClick={() => selectProfile(p.id)}
                  className={`w-full text-left px-4 py-3 rounded-lg transition-colors ${
                    selectedProfile === p.id
                      ? 'bg-cyan-600/30 border border-cyan-400/50 text-white'
                      : 'bg-white/5 border border-white/10 text-white/80 hover:bg-white/10'
                  }`}
                >
                  {p.name}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
}

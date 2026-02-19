'use client';

import { useState, useEffect, useCallback } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { useStretchSession } from '@/hooks/useStretchSession';
import {
  profilesApi,
  stretchSequencesApi,
  stretchMovementsApi,
  stretchSessionsApi,
} from '@/lib/api';
import type { Profile, StretchSequence, StretchMovement, StretchSession } from '@/types/health';

function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

type Tab = 'sequences' | 'session' | 'history';

export default function AlongamentoPage() {
  const session = useStretchSession();
  const { state: sessionState } = session;

  const [tab, setTab] = useState<Tab>('sequences');
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState('');
  const [sequences, setSequences] = useState<StretchSequence[]>([]);
  const [sessions, setSessions] = useState<StretchSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingSequence, setEditingSequence] = useState<StretchSequence | null>(null);

  // New sequence form
  const [newSeqName, setNewSeqName] = useState('');
  const [newSeqDesc, setNewSeqDesc] = useState('');
  const [showNewSeqForm, setShowNewSeqForm] = useState(false);

  // New movement form
  const [newMovName, setNewMovName] = useState('');
  const [newMovDesc, setNewMovDesc] = useState('');
  const [newMovDuration, setNewMovDuration] = useState(30);

  // Session save
  const [rating, setRating] = useState(0);
  const [notes, setNotes] = useState('');
  const [saving, setSaving] = useState(false);

  const PROFILE_STORAGE_KEY = 'health-selected-profile';

  useEffect(() => {
    profilesApi.list().then(data => {
      setProfiles(data);
      const savedId = localStorage.getItem(PROFILE_STORAGE_KEY);
      const savedExists = savedId && data.some(p => p.id === savedId);
      if (savedExists) {
        setSelectedProfile(savedId!);
      } else if (data.length >= 1) {
        setSelectedProfile(data[0].id);
        localStorage.setItem(PROFILE_STORAGE_KEY, data[0].id);
      }
    }).finally(() => setLoading(false));
  }, []);

  const loadSequences = useCallback(async () => {
    if (!selectedProfile) return;
    try {
      const data = await stretchSequencesApi.list(selectedProfile);
      setSequences(data);
    } catch (e) {
      console.error('Failed to load sequences', e);
    }
  }, [selectedProfile]);

  const loadSessions = useCallback(async () => {
    if (!selectedProfile) return;
    try {
      const data = await stretchSessionsApi.list({ profileId: selectedProfile });
      setSessions(data);
    } catch (e) {
      console.error('Failed to load sessions', e);
    }
  }, [selectedProfile]);

  useEffect(() => {
    if (selectedProfile) {
      loadSequences();
      loadSessions();
    }
  }, [selectedProfile, loadSequences, loadSessions]);

  const handleCreateSequence = async () => {
    if (!newSeqName.trim() || !selectedProfile) return;
    try {
      await stretchSequencesApi.create({
        profileId: selectedProfile,
        name: newSeqName.trim(),
        description: newSeqDesc.trim() || undefined,
      });
      setNewSeqName('');
      setNewSeqDesc('');
      setShowNewSeqForm(false);
      loadSequences();
    } catch (e) {
      console.error('Failed to create sequence', e);
    }
  };

  const handleDeleteSequence = async (id: string) => {
    try {
      await stretchSequencesApi.delete(id);
      if (editingSequence?.id === id) setEditingSequence(null);
      loadSequences();
    } catch (e) {
      console.error('Failed to delete sequence', e);
    }
  };

  const handleAddMovement = async () => {
    if (!editingSequence || !newMovName.trim()) return;
    try {
      await stretchSequencesApi.addMovement(editingSequence.id, {
        name: newMovName.trim(),
        description: newMovDesc.trim() || undefined,
        durationSeconds: newMovDuration,
        orderIndex: editingSequence.movements?.length || 0,
      });
      setNewMovName('');
      setNewMovDesc('');
      setNewMovDuration(30);
      // Reload sequence
      const updated = await stretchSequencesApi.get(editingSequence.id);
      setEditingSequence(updated);
      loadSequences();
    } catch (e) {
      console.error('Failed to add movement', e);
    }
  };

  const handleDeleteMovement = async (movId: string) => {
    if (!editingSequence) return;
    try {
      await stretchMovementsApi.delete(movId);
      const updated = await stretchSequencesApi.get(editingSequence.id);
      setEditingSequence(updated);
      loadSequences();
    } catch (e) {
      console.error('Failed to delete movement', e);
    }
  };

  const handleUpdateMovementDuration = async (mov: StretchMovement, delta: number) => {
    const newDuration = Math.max(5, mov.durationSeconds + delta);
    try {
      await stretchMovementsApi.update(mov.id, {
        name: mov.name,
        description: mov.description,
        durationSeconds: newDuration,
        orderIndex: mov.orderIndex,
      });
      if (editingSequence) {
        const updated = await stretchSequencesApi.get(editingSequence.id);
        setEditingSequence(updated);
        loadSequences();
      }
    } catch (e) {
      console.error('Failed to update movement', e);
    }
  };

  const handleStartSession = (seq: StretchSequence) => {
    session.startSession(seq);
    setTab('session');
    setRating(0);
    setNotes('');
  };

  const handleSaveSession = async () => {
    const data = session.getSessionData();
    if (!data) return;
    setSaving(true);
    try {
      await stretchSessionsApi.create({
        ...data,
        rating: rating || undefined,
        notes: notes.trim() || undefined,
      });
      session.reset();
      setTab('history');
      loadSessions();
    } catch (e) {
      console.error('Failed to save session', e);
    } finally {
      setSaving(false);
    }
  };

  const totalSeqTime = (seq: StretchSequence) =>
    (seq.movements || []).reduce((sum, m) => sum + m.durationSeconds, 0);

  if (loading) {
    return (
      <AppLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-white/60">Carregando...</div>
        </div>
      </AppLayout>
    );
  }

  // ── Fullscreen Session Overlay ──
  if (sessionState.phase === 'ACTIVE' || sessionState.phase === 'REST') {
    const mov = session.currentMovement;
    return (
      <AppLayout>
        <div className="fixed inset-0 z-50 bg-black/95 flex flex-col items-center justify-center text-white">
          {/* Progress */}
          <div className="absolute top-6 left-6 text-white/50 text-sm">
            {session.completedCount + 1} / {session.totalMovements} movimentos
          </div>
          <button
            onClick={() => { session.reset(); setTab('sequences'); }}
            className="absolute top-6 right-6 text-white/40 hover:text-white/80 text-sm"
          >
            Cancelar
          </button>

          {sessionState.phase === 'ACTIVE' && mov && (
            <>
              {/* Circle timer */}
              <div className="w-52 h-52 rounded-full border-4 border-blue-400/60 shadow-[0_0_30px_rgba(59,130,246,0.3)] flex flex-col items-center justify-center mb-8">
                <span className="text-5xl font-bold text-blue-300 font-mono">
                  {formatTime(sessionState.elapsedSeconds)}
                </span>
                <span className="text-sm text-white/40 mt-1">
                  meta: {formatTime(mov.durationSeconds)}
                </span>
              </div>

              <h2 className="text-2xl font-bold mb-2">{mov.name}</h2>
              {mov.description && (
                <p className="text-white/50 text-center max-w-md mb-8">{mov.description}</p>
              )}

              <div className="flex gap-3">
                <button
                  onClick={session.completeMovement}
                  className="px-8 py-4 bg-green-600 hover:bg-green-500 rounded-xl text-lg font-semibold transition-colors"
                >
                  Concluir
                </button>
                <button
                  onClick={session.skipMovement}
                  className="px-6 py-4 bg-white/10 hover:bg-white/20 rounded-xl text-lg transition-colors"
                >
                  Pular
                </button>
                <button
                  onClick={session.addRep}
                  className="px-6 py-4 bg-blue-600/30 hover:bg-blue-600/50 rounded-xl text-lg transition-colors"
                >
                  Repetir
                </button>
              </div>
            </>
          )}

          {sessionState.phase === 'REST' && (
            <>
              <div className="w-52 h-52 rounded-full border-4 border-green-400/60 shadow-[0_0_30px_rgba(34,197,94,0.3)] flex flex-col items-center justify-center mb-8">
                <span className="text-6xl font-bold text-green-300 font-mono">
                  {sessionState.restCountdown}
                </span>
              </div>
              <h2 className="text-xl text-white/60">Descanso</h2>
              <p className="text-white/40 mt-2">
                Proximo: {sessionState.sequence?.movements?.[sessionState.currentMovementIndex + 1]?.name}
              </p>
            </>
          )}
        </div>
      </AppLayout>
    );
  }

  // ── Tab Layout ──
  return (
    <AppLayout>
      <div className="max-w-4xl mx-auto">
        {/* Profile selector */}
        {profiles.length > 1 && (
          <div className="mb-4">
            <select
              value={selectedProfile}
              onChange={e => {
                setSelectedProfile(e.target.value);
                localStorage.setItem(PROFILE_STORAGE_KEY, e.target.value);
              }}
              className="bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-sm"
            >
              {profiles.map(p => (
                <option key={p.id} value={p.id} className="bg-gray-900">{p.name}</option>
              ))}
            </select>
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          {[
            { key: 'sequences' as Tab, label: 'Sequencias' },
            { key: 'session' as Tab, label: 'Sessao' },
            { key: 'history' as Tab, label: 'Historico' },
          ].map(t => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                tab === t.key
                  ? 'bg-violet-500/30 border border-violet-400/50 text-violet-200'
                  : 'bg-white/5 border border-white/10 text-white/60 hover:bg-white/10'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* ── Tab: Sequences ── */}
        {tab === 'sequences' && (
          <div className="space-y-4">
            {/* Editing a sequence */}
            {editingSequence ? (
              <div className="bg-white/5 rounded-xl border border-white/10 p-6">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-xl font-bold text-white">{editingSequence.name}</h2>
                  <button
                    onClick={() => setEditingSequence(null)}
                    className="text-white/40 hover:text-white/80 text-sm"
                  >
                    Voltar
                  </button>
                </div>
                {editingSequence.description && (
                  <p className="text-white/50 text-sm mb-4">{editingSequence.description}</p>
                )}

                {/* Movements list */}
                <div className="space-y-2 mb-4">
                  {(editingSequence.movements || []).map((mov, i) => (
                    <div key={mov.id} className="flex items-center gap-3 bg-white/5 rounded-lg px-4 py-3">
                      <span className="text-white/30 text-sm w-6">{i + 1}.</span>
                      <div className="flex-1">
                        <span className="text-white font-medium">{mov.name}</span>
                        {mov.description && (
                          <span className="text-white/40 text-sm ml-2">{mov.description}</span>
                        )}
                      </div>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => handleUpdateMovementDuration(mov, -5)}
                          className="w-7 h-7 rounded bg-white/10 hover:bg-white/20 text-white/60 text-sm"
                        >
                          -
                        </button>
                        <span className="text-white/70 text-sm w-10 text-center">{mov.durationSeconds}s</span>
                        <button
                          onClick={() => handleUpdateMovementDuration(mov, 5)}
                          className="w-7 h-7 rounded bg-white/10 hover:bg-white/20 text-white/60 text-sm"
                        >
                          +
                        </button>
                      </div>
                      <button
                        onClick={() => handleDeleteMovement(mov.id)}
                        className="text-red-400/60 hover:text-red-400 text-sm ml-2"
                      >
                        Remover
                      </button>
                    </div>
                  ))}
                  {(!editingSequence.movements || editingSequence.movements.length === 0) && (
                    <p className="text-white/30 text-sm text-center py-4">Nenhum movimento. Adicione abaixo.</p>
                  )}
                </div>

                {/* Add movement form */}
                <div className="border-t border-white/10 pt-4">
                  <h3 className="text-white/70 text-sm font-medium mb-2">Novo Movimento</h3>
                  <div className="flex gap-2">
                    <input
                      value={newMovName}
                      onChange={e => setNewMovName(e.target.value)}
                      placeholder="Nome do movimento"
                      className="flex-1 bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-sm placeholder:text-white/30"
                    />
                    <input
                      value={newMovDuration}
                      onChange={e => setNewMovDuration(Number(e.target.value) || 30)}
                      type="number"
                      min={5}
                      className="w-20 bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-sm text-center"
                    />
                    <span className="text-white/40 self-center text-sm">seg</span>
                    <button
                      onClick={handleAddMovement}
                      disabled={!newMovName.trim()}
                      className="px-4 py-2 bg-violet-600 hover:bg-violet-500 disabled:opacity-40 rounded-lg text-white text-sm font-medium transition-colors"
                    >
                      Adicionar
                    </button>
                  </div>
                </div>

                {/* Start session button */}
                {editingSequence.movements && editingSequence.movements.length > 0 && (
                  <button
                    onClick={() => handleStartSession(editingSequence)}
                    className="mt-4 w-full py-3 bg-green-600 hover:bg-green-500 rounded-xl text-white font-semibold transition-colors"
                  >
                    Iniciar Sessao ({editingSequence.movements.length} movimentos, {formatTime(totalSeqTime(editingSequence))})
                  </button>
                )}
              </div>
            ) : (
              <>
                {/* Sequence list */}
                {sequences.map(seq => (
                  <div
                    key={seq.id}
                    className="bg-white/5 rounded-xl border border-white/10 p-4 hover:bg-white/8 transition-colors cursor-pointer"
                    onClick={() => setEditingSequence(seq)}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <h3 className="text-white font-semibold">{seq.name}</h3>
                        <p className="text-white/40 text-sm">
                          {seq.movements?.length || 0} movimentos &middot; {formatTime(totalSeqTime(seq))}
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <button
                          onClick={e => { e.stopPropagation(); handleStartSession(seq); }}
                          disabled={!seq.movements || seq.movements.length === 0}
                          className="px-4 py-2 bg-green-600/80 hover:bg-green-500 disabled:opacity-30 rounded-lg text-white text-sm font-medium transition-colors"
                        >
                          Iniciar
                        </button>
                        <button
                          onClick={e => { e.stopPropagation(); handleDeleteSequence(seq.id); }}
                          className="px-3 py-2 bg-red-600/20 hover:bg-red-600/40 rounded-lg text-red-300 text-sm transition-colors"
                        >
                          Excluir
                        </button>
                      </div>
                    </div>
                  </div>
                ))}

                {sequences.length === 0 && !showNewSeqForm && (
                  <p className="text-white/30 text-center py-8">Nenhuma sequencia criada ainda.</p>
                )}

                {/* New sequence form */}
                {showNewSeqForm ? (
                  <div className="bg-white/5 rounded-xl border border-white/10 p-4 space-y-3">
                    <input
                      value={newSeqName}
                      onChange={e => setNewSeqName(e.target.value)}
                      placeholder="Nome da sequencia"
                      className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white placeholder:text-white/30"
                      autoFocus
                    />
                    <input
                      value={newSeqDesc}
                      onChange={e => setNewSeqDesc(e.target.value)}
                      placeholder="Descricao (opcional)"
                      className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white placeholder:text-white/30"
                    />
                    <div className="flex gap-2">
                      <button
                        onClick={handleCreateSequence}
                        disabled={!newSeqName.trim()}
                        className="px-4 py-2 bg-violet-600 hover:bg-violet-500 disabled:opacity-40 rounded-lg text-white text-sm font-medium transition-colors"
                      >
                        Criar
                      </button>
                      <button
                        onClick={() => { setShowNewSeqForm(false); setNewSeqName(''); setNewSeqDesc(''); }}
                        className="px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg text-white/60 text-sm transition-colors"
                      >
                        Cancelar
                      </button>
                    </div>
                  </div>
                ) : (
                  <button
                    onClick={() => setShowNewSeqForm(true)}
                    className="w-full py-3 border-2 border-dashed border-white/20 rounded-xl text-white/40 hover:text-white/70 hover:border-white/40 transition-colors"
                  >
                    + Nova Sequencia
                  </button>
                )}
              </>
            )}
          </div>
        )}

        {/* ── Tab: Session ── */}
        {tab === 'session' && (
          <div>
            {sessionState.phase === 'IDLE' && (
              <div className="text-center py-12">
                <div className="text-6xl mb-4">🤸</div>
                <h2 className="text-2xl font-bold text-white mb-2">Sessao de Alongamento</h2>
                <p className="text-white/50 mb-8">Selecione uma sequencia para iniciar</p>
                <div className="space-y-3 max-w-md mx-auto">
                  {sequences.filter(s => s.movements && s.movements.length > 0).map(seq => (
                    <button
                      key={seq.id}
                      onClick={() => handleStartSession(seq)}
                      className="w-full py-3 px-4 bg-white/5 hover:bg-white/10 border border-white/10 rounded-xl text-left transition-colors"
                    >
                      <span className="text-white font-medium">{seq.name}</span>
                      <span className="text-white/40 text-sm ml-2">
                        {seq.movements?.length} mov &middot; {formatTime(totalSeqTime(seq))}
                      </span>
                    </button>
                  ))}
                  {sequences.filter(s => s.movements && s.movements.length > 0).length === 0 && (
                    <p className="text-white/30 text-sm">
                      Crie uma sequencia com movimentos na aba &quot;Sequencias&quot; primeiro.
                    </p>
                  )}
                </div>
              </div>
            )}

            {sessionState.phase === 'COMPLETE' && (
              <div className="max-w-md mx-auto text-center py-8">
                <div className="text-6xl mb-4">🎉</div>
                <h2 className="text-2xl font-bold text-white mb-2">Sessao Completa!</h2>
                <p className="text-white/50 mb-6">
                  {sessionState.sequence?.name} &middot; {formatTime(sessionState.totalElapsedSeconds)}
                </p>

                {/* Stats */}
                <div className="grid grid-cols-3 gap-3 mb-6">
                  <div className="bg-white/5 rounded-lg p-3">
                    <div className="text-2xl font-bold text-green-400">{session.completedCount - session.skippedCount}</div>
                    <div className="text-white/40 text-xs">Completos</div>
                  </div>
                  <div className="bg-white/5 rounded-lg p-3">
                    <div className="text-2xl font-bold text-yellow-400">{session.skippedCount}</div>
                    <div className="text-white/40 text-xs">Pulados</div>
                  </div>
                  <div className="bg-white/5 rounded-lg p-3">
                    <div className="text-2xl font-bold text-blue-400">{formatTime(sessionState.totalElapsedSeconds)}</div>
                    <div className="text-white/40 text-xs">Duracao</div>
                  </div>
                </div>

                {/* Movement results */}
                <div className="text-left bg-white/5 rounded-xl p-4 mb-6">
                  {sessionState.movementResults.map((r, i) => (
                    <div key={i} className="flex items-center justify-between py-2 border-b border-white/5 last:border-0">
                      <span className={`text-sm ${r.skipped ? 'text-white/30 line-through' : 'text-white/80'}`}>
                        {r.movementName}
                      </span>
                      <span className="text-white/40 text-sm">
                        {r.skipped ? 'pulado' : `${r.actualDuration}s / ${r.plannedDuration}s`}
                      </span>
                    </div>
                  ))}
                </div>

                {/* Rating */}
                <div className="flex justify-center gap-2 mb-4">
                  {[1, 2, 3, 4, 5].map(star => (
                    <button
                      key={star}
                      onClick={() => setRating(star)}
                      className={`text-2xl transition-colors ${star <= rating ? 'text-yellow-400' : 'text-white/20'}`}
                    >
                      ★
                    </button>
                  ))}
                </div>

                {/* Notes */}
                <textarea
                  value={notes}
                  onChange={e => setNotes(e.target.value)}
                  placeholder="Notas (opcional)"
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-sm placeholder:text-white/30 mb-4 h-20 resize-none"
                />

                <div className="flex gap-3">
                  <button
                    onClick={handleSaveSession}
                    disabled={saving}
                    className="flex-1 py-3 bg-violet-600 hover:bg-violet-500 disabled:opacity-50 rounded-xl text-white font-semibold transition-colors"
                  >
                    {saving ? 'Salvando...' : 'Salvar'}
                  </button>
                  <button
                    onClick={() => { session.reset(); setTab('sequences'); }}
                    className="px-6 py-3 bg-white/10 hover:bg-white/20 rounded-xl text-white/60 transition-colors"
                  >
                    Descartar
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Tab: History ── */}
        {tab === 'history' && (
          <div className="space-y-3">
            {sessions.length === 0 && (
              <p className="text-white/30 text-center py-8">Nenhuma sessao registrada ainda.</p>
            )}
            {sessions.map(s => (
              <HistoryItem key={s.id} session={s} />
            ))}
          </div>
        )}
      </div>
    </AppLayout>
  );
}

function HistoryItem({ session: s }: { session: StretchSession }) {
  const [expanded, setExpanded] = useState(false);
  const [details, setDetails] = useState<StretchSession | null>(null);

  const handleExpand = async () => {
    if (expanded) {
      setExpanded(false);
      return;
    }
    if (!details) {
      try {
        const data = await stretchSessionsApi.get(s.id);
        setDetails(data);
      } catch (e) {
        console.error('Failed to load session details', e);
      }
    }
    setExpanded(true);
  };

  return (
    <div className="bg-white/5 rounded-xl border border-white/10 overflow-hidden">
      <button
        onClick={handleExpand}
        className="w-full p-4 text-left hover:bg-white/5 transition-colors"
      >
        <div className="flex items-center justify-between">
          <div>
            <span className="text-white font-medium">{s.sequenceName || 'Sessao'}</span>
            <span className="text-white/40 text-sm ml-2">{s.sessionDate}</span>
          </div>
          <div className="flex items-center gap-3">
            {s.durationSeconds != null && (
              <span className="text-white/50 text-sm">{formatTime(s.durationSeconds)}</span>
            )}
            {s.rating && (
              <span className="text-yellow-400 text-sm">{'★'.repeat(s.rating)}</span>
            )}
            <span className="text-white/30">{expanded ? '▲' : '▼'}</span>
          </div>
        </div>
      </button>
      {expanded && details && (
        <div className="px-4 pb-4 border-t border-white/5">
          {details.notes && (
            <p className="text-white/40 text-sm mt-2 mb-2">{details.notes}</p>
          )}
          {(details.movements || []).map((m, i) => (
            <div key={i} className="flex items-center justify-between py-1.5 text-sm">
              <span className={m.skipped ? 'text-white/30 line-through' : 'text-white/70'}>
                {m.movementName}
              </span>
              <span className="text-white/40">
                {m.skipped ? 'pulado' : `${m.actualDurationSeconds || 0}s / ${m.plannedDurationSeconds}s`}
                {m.reps > 1 && ` x${m.reps}`}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

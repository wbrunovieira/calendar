'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import { profilesApi, exerciseTypesApi, bodyMeasurementsApi } from '@/lib/api';
import type { Profile, ExerciseType, BodyMeasurement } from '@/types/health';

interface Calendar {
  id: string;
  name: string;
  email: string | null;
}

export default function ConfiguracoesPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [exerciseTypes, setExerciseTypes] = useState<ExerciseType[]>([]);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [loading, setLoading] = useState(true);

  const [showProfileForm, setShowProfileForm] = useState(false);
  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [profileForm, setProfileForm] = useState({
    calendarId: '',
    name: '',
    birthDate: '',
    height: '',
    currentWeight: '',
    targetWeight: '',
  });

  const [measurements, setMeasurements] = useState<BodyMeasurement[]>([]);
  const [showMeasurementForm, setShowMeasurementForm] = useState(false);
  const [measurementForm, setMeasurementForm] = useState({
    measuredAt: new Date().toISOString().split('T')[0],
    weight: '',
    bodyFatPct: '',
    notes: '',
  });

  const [showTypeForm, setShowTypeForm] = useState(false);
  const [typeForm, setTypeForm] = useState({ name: '', description: '', icon: '' });

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      const [profilesData, typesData] = await Promise.all([
        profilesApi.list(),
        exerciseTypesApi.list(),
      ]);
      setProfiles(profilesData);
      setExerciseTypes(typesData);

      // Load measurements for first profile
      if (profilesData.length > 0) {
        try {
          const m = await bodyMeasurementsApi.list(profilesData[0].id);
          setMeasurements(m);
        } catch (e) {
          console.warn('Could not load measurements:', e);
        }
      }

      // Load calendars from calendar-core
      try {
        const calendarApiUrl = process.env.NEXT_PUBLIC_CALENDAR_API_URL || 'http://localhost:3334';
        const calendarApiToken = process.env.NEXT_PUBLIC_CALENDAR_API_TOKEN;
        const response = await fetch(`${calendarApiUrl}/calendars`, {
          headers: calendarApiToken ? { Authorization: `Bearer ${calendarApiToken}` } : {},
        });
        const data = await response.json();
        setCalendars(data.data || []);
      } catch (e) {
        console.warn('Could not load calendars:', e);
      }
    } catch (error) {
      console.error('Error loading data:', error);
    } finally {
      setLoading(false);
    }
  }

  async function handleProfileSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      const data = {
        calendarId: profileForm.calendarId,
        name: profileForm.name,
        birthDate: profileForm.birthDate || undefined,
        height: profileForm.height ? parseFloat(profileForm.height) : undefined,
        currentWeight: profileForm.currentWeight ? parseFloat(profileForm.currentWeight) : undefined,
        targetWeight: profileForm.targetWeight ? parseFloat(profileForm.targetWeight) : undefined,
      };

      if (editingProfile) {
        await profilesApi.update(editingProfile.id, data);
      } else {
        await profilesApi.create(data);
      }
      await loadData();
      resetProfileForm();
    } catch (error) {
      console.error('Error saving profile:', error);
      alert('Erro ao salvar perfil');
    }
  }

  async function handleDeleteProfile(id: string) {
    if (!confirm('Excluir este perfil?')) return;
    try {
      await profilesApi.delete(id);
      await loadData();
    } catch (error) {
      console.error('Error deleting profile:', error);
      alert('Erro ao excluir perfil');
    }
  }

  function resetProfileForm() {
    setShowProfileForm(false);
    setEditingProfile(null);
    setProfileForm({
      calendarId: '',
      name: '',
      birthDate: '',
      height: '',
      currentWeight: '',
      targetWeight: '',
    });
  }

  function startEditProfile(profile: Profile) {
    setEditingProfile(profile);
    setProfileForm({
      calendarId: profile.calendarId,
      name: profile.name,
      birthDate: profile.birthDate || '',
      height: profile.height?.toString() || '',
      currentWeight: profile.currentWeight?.toString() || '',
      targetWeight: profile.targetWeight?.toString() || '',
    });
    setShowProfileForm(true);
  }

  async function handleTypeSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await exerciseTypesApi.create(typeForm);
      await loadData();
      setShowTypeForm(false);
      setTypeForm({ name: '', description: '', icon: '' });
    } catch (error) {
      console.error('Error saving exercise type:', error);
      alert('Erro ao salvar tipo');
    }
  }

  async function handleDeleteType(id: string) {
    if (!confirm('Excluir este tipo de exercicio?')) return;
    try {
      await exerciseTypesApi.delete(id);
      await loadData();
    } catch (error) {
      console.error('Error deleting type:', error);
      alert('Erro ao excluir tipo');
    }
  }

  async function handleMeasurementSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (profiles.length === 0) return;
    try {
      await bodyMeasurementsApi.create({
        profileId: profiles[0].id,
        measuredAt: measurementForm.measuredAt,
        weight: measurementForm.weight ? parseFloat(measurementForm.weight) : undefined,
        bodyFatPct: measurementForm.bodyFatPct ? parseFloat(measurementForm.bodyFatPct) : undefined,
        notes: measurementForm.notes || undefined,
      });
      const m = await bodyMeasurementsApi.list(profiles[0].id);
      setMeasurements(m);
      setShowMeasurementForm(false);
      setMeasurementForm({
        measuredAt: new Date().toISOString().split('T')[0],
        weight: '',
        bodyFatPct: '',
        notes: '',
      });
    } catch (error) {
      console.error('Error saving measurement:', error);
      alert('Erro ao salvar medicao');
    }
  }

  async function handleDeleteMeasurement(id: string) {
    if (!confirm('Excluir esta medicao?')) return;
    try {
      await bodyMeasurementsApi.delete(id);
      if (profiles.length > 0) {
        const m = await bodyMeasurementsApi.list(profiles[0].id);
        setMeasurements(m);
      }
    } catch (error) {
      console.error('Error deleting measurement:', error);
      alert('Erro ao excluir medicao');
    }
  }

  if (loading) {
    return (
      <AppLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-white/60">Carregando...</div>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <div className="py-6 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-white">Configuracoes</h1>
            <p className="text-white/60 text-sm">Gerencie perfis e tipos de exercicio</p>
          </div>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">
            ← Voltar
          </Link>
        </div>

        {/* Perfis */}
        <div className="bg-white/5 border border-white/10 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-xl font-bold text-white">Perfis</h2>
              <p className="text-white/60 text-sm">Vincule ao calendario para rastreamento</p>
            </div>
            <button
              onClick={() => setShowProfileForm(true)}
              className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
            >
              + Novo Perfil
            </button>
          </div>

          {showProfileForm && (
            <form onSubmit={handleProfileSubmit} className="bg-white/5 rounded-lg p-4 mb-4 space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/60 text-sm mb-1">Nome *</label>
                  <input
                    type="text"
                    value={profileForm.name}
                    onChange={(e) => setProfileForm({ ...profileForm, name: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="Seu nome"
                    required
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Calendario *</label>
                  <select
                    value={profileForm.calendarId}
                    onChange={(e) => setProfileForm({ ...profileForm, calendarId: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    required
                  >
                    <option value="" className="bg-gray-800">Selecione...</option>
                    {calendars.map((cal) => (
                      <option key={cal.id} value={cal.id} className="bg-gray-800">
                        {cal.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Data de nascimento</label>
                  <input
                    type="date"
                    value={profileForm.birthDate}
                    onChange={(e) => setProfileForm({ ...profileForm, birthDate: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Altura (cm)</label>
                  <input
                    type="number"
                    value={profileForm.height}
                    onChange={(e) => setProfileForm({ ...profileForm, height: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="175"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Peso atual (kg)</label>
                  <input
                    type="number"
                    step="0.1"
                    value={profileForm.currentWeight}
                    onChange={(e) => setProfileForm({ ...profileForm, currentWeight: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="80"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Peso meta (kg)</label>
                  <input
                    type="number"
                    step="0.1"
                    value={profileForm.targetWeight}
                    onChange={(e) => setProfileForm({ ...profileForm, targetWeight: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="75"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
                >
                  {editingProfile ? 'Salvar' : 'Criar'}
                </button>
                <button
                  type="button"
                  onClick={resetProfileForm}
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-sm"
                >
                  Cancelar
                </button>
              </div>
            </form>
          )}

          {profiles.length === 0 ? (
            <div className="text-center py-8 text-white/60">
              Nenhum perfil cadastrado. Crie um para comecar a registrar treinos.
            </div>
          ) : (
            <div className="space-y-3">
              {profiles.map((profile) => {
                const calendar = calendars.find((c) => c.id === profile.calendarId);
                return (
                  <div
                    key={profile.id}
                    className="bg-white/5 border border-white/10 rounded-lg p-4 flex items-center justify-between"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-12 h-12 rounded-xl bg-violet-600/30 flex items-center justify-center text-xl">
                        💪
                      </div>
                      <div>
                        <p className="text-white font-medium">{profile.name}</p>
                        <p className="text-white/60 text-sm">
                          {calendar?.name || 'Sem calendario'}
                          {profile.currentWeight && ` • ${profile.currentWeight}kg`}
                          {profile.height && ` • ${profile.height}cm`}
                        </p>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => startEditProfile(profile)}
                        className="p-2 bg-white/10 hover:bg-white/20 rounded text-white/60"
                      >
                        ✏️
                      </button>
                      <button
                        onClick={() => handleDeleteProfile(profile.id)}
                        className="p-2 bg-red-500/20 hover:bg-red-500/30 rounded text-red-300"
                      >
                        🗑️
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Historico de Peso */}
        {profiles.length > 0 && (
          <div className="bg-white/5 border border-white/10 rounded-xl p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-xl font-bold text-white">Historico de Peso</h2>
                <p className="text-white/60 text-sm">Acompanhe sua evolucao</p>
              </div>
              <button
                onClick={() => setShowMeasurementForm(true)}
                className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
              >
                + Registrar Peso
              </button>
            </div>

            {showMeasurementForm && (
              <form onSubmit={handleMeasurementSubmit} className="bg-white/5 rounded-lg p-4 mb-4 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                  <div>
                    <label className="block text-white/60 text-sm mb-1">Data *</label>
                    <input
                      type="date"
                      value={measurementForm.measuredAt}
                      onChange={(e) => setMeasurementForm({ ...measurementForm, measuredAt: e.target.value })}
                      className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-white/60 text-sm mb-1">Peso (kg)</label>
                    <input
                      type="number"
                      step="0.1"
                      value={measurementForm.weight}
                      onChange={(e) => setMeasurementForm({ ...measurementForm, weight: e.target.value })}
                      className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                      placeholder="82.5"
                    />
                  </div>
                  <div>
                    <label className="block text-white/60 text-sm mb-1">Gordura (%)</label>
                    <input
                      type="number"
                      step="0.1"
                      value={measurementForm.bodyFatPct}
                      onChange={(e) => setMeasurementForm({ ...measurementForm, bodyFatPct: e.target.value })}
                      className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                      placeholder="15.0"
                    />
                  </div>
                  <div>
                    <label className="block text-white/60 text-sm mb-1">Notas</label>
                    <input
                      type="text"
                      value={measurementForm.notes}
                      onChange={(e) => setMeasurementForm({ ...measurementForm, notes: e.target.value })}
                      className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                      placeholder="Manha, em jejum"
                    />
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    type="submit"
                    className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
                  >
                    Registrar
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowMeasurementForm(false)}
                    className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-sm"
                  >
                    Cancelar
                  </button>
                </div>
              </form>
            )}

            {measurements.length === 0 ? (
              <div className="text-center py-8 text-white/60">
                Nenhuma medicao registrada. Registre seu peso para acompanhar a evolucao.
              </div>
            ) : (
              <div className="space-y-2">
                {measurements.map((m, i) => {
                  const prev = measurements[i + 1];
                  const diff = m.weight && prev?.weight ? m.weight - prev.weight : null;
                  return (
                    <div
                      key={m.id}
                      className="bg-white/5 border border-white/10 rounded-lg px-4 py-3 flex items-center justify-between"
                    >
                      <div className="flex items-center gap-4">
                        <span className="text-white/50 text-sm w-24">{m.measuredAt}</span>
                        <span className="text-white font-medium">
                          {m.weight ? `${m.weight} kg` : '-'}
                        </span>
                        {diff !== null && (
                          <span className={`text-sm ${diff < 0 ? 'text-green-400' : diff > 0 ? 'text-red-400' : 'text-white/40'}`}>
                            {diff > 0 ? '+' : ''}{diff.toFixed(1)}
                          </span>
                        )}
                        {m.bodyFatPct && (
                          <span className="text-white/50 text-sm">{m.bodyFatPct}% gordura</span>
                        )}
                        {m.notes && (
                          <span className="text-white/40 text-sm">{m.notes}</span>
                        )}
                      </div>
                      <button
                        onClick={() => handleDeleteMeasurement(m.id)}
                        className="p-1 hover:bg-red-500/20 rounded text-red-300 text-xs"
                      >
                        ✕
                      </button>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* Tipos de Exercicio */}
        <div className="bg-white/5 border border-white/10 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-xl font-bold text-white">Tipos de Exercicio</h2>
              <p className="text-white/60 text-sm">Categorize seus exercicios</p>
            </div>
            <button
              onClick={() => setShowTypeForm(true)}
              className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
            >
              + Novo Tipo
            </button>
          </div>

          {showTypeForm && (
            <form onSubmit={handleTypeSubmit} className="bg-white/5 rounded-lg p-4 mb-4 space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label className="block text-white/60 text-sm mb-1">Nome *</label>
                  <input
                    type="text"
                    value={typeForm.name}
                    onChange={(e) => setTypeForm({ ...typeForm, name: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="Ex: Musculacao"
                    required
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Descricao</label>
                  <input
                    type="text"
                    value={typeForm.description}
                    onChange={(e) => setTypeForm({ ...typeForm, description: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="Treino de forca"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Icone (emoji)</label>
                  <input
                    type="text"
                    value={typeForm.icon}
                    onChange={(e) => setTypeForm({ ...typeForm, icon: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="🏋️"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
                >
                  Criar
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowTypeForm(false);
                    setTypeForm({ name: '', description: '', icon: '' });
                  }}
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-sm"
                >
                  Cancelar
                </button>
              </div>
            </form>
          )}

          {exerciseTypes.length === 0 ? (
            <div className="text-center py-8 text-white/60">
              Nenhum tipo cadastrado. Adicione tipos como Musculacao, Cardio, etc.
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {exerciseTypes.map((type) => (
                <div
                  key={type.id}
                  className="flex items-center gap-2 bg-white/5 border border-white/10 rounded-lg px-3 py-2"
                >
                  <span className="text-lg">{type.icon || '🏋️'}</span>
                  <span className="text-white">{type.name}</span>
                  <button
                    onClick={() => handleDeleteType(type.id)}
                    className="p-1 hover:bg-red-500/20 rounded text-red-300 text-xs"
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </AppLayout>
  );
}

'use client';

import { useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { profilesApi, workoutsApi } from '@/lib/api';
import { Profile, Workout, WorkoutFormData } from '@/types/health';

export default function WorkoutsPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState<string>('');
  const [workouts, setWorkouts] = useState<Workout[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editingWorkout, setEditingWorkout] = useState<Workout | null>(null);
  const [formData, setFormData] = useState<WorkoutFormData>({
    profileId: '',
    name: '',
    workoutDate: new Date().toISOString().split('T')[0],
  });

  useEffect(() => {
    loadProfiles();
  }, []);

  useEffect(() => {
    if (selectedProfile) {
      loadWorkouts();
      setFormData(prev => ({ ...prev, profileId: selectedProfile }));
    }
  }, [selectedProfile]);

  async function loadProfiles() {
    try {
      const data = await profilesApi.list();
      setProfiles(data);
      if (data.length > 0) {
        setSelectedProfile(data[0].id);
      }
    } catch (error) {
      console.error('Error loading profiles:', error);
    } finally {
      setLoading(false);
    }
  }

  async function loadWorkouts() {
    try {
      const data = await workoutsApi.list({ profileId: selectedProfile });
      setWorkouts(data);
    } catch (error) {
      console.error('Error loading workouts:', error);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (editingWorkout) {
        await workoutsApi.update(editingWorkout.id, formData);
      } else {
        await workoutsApi.create(formData);
      }
      await loadWorkouts();
      resetForm();
    } catch (error) {
      console.error('Error saving workout:', error);
      alert('Erro ao salvar treino');
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Excluir este treino?')) return;
    try {
      await workoutsApi.delete(id);
      await loadWorkouts();
    } catch (error) {
      console.error('Error deleting workout:', error);
      alert('Erro ao excluir treino');
    }
  }

  async function handleStatusChange(workout: Workout, status: string) {
    try {
      await workoutsApi.update(workout.id, {
        name: workout.name,
        workoutDate: workout.workoutDate,
        status,
        endTime: status === 'COMPLETED' ? new Date().toISOString().split('T')[1].substring(0, 5) : undefined,
      });
      await loadWorkouts();
    } catch (error) {
      console.error('Error updating status:', error);
    }
  }

  function resetForm() {
    setShowForm(false);
    setEditingWorkout(null);
    setFormData({
      profileId: selectedProfile,
      name: '',
      workoutDate: new Date().toISOString().split('T')[0],
    });
  }

  function startEdit(workout: Workout) {
    setEditingWorkout(workout);
    setFormData({
      profileId: workout.profileId,
      name: workout.name,
      description: workout.description,
      workoutDate: workout.workoutDate,
      startTime: workout.startTime,
      notes: workout.notes,
    });
    setShowForm(true);
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'COMPLETED': return '✅';
      case 'IN_PROGRESS': return '🏃';
      case 'SKIPPED': return '⏭️';
      default: return '📅';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'COMPLETED': return 'Concluido';
      case 'IN_PROGRESS': return 'Em andamento';
      case 'SKIPPED': return 'Pulado';
      default: return 'Planejado';
    }
  };

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
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-white">Treinos</h2>
            <p className="text-white/60 text-sm">Registre seus treinos de academia</p>
          </div>
          <div className="flex items-center gap-3">
            {profiles.length > 0 && (
              <select
                value={selectedProfile}
                onChange={(e) => setSelectedProfile(e.target.value)}
                className="bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-sm"
              >
                {profiles.map((profile) => (
                  <option key={profile.id} value={profile.id} className="bg-gray-800">
                    {profile.name}
                  </option>
                ))}
              </select>
            )}
            <button
              onClick={() => setShowForm(true)}
              className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
            >
              + Novo Treino
            </button>
          </div>
        </div>

        {/* Form */}
        {showForm && (
          <div className="bg-white/5 border border-white/10 rounded-xl p-6">
            <h3 className="text-lg font-semibold text-white mb-4">
              {editingWorkout ? 'Editar Treino' : 'Novo Treino'}
            </h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/60 text-sm mb-1">Nome *</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="Ex: Treino A - Peito e Triceps"
                    required
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Data *</label>
                  <input
                    type="date"
                    value={formData.workoutDate}
                    onChange={(e) => setFormData({ ...formData, workoutDate: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    required
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Horario Inicio</label>
                  <input
                    type="time"
                    value={formData.startTime || ''}
                    onChange={(e) => setFormData({ ...formData, startTime: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Descricao</label>
                  <input
                    type="text"
                    value={formData.description || ''}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="Descricao opcional"
                  />
                </div>
              </div>
              <div>
                <label className="block text-white/60 text-sm mb-1">Notas</label>
                <textarea
                  value={formData.notes || ''}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  rows={2}
                  placeholder="Anotacoes sobre o treino"
                />
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
                >
                  {editingWorkout ? 'Salvar' : 'Criar'}
                </button>
                <button
                  type="button"
                  onClick={resetForm}
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-sm"
                >
                  Cancelar
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Workouts List */}
        <div className="space-y-3">
          {workouts.length === 0 ? (
            <div className="bg-white/5 border border-white/10 rounded-xl p-8 text-center">
              <p className="text-white/60">Nenhum treino registrado</p>
            </div>
          ) : (
            workouts.map((workout) => (
              <div
                key={workout.id}
                className="bg-white/5 border border-white/10 rounded-xl p-4"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">{getStatusIcon(workout.status)}</span>
                    <div>
                      <p className="text-white font-medium">{workout.name}</p>
                      <p className="text-white/60 text-sm">
                        {workout.workoutDate}
                        {workout.startTime && ` - ${workout.startTime}`}
                        {workout.durationMinutes && ` - ${workout.durationMinutes} min`}
                      </p>
                      {workout.description && (
                        <p className="text-white/40 text-xs mt-1">{workout.description}</p>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`px-2 py-1 rounded text-xs ${
                      workout.status === 'COMPLETED' ? 'bg-green-500/20 text-green-300' :
                      workout.status === 'IN_PROGRESS' ? 'bg-yellow-500/20 text-yellow-300' :
                      workout.status === 'SKIPPED' ? 'bg-gray-500/20 text-gray-300' :
                      'bg-blue-500/20 text-blue-300'
                    }`}>
                      {getStatusLabel(workout.status)}
                    </span>
                    {workout.status === 'PLANNED' && (
                      <button
                        onClick={() => handleStatusChange(workout, 'IN_PROGRESS')}
                        className="p-1.5 bg-yellow-500/20 hover:bg-yellow-500/30 rounded text-yellow-300"
                        title="Iniciar"
                      >
                        🏃
                      </button>
                    )}
                    {workout.status === 'IN_PROGRESS' && (
                      <button
                        onClick={() => handleStatusChange(workout, 'COMPLETED')}
                        className="p-1.5 bg-green-500/20 hover:bg-green-500/30 rounded text-green-300"
                        title="Concluir"
                      >
                        ✅
                      </button>
                    )}
                    <button
                      onClick={() => startEdit(workout)}
                      className="p-1.5 bg-white/10 hover:bg-white/20 rounded text-white/60"
                      title="Editar"
                    >
                      ✏️
                    </button>
                    <button
                      onClick={() => handleDelete(workout.id)}
                      className="p-1.5 bg-red-500/20 hover:bg-red-500/30 rounded text-red-300"
                      title="Excluir"
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                {workout.notes && (
                  <p className="text-white/50 text-sm mt-2 pl-10">{workout.notes}</p>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </AppLayout>
  );
}

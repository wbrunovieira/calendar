'use client';

import { useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { profilesApi, activitiesApi } from '@/lib/api';
import { Profile, Activity, ActivityFormData, ACTIVITY_TYPES } from '@/types/health';

export default function ActivitiesPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState<string>('');
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editingActivity, setEditingActivity] = useState<Activity | null>(null);
  const [formData, setFormData] = useState<ActivityFormData>({
    profileId: '',
    name: '',
    activityType: 'WIM_HOF',
    activityDate: new Date().toISOString().split('T')[0],
  });

  useEffect(() => {
    loadProfiles();
  }, []);

  useEffect(() => {
    if (selectedProfile) {
      loadActivities();
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

  async function loadActivities() {
    try {
      const data = await activitiesApi.list({ profileId: selectedProfile });
      setActivities(data);
    } catch (error) {
      console.error('Error loading activities:', error);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (editingActivity) {
        await activitiesApi.update(editingActivity.id, formData);
      } else {
        await activitiesApi.create(formData);
      }
      await loadActivities();
      resetForm();
    } catch (error) {
      console.error('Error saving activity:', error);
      alert('Erro ao salvar atividade');
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Excluir esta atividade?')) return;
    try {
      await activitiesApi.delete(id);
      await loadActivities();
    } catch (error) {
      console.error('Error deleting activity:', error);
      alert('Erro ao excluir atividade');
    }
  }

  function resetForm() {
    setShowForm(false);
    setEditingActivity(null);
    setFormData({
      profileId: selectedProfile,
      name: '',
      activityType: 'WIM_HOF',
      activityDate: new Date().toISOString().split('T')[0],
    });
  }

  function startEdit(activity: Activity) {
    setEditingActivity(activity);
    setFormData({
      profileId: activity.profileId,
      name: activity.name,
      activityType: activity.activityType,
      activityDate: activity.activityDate,
      startTime: activity.startTime,
      durationMinutes: activity.durationMinutes,
      rounds: activity.rounds,
      notes: activity.notes,
      rating: activity.rating,
    });
    setShowForm(true);
  }

  const getActivityTypeInfo = (type: string) => {
    return ACTIVITY_TYPES.find(t => t.value === type) || { label: type, icon: '💪' };
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
            <h2 className="text-xl font-semibold text-white">Atividades</h2>
            <p className="text-white/60 text-sm">Wim Hof, HIIT, Meditacao e mais</p>
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
              + Nova Atividade
            </button>
          </div>
        </div>

        {/* Form */}
        {showForm && (
          <div className="bg-white/5 border border-white/10 rounded-xl p-6">
            <h3 className="text-lg font-semibold text-white mb-4">
              {editingActivity ? 'Editar Atividade' : 'Nova Atividade'}
            </h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/60 text-sm mb-1">Tipo *</label>
                  <select
                    value={formData.activityType}
                    onChange={(e) => setFormData({ ...formData, activityType: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  >
                    {ACTIVITY_TYPES.map((type) => (
                      <option key={type.value} value={type.value} className="bg-gray-800">
                        {type.icon} {type.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Nome *</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="Ex: Wim Hof Manha"
                    required
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Data *</label>
                  <input
                    type="date"
                    value={formData.activityDate}
                    onChange={(e) => setFormData({ ...formData, activityDate: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    required
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Horario</label>
                  <input
                    type="time"
                    value={formData.startTime || ''}
                    onChange={(e) => setFormData({ ...formData, startTime: e.target.value })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Duracao (min)</label>
                  <input
                    type="number"
                    value={formData.durationMinutes || ''}
                    onChange={(e) => setFormData({ ...formData, durationMinutes: parseInt(e.target.value) || undefined })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="30"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Rounds</label>
                  <input
                    type="number"
                    value={formData.rounds || ''}
                    onChange={(e) => setFormData({ ...formData, rounds: parseInt(e.target.value) || undefined })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                    placeholder="3"
                  />
                </div>
                <div>
                  <label className="block text-white/60 text-sm mb-1">Avaliacao</label>
                  <select
                    value={formData.rating || ''}
                    onChange={(e) => setFormData({ ...formData, rating: parseInt(e.target.value) || undefined })}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  >
                    <option value="" className="bg-gray-800">Sem avaliacao</option>
                    <option value="1" className="bg-gray-800">1 - Ruim</option>
                    <option value="2" className="bg-gray-800">2 - Regular</option>
                    <option value="3" className="bg-gray-800">3 - Bom</option>
                    <option value="4" className="bg-gray-800">4 - Muito bom</option>
                    <option value="5" className="bg-gray-800">5 - Excelente</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="block text-white/60 text-sm mb-1">Notas</label>
                <textarea
                  value={formData.notes || ''}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white"
                  rows={2}
                  placeholder="Anotacoes sobre a atividade"
                />
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white rounded-lg text-sm font-medium"
                >
                  {editingActivity ? 'Salvar' : 'Criar'}
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

        {/* Activities List */}
        <div className="space-y-3">
          {activities.length === 0 ? (
            <div className="bg-white/5 border border-white/10 rounded-xl p-8 text-center">
              <p className="text-white/60">Nenhuma atividade registrada</p>
            </div>
          ) : (
            activities.map((activity) => {
              const typeInfo = getActivityTypeInfo(activity.activityType);
              return (
                <div
                  key={activity.id}
                  className="bg-white/5 border border-white/10 rounded-xl p-4"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-2xl">{typeInfo.icon}</span>
                      <div>
                        <p className="text-white font-medium">{activity.name}</p>
                        <p className="text-white/60 text-sm">
                          {typeInfo.label} - {activity.activityDate}
                          {activity.startTime && ` - ${activity.startTime}`}
                        </p>
                        <div className="flex gap-3 text-white/40 text-xs mt-1">
                          {activity.durationMinutes && <span>{activity.durationMinutes} min</span>}
                          {activity.rounds && <span>{activity.rounds} rounds</span>}
                          {activity.rating && <span>{'⭐'.repeat(activity.rating)}</span>}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => startEdit(activity)}
                        className="p-1.5 bg-white/10 hover:bg-white/20 rounded text-white/60"
                        title="Editar"
                      >
                        ✏️
                      </button>
                      <button
                        onClick={() => handleDelete(activity.id)}
                        className="p-1.5 bg-red-500/20 hover:bg-red-500/30 rounded text-red-300"
                        title="Excluir"
                      >
                        🗑️
                      </button>
                    </div>
                  </div>
                  {activity.notes && (
                    <p className="text-white/50 text-sm mt-2 pl-10">{activity.notes}</p>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </AppLayout>
  );
}

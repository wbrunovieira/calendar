'use client';

import { useEffect, useState, useMemo } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { profilesApi, workoutsApi, activitiesApi } from '@/lib/api';
import { Profile, Workout, Activity, ACTIVITY_TYPES } from '@/types/health';

export default function DashboardPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState<string>('');
  const [workouts, setWorkouts] = useState<Workout[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadProfiles();
  }, []);

  useEffect(() => {
    if (selectedProfile) {
      loadData();
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

  async function loadData() {
    try {
      const today = new Date();
      const startOfWeek = new Date(today);
      startOfWeek.setDate(today.getDate() - today.getDay());
      const endOfWeek = new Date(startOfWeek);
      endOfWeek.setDate(startOfWeek.getDate() + 6);

      const [workoutsData, activitiesData] = await Promise.all([
        workoutsApi.list({
          profileId: selectedProfile,
          startDate: startOfWeek.toISOString().split('T')[0],
          endDate: endOfWeek.toISOString().split('T')[0],
        }),
        activitiesApi.list({
          profileId: selectedProfile,
          startDate: startOfWeek.toISOString().split('T')[0],
          endDate: endOfWeek.toISOString().split('T')[0],
        }),
      ]);

      setWorkouts(workoutsData);
      setActivities(activitiesData);
    } catch (error) {
      console.error('Error loading data:', error);
    }
  }

  const weekStats = useMemo(() => {
    const completedWorkouts = workouts.filter(w => w.status === 'COMPLETED').length;
    const totalDuration = workouts
      .filter(w => w.status === 'COMPLETED')
      .reduce((sum, w) => sum + (w.durationMinutes || 0), 0);
    const activitiesCount = activities.length;

    const activityTypeStats = activities.reduce((acc, activity) => {
      const type = activity.activityType;
      if (!acc[type]) acc[type] = 0;
      acc[type]++;
      return acc;
    }, {} as Record<string, number>);

    return {
      completedWorkouts,
      totalDuration,
      activitiesCount,
      activityTypeStats,
    };
  }, [workouts, activities]);

  const getActivityTypeLabel = (type: string) => {
    return ACTIVITY_TYPES.find(t => t.value === type)?.label || type;
  };

  const getActivityTypeIcon = (type: string) => {
    return ACTIVITY_TYPES.find(t => t.value === type)?.icon || '💪';
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
        {/* Profile Selector */}
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-white">Dashboard</h2>
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
        </div>

        {profiles.length === 0 ? (
          <div className="bg-white/5 border border-white/10 rounded-xl p-8 text-center">
            <p className="text-white/60">Nenhum perfil encontrado.</p>
            <a href="/configuracoes" className="text-violet-400 hover:underline mt-2 inline-block">
              Criar perfil
            </a>
          </div>
        ) : (
          <>
            {/* Week Stats */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <div className="flex items-center gap-3">
                  <div className="text-3xl">🏋️</div>
                  <div>
                    <p className="text-white/60 text-sm">Treinos da Semana</p>
                    <p className="text-3xl font-bold text-white">{weekStats.completedWorkouts}</p>
                  </div>
                </div>
              </div>

              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <div className="flex items-center gap-3">
                  <div className="text-3xl">⏱️</div>
                  <div>
                    <p className="text-white/60 text-sm">Tempo Total</p>
                    <p className="text-3xl font-bold text-white">
                      {Math.floor(weekStats.totalDuration / 60)}h {weekStats.totalDuration % 60}m
                    </p>
                  </div>
                </div>
              </div>

              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <div className="flex items-center gap-3">
                  <div className="text-3xl">🧘</div>
                  <div>
                    <p className="text-white/60 text-sm">Atividades</p>
                    <p className="text-3xl font-bold text-white">{weekStats.activitiesCount}</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Activity Types Breakdown */}
            {Object.keys(weekStats.activityTypeStats).length > 0 && (
              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <h3 className="text-lg font-semibold text-white mb-4">Atividades por Tipo</h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  {Object.entries(weekStats.activityTypeStats).map(([type, count]) => (
                    <div key={type} className="bg-white/5 rounded-lg p-3 flex items-center gap-2">
                      <span className="text-2xl">{getActivityTypeIcon(type)}</span>
                      <div>
                        <p className="text-white/60 text-xs">{getActivityTypeLabel(type)}</p>
                        <p className="text-white font-semibold">{count}x</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Recent Workouts */}
            <div className="bg-white/5 border border-white/10 rounded-xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Treinos Recentes</h3>
                <a href="/workouts" className="text-violet-400 text-sm hover:underline">
                  Ver todos
                </a>
              </div>
              {workouts.length === 0 ? (
                <p className="text-white/60 text-center py-8">Nenhum treino esta semana</p>
              ) : (
                <div className="space-y-2">
                  {workouts.slice(0, 5).map((workout) => (
                    <div
                      key={workout.id}
                      className="flex items-center justify-between bg-white/5 rounded-lg p-3"
                    >
                      <div className="flex items-center gap-3">
                        <span className="text-xl">
                          {workout.status === 'COMPLETED' ? '✅' : workout.status === 'IN_PROGRESS' ? '🏃' : '📅'}
                        </span>
                        <div>
                          <p className="text-white font-medium">{workout.name}</p>
                          <p className="text-white/60 text-sm">{workout.workoutDate}</p>
                        </div>
                      </div>
                      {workout.durationMinutes && (
                        <span className="text-white/60 text-sm">{workout.durationMinutes} min</span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Recent Activities */}
            <div className="bg-white/5 border border-white/10 rounded-xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Atividades Recentes</h3>
                <a href="/activities" className="text-violet-400 text-sm hover:underline">
                  Ver todas
                </a>
              </div>
              {activities.length === 0 ? (
                <p className="text-white/60 text-center py-8">Nenhuma atividade esta semana</p>
              ) : (
                <div className="space-y-2">
                  {activities.slice(0, 5).map((activity) => (
                    <div
                      key={activity.id}
                      className="flex items-center justify-between bg-white/5 rounded-lg p-3"
                    >
                      <div className="flex items-center gap-3">
                        <span className="text-xl">{getActivityTypeIcon(activity.activityType)}</span>
                        <div>
                          <p className="text-white font-medium">{activity.name}</p>
                          <p className="text-white/60 text-sm">
                            {getActivityTypeLabel(activity.activityType)} - {activity.activityDate}
                          </p>
                        </div>
                      </div>
                      {activity.durationMinutes && (
                        <span className="text-white/60 text-sm">{activity.durationMinutes} min</span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </AppLayout>
  );
}

import {
  Profile,
  ExerciseType,
  Exercise,
  Workout,
  WorkoutExercise,
  ExerciseSet,
  Activity,
  PersonalRecord,
  BodyMeasurement,
  StretchSequence,
  StretchMovement,
  StretchSession,
  WorkoutFormData,
  ActivityFormData,
  ExerciseFormData,
} from '@/types/health';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3336/api/v1';

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    // Carry the Authelia session cookie — same reason as the finances client.
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || `API Error: ${response.status}`);
  }

  if (response.status === 204) {
    return null as T;
  }

  return response.json();
}

// Profiles
export const profilesApi = {
  list: () => fetchApi<Profile[]>('/profiles'),
  get: (id: string) => fetchApi<Profile>(`/profiles/${id}`),
  create: (data: Partial<Profile>) =>
    fetchApi<Profile>('/profiles', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<Profile>) =>
    fetchApi<Profile>(`/profiles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/profiles/${id}`, {
      method: 'DELETE',
    }),
};

// Exercise Types
export const exerciseTypesApi = {
  list: () => fetchApi<ExerciseType[]>('/exercise-types'),
  get: (id: string) => fetchApi<ExerciseType>(`/exercise-types/${id}`),
  create: (data: Partial<ExerciseType>) =>
    fetchApi<ExerciseType>('/exercise-types', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<ExerciseType>) =>
    fetchApi<ExerciseType>(`/exercise-types/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/exercise-types/${id}`, {
      method: 'DELETE',
    }),
};

// Exercises
export const exercisesApi = {
  list: (params?: { exerciseTypeId?: string; muscleGroup?: string }) => {
    const searchParams = new URLSearchParams();
    if (params?.exerciseTypeId) searchParams.set('exerciseTypeId', params.exerciseTypeId);
    if (params?.muscleGroup) searchParams.set('muscleGroup', params.muscleGroup);
    const query = searchParams.toString();
    return fetchApi<Exercise[]>(`/exercises${query ? `?${query}` : ''}`);
  },
  get: (id: string) => fetchApi<Exercise>(`/exercises/${id}`),
  create: (data: ExerciseFormData) =>
    fetchApi<Exercise>('/exercises', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<ExerciseFormData>) =>
    fetchApi<Exercise>(`/exercises/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/exercises/${id}`, {
      method: 'DELETE',
    }),
};

// Workouts
export const workoutsApi = {
  list: (params?: { profileId?: string; startDate?: string; endDate?: string }) => {
    const searchParams = new URLSearchParams();
    if (params?.profileId) searchParams.set('profileId', params.profileId);
    if (params?.startDate) searchParams.set('startDate', params.startDate);
    if (params?.endDate) searchParams.set('endDate', params.endDate);
    const query = searchParams.toString();
    return fetchApi<Workout[]>(`/workouts${query ? `?${query}` : ''}`);
  },
  get: (id: string) => fetchApi<Workout>(`/workouts/${id}`),
  create: (data: WorkoutFormData) =>
    fetchApi<Workout>('/workouts', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<WorkoutFormData & { status?: string; endTime?: string; durationMinutes?: number; rating?: number }>) =>
    fetchApi<Workout>(`/workouts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/workouts/${id}`, {
      method: 'DELETE',
    }),
};

// Workout Exercises
export const workoutExercisesApi = {
  list: (workoutId: string) =>
    fetchApi<WorkoutExercise[]>(`/workouts/${workoutId}/exercises`),
  add: (workoutId: string, data: { exerciseId: string; exerciseOrder: number; notes?: string }) =>
    fetchApi<WorkoutExercise>(`/workouts/${workoutId}/exercises`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (workoutId: string, exerciseId: string, data: { exerciseOrder?: number; notes?: string }) =>
    fetchApi<WorkoutExercise>(`/workouts/${workoutId}/exercises/${exerciseId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  remove: (workoutId: string, exerciseId: string) =>
    fetchApi<void>(`/workouts/${workoutId}/exercises/${exerciseId}`, {
      method: 'DELETE',
    }),
};

// Exercise Sets
export const setsApi = {
  list: (workoutExerciseId: string) =>
    fetchApi<ExerciseSet[]>(`/workout-exercises/${workoutExerciseId}/sets`),
  add: (workoutExerciseId: string, data: Partial<ExerciseSet>) =>
    fetchApi<ExerciseSet>(`/workout-exercises/${workoutExerciseId}/sets`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (setId: string, data: Partial<ExerciseSet>) =>
    fetchApi<ExerciseSet>(`/sets/${setId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (setId: string) =>
    fetchApi<void>(`/sets/${setId}`, {
      method: 'DELETE',
    }),
};

// Activities
export const activitiesApi = {
  list: (params?: { profileId?: string; activityType?: string; startDate?: string; endDate?: string }) => {
    const searchParams = new URLSearchParams();
    if (params?.profileId) searchParams.set('profileId', params.profileId);
    if (params?.activityType) searchParams.set('activityType', params.activityType);
    if (params?.startDate) searchParams.set('startDate', params.startDate);
    if (params?.endDate) searchParams.set('endDate', params.endDate);
    const query = searchParams.toString();
    return fetchApi<Activity[]>(`/activities${query ? `?${query}` : ''}`);
  },
  get: (id: string) => fetchApi<Activity>(`/activities/${id}`),
  create: (data: ActivityFormData) =>
    fetchApi<Activity>('/activities', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: Partial<ActivityFormData>) =>
    fetchApi<Activity>(`/activities/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/activities/${id}`, {
      method: 'DELETE',
    }),
};

// Body Measurements
export const bodyMeasurementsApi = {
  list: (profileId: string) =>
    fetchApi<BodyMeasurement[]>(`/body-measurements?profileId=${profileId}`),
  create: (data: { profileId: string; measuredAt: string; weight?: number; bodyFatPct?: number; notes?: string }) =>
    fetchApi<BodyMeasurement>('/body-measurements', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/body-measurements/${id}`, {
      method: 'DELETE',
    }),
};

// Personal Records
export const recordsApi = {
  list: (params?: { profileId?: string; exerciseId?: string }) => {
    const searchParams = new URLSearchParams();
    if (params?.profileId) searchParams.set('profileId', params.profileId);
    if (params?.exerciseId) searchParams.set('exerciseId', params.exerciseId);
    const query = searchParams.toString();
    return fetchApi<PersonalRecord[]>(`/personal-records${query ? `?${query}` : ''}`);
  },
  create: (data: Partial<PersonalRecord>) =>
    fetchApi<PersonalRecord>('/personal-records', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/personal-records/${id}`, {
      method: 'DELETE',
    }),
};

// Stretch Sequences
export const stretchSequencesApi = {
  list: (profileId?: string) => {
    const query = profileId ? `?profileId=${profileId}` : '';
    return fetchApi<StretchSequence[]>(`/stretch-sequences${query}`);
  },
  get: (id: string) => fetchApi<StretchSequence>(`/stretch-sequences/${id}`),
  create: (data: { profileId: string; name: string; description?: string }) =>
    fetchApi<StretchSequence>('/stretch-sequences', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (id: string, data: { name: string; description?: string; isActive?: boolean }) =>
    fetchApi<StretchSequence>(`/stretch-sequences/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/stretch-sequences/${id}`, {
      method: 'DELETE',
    }),
  addMovement: (sequenceId: string, data: { name: string; description?: string; durationSeconds: number; orderIndex: number }) =>
    fetchApi<StretchMovement>(`/stretch-sequences/${sequenceId}/movements`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
};

// Stretch Movements
export const stretchMovementsApi = {
  update: (id: string, data: { name: string; description?: string; durationSeconds: number; orderIndex: number }) =>
    fetchApi<StretchMovement>(`/stretch-movements/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  delete: (id: string) =>
    fetchApi<void>(`/stretch-movements/${id}`, {
      method: 'DELETE',
    }),
};

// Stretch Sessions
export const stretchSessionsApi = {
  list: (params?: { profileId?: string; sequenceId?: string; startDate?: string; endDate?: string }) => {
    const searchParams = new URLSearchParams();
    if (params?.profileId) searchParams.set('profileId', params.profileId);
    if (params?.sequenceId) searchParams.set('sequenceId', params.sequenceId);
    if (params?.startDate) searchParams.set('startDate', params.startDate);
    if (params?.endDate) searchParams.set('endDate', params.endDate);
    const query = searchParams.toString();
    return fetchApi<StretchSession[]>(`/stretch-sessions${query ? `?${query}` : ''}`);
  },
  get: (id: string) => fetchApi<StretchSession>(`/stretch-sessions/${id}`),
  create: (data: {
    profileId: string;
    sequenceId?: string;
    sessionDate: string;
    durationSeconds?: number;
    rating?: number;
    notes?: string;
    movements?: Array<{
      movementId?: string;
      movementName: string;
      plannedDurationSeconds: number;
      actualDurationSeconds?: number;
      reps: number;
      skipped: boolean;
      orderIndex: number;
    }>;
  }) =>
    fetchApi<StretchSession>('/stretch-sessions', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
};

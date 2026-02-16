export interface Profile {
  id: string;
  calendarId: string;
  name: string;
  birthDate?: string;
  height?: number;
  currentWeight?: number;
  targetWeight?: number;
  createdAt: string;
  updatedAt: string;
}

export interface ExerciseType {
  id: string;
  name: string;
  description?: string;
  icon?: string;
  createdAt: string;
}

export interface Exercise {
  id: string;
  exerciseTypeId: string;
  name: string;
  description?: string;
  muscleGroup?: string;
  equipment?: string;
  instructions?: string;
  videoUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export type WorkoutStatus = 'PLANNED' | 'IN_PROGRESS' | 'COMPLETED' | 'SKIPPED';

export interface Workout {
  id: string;
  profileId: string;
  name: string;
  description?: string;
  workoutDate: string;
  startTime?: string;
  endTime?: string;
  durationMinutes?: number;
  notes?: string;
  rating?: number;
  status: WorkoutStatus;
  createdAt: string;
  updatedAt: string;
  exercises?: WorkoutExercise[];
}

export type SetType = 'NORMAL' | 'WARMUP' | 'DROP' | 'FAILURE';

export interface ExerciseSet {
  id: string;
  workoutExerciseId: string;
  setNumber: number;
  setType: SetType;
  weight?: number;
  reps?: number;
  durationSeconds?: number;
  distance?: number;
  restSeconds?: number;
  notes?: string;
  createdAt: string;
}

export interface WorkoutExercise {
  id: string;
  workoutId: string;
  exerciseId: string;
  exerciseOrder: number;
  notes?: string;
  createdAt: string;
  sets?: ExerciseSet[];
  exercise?: Exercise;
}

export interface Activity {
  id: string;
  profileId: string;
  exerciseId?: string;
  name: string;
  activityType: string;
  activityDate: string;
  startTime?: string;
  durationMinutes?: number;
  rounds?: number;
  notes?: string;
  metrics?: ActivityMetrics;
  rating?: number;
  createdAt: string;
  updatedAt: string;
}

export interface ActivityMetrics {
  // Wim Hof specific
  breathingRounds?: number;
  breathsPerRound?: number[];
  retentionTimes?: number[];
  coldExposureSeconds?: number;
  waterTemperature?: number;
  pushUps?: number;

  // HIIT specific
  intervals?: number;
  workSeconds?: number;
  restSeconds?: number;
  maxHeartRate?: number;
  avgHeartRate?: number;

  // General
  caloriesBurned?: number;
  avgPace?: string;
  maxSpeed?: number;
  elevationGain?: number;
}

export interface PersonalRecord {
  id: string;
  profileId: string;
  exerciseId: string;
  recordType: string;
  value: number;
  unit: string;
  achievedAt: string;
  notes?: string;
  createdAt: string;
}

// Activity types
export const ACTIVITY_TYPES = [
  { value: 'WIM_HOF', label: 'Wim Hof Method', icon: '🧊' },
  { value: 'HIIT', label: 'HIIT', icon: '🔥' },
  { value: 'MEDITATION', label: 'Meditacao', icon: '🧘' },
  { value: 'YOGA', label: 'Yoga', icon: '🧘‍♀️' },
  { value: 'RUNNING', label: 'Corrida', icon: '🏃' },
  { value: 'CYCLING', label: 'Ciclismo', icon: '🚴' },
  { value: 'SWIMMING', label: 'Natacao', icon: '🏊' },
  { value: 'STRETCHING', label: 'Alongamento', icon: '🤸' },
  { value: 'OTHER', label: 'Outro', icon: '💪' },
] as const;

// Exercise muscle groups
export const MUSCLE_GROUPS = [
  'Peito',
  'Costas',
  'Ombros',
  'Biceps',
  'Triceps',
  'Pernas',
  'Gluteos',
  'Abdomen',
  'Core',
  'Full Body',
] as const;

// Form types
export interface WorkoutFormData {
  profileId: string;
  name: string;
  description?: string;
  workoutDate: string;
  startTime?: string;
  notes?: string;
}

export interface ActivityFormData {
  profileId: string;
  name: string;
  activityType: string;
  activityDate: string;
  startTime?: string;
  durationMinutes?: number;
  rounds?: number;
  notes?: string;
  metrics?: ActivityMetrics;
  rating?: number;
}

export interface ExerciseFormData {
  exerciseTypeId: string;
  name: string;
  description?: string;
  muscleGroup?: string;
  equipment?: string;
  instructions?: string;
}

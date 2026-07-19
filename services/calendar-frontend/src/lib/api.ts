import type { Event, Category, CategoryType, HabitStats, FlexibleHabitProgress, Calendar, Label } from '@/types/calendar';

/**
 * API client for calendar backend
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3334';
// The backend requires this token once configured. It ships in the browser bundle, so it is a
// gateway key (keeps anonymous traffic out), not a secret — the frontend itself sits behind Authelia.
const API_TOKEN = process.env.NEXT_PUBLIC_API_TOKEN;

/**
 * Base fetch wrapper with error handling
 */
async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = `${API_URL}${endpoint}`;

  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...(API_TOKEN ? { Authorization: `Bearer ${API_TOKEN}` } : {}),
        ...options?.headers,
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`);
    }

    // Status 204 (No Content) não tem corpo na resposta
    if (response.status === 204) {
      return null as T;
    }

    const data = await response.json();
    return data;
  } catch (error) {
    throw error;
  }
}

interface ListEventsParams {
  calendarId?: string;
  categoryId?: string;
  search?: string;
  startDate?: string; // YYYY-MM-DD
  endDate?: string;   // YYYY-MM-DD
  eventType?: 'EVENT' | 'HABIT' | 'TODO' | 'REMINDER';
}

export interface EventExecution {
  id: string;
  eventId: string;
  executionDate: string;
  completed: boolean;
  completedAt?: string;
  notes?: string;
}

export interface StatsData {
  key: string;
  label: string;
  total: number;
  completed: number;
  pending: number;
  percentage: number;
  averageCompletionTime?: number;
  streak?: number;
  breakdown?: {
    [key: string]: {
      total: number;
      completed: number;
    }
  };
}

interface GetStatsParams {
  startDate: string;
  endDate: string;
  calendarId?: string;
  categoryId?: string;
  categoryTypeId?: string;
  groupBy?: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';
  includeBreakdown?: boolean;
}

/**
 * API methods
 */
export const api = {
  // Events
  events: {
    list: (params?: ListEventsParams) => {
      const queryParams = new URLSearchParams();
      if (params?.calendarId) queryParams.append('calendarId', params.calendarId);
      if (params?.categoryId) queryParams.append('categoryId', params.categoryId);
      if (params?.search) queryParams.append('search', params.search);
      if (params?.startDate) queryParams.append('startDate', params.startDate);
      if (params?.endDate) queryParams.append('endDate', params.endDate);
      if (params?.eventType) queryParams.append('eventType', params.eventType);

      const query = queryParams.toString();
      return fetchAPI<Event[]>(`/events${query ? `?${query}` : ''}`);
    },
    listHabits: (params?: Omit<ListEventsParams, 'eventType'>) => {
      return api.events.list({ ...params, eventType: 'HABIT' });
    },
    listTodos: (params?: Omit<ListEventsParams, 'eventType'>) => {
      return api.events.list({ ...params, eventType: 'TODO' });
    },
    listCalendarEvents: (params?: Omit<ListEventsParams, 'eventType'>) => {
      return api.events.list({ ...params, eventType: 'EVENT' });
    },
    listReminders: (params?: Omit<ListEventsParams, 'eventType'>) => {
      return api.events.list({ ...params, eventType: 'REMINDER' });
    },
    create: (data: Record<string, unknown>) =>
      fetchAPI<Event>('/events', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Record<string, unknown>) =>
      fetchAPI<Event>(`/events/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchAPI<void>(`/events/${id}`, {
        method: 'DELETE',
      }),
    deleteRecurring: (id: string, scope: 'this' | 'future' | 'all', occurrenceDate: string) =>
      fetchAPI<void>(`/events/${id}/recurring?scope=${scope}&occurrenceDate=${occurrenceDate}`, {
        method: 'DELETE',
      }),
    toggleExecution: (eventId: string, executionDate: string, completed: boolean, notes?: string) =>
      fetchAPI<EventExecution>('/events/executions/toggle', {
        method: 'POST',
        body: JSON.stringify({ eventId, executionDate, completed, notes }),
      }),
    getExecutions: (eventId: string) =>
      fetchAPI<EventExecution[]>(`/events/${eventId}/executions`),
    getStats: (params: GetStatsParams) => {
      const queryParams = new URLSearchParams();
      queryParams.append('startDate', params.startDate);
      queryParams.append('endDate', params.endDate);
      if (params.calendarId) queryParams.append('calendarId', params.calendarId);
      if (params.categoryId) queryParams.append('categoryId', params.categoryId);
      if (params.categoryTypeId) queryParams.append('categoryTypeId', params.categoryTypeId);
      if (params.groupBy) queryParams.append('groupBy', params.groupBy);
      if (params.includeBreakdown !== undefined) queryParams.append('includeBreakdown', String(params.includeBreakdown));

      return fetchAPI<StatsData[]>(`/events/stats?${queryParams.toString()}`);
    },
    getHabitsStats: (params?: { calendarId?: string; categoryId?: string }) => {
      const queryParams = new URLSearchParams();
      if (params?.calendarId) queryParams.append('calendarId', params.calendarId);
      if (params?.categoryId) queryParams.append('categoryId', params.categoryId);

      const query = queryParams.toString();
      return fetchAPI<HabitStats[]>(`/events/habits/stats${query ? `?${query}` : ''}`);
    },
    getWeeklyProgress: (params?: { calendarId?: string; categoryId?: string }) => {
      const queryParams = new URLSearchParams();
      if (params?.calendarId) queryParams.append('calendarId', params.calendarId);
      if (params?.categoryId) queryParams.append('categoryId', params.categoryId);

      const query = queryParams.toString();
      return fetchAPI<FlexibleHabitProgress[]>(`/events/habits/weekly-progress${query ? `?${query}` : ''}`);
    },
    reorder: (orderedIds: string[]) =>
      fetchAPI<{ success: boolean }>('/events/reorder', {
        method: 'POST',
        body: JSON.stringify({ orderedIds }),
      }),
    upcomingReminders: (windowMinutes = 2) =>
      fetchAPI<Array<{
        eventId: string;
        eventTitle: string;
        startTime: string;
        startDate: string;
        minutesBefore: number;
        triggerAt: string;
      }>>(`/events/upcoming-reminders?windowMinutes=${windowMinutes}`),
  },

  // Calendars (Profiles)
  calendars: {
    list: () =>
      fetchAPI<{ data: Calendar[]; total: number }>('/calendars').then((res) => res.data),
  },

  // Google Calendar integration
  googleCalendar: {
    getConnectUrl: (calendarId: string) =>
      `${API_URL}/google-calendar/auth/connect?calendarId=${calendarId}`,
    disconnect: (calendarId: string) =>
      fetchAPI<void>(`/google-calendar/auth/disconnect/${calendarId}`, {
        method: 'DELETE',
      }),
    pullSync: (calendarId: string) =>
      fetchAPI<{ data: { created: number; updated: number; deleted: number } }>(
        `/google-calendar/sync/${calendarId}/pull`,
        { method: 'POST' },
      ),
  },

  // Categories
  categories: {
    list: (calendarId?: string) => {
      const query = calendarId ? `?calendarId=${calendarId}` : '';
      return fetchAPI<Category[]>(`/categories${query}`);
    },
    create: (data: Omit<Category, 'id' | 'isActive'>) =>
      fetchAPI<Category>('/categories', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<Omit<Category, 'id' | 'isActive'>>) =>
      fetchAPI<Category>(`/categories/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchAPI<void>(`/categories/${id}`, {
        method: 'DELETE',
      }),
  },

  // Category Types
  categoryTypes: {
    list: (calendarId?: string) => {
      const query = calendarId ? `?calendarId=${calendarId}` : '';
      return fetchAPI<CategoryType[]>(`/category-types${query}`);
    },
    create: (data: { calendarId: string; name: string; value: string; color: string; icon?: string }) =>
      fetchAPI<CategoryType>('/category-types', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: { name?: string; value?: string; color?: string; icon?: string }) =>
      fetchAPI<CategoryType>(`/category-types/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchAPI<void>(`/category-types/${id}`, {
        method: 'DELETE',
      }),
  },

  // Labels
  labels: {
    list: (calendarId?: string) => {
      const query = calendarId ? `?calendarId=${calendarId}` : '';
      return fetchAPI<Label[]>(`/labels${query}`);
    },
    create: (data: { calendarId: string; name: string; color: string }) =>
      fetchAPI<Label>('/labels', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: { name?: string; color?: string }) =>
      fetchAPI<Label>(`/labels/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchAPI<void>(`/labels/${id}`, {
        method: 'DELETE',
      }),
  },
};

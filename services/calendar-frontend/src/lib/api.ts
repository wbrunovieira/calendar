import type { Event, Category, CategoryType } from '@/types/calendar';

/**
 * API client for calendar backend
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3334';

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
}

export interface EventExecution {
  id: string;
  eventId: string;
  executionDate: string;
  completed: boolean;
  completedAt?: string;
  notes?: string;
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

      const query = queryParams.toString();
      return fetchAPI<Event[]>(`/events${query ? `?${query}` : ''}`);
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
};

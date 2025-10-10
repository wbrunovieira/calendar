import type { Event, Category, Calendar } from '@/types/calendar';

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
      throw new Error(`API error: ${response.status} ${response.statusText}`);
    }

    return response.json();
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
}

interface ListEventsParams {
  calendarId?: string;
  categoryId?: string;
  search?: string;
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

      const query = queryParams.toString();
      return fetchAPI<Event[]>(`/events${query ? `?${query}` : ''}`);
    },
    create: (data: Omit<Event, 'id' | 'isActive'>) =>
      fetchAPI<Event>('/events', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchAPI<void>(`/events/${id}`, {
        method: 'DELETE',
      }),
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
    delete: (id: string) =>
      fetchAPI<void>(`/categories/${id}`, {
        method: 'DELETE',
      }),
  },
};

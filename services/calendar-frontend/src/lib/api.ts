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

  console.log('=== FETCH API ===');
  console.log('URL:', url);
  console.log('Method:', options?.method || 'GET');
  console.log('Headers:', options?.headers);
  console.log('Body:', options?.body);

  try {
    console.log('Iniciando fetch...');
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    console.log('Response status:', response.status);
    console.log('Response ok:', response.ok);
    console.log('Response headers:', response.headers);

    if (!response.ok) {
      const errorText = await response.text();
      console.error('Erro na resposta:', errorText);
      throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`);
    }

    // Status 204 (No Content) não tem corpo na resposta
    if (response.status === 204) {
      console.log('=== FETCH API SUCESSO (204 No Content) ===');
      return null as T;
    }

    const data = await response.json();
    console.log('Response data:', data);
    console.log('=== FETCH API SUCESSO ===');
    return data;
  } catch (error) {
    console.error('=== FETCH API ERRO ===');
    console.error('API request failed:', error);
    console.error('Error name:', (error as Error).name);
    console.error('Error message:', (error as Error).message);
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
    update: (id: string, data: Partial<Event>) =>
      fetchAPI<Event>(`/events/${id}`, {
        method: 'PUT',
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

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

/**
 * API methods
 */
export const api = {
  // Health check
  health: () => fetchAPI<{ status: string }>('/'),

  // Calendar events (to be implemented)
  // events: {
  //   list: () => fetchAPI<Event[]>('/api/events'),
  //   get: (id: string) => fetchAPI<Event>(`/api/events/${id}`),
  //   create: (data: CreateEventDto) => fetchAPI<Event>('/api/events', {
  //     method: 'POST',
  //     body: JSON.stringify(data),
  //   }),
  // },
};

/**
 * Custom hook for managing calendar events and categories data
 */

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Event, Category } from '@/types/calendar';
import { getDefaultDateRange } from '@/utils/calendar/dateRanges';

export function useCalendarData() {
  const [events, setEvents] = useState<Event[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    try {
      setLoading(true);

      const dateRange = getDefaultDateRange();

      const [fetchedEvents, fetchedCategories] = await Promise.all([api.events.list(dateRange), api.categories.list()]);
      setEvents(fetchedEvents);
      setCategories(fetchedCategories);
    } catch {
      // Error fetching data
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return {
    events,
    categories,
    loading,
    refetch: fetchData,
  };
}

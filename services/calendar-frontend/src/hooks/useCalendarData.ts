/**
 * Custom hook for managing calendar events and categories data
 */

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Event, Category, CategoryType } from '@/types/calendar';
import { getDefaultDateRange } from '@/utils/calendar/dateRanges';

export function useCalendarData() {
  const [events, setEvents] = useState<Event[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [categoryTypes, setCategoryTypes] = useState<CategoryType[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    try {
      setLoading(true);

      const dateRange = getDefaultDateRange();

      const [fetchedEvents, fetchedCategories, fetchedCategoryTypes] = await Promise.all([
        api.events.list({ ...dateRange, eventType: 'EVENT' }),
        api.categories.list(),
        api.categoryTypes.list(),
      ]);

      // Debug: verificar se os eventos têm executions
      console.log('[useCalendarData] Events fetched:', {
        totalEvents: fetchedEvents.length,
        eventsWithExecutions: fetchedEvents.filter(e => e.executions && e.executions.length > 0).length,
        sampleEvent: fetchedEvents.find(e => e.executions && e.executions.length > 0) || fetchedEvents[0]
      });

      setEvents(fetchedEvents);
      setCategories(fetchedCategories);
      setCategoryTypes(fetchedCategoryTypes);
    } catch (error) {
      console.error('[useCalendarData] Error fetching data', error);
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
    categoryTypes,
    loading,
    refetch: fetchData,
  };
}

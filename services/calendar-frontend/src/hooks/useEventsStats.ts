/**
 * Custom hook for fetching event statistics
 */

import { useState, useEffect, useCallback } from 'react';
import { api, StatsData } from '@/lib/api';

interface UseEventsStatsParams {
  startDate: string;
  endDate: string;
  calendarId?: string;
  categoryId?: string;
  categoryTypeId?: string;
  groupBy?: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';
  includeBreakdown?: boolean;
  enabled?: boolean; // Para controlar quando buscar
}

export function useEventsStats(params: UseEventsStatsParams) {
  const [stats, setStats] = useState<StatsData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const fetchStats = useCallback(async () => {
    if (params.enabled === false) return;

    try {
      setLoading(true);
      setError(null);

      const data = await api.events.getStats({
        startDate: params.startDate,
        endDate: params.endDate,
        calendarId: params.calendarId,
        categoryId: params.categoryId,
        categoryTypeId: params.categoryTypeId,
        groupBy: params.groupBy || 'day',
        includeBreakdown: params.includeBreakdown,
      });

      setStats(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch stats'));
    } finally {
      setLoading(false);
    }
  }, [
    params.startDate,
    params.endDate,
    params.calendarId,
    params.categoryId,
    params.categoryTypeId,
    params.groupBy,
    params.includeBreakdown,
    params.enabled,
  ]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return {
    stats,
    loading,
    error,
    refetch: fetchStats,
  };
}

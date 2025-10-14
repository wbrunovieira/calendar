/**
 * Custom hook for fetching category types
 */

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import { CategoryType } from '@/types/calendar';

export function useCategoryTypes(calendarId?: string) {
  const [categoryTypes, setCategoryTypes] = useState<CategoryType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchCategoryTypes = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.categoryTypes.list(calendarId);
      setCategoryTypes(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch category types'));
    } finally {
      setLoading(false);
    }
  }, [calendarId]);

  useEffect(() => {
    fetchCategoryTypes();
  }, [fetchCategoryTypes]);

  return {
    categoryTypes,
    loading,
    error,
    refetch: fetchCategoryTypes,
  };
}

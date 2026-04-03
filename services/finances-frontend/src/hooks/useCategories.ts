'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { Category } from '@/types/finances';

export function useCategories(profileId: string | null, type?: string) {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchCategories = useCallback(async () => {
    if (!profileId) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({ profileId });
      if (type) params.set('type', type);
      const data = await api.get<{ data: Category[] }>(`/categories?${params}`);
      setCategories(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar categorias:', error);
    } finally {
      setLoading(false);
    }
  }, [profileId, type]);

  useEffect(() => {
    fetchCategories();
  }, [fetchCategories]);

  return { categories, loading, refetch: fetchCategories };
}

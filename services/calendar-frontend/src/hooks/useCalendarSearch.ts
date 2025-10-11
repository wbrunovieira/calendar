/**
 * Custom hook for managing calendar search functionality
 */

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Event, Category } from '@/types/calendar';
import { getDefaultDateRange } from '@/utils/calendar/dateRanges';
import { SEARCH_CONFIG } from '@/constants/calendar';

interface UseCalendarSearchProps {
  categories: Category[];
  onResultClick: (event: Event) => void;
}

export function useCalendarSearch({ onResultClick }: UseCalendarSearchProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Event[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showSearchResults, setShowSearchResults] = useState(false);

  // Search handler with debounce
  useEffect(() => {
    const searchEvents = async () => {
      if (searchQuery.trim().length === 0) {
        setSearchResults([]);
        setShowSearchResults(false);
        return;
      }

      if (searchQuery.trim().length < SEARCH_CONFIG.MIN_QUERY_LENGTH) {
        return; // Don't search for single character
      }

      setIsSearching(true);
      try {
        const dateRange = getDefaultDateRange();

        const results = await api.events.list({
          search: searchQuery,
          ...dateRange,
        });
        // Backend already returns expanded occurrences, just sort by date (closest first)
        results.sort((a, b) => new Date(a.startDate).getTime() - new Date(b.startDate).getTime());
        setSearchResults(results);
        setShowSearchResults(true);
      } catch {
        // Error searching events
      } finally {
        setIsSearching(false);
      }
    };

    const debounceTimer = setTimeout(searchEvents, SEARCH_CONFIG.DEBOUNCE_MS);
    return () => clearTimeout(debounceTimer);
  }, [searchQuery]);

  // Close search results when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('.search-container')) {
        setShowSearchResults(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSearchResultClick = (event: Event) => {
    onResultClick(event);
    setShowSearchResults(false);
    setSearchQuery('');
  };

  const clearSearch = () => {
    setSearchQuery('');
    setShowSearchResults(false);
  };

  return {
    searchQuery,
    setSearchQuery,
    searchResults,
    isSearching,
    showSearchResults,
    setShowSearchResults,
    handleSearchResultClick,
    clearSearch,
  };
}

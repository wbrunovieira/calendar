/**
 * Calendar Search Component
 * Displays search bar and search results dropdown
 */

import { calendars } from '@/data/calendars';
import { Category, Event } from '@/types/calendar';
import { DAYS_OF_WEEK_SHORT } from '@/constants/calendar';

interface SearchState {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  searchResults: Event[];
  isSearching: boolean;
  showSearchResults: boolean;
  setShowSearchResults: (show: boolean) => void;
  handleSearchResultClick: (event: Event) => void;
  clearSearch: () => void;
}

interface CalendarSearchProps {
  search: SearchState;
  categories: Category[];
}

export default function CalendarSearch({ search, categories }: CalendarSearchProps) {
  const {
    searchQuery,
    setSearchQuery,
    searchResults,
    isSearching,
    showSearchResults,
    setShowSearchResults,
    handleSearchResultClick,
    clearSearch,
  } = search;
  return (
    <div className="fixed top-6 left-1/2 transform -translate-x-1/2 z-50 w-full max-w-md px-4 search-container">
      <div className="relative">
        <input
          type="text"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          onFocus={() => searchQuery && setShowSearchResults(true)}
          placeholder="Buscar eventos..."
          className="w-full px-4 py-3 pl-12 bg-white/10 backdrop-blur-md border border-white/20 text-white placeholder-white/50 rounded-xl focus:ring-2 focus:ring-[#792990] focus:border-transparent shadow-2xl"
        />
        <svg
          className="w-5 h-5 absolute left-4 top-1/2 transform -translate-y-1/2 text-white/50"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        {searchQuery && (
          <button
            onClick={clearSearch}
            className="absolute right-4 top-1/2 transform -translate-y-1/2 text-white/50 hover:text-white"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>

      {/* Search Results Dropdown */}
      {showSearchResults && (
        <div className="absolute top-full mt-2 w-full bg-[#350545] border border-white/20 rounded-xl shadow-2xl max-h-96 overflow-y-auto">
          {isSearching ? (
            <div className="p-4 text-center text-white/70">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white mx-auto"></div>
            </div>
          ) : searchResults.length === 0 ? (
            <div className="p-4 text-center text-white/70">Nenhum evento encontrado</div>
          ) : (
            <div className="divide-y divide-white/10">
              {searchResults.map(event => {
                const category = categories.find(c => c.id === event.categoryId);
                const calendar = calendars.find(c => c.id === event.calendarId);

                // Parse date string as local date to avoid timezone issues
                // "2025-10-20" should be Oct 20, not Oct 19 (which happens with UTC interpretation)
                const [year, month, day] = event.startDate.split('-').map(Number);
                const eventDate = new Date(year, month - 1, day);
                const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

                // Format date to show day of week
                const dayOfWeek = DAYS_OF_WEEK_SHORT[eventDate.getDay()];
                const isToday = eventDate.toDateString() === new Date().toDateString();
                const isTomorrow = eventDate.toDateString() === new Date(Date.now() + 86400000).toDateString();

                return (
                  <button
                    key={event.id}
                    onClick={() => handleSearchResultClick(event)}
                    className="w-full px-4 py-3 hover:bg-white/10 transition-colors text-left flex items-center gap-3"
                  >
                    <div
                      className="w-10 h-10 rounded-lg flex items-center justify-center text-lg flex-shrink-0"
                      style={{ backgroundColor: category?.color + '80' }}
                    >
                      {category?.icon || '📅'}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-white font-semibold truncate">{event.title}</span>
                        {event.isRecurring && (
                          <span className="text-xs bg-white/20 px-1.5 py-0.5 rounded" title="Evento recorrente">
                            🔄
                          </span>
                        )}
                        <span className="text-xs" style={{ color: calendar?.color }}>
                          {calendarIcon}
                        </span>
                      </div>
                      <div className="text-xs text-white/70 flex items-center gap-2">
                        {isToday && <span className="text-green-400 font-semibold">Hoje</span>}
                        {isTomorrow && <span className="text-blue-400 font-semibold">Amanhã</span>}
                        {!isToday && !isTomorrow && (
                          <>
                            <span>{dayOfWeek}</span>
                            <span>•</span>
                          </>
                        )}
                        <span>{eventDate.toLocaleDateString('pt-BR')}</span>
                        <span>•</span>
                        <span>{event.startTime}</span>
                        {event.description && (
                          <>
                            <span>•</span>
                            <span className="truncate">{event.description}</span>
                          </>
                        )}
                      </div>
                    </div>
                    <svg
                      className="w-5 h-5 text-white/50 flex-shrink-0"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

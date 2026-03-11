'use client';

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { API_BASE } from '@/lib/api';

interface Transaction {
  id: string;
  description: string;
  amount: number;
  type: 'INCOME' | 'EXPENSE' | 'TRANSFER';
  status: 'PLANNED' | 'CONFIRMED' | 'CANCELLED';
  occurredOn: string;
  categoryId?: string;
  bankAccountId: string;
}

interface RecurringTransaction {
  id: string;
  description: string;
  amount: number;
  type: 'INCOME' | 'EXPENSE';
  status: 'ACTIVE' | 'INACTIVE';
  recurrenceRule: string;
  categoryId?: string;
}

interface Category {
  id: string;
  name: string;
}

interface BankAccount {
  id: string;
  name: string;
}

interface SearchResult {
  id: string;
  type: 'transaction' | 'recurring';
  title: string;
  subtitle: string;
  amount: number;
  transactionType: 'INCOME' | 'EXPENSE' | 'TRANSFER';
  status: string;
  date?: string;
  matchedField: string;
}

interface GlobalSearchProps {
  profileId: string | null;
  categories: Category[];
  accounts: BankAccount[];
  onSelectTransaction?: (id: string) => void;
  onSelectRecurring?: (id: string) => void;
  refreshKey?: number;
}

const formatCurrency = (value: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(value);

const formatDate = (value: string) => {
  const datePart = value.split('T')[0];
  const [year, month, day] = datePart.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  return new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short' }).format(date);
};

const highlightMatch = (text: string, query: string) => {
  if (!query.trim()) return text;
  const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
  const parts = text.split(regex);
  return parts.map((part, i) =>
    regex.test(part) ? (
      <mark key={i} className="bg-yellow-400/30 text-yellow-200 rounded px-0.5">
        {part}
      </mark>
    ) : (
      part
    )
  );
};

export default function GlobalSearch({
  profileId,
  categories,
  accounts,
  onSelectTransaction,
  onSelectRecurring,
  refreshKey,
}: GlobalSearchProps) {
  const [query, setQuery] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);

  const categoryMap = useMemo(() => {
    const map = new Map<string, string>();
    categories.forEach((c) => map.set(c.id, c.name));
    return map;
  }, [categories]);

  const accountMap = useMemo(() => {
    const map = new Map<string, string>();
    accounts.forEach((a) => map.set(a.id, a.name));
    return map;
  }, [accounts]);

  // Fetch all data when profile changes
  useEffect(() => {
    if (!profileId) return;

    const fetchData = async () => {
      try {
        const [txRes, recRes] = await Promise.all([
          fetch(`${API_BASE}/transactions?profileId=${profileId}`),
          fetch(`${API_BASE}/recurring-transactions?profileId=${profileId}`),
        ]);

        if (txRes.ok) {
          const data = await txRes.json();
          setTransactions(data.data || []);
        }
        if (recRes.ok) {
          const data = await recRes.json();
          setRecurrings(data.data || []);
        }
      } catch (e) {
        console.warn('Error fetching search data:', e);
      }
    };

    fetchData();
  }, [profileId, refreshKey]);

  // Search logic
  const searchResults = useCallback(
    (searchQuery: string): SearchResult[] => {
      if (!searchQuery.trim() || searchQuery.length < 2) return [];

      const q = searchQuery.toLowerCase();
      const results: SearchResult[] = [];

      // Search transactions
      transactions.forEach((tx) => {
        const descMatch = tx.description.toLowerCase().includes(q);
        const categoryName = tx.categoryId ? categoryMap.get(tx.categoryId) : '';
        const catMatch = categoryName?.toLowerCase().includes(q);
        const accountName = accountMap.get(tx.bankAccountId) || '';
        const accMatch = accountName.toLowerCase().includes(q);
        const amountStr = tx.amount.toString();
        const amountMatch = amountStr.includes(q);

        if (descMatch || catMatch || accMatch || amountMatch) {
          let matchedField = '';
          if (descMatch) matchedField = 'descricao';
          else if (catMatch) matchedField = 'categoria';
          else if (accMatch) matchedField = 'conta';
          else if (amountMatch) matchedField = 'valor';

          results.push({
            id: tx.id,
            type: 'transaction',
            title: tx.description,
            subtitle: `${categoryName || 'Sem categoria'} • ${accountName}`,
            amount: tx.amount,
            transactionType: tx.type,
            status: tx.status,
            date: tx.occurredOn,
            matchedField,
          });
        }
      });

      // Search recurring transactions
      recurrings.forEach((rec) => {
        const descMatch = rec.description.toLowerCase().includes(q);
        const categoryName = rec.categoryId ? categoryMap.get(rec.categoryId) : '';
        const catMatch = categoryName?.toLowerCase().includes(q);
        const amountStr = rec.amount.toString();
        const amountMatch = amountStr.includes(q);

        if (descMatch || catMatch || amountMatch) {
          let matchedField = '';
          if (descMatch) matchedField = 'descricao';
          else if (catMatch) matchedField = 'categoria';
          else if (amountMatch) matchedField = 'valor';

          results.push({
            id: rec.id,
            type: 'recurring',
            title: rec.description,
            subtitle: `${categoryName || 'Sem categoria'} • Fixa ${rec.status === 'ACTIVE' ? 'ativa' : 'inativa'}`,
            amount: rec.amount,
            transactionType: rec.type,
            status: rec.status,
            matchedField,
          });
        }
      });

      // Sort by relevance (description matches first, then by date)
      return results
        .sort((a, b) => {
          if (a.matchedField === 'descricao' && b.matchedField !== 'descricao') return -1;
          if (b.matchedField === 'descricao' && a.matchedField !== 'descricao') return 1;
          if (a.date && b.date) return new Date(b.date).getTime() - new Date(a.date).getTime();
          return 0;
        })
        .slice(0, 15); // Limit results
    },
    [transactions, recurrings, categoryMap, accountMap]
  );

  // Debounced search
  useEffect(() => {
    if (!query.trim() || query.length < 2) {
      setResults([]);
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    const timer = setTimeout(() => {
      const searchRes = searchResults(query);
      setResults(searchRes);
      setIsLoading(false);
      setSelectedIndex(-1);
    }, 150);

    return () => clearTimeout(timer);
  }, [query, searchResults]);

  // Click outside handler
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Global "/" keyboard shortcut to focus search
  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      // Don't trigger if user is typing in an input/textarea
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        return;
      }

      if (e.key === '/') {
        e.preventDefault();
        inputRef.current?.focus();
        setIsOpen(true);
      }
    };

    document.addEventListener('keydown', handleGlobalKeyDown);
    return () => document.removeEventListener('keydown', handleGlobalKeyDown);
  }, []);

  // Keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isOpen || results.length === 0) {
      if (e.key === 'Escape') {
        setQuery('');
        inputRef.current?.blur();
      }
      return;
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((prev) => (prev < results.length - 1 ? prev + 1 : 0));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((prev) => (prev > 0 ? prev - 1 : results.length - 1));
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && results[selectedIndex]) {
          handleSelect(results[selectedIndex]);
        }
        break;
      case 'Escape':
        setIsOpen(false);
        setQuery('');
        inputRef.current?.blur();
        break;
    }
  };

  // Scroll selected item into view
  useEffect(() => {
    if (selectedIndex >= 0 && resultsRef.current) {
      const selectedElement = resultsRef.current.children[selectedIndex] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    }
  }, [selectedIndex]);

  const handleSelect = (result: SearchResult) => {
    if (result.type === 'transaction') {
      onSelectTransaction?.(result.id);
    } else {
      onSelectRecurring?.(result.id);
    }
    setIsOpen(false);
    setQuery('');
  };

  const getStatusIcon = (result: SearchResult) => {
    if (result.type === 'recurring') {
      return result.status === 'ACTIVE' ? '🔄' : '⏸️';
    }
    switch (result.status) {
      case 'CONFIRMED':
        return '✅';
      case 'PLANNED':
        return '🗓️';
      case 'CANCELLED':
        return '🛑';
      default:
        return '📝';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'INCOME':
        return 'text-emerald-400';
      case 'EXPENSE':
        return 'text-rose-400';
      default:
        return 'text-blue-400';
    }
  };

  return (
    <div ref={containerRef} className="relative w-full max-w-2xl mx-auto">
      {/* Search Input */}
      <div className="relative group">
        <div className="absolute inset-0 bg-gradient-to-r from-blue-500/20 via-purple-500/20 to-pink-500/20 rounded-2xl blur-xl opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity duration-500" />
        <div className="relative">
          <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            {isLoading ? (
              <svg className="animate-spin h-5 w-5 text-white/50" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            ) : (
              <svg className="h-5 w-5 text-white/50 group-focus-within:text-white/80 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            )}
          </div>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setIsOpen(true);
            }}
            onFocus={() => setIsOpen(true)}
            onKeyDown={handleKeyDown}
            placeholder="Buscar transacoes, fixas, categorias..."
            className="w-full pl-12 pr-12 py-4 bg-white/5 hover:bg-white/10 focus:bg-white/10 border border-white/10 hover:border-white/20 focus:border-white/30 rounded-2xl text-white placeholder-white/40 outline-none transition-all duration-300 text-lg backdrop-blur-sm"
          />
          {query && (
            <button
              onClick={() => {
                setQuery('');
                setResults([]);
                inputRef.current?.focus();
              }}
              className="absolute inset-y-0 right-0 pr-4 flex items-center text-white/40 hover:text-white/80 transition-colors"
            >
              <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Keyboard hint */}
      {!isOpen && !query && (
        <div className="absolute -bottom-6 left-0 right-0 flex justify-center">
          <span className="text-xs text-white/30">
            Pressione <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/50 font-mono text-[10px]">/</kbd> para focar
          </span>
        </div>
      )}

      {/* Results Dropdown */}
      {isOpen && query.length >= 2 && (
        <div className="absolute z-50 w-full mt-2 bg-slate-900/95 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl overflow-hidden animate-in fade-in slide-in-from-top-2 duration-200">
          {results.length === 0 && !isLoading && (
            <div className="px-6 py-8 text-center">
              <div className="text-4xl mb-3">🔍</div>
              <p className="text-white/60">Nenhum resultado para &quot;{query}&quot;</p>
              <p className="text-white/40 text-sm mt-1">Tente buscar por descricao, categoria ou valor</p>
            </div>
          )}

          {results.length > 0 && (
            <>
              <div className="px-4 py-2 border-b border-white/10 flex items-center justify-between">
                <span className="text-xs text-white/50">
                  {results.length} {results.length === 1 ? 'resultado' : 'resultados'}
                </span>
                <span className="text-xs text-white/30">
                  <kbd className="px-1 py-0.5 rounded bg-white/10 text-white/50 font-mono text-[10px]">↑↓</kbd> navegar
                  <kbd className="px-1 py-0.5 rounded bg-white/10 text-white/50 font-mono text-[10px] ml-2">Enter</kbd> selecionar
                </span>
              </div>
              <div ref={resultsRef} className="max-h-[400px] overflow-y-auto">
                {results.map((result, index) => (
                  <button
                    key={`${result.type}-${result.id}`}
                    onClick={() => handleSelect(result)}
                    onMouseEnter={() => setSelectedIndex(index)}
                    className={`w-full px-4 py-3 flex items-center gap-4 text-left transition-colors ${
                      index === selectedIndex
                        ? 'bg-white/10'
                        : 'hover:bg-white/5'
                    }`}
                  >
                    <div className="flex-shrink-0 w-10 h-10 rounded-xl bg-white/5 flex items-center justify-center text-lg">
                      {getStatusIcon(result)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-white font-medium truncate">
                          {highlightMatch(result.title, query)}
                        </span>
                        {result.type === 'recurring' && (
                          <span className="flex-shrink-0 px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-300 text-[10px] font-medium">
                            FIXA
                          </span>
                        )}
                        {result.status === 'PLANNED' && (
                          <span className="flex-shrink-0 px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 text-[10px] font-medium">
                            PLANEJADO
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 text-sm text-white/50">
                        <span className="truncate">{highlightMatch(result.subtitle, query)}</span>
                        {result.date && (
                          <>
                            <span>•</span>
                            <span className="flex-shrink-0">{formatDate(result.date)}</span>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="flex-shrink-0 text-right">
                      <span className={`font-semibold ${getTypeColor(result.transactionType)}`}>
                        {result.transactionType === 'EXPENSE' ? '-' : ''}
                        {formatCurrency(result.amount)}
                      </span>
                      <div className="text-[10px] text-white/40 mt-0.5">
                        encontrado em {result.matchedField}
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

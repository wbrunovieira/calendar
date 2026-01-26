'use client';

import { useState, useEffect, useRef } from 'react';
import { api } from '@/lib/api';
import type { Label } from '@/types/calendar';

interface LabelSelectorProps {
  calendarId: string;
  selectedLabelId?: string;
  onSelect: (labelId: string | undefined) => void;
}

const PRESET_COLORS = [
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#14b8a6', // teal
  '#3b82f6', // blue
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#6b7280', // gray
];

export default function LabelSelector({
  calendarId,
  selectedLabelId,
  onSelect,
}: LabelSelectorProps) {
  const [labels, setLabels] = useState<Label[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [newLabelName, setNewLabelName] = useState('');
  const [newLabelColor, setNewLabelColor] = useState(PRESET_COLORS[0]);
  const [loading, setLoading] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const selectedLabel = labels.find((l) => l.id === selectedLabelId);

  // Fetch labels when calendarId changes
  useEffect(() => {
    if (calendarId) {
      api.labels.list(calendarId).then(setLabels).catch(console.error);
    } else {
      setLabels([]);
    }
  }, [calendarId]);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setIsCreating(false);
        setNewLabelName('');
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Focus input when creating
  useEffect(() => {
    if (isCreating && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isCreating]);

  const handleCreateLabel = async () => {
    if (!newLabelName.trim() || !calendarId) return;

    setLoading(true);
    try {
      const newLabel = await api.labels.create({
        calendarId,
        name: newLabelName.trim(),
        color: newLabelColor,
      });
      setLabels((prev) => [...prev, newLabel]);
      onSelect(newLabel.id);
      setIsCreating(false);
      setNewLabelName('');
      setIsOpen(false);
    } catch (error) {
      console.error('Error creating label:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleCreateLabel();
    } else if (e.key === 'Escape') {
      setIsCreating(false);
      setNewLabelName('');
    }
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <label className="block text-sm font-medium text-white/70 mb-1">Label</label>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-left text-white focus:outline-none focus:ring-2 focus:ring-purple-500 flex items-center gap-2"
      >
        {selectedLabel ? (
          <>
            <span
              className="w-3 h-3 rounded-full flex-shrink-0"
              style={{ backgroundColor: selectedLabel.color }}
            />
            <span className="truncate">{selectedLabel.name}</span>
          </>
        ) : (
          <span className="text-white/40">Sem label</span>
        )}
        <svg
          className={`w-4 h-4 ml-auto transition-transform ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div className="absolute z-50 mt-1 w-full bg-[#2a0a3a] border border-white/20 rounded-lg shadow-xl overflow-hidden">
          <div className="max-h-48 overflow-y-auto">
            {/* No label option */}
            <button
              type="button"
              onClick={() => {
                onSelect(undefined);
                setIsOpen(false);
              }}
              className={`w-full px-4 py-2 text-left hover:bg-white/10 flex items-center gap-2 ${
                !selectedLabelId ? 'bg-white/5' : ''
              }`}
            >
              <span className="w-3 h-3 rounded-full bg-white/20 flex-shrink-0" />
              <span className="text-white/70">Sem label</span>
            </button>

            {/* Existing labels */}
            {labels.map((label) => (
              <button
                key={label.id}
                type="button"
                onClick={() => {
                  onSelect(label.id);
                  setIsOpen(false);
                }}
                className={`w-full px-4 py-2 text-left hover:bg-white/10 flex items-center gap-2 ${
                  selectedLabelId === label.id ? 'bg-white/5' : ''
                }`}
              >
                <span
                  className="w-3 h-3 rounded-full flex-shrink-0"
                  style={{ backgroundColor: label.color }}
                />
                <span className="text-white truncate">{label.name}</span>
              </button>
            ))}
          </div>

          {/* Create new label */}
          <div className="border-t border-white/10">
            {!isCreating ? (
              <button
                type="button"
                onClick={() => setIsCreating(true)}
                className="w-full px-4 py-2 text-left hover:bg-white/10 flex items-center gap-2 text-purple-400"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 4v16m8-8H4"
                  />
                </svg>
                <span>Criar novo label</span>
              </button>
            ) : (
              <div className="p-3 space-y-3">
                <input
                  ref={inputRef}
                  type="text"
                  value={newLabelName}
                  onChange={(e) => setNewLabelName(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="Nome do label"
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500"
                />

                {/* Color picker */}
                <div className="flex flex-wrap gap-1.5">
                  {PRESET_COLORS.map((color) => (
                    <button
                      key={color}
                      type="button"
                      onClick={() => setNewLabelColor(color)}
                      className={`w-6 h-6 rounded-full transition-transform ${
                        newLabelColor === color ? 'ring-2 ring-white scale-110' : ''
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>

                {/* Create button */}
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => {
                      setIsCreating(false);
                      setNewLabelName('');
                    }}
                    className="flex-1 px-3 py-1.5 bg-white/10 text-white text-sm rounded-lg hover:bg-white/20 transition-colors"
                  >
                    Cancelar
                  </button>
                  <button
                    type="button"
                    onClick={handleCreateLabel}
                    disabled={!newLabelName.trim() || loading}
                    className="flex-1 px-3 py-1.5 bg-purple-600 text-white text-sm rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {loading ? 'Criando...' : 'Criar'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

'use client';

import { useState } from 'react';
import { calendars } from '@/data/calendars';

interface CreateCategoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (category: {
    calendarId: string;
    name: string;
    icon: string;
    color: string;
    type: string;
  }) => Promise<void>;
}

const EMOJI_LIST = [
  '💼', '💰', '💻', '📞', '📧', '📝', '📊', '📈', '📉', '🎯',
  '🏃', '🥋', '🏊', '🚴', '⚽', '🏀', '🎾', '🏋️', '🧘', '🤸',
  '📖', '🎨', '🎵', '🎮', '🎬', '📺', '🎭', '🎪', '🎡', '🎢',
  '🍔', '🍕', '🍜', '🍱', '🍰', '☕', '🍷', '🥗', '🍎', '🥑',
  '🏠', '🏢', '🏥', '🏫', '🏪', '🏨', '⛪', '🏛️', '🏗️', '🏭',
  '✈️', '🚗', '🚕', '🚌', '🚎', '🏍️', '🚲', '🚁', '⛵', '🚀',
  '💊', '🩺', '🧬', '🔬', '🧪', '💉', '🩹', '🧴', '🪥', '🧼',
  '👨‍👩‍👧‍👦', '👶', '👦', '👧', '🧒', '👨', '👩', '🧑', '👴', '👵',
  '🎓', '📚', '✏️', '📐', '📏', '🖊️', '🖍️', '🖌️', '🖋️', '✒️',
  '💡', '🔦', '🕯️', '💡', '🔌', '🔋', '🪔', '🧯', '🛠️', '🔧',
];

const COLORS = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E2', '#F8B195', '#C06C84',
  '#6C5B7B', '#355C7D', '#2C3E50', '#E74C3C', '#3498DB',
  '#2ECC71', '#F39C12', '#9B59B6', '#1ABC9C', '#E67E22',
];

const CATEGORY_TYPES = [
  { value: 'work', label: 'Trabalho' },
  { value: 'health', label: 'Saúde' },
  { value: 'leisure', label: 'Lazer' },
  { value: 'wellness', label: 'Bem-estar' },
  { value: 'education', label: 'Educação' },
  { value: 'finance', label: 'Finanças' },
  { value: 'family', label: 'Família' },
  { value: 'home', label: 'Casa' },
  { value: 'travel', label: 'Viagem' },
  { value: 'other', label: 'Outro' },
];

export default function CreateCategoryModal({ isOpen, onClose, onSave }: CreateCategoryModalProps) {
  const [calendarId, setCalendarId] = useState(calendars[0]?.id || '');
  const [name, setName] = useState('');
  const [icon, setIcon] = useState('💼');
  const [color, setColor] = useState(COLORS[0]);
  const [type, setType] = useState('work');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) return;

    setSaving(true);
    try {
      await onSave({ calendarId, name: name.trim(), icon, color, type });

      // Reset form
      setName('');
      setIcon('💼');
      setColor(COLORS[0]);
      setType('work');

      onClose();
    } catch (error) {
      console.error('Error creating category:', error);
      alert('Erro ao criar categoria. Tente novamente.');
    } finally {
      setSaving(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div className="bg-gradient-to-br from-[#350545] via-[#4a0860] to-[#792990] rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto border border-purple-600/20">
        {/* Header */}
        <div className="sticky top-0 bg-gradient-to-r from-[#350545] to-[#792990] px-6 py-4 border-b border-white/10">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-bold text-white">Nova Categoria</h2>
            <button
              onClick={onClose}
              className="text-white/70 hover:text-white transition-colors"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {/* Calendar Selection */}
          <div>
            <label className="block text-white font-semibold mb-2">Calendário</label>
            <select
              value={calendarId}
              onChange={(e) => setCalendarId(e.target.value)}
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              {calendars.map((calendar) => (
                <option key={calendar.id} value={calendar.id} className="bg-[#350545]">
                  {calendar.name}
                </option>
              ))}
            </select>
          </div>

          {/* Category Name */}
          <div>
            <label className="block text-white font-semibold mb-2">Nome da Categoria</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ex: Programação, Academia, Leitura..."
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-purple-500"
              required
            />
          </div>

          {/* Icon Selection */}
          <div>
            <label className="block text-white font-semibold mb-2">Ícone</label>
            <div className="grid grid-cols-10 gap-2 max-h-48 overflow-y-auto p-2 bg-white/5 rounded-lg border border-white/10">
              {EMOJI_LIST.map((emoji) => (
                <button
                  key={emoji}
                  type="button"
                  onClick={() => setIcon(emoji)}
                  className={`text-2xl p-2 rounded-lg transition-all hover:scale-110 ${
                    icon === emoji
                      ? 'bg-white/30 ring-2 ring-white/50'
                      : 'bg-white/10 hover:bg-white/20'
                  }`}
                >
                  {emoji}
                </button>
              ))}
            </div>
          </div>

          {/* Color Selection */}
          <div>
            <label className="block text-white font-semibold mb-2">Cor</label>
            <div className="grid grid-cols-10 gap-2">
              {COLORS.map((colorOption) => (
                <button
                  key={colorOption}
                  type="button"
                  onClick={() => setColor(colorOption)}
                  className={`w-10 h-10 rounded-lg transition-all hover:scale-110 ${
                    color === colorOption
                      ? 'ring-2 ring-white ring-offset-2 ring-offset-purple-900'
                      : ''
                  }`}
                  style={{ backgroundColor: colorOption }}
                />
              ))}
            </div>
          </div>

          {/* Type Selection */}
          <div>
            <label className="block text-white font-semibold mb-2">Tipo</label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              {CATEGORY_TYPES.map((typeOption) => (
                <option key={typeOption.value} value={typeOption.value} className="bg-[#350545]">
                  {typeOption.label}
                </option>
              ))}
            </select>
          </div>

          {/* Actions */}
          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-6 py-3 bg-white/10 hover:bg-white/20 text-white rounded-lg font-semibold transition-all"
              disabled={saving}
            >
              Cancelar
            </button>
            <button
              type="submit"
              className="flex-1 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-lg font-semibold transition-all shadow-lg disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={saving || !name.trim()}
            >
              {saving ? 'Salvando...' : 'Criar Categoria'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

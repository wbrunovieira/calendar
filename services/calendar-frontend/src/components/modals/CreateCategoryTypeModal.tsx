'use client';

import { useState, useEffect } from 'react';
import { CategoryType, Category } from '@/types/calendar';

interface CreateCategoryTypeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (type: {
    calendarId: string;
    name: string;
    value: string;
    color: string;
    icon?: string;
  }) => Promise<void>;
  calendars: { id: string; name: string; email: string; color: string }[];
  categories?: Category[];
  initialData?: CategoryType;
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
  '💡', '🔦', '🕯️', '🔌', '🔋', '🪔', '🧯', '🛠️', '🔧', '⚙️',
];

const COLORS = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E2', '#F8B195', '#C06C84',
  '#6C5B7B', '#355C7D', '#2C3E50', '#E74C3C', '#3498DB',
  '#2ECC71', '#F39C12', '#9B59B6', '#1ABC9C', '#E67E22',
];

export default function CreateCategoryTypeModal({ isOpen, onClose, onSave, calendars, categories = [], initialData }: CreateCategoryTypeModalProps) {
  const [calendarId, setCalendarId] = useState(calendars[0]?.id || '');
  const [selectedCategoryId, setSelectedCategoryId] = useState('');
  const [name, setName] = useState('');
  const [value, setValue] = useState('');
  const [icon, setIcon] = useState('💼');
  const [color, setColor] = useState(COLORS[0]);
  const [saving, setSaving] = useState(false);

  // Filter categories by selected calendar
  const filteredCategories = categories.filter(cat => cat.calendarId === calendarId);

  // Populate form with initialData when editing
  useEffect(() => {
    if (initialData) {
      setCalendarId(initialData.calendarId);
      setName(initialData.name);
      setValue(initialData.value);
      setIcon(initialData.icon || '💼');
      setColor(initialData.color);
    } else {
      // Reset form when not editing
      setCalendarId(calendars[0]?.id || '');
      setSelectedCategoryId('');
      setName('');
      setValue('');
      setIcon('💼');
      setColor(COLORS[0]);
    }
  }, [initialData, calendars]);

  // Reset selected category when calendar changes
  useEffect(() => {
    setSelectedCategoryId('');
  }, [calendarId]);

  // Auto-generate value from name
  const handleNameChange = (newName: string) => {
    setName(newName);
    // Generate slug: lowercase, replace spaces with hyphens, remove special chars
    const slug = newName
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '') // Remove accents
      .replace(/[^a-z0-9\s-]/g, '') // Keep only letters, numbers, spaces, hyphens
      .trim()
      .replace(/\s+/g, '-'); // Replace spaces with hyphens
    setValue(slug);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim() || !value.trim()) return;

    setSaving(true);
    try {
      await onSave({ calendarId, name: name.trim(), value: value.trim(), color, icon });

      // Reset form
      setCalendarId(calendars[0]?.id || '');
      setSelectedCategoryId('');
      setName('');
      setValue('');
      setIcon('💼');
      setColor(COLORS[0]);

      onClose();
    } catch (error) {
      console.error('Error creating category type:', error);
      alert('Erro ao criar tipo. Tente novamente.');
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
            <h2 className="text-2xl font-bold text-white">
              {initialData ? 'Editar Tipo' : 'Novo Tipo'}
            </h2>
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

          {/* Category Selection (Optional) */}
          {filteredCategories.length > 0 && (
            <div>
              <label className="block text-white font-semibold mb-2">
                Categoria
                <span className="text-white/60 text-sm font-normal ml-2">
                  - opcional, para sugerir nome e ícone
                </span>
              </label>
              <select
                value={selectedCategoryId}
                onChange={(e) => {
                  const categoryId = e.target.value;
                  setSelectedCategoryId(categoryId);

                  // Auto-populate name and icon from selected category
                  if (categoryId) {
                    const category = filteredCategories.find(c => c.id === categoryId);
                    if (category) {
                      handleNameChange(category.name);
                      if (category.icon) {
                        setIcon(category.icon);
                      }
                      if (category.color) {
                        setColor(category.color);
                      }
                    }
                  }
                }}
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              >
                <option value="" className="bg-[#350545]">
                  Selecione uma categoria (ou deixe vazio)
                </option>
                {filteredCategories.map((category) => (
                  <option key={category.id} value={category.id} className="bg-[#350545]">
                    {category.icon} {category.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Type Name */}
          <div>
            <label className="block text-white font-semibold mb-2">Nome do Tipo</label>
            <input
              type="text"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="Ex: Trabalho, Saúde, Lazer..."
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-purple-500"
              required
            />
          </div>

          {/* Type Value (Slug) */}
          <div>
            <label className="block text-white font-semibold mb-2">
              Valor (slug)
              <span className="text-white/60 text-sm font-normal ml-2">
                - usado internamente
              </span>
            </label>
            <input
              type="text"
              value={value}
              onChange={(e) => setValue(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
              placeholder="Ex: trabalho, saude, lazer"
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono"
              required
            />
            <p className="text-white/50 text-xs mt-1">
              Apenas letras minúsculas, números e hífens
            </p>
          </div>

          {/* Icon Selection */}
          <div>
            <label className="block text-white font-semibold mb-2">Ícone</label>

            {/* Custom emoji input */}
            <div className="mb-3">
              <input
                type="text"
                value={icon}
                onChange={(e) => setIcon(e.target.value)}
                placeholder="Cole ou digite seu emoji aqui... (ex: ⚕️, 🏥, 💊)"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-purple-500 text-2xl text-center"
                maxLength={10}
              />
              <p className="text-white/50 text-xs mt-1 text-center">
                Ou escolha um da lista abaixo
              </p>
            </div>

            {/* Emoji grid */}
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
              disabled={saving || !name.trim() || !value.trim()}
            >
              {saving ? 'Salvando...' : (initialData ? 'Salvar Alterações' : 'Criar Tipo')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

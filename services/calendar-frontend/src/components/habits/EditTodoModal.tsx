'use client';

import { useState, useEffect, useMemo } from 'react';
import { api } from '@/lib/api';
import type { Calendar, Category, CategoryType, Event } from '@/types/calendar';
import LabelSelector from '@/components/labels/LabelSelector';

interface EditTodoModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdated: () => void;
  todo: Event | null;
  calendars: Calendar[];
}

export default function EditTodoModal({
  isOpen,
  onClose,
  onUpdated,
  todo,
  calendars,
}: EditTodoModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [calendarId, setCalendarId] = useState('');
  const [priority, setPriority] = useState<number | undefined>(undefined);
  const [dueDate, setDueDate] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [categoryTypeId, setCategoryTypeId] = useState('');
  const [labelId, setLabelId] = useState<string | undefined>(undefined);
  const [categories, setCategories] = useState<Category[]>([]);
  const [categoryTypes, setCategoryTypes] = useState<CategoryType[]>([]);
  const [loading, setLoading] = useState(false);

  // Load todo data when modal opens
  useEffect(() => {
    if (isOpen && todo) {
      setTitle(todo.title || '');
      setDescription(todo.description || '');
      setCalendarId(todo.calendarId || '');
      setPriority(todo.priority || undefined);
      setDueDate(todo.dueDate ? todo.dueDate.split('T')[0] : '');
      setCategoryId(todo.categoryId || '');
      setCategoryTypeId(todo.categoryTypeId || '');
      setLabelId(todo.labelId || undefined);
    }
  }, [isOpen, todo]);

  // Fetch categories and category types when calendar changes
  useEffect(() => {
    if (calendarId) {
      Promise.all([
        api.categories.list(calendarId),
        api.categoryTypes.list(calendarId),
      ])
        .then(([cats, types]) => {
          setCategories(cats);
          setCategoryTypes(types);
        })
        .catch(console.error);
    } else {
      setCategories([]);
      setCategoryTypes([]);
    }
  }, [calendarId]);

  // Filter categories based on selected categoryType
  const filteredCategories = useMemo(() => {
    if (!categoryTypeId) return categories;
    return categories.filter(cat =>
      cat.categoryTypes?.some(ct => ct.id === categoryTypeId)
    );
  }, [categories, categoryTypeId]);

  const canSave = title.trim() !== '' && calendarId !== '';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSave || !todo) return;

    setLoading(true);
    try {
      await api.events.update(todo.id, {
        calendarId,
        title: title.trim(),
        description: description.trim() || undefined,
        priority: priority || null,
        dueDate: dueDate || null,
        categoryId: categoryId || null,
        categoryTypeId: categoryTypeId || null,
        labelId: labelId || null,
      });

      onUpdated();
      onClose();
    } catch (error) {
      console.error('Error updating todo:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen || !todo) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-[#2a0a3a] rounded-2xl p-6 w-full max-w-md border border-white/20 shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold text-white mb-6">Editar Tarefa</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Calendar Selector */}
          <div>
            <label htmlFor="calendar" className="block text-sm font-medium text-white/70 mb-1">
              Perfil
            </label>
            <select
              id="calendar"
              value={calendarId}
              onChange={(e) => setCalendarId(e.target.value)}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              <option value="">Selecione um perfil</option>
              {calendars.map((cal) => (
                <option key={cal.id} value={cal.id}>
                  {cal.name}
                </option>
              ))}
            </select>
          </div>

          {/* Title */}
          <div>
            <label htmlFor="title" className="block text-sm font-medium text-white/70 mb-1">
              Titulo
            </label>
            <input
              type="text"
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Ex: Criar post LinkedIn"
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
          </div>

          {/* Description */}
          <div>
            <label htmlFor="description" className="block text-sm font-medium text-white/70 mb-1">
              Descricao (opcional)
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500 resize-none"
            />
          </div>

          {/* Priority */}
          <div>
            <label htmlFor="priority" className="block text-sm font-medium text-white/70 mb-1">
              Prioridade
            </label>
            <select
              id="priority"
              value={priority || ''}
              onChange={(e) => setPriority(e.target.value ? Number(e.target.value) : undefined)}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              <option value="">Sem prioridade</option>
              <option value="1">Alta</option>
              <option value="2">Media</option>
              <option value="3">Baixa</option>
            </select>
          </div>

          {/* Due Date */}
          <div>
            <label htmlFor="dueDate" className="block text-sm font-medium text-white/70 mb-1">
              Data de Vencimento
            </label>
            <input
              type="date"
              id="dueDate"
              value={dueDate}
              onChange={(e) => setDueDate(e.target.value)}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
          </div>

          {/* Category Type */}
          {categoryTypes.length > 0 && (
            <div>
              <label htmlFor="categoryType" className="block text-sm font-medium text-white/70 mb-1">
                Tipo
              </label>
              <select
                id="categoryType"
                value={categoryTypeId}
                onChange={(e) => {
                  setCategoryTypeId(e.target.value);
                  setCategoryId(''); // Reset category when type changes
                }}
                className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              >
                <option value="">Selecione um tipo</option>
                {categoryTypes.map((type) => (
                  <option key={type.id} value={type.id}>
                    {type.icon} {type.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Category - only shown when type is selected */}
          {categoryTypeId && filteredCategories.length > 0 && (
            <div>
              <label htmlFor="category" className="block text-sm font-medium text-white/70 mb-1">
                Categoria
              </label>
              <select
                id="category"
                value={categoryId}
                onChange={(e) => setCategoryId(e.target.value)}
                className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              >
                <option value="">Selecione uma categoria</option>
                {filteredCategories.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.icon} {cat.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Label Selector */}
          {calendarId && (
            <LabelSelector
              calendarId={calendarId}
              selectedLabelId={labelId}
              onSelect={setLabelId}
            />
          )}

          {/* Buttons */}
          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={!canSave || loading}
              className="flex-1 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Salvando...' : 'Salvar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

/**
 * Custom hook for edit event form management
 * Handles state, validation, and submission logic for editing events
 */

import { useState, useEffect, useMemo } from 'react';
import { api } from '@/lib/api';
import { Event, Category, CategoryType } from '@/types/calendar';
import { calculateEndTime } from '@/utils/calendar';
import { buildEventPayload, validateEventForm } from '@/utils/eventHelpers';
import { RecurringEventAction } from '@/components/modals/RecurringEventActionModal';

interface UseEditEventFormProps {
  isOpen: boolean;
  event: Event | null;
  calendars: Array<{ id: string; name: string; color: string; type: string }>;
  categories: Category[];
  categoryTypes: CategoryType[];
  onEventUpdated: () => void;
  onClose: () => void;
}

export function useEditEventForm({
  isOpen,
  event,
  calendars,
  categories,
  categoryTypes,
  onEventUpdated,
  onClose,
}: UseEditEventFormProps) {
  const [formData, setFormData] = useState({
    calendarId: '',
    categoryId: '',
    categoryTypeId: '',
    title: '',
    description: '',
    startTime: '',
    endTime: '',
    startDate: '',
    endDate: '',
    isRecurring: false,
    recurrenceFrequency: 'weekly' as 'daily' | 'weekly' | 'monthly' | 'yearly',
    recurrenceInterval: 1,
    recurrenceDaysOfWeek: [] as number[],
    recurrenceEndDate: '',
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showRecurringActionModal, setShowRecurringActionModal] = useState(false);
  const [recurringAction, setRecurringAction] = useState<RecurringEventAction | null>(null);

  // Load event data when modal opens
  useEffect(() => {
    if (isOpen && event) {
      setFormData({
        calendarId: event.calendarId || '',
        categoryId: event.categoryId || '',
        categoryTypeId: event.categoryTypeId || '',
        title: event.title || '',
        description: event.description || '',
        startTime: event.startTime || '',
        endTime: event.endTime || '',
        startDate: event.startDate?.split('T')[0] || '',
        endDate: event.endDate?.split('T')[0] || '',
        isRecurring: event.isRecurring || false,
        recurrenceFrequency: event.recurrenceFrequency || 'weekly',
        recurrenceInterval: event.recurrenceInterval || 1,
        recurrenceDaysOfWeek: event.recurrenceDaysOfWeek || [],
        recurrenceEndDate: event.recurrenceEndDate?.split('T')[0] || '',
      });
      setError('');

      // If event is recurring, show the action modal first
      if (event.isRecurring) {
        setShowRecurringActionModal(true);
        setRecurringAction(null);
      } else {
        setShowRecurringActionModal(false);
      }
    }
  }, [isOpen, event]);

  // Filtered categories based on selected calendar
  const filteredCategories = useMemo(
    () => (formData.calendarId ? categories.filter(c => c.calendarId === formData.calendarId) : []),
    [formData.calendarId, categories]
  );

  // Filtered category types based on selected calendar
  const filteredCategoryTypes = useMemo(
    () => (formData.calendarId ? categoryTypes.filter(ct => ct.calendarId === formData.calendarId) : []),
    [formData.calendarId, categoryTypes]
  );

  // Get available types for the selected category
  const availableTypes = useMemo(() => {
    if (!formData.categoryId) return [];

    const selectedCategory = categories.find(c => c.id === formData.categoryId);
    if (!selectedCategory || !selectedCategory.categoryTypes || selectedCategory.categoryTypes.length === 0) {
      return [];
    }

    return selectedCategory.categoryTypes;
  }, [formData.categoryId, categories]);

  // Memoized options for selects
  const calendarOptions = useMemo(() => calendars.map(cal => ({ value: cal.id, label: cal.name })), [calendars]);

  const categoryOptions = useMemo(
    () => filteredCategories.map(cat => ({ value: cat.id, label: `${cat.icon} ${cat.name}` })),
    [filteredCategories]
  );

  const categoryTypeOptions = useMemo(
    () => filteredCategoryTypes.map(type => ({
      value: type.id,
      label: `${type.icon ? type.icon + ' ' : ''}${type.name}`
    })),
    [filteredCategoryTypes]
  );

  // Form field handlers
  const handleCalendarChange = (value: string) => {
    setFormData({ ...formData, calendarId: value, categoryId: '', categoryTypeId: '' });
  };

  const handleCategoryTypeChange = (typeId: string) => {
    // Se um tipo foi selecionado, buscar a categoria que possui esse tipo
    const categoryWithType = filteredCategories.find(cat =>
      cat.categoryTypes && cat.categoryTypes.some((ct: CategoryType) => ct.id === typeId)
    );

    setFormData({
      ...formData,
      categoryTypeId: typeId,
      categoryId: categoryWithType ? categoryWithType.id : ''
    });
  };

  const handleCategoryChange = (categoryId: string) => {
    // Se uma categoria foi selecionada, limpar o tipo (será escolhido depois)
    setFormData({
      ...formData,
      categoryId,
      categoryTypeId: ''
    });
  };

  const handleStartTimeChange = (time: string) => {
    setFormData({ ...formData, startTime: time, endTime: calculateEndTime(time) });
  };

  const toggleDayOfWeek = (day: number) => {
    setFormData(prev => ({
      ...prev,
      recurrenceDaysOfWeek: prev.recurrenceDaysOfWeek.includes(day)
        ? prev.recurrenceDaysOfWeek.filter(d => d !== day)
        : [...prev.recurrenceDaysOfWeek, day].sort(),
    }));
  };

  // Recurring action handlers
  const handleRecurringActionSelect = (action: RecurringEventAction) => {
    setRecurringAction(action);
    setShowRecurringActionModal(false);
  };

  const handleRecurringActionClose = () => {
    setShowRecurringActionModal(false);
    onClose(); // Close the entire edit modal if user cancels recurring action
  };

  // Extract event ID (handle expanded recurring event IDs)
  const extractEventId = (eventId: string): string => {
    // Check if this looks like an expanded recurring event ID (has date pattern at the end)
    const datePattern = /-\d{4}-\d{2}-\d{2}$/;
    if (datePattern.test(eventId)) {
      // Extract the UUID part before the date
      return eventId.replace(datePattern, '');
    }
    return eventId;
  };

  // Submit handler
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!event) return;

    const validationError = validateEventForm(formData);
    if (validationError) {
      setError(validationError);
      return;
    }

    const payload = buildEventPayload(formData);

    // Add recurring action scope if this is a recurring event
    if (event.isRecurring && recurringAction) {
      payload.recurringEditScope = recurringAction;
    }

    try {
      setLoading(true);
      const eventId = extractEventId(event.id);
      await api.events.update(eventId, payload);
      onEventUpdated();
      onClose();
    } catch {
      setError('Erro ao atualizar evento. Tente novamente.');
    } finally {
      setLoading(false);
    }
  };

  return {
    formData,
    setFormData,
    loading,
    error,
    showRecurringActionModal,
    recurringAction,
    calendarOptions,
    categoryOptions,
    categoryTypeOptions,
    availableTypes,
    handleCalendarChange,
    handleCategoryChange,
    handleCategoryTypeChange,
    handleStartTimeChange,
    toggleDayOfWeek,
    handleSubmit,
    handleRecurringActionSelect,
    handleRecurringActionClose,
  };
}

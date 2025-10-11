/**
 * Custom hook for event form management
 * Handles state, validation, and submission logic
 */

import { useState, useEffect, useMemo } from 'react';
import { api } from '@/lib/api';
import { Category } from '@/types/calendar';
import { calculateEndTime, incrementTime } from '@/utils/calendar';
import { buildEventPayload, validateEventForm } from '@/utils/eventHelpers';

interface UseEventFormProps {
  isOpen: boolean;
  initialDate?: string;
  initialTime?: string;
  preservedFormData?: Record<string, unknown>;
  calendars: Array<{ id: string; name: string; color: string; type: string }>;
  categories: Category[];
  onEventCreated: (preservedData?: Record<string, unknown>) => void;
  onClose: () => void;
}

export function useEventForm({
  isOpen,
  initialDate,
  initialTime,
  preservedFormData,
  calendars,
  categories,
  onEventCreated,
  onClose,
}: UseEventFormProps) {
  const [formData, setFormData] = useState({
    calendarId: '',
    categoryId: '',
    title: '',
    description: '',
    startTime: initialTime || '',
    endTime: '',
    startDate: initialDate || '',
    endDate: '',
    isRecurring: false,
    recurrenceFrequency: 'weekly' as 'daily' | 'weekly' | 'monthly' | 'yearly',
    recurrenceInterval: 1,
    recurrenceDaysOfWeek: [] as number[],
    recurrenceEndDate: '',
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [createAnother, setCreateAnother] = useState(false);

  // Update form data when modal opens
  useEffect(() => {
    if (isOpen) {
      if (preservedFormData) {
        setFormData(preservedFormData as typeof formData);
      } else if (initialDate || initialTime) {
        setFormData(prev => ({
          ...prev,
          startDate: initialDate || prev.startDate,
          startTime: initialTime || prev.startTime,
          endTime: initialTime ? calculateEndTime(initialTime) : prev.endTime,
        }));
      }
    }
  }, [isOpen, initialDate, initialTime, preservedFormData]);

  // Filtered categories based on selected calendar
  const filteredCategories = useMemo(
    () => (formData.calendarId ? categories.filter(c => c.calendarId === formData.calendarId) : []),
    [formData.calendarId, categories]
  );

  // Memoized options for selects
  const calendarOptions = useMemo(() => calendars.map(cal => ({ value: cal.id, label: cal.name })), [calendars]);

  const categoryOptions = useMemo(
    () => filteredCategories.map(cat => ({ value: cat.id, label: `${cat.icon} ${cat.name}` })),
    [filteredCategories]
  );

  // Form field handlers
  const handleCalendarChange = (value: string) => {
    setFormData({ ...formData, calendarId: value, categoryId: '' });
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

  // Submit handler
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const validationError = validateEventForm(formData);
    if (validationError) {
      setError(validationError);
      return;
    }

    const payload = buildEventPayload(formData);

    try {
      setLoading(true);
      await api.events.create(payload);

      // Keep form data for next similar event, but clear title and description
      const updatedFormData = {
        ...formData,
        title: '',
        description: '',
        startTime: createAnother ? incrementTime(formData.startTime) : formData.startTime,
        endTime: createAnother && formData.endTime ? incrementTime(formData.endTime) : formData.endTime,
      };

      // Pass preserved data to parent if "create another" is checked
      if (createAnother) {
        onEventCreated(updatedFormData);
      } else {
        onEventCreated();
        onClose();
      }
    } catch {
      setError('Erro ao criar evento. Tente novamente.');
    } finally {
      setLoading(false);
    }
  };

  return {
    formData,
    setFormData,
    loading,
    error,
    createAnother,
    setCreateAnother,
    calendarOptions,
    categoryOptions,
    handleCalendarChange,
    handleStartTimeChange,
    toggleDayOfWeek,
    handleSubmit,
  };
}

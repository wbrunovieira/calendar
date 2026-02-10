import { describe, it, expect } from 'vitest';
import { buildEventPayload, validateEventForm } from './eventHelpers';

describe('buildEventPayload', () => {
  const baseFormData = {
    calendarId: 'cal-1',
    categoryId: '',
    categoryTypeId: '',
    title: 'Test Event',
    description: '',
    startTime: '10:00',
    endTime: '11:00',
    startDate: '2024-11-16',
    endDate: '',
    isRecurring: false,
    recurrenceFrequency: 'weekly' as const,
    recurrenceInterval: 1,
    recurrenceDaysOfWeek: [] as number[],
    recurrenceEndDate: '',
  };

  it('should include required fields', () => {
    const payload = buildEventPayload(baseFormData);
    expect(payload.calendarId).toBe('cal-1');
    expect(payload.title).toBe('Test Event');
    expect(payload.startTime).toBe('10:00');
    expect(payload.startDate).toBe('2024-11-16');
    expect(payload.isRecurring).toBe(false);
  });

  it('should include optional fields when present', () => {
    const payload = buildEventPayload({
      ...baseFormData,
      categoryId: 'cat-1',
      description: 'A description',
      endTime: '11:00',
      endDate: '2024-11-17',
    });
    expect(payload.categoryId).toBe('cat-1');
    expect(payload.description).toBe('A description');
    expect(payload.endTime).toBe('11:00');
    expect(payload.endDate).toBe('2024-11-17');
  });

  it('should not include empty optional fields', () => {
    const payload = buildEventPayload(baseFormData);
    expect(payload).not.toHaveProperty('categoryId');
    expect(payload).not.toHaveProperty('description');
    expect(payload).not.toHaveProperty('endDate');
  });

  it('should include reminders when present', () => {
    const payload = buildEventPayload({
      ...baseFormData,
      reminders: [
        { minutesBefore: 30, method: 'notification' },
        { minutesBefore: 60, method: 'notification' },
      ],
    });
    expect(payload.reminders).toEqual([
      { minutesBefore: 30, method: 'notification' },
      { minutesBefore: 60, method: 'notification' },
    ]);
  });

  it('should not include reminders when empty array', () => {
    const payload = buildEventPayload({
      ...baseFormData,
      reminders: [],
    });
    expect(payload).not.toHaveProperty('reminders');
  });

  it('should not include reminders when undefined', () => {
    const payload = buildEventPayload(baseFormData);
    expect(payload).not.toHaveProperty('reminders');
  });

  it('should include single reminder with 0 minutes', () => {
    const payload = buildEventPayload({
      ...baseFormData,
      reminders: [{ minutesBefore: 0, method: 'notification' }],
    });
    expect(payload.reminders).toEqual([{ minutesBefore: 0, method: 'notification' }]);
  });

  it('should include recurrence fields when recurring', () => {
    const payload = buildEventPayload({
      ...baseFormData,
      isRecurring: true,
      recurrenceFrequency: 'weekly',
      recurrenceInterval: 1,
      recurrenceDaysOfWeek: [1, 3, 5],
    });
    expect(payload.recurrenceFrequency).toBe('weekly');
    expect(payload.recurrenceInterval).toBe(1);
    expect(payload.recurrenceDaysOfWeek).toEqual([1, 3, 5]);
  });
});

describe('validateEventForm', () => {
  const validForm = {
    calendarId: 'cal-1',
    categoryId: '',
    categoryTypeId: '',
    title: 'Test',
    description: '',
    startTime: '10:00',
    endTime: '',
    startDate: '2024-11-16',
    endDate: '',
    isRecurring: false,
    recurrenceFrequency: 'weekly' as const,
    recurrenceInterval: 1,
    recurrenceDaysOfWeek: [] as number[],
    recurrenceEndDate: '',
  };

  it('should return null for valid form', () => {
    expect(validateEventForm(validForm)).toBeNull();
  });

  it('should return error when calendarId is missing', () => {
    expect(validateEventForm({ ...validForm, calendarId: '' })).toBe('Selecione um calendário');
  });

  it('should return error when title is missing', () => {
    expect(validateEventForm({ ...validForm, title: '' })).toBe('Digite um título');
  });

  it('should return error when startTime is missing', () => {
    expect(validateEventForm({ ...validForm, startTime: '' })).toBe('Selecione a hora de início');
  });

  it('should return error when startDate is missing', () => {
    expect(validateEventForm({ ...validForm, startDate: '' })).toBe('Selecione a data de início');
  });
});

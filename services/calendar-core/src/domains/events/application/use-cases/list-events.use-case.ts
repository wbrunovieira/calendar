import { Injectable } from '@nestjs/common';
import { Event } from '../../domain/entities/event.entity';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

/**
 * Formats a Date object to YYYY-MM-DD string in local timezone
 * This avoids UTC conversion issues that can shift dates by a day
 */
function formatLocalDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export interface ListEventsFilters {
  calendarId?: string;
  categoryId?: string;
  search?: string;
  startDate?: string; // Date range start (YYYY-MM-DD)
  endDate?: string; // Date range end (YYYY-MM-DD)
  eventType?: string; // 'EVENT' | 'HABIT' | 'TODO'
}

@Injectable()
export class ListEventsUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(filters?: ListEventsFilters): Promise<any[]> {
    const events = await this.eventRepository.findAll(filters);

    // Se não tem date range, retorna eventos como estão (backward compatibility)
    if (!filters?.startDate || !filters?.endDate) {
      return events.map((event) => ({
        ...event,
        executions: event.executions || [],
      }));
    }

    // Garantir que datas sejam interpretadas no timezone local, não UTC
    // Parse date string as local date components
    const [startYear, startMonth, startDay] = filters.startDate
      .split('-')
      .map(Number);
    const rangeStart = new Date(
      startYear,
      startMonth - 1,
      startDay,
      0,
      0,
      0,
      0,
    );

    const [endYear, endMonth, endDay] = filters.endDate.split('-').map(Number);
    const rangeEnd = new Date(endYear, endMonth - 1, endDay, 23, 59, 59, 999);

    const occurrences: any[] = [];

    for (const event of events) {
      // Evento one-time normal
      if (!event.recurrenceRule) {
        const eventDate = new Date(event.startDate);
        eventDate.setHours(0, 0, 0, 0); // Force local timezone
        if (eventDate >= rangeStart && eventDate <= rangeEnd) {
          // Include executions for non-recurring events
          occurrences.push({
            ...event,
            executions: event.executions || [],
          });
        }
        continue;
      }

      // Evento recorrente: expandir usando RRULE
      const expandedOccurrences = this.expandRecurringEvent(
        event,
        rangeStart,
        rangeEnd,
      );
      occurrences.push(...expandedOccurrences);
    }

    return occurrences;
  }

  private expandRecurringEvent(
    event: any,
    rangeStart: Date,
    rangeEnd: Date,
  ): any[] {
    const occurrences: any[] = [];

    // Gerar todas as datas usando RRULE
    const dates = RRuleHelper.generateOccurrences(
      event.recurrenceRule,
      new Date(event.startDate),
      rangeStart,
      rangeEnd,
    );

    // Criar set de exceções
    const exceptions = new Set<string>();
    if (event.exceptions) {
      for (const exception of event.exceptions) {
        const exceptionDate = new Date(exception.exceptionDate);
        exceptions.add(formatLocalDate(exceptionDate));
      }
    }

    // Criar map de overrides
    const overridesMap = new Map<string, any>();
    if (event.overrides) {
      for (const override of event.overrides) {
        const overrideDate = new Date(override.occurrenceDate);
        const dateStr = formatLocalDate(overrideDate);
        overridesMap.set(dateStr, override.overrideEvent);
      }
    }

    // Para cada data gerada pela RRULE
    for (const date of dates) {
      const dateStr = formatLocalDate(date);

      // Se está nas exceções, pula
      if (exceptions.has(dateStr)) {
        continue;
      }

      // Se tem override, usa o evento override
      if (overridesMap.has(dateStr)) {
        const overrideEvent = overridesMap.get(dateStr);
        occurrences.push({
          ...overrideEvent,
          originalEventId: event.id,
          occurrenceDate: dateStr,
        });
        continue;
      }

      // Usa o evento master para esta ocorrência
      // Converter RRULE para formato legado para o frontend
      const legacy = this.convertRRuleToLegacy(event.recurrenceRule);

      occurrences.push({
        id: `${event.id}-${dateStr}`, // ID único para esta ocorrência
        calendarId: event.calendarId,
        categoryId: event.categoryId,
        categoryTypeId: event.categoryTypeId,
        category: event.category, // Incluir categoria completa
        categoryType: event.categoryType, // Incluir tipo de categoria completa
        title: event.title,
        description: event.description,
        startTime: event.startTime,
        endTime: event.endTime,
        startDate: dateStr, // Retornar como string YYYY-MM-DD, não Date object
        endDate: event.endDate,
        ...legacy, // Campos do formato legado (isRecurring, recurrenceFrequency, etc.)
        eventType: event.eventType || 'EVENT',
        priority: event.priority,
        dueDate: event.dueDate,
        // Flexible habit fields
        recurrenceType: event.recurrenceType || 'FIXED',
        weeklyTargetCount: event.weeklyTargetCount || null,
        weeklyPreferredDays: event.weeklyPreferredDays || null,
        googleEventId: event.googleEventId,
        isActive: event.isActive,
        createdAt: event.createdAt,
        updatedAt: event.updatedAt,
        originalEventId: event.id, // Referência ao evento master
        occurrenceDate: dateStr, // Data específica desta ocorrência
        executions: (() => {
          return (
            event.executions
              ?.filter((c) => {
                // Handle executionDate which can be a Date object or string
                // If it's a Date, Prisma returns it as UTC midnight, so we need to extract the date parts carefully
                let compDateStr: string;
                if (typeof c.executionDate === 'string') {
                  // If it's already a string, extract the date part
                  compDateStr = c.executionDate.split('T')[0];
                } else if (c.executionDate instanceof Date) {
                  // If it's a Date object, use UTC date to avoid timezone issues
                  // Because Prisma stores DATE type as UTC midnight
                  const d = c.executionDate;
                  compDateStr = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`;
                } else {
                  compDateStr = '';
                }
                const matches = compDateStr === dateStr;
                return matches;
              })
              .map((c) => ({
                id: c.id,
                eventId: c.eventId,
                executionDate: c.executionDate,
                completed: c.completed,
                completedAt: c.completedAt,
                notes: c.notes,
                createdAt: c.createdAt,
                updatedAt: c.updatedAt,
              })) || []
          );
        })(),
      });
    }

    return occurrences;
  }

  /**
   * Converte RRULE para formato legado (compatibilidade frontend)
   */
  private convertRRuleToLegacy(recurrenceRule: string | null): any {
    if (!recurrenceRule) {
      return {
        isRecurring: false,
        recurrenceFrequency: null,
        recurrenceInterval: null,
        recurrenceDaysOfWeek: [],
        recurrenceDayOfMonth: null,
        recurrenceWeekOfMonth: null,
        recurrenceEndDate: null,
      };
    }

    const rule = RRuleHelper.parse(recurrenceRule);
    if (!rule) {
      return {
        isRecurring: false,
        recurrenceFrequency: null,
        recurrenceInterval: null,
        recurrenceDaysOfWeek: [],
        recurrenceDayOfMonth: null,
        recurrenceWeekOfMonth: null,
        recurrenceEndDate: null,
      };
    }

    return {
      isRecurring: true,
      recurrenceFrequency: rule.freq.toLowerCase(),
      recurrenceInterval: rule.interval || 1,
      recurrenceDaysOfWeek: rule.byweekday || [],
      recurrenceDayOfMonth: rule.bymonthday || null,
      recurrenceWeekOfMonth: null,
      recurrenceEndDate: rule.until || null,
    };
  }
}

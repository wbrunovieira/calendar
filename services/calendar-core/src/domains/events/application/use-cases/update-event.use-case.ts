import { Injectable, NotFoundException } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { UpdateEventDto } from '../../infrastructure/dtos/update-event.dto';
import { PrismaClient } from '@prisma/client';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

@Injectable()
export class UpdateEventUseCase {
  private prisma: PrismaClient;

  constructor(private readonly eventRepository: EventRepository) {
    this.prisma = new PrismaClient();
  }

  async execute(id: string, dto: UpdateEventDto) {
    const existingEvent = await this.eventRepository.findById(id);

    if (!existingEvent) {
      throw new NotFoundException(`Event with ID ${id} not found`);
    }

    // Verificar se é evento recorrente E tem escopo de edição
    if (existingEvent.recurrenceRule && dto.recurringEditScope) {
      return await this.handleRecurringEventEdit(id, existingEvent, dto);
    }

    // Evento regular (one-time) ou recorrente sem escopo específico
    const updateData: any = {};

    if (dto.calendarId !== undefined) updateData.calendarId = dto.calendarId;
    if (dto.categoryId !== undefined) updateData.categoryId = dto.categoryId;
    if (dto.categoryTypeId !== undefined) updateData.categoryTypeId = dto.categoryTypeId;
    if (dto.title !== undefined) updateData.title = dto.title;
    if (dto.description !== undefined) updateData.description = dto.description;
    if (dto.startTime !== undefined) updateData.startTime = dto.startTime;
    if (dto.endTime !== undefined) updateData.endTime = dto.endTime;
    if (dto.startDate !== undefined) {
      // Parse date string as local date, not UTC
      const [year, month, day] = dto.startDate.split('-').map(Number);
      updateData.startDate = new Date(year, month - 1, day);
    }
    if (dto.endDate !== undefined) {
      if (dto.endDate) {
        const [year, month, day] = dto.endDate.split('-').map(Number);
        updateData.endDate = new Date(year, month - 1, day);
      } else {
        updateData.endDate = null;
      }
    }

    // Atualizar RRULE se necessário
    if (dto.isRecurring && dto.recurrenceFrequency) {
      const freq = dto.recurrenceFrequency.toUpperCase() as 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY';
      updateData.recurrenceRule = RRuleHelper.toString({
        freq,
        interval: dto.recurrenceInterval || 1,
        byweekday: dto.recurrenceDaysOfWeek,
        bymonthday: dto.recurrenceDayOfMonth,
        until: dto.recurrenceEndDate ? new Date(dto.recurrenceEndDate) : undefined,
      });
    }

    const updatedEvent = await this.eventRepository.update(id, updateData);
    return updatedEvent;
  }

  private async handleRecurringEventEdit(id: string, existingEvent: any, dto: UpdateEventDto) {
    const scope = dto.recurringEditScope;

    console.log('[UpdateEventUseCase] handleRecurringEventEdit called', {
      id,
      scope,
      existingEventStartDate: existingEvent.startDate,
      existingRecurrenceRule: existingEvent.recurrenceRule,
      dtoStartDate: dto.startDate,
      dtoOccurrenceDate: dto.occurrenceDate,
    });

    if (scope === 'this') {
      // Google Calendar Pattern: "Apenas este evento"
      // 1. Criar exceção na data original
      // 2. Criar novo evento one-time com as modificações

      // Parse dates as local dates
      let originalOccurrenceDate: Date;
      if (dto.occurrenceDate) {
        const [y, m, d] = dto.occurrenceDate.split('-').map(Number);
        originalOccurrenceDate = new Date(y, m - 1, d);
      } else {
        originalOccurrenceDate = new Date(existingEvent.startDate);
      }
      originalOccurrenceDate.setHours(0, 0, 0, 0);

      // Criar exceção
      await this.prisma.recurrenceException.create({
        data: {
          eventId: id,
          exceptionDate: originalOccurrenceDate,
        },
      });

      // Criar novo evento one-time
      let newStartDate: Date;
      if (dto.startDate) {
        const [y, m, d] = dto.startDate.split('-').map(Number);
        newStartDate = new Date(y, m - 1, d);
      } else {
        newStartDate = existingEvent.startDate;
      }

      let newEndDate: Date | null;
      if (dto.endDate) {
        const [y, m, d] = dto.endDate.split('-').map(Number);
        newEndDate = new Date(y, m - 1, d);
      } else {
        newEndDate = existingEvent.endDate;
      }

      const newEvent = await this.prisma.event.create({
        data: {
          calendarId: dto.calendarId ?? existingEvent.calendarId,
          categoryId: dto.categoryId ?? existingEvent.categoryId,
          categoryTypeId: dto.categoryTypeId ?? existingEvent.categoryTypeId,
          title: dto.title ?? existingEvent.title,
          description: dto.description ?? existingEvent.description,
          startTime: dto.startTime ?? existingEvent.startTime,
          endTime: dto.endTime ?? existingEvent.endTime,
          startDate: newStartDate,
          endDate: newEndDate,
          recurrenceRule: null, // One-time event
          recurrenceMasterId: id, // Aponta para o evento master
          status: 'CONFIRMED',
        },
      });

      // Criar override
      await this.prisma.recurrenceOverride.create({
        data: {
          masterEventId: id,
          occurrenceDate: originalOccurrenceDate,
          overrideEventId: newEvent.id,
        },
      });

      return newEvent;
    }

    if (scope === 'future') {
      // Google Calendar Pattern: "Este e os próximos"
      // 1. Terminar série antiga (adicionar UNTIL)
      // 2. Criar nova série derivada a partir da nova data

      // Parse dates as local dates
      let originalOccurrenceDate: Date;
      if (dto.occurrenceDate) {
        const [y, m, d] = dto.occurrenceDate.split('-').map(Number);
        originalOccurrenceDate = new Date(y, m - 1, d);
      } else {
        originalOccurrenceDate = new Date(existingEvent.startDate);
      }
      originalOccurrenceDate.setHours(0, 0, 0, 0);

      // Calcular o dia antes da ocorrência para terminar a série antiga
      const newEndDate = new Date(originalOccurrenceDate);
      newEndDate.setDate(newEndDate.getDate() - 1);

      // Parse RRULE atual
      const currentRule = RRuleHelper.parse(existingEvent.recurrenceRule);
      if (!currentRule) {
        throw new Error('Invalid recurrence rule');
      }

      // Atualizar série antiga com UNTIL
      const updatedRule = RRuleHelper.toString({
        ...currentRule,
        until: newEndDate,
      });

      await this.eventRepository.update(id, {
        recurrenceRule: updatedRule,
      });

      // Criar nova série derivada
      let newStartDate: Date;
      if (dto.startDate) {
        const [y, m, d] = dto.startDate.split('-').map(Number);
        newStartDate = new Date(y, m - 1, d);
      } else {
        newStartDate = originalOccurrenceDate;
      }
      newStartDate.setHours(0, 0, 0, 0);

      // Ajustar dia da semana se for semanal e a data mudou
      let newRule = { ...currentRule };
      if (currentRule.freq === 'WEEKLY' && dto.startDate) {
        const newDayOfWeek = newStartDate.getDay();
        const originalDayOfWeek = originalOccurrenceDate.getDay();

        if (newDayOfWeek !== originalDayOfWeek && currentRule.byweekday) {
          // Substituir dia antigo pelo novo
          newRule.byweekday = currentRule.byweekday
            .filter(day => day !== originalDayOfWeek)
            .concat([newDayOfWeek])
            .sort();

          if (newRule.byweekday.length === 0) {
            newRule.byweekday = [newDayOfWeek];
          }
        }
      }

      let newSeriesEndDate: Date | null;
      if (dto.endDate) {
        const [y, m, d] = dto.endDate.split('-').map(Number);
        newSeriesEndDate = new Date(y, m - 1, d);
      } else {
        newSeriesEndDate = existingEvent.endDate;
      }

      const newEvent = await this.prisma.event.create({
        data: {
          calendarId: dto.calendarId ?? existingEvent.calendarId,
          categoryId: dto.categoryId ?? existingEvent.categoryId,
          categoryTypeId: dto.categoryTypeId ?? existingEvent.categoryTypeId,
          title: dto.title ?? existingEvent.title,
          description: dto.description ?? existingEvent.description,
          startTime: dto.startTime ?? existingEvent.startTime,
          endTime: dto.endTime ?? existingEvent.endTime,
          startDate: newStartDate,
          endDate: newSeriesEndDate,
          recurrenceRule: RRuleHelper.toString(newRule),
          recurrenceMasterId: id, // Referencia o master original
          status: 'CONFIRMED',
        },
      });

      return newEvent;
    }

    if (scope === 'all') {
      // Google Calendar Pattern: "Todos os eventos"
      // Atualizar o evento master e preservar exceptions/overrides

      const updateData: any = {};

      if (dto.calendarId !== undefined) updateData.calendarId = dto.calendarId;
      if (dto.categoryId !== undefined) updateData.categoryId = dto.categoryId;
      if (dto.categoryTypeId !== undefined) updateData.categoryTypeId = dto.categoryTypeId;
      if (dto.title !== undefined) updateData.title = dto.title;
      if (dto.description !== undefined) updateData.description = dto.description;
      if (dto.startTime !== undefined) updateData.startTime = dto.startTime;
      if (dto.endTime !== undefined) updateData.endTime = dto.endTime;

      // Se mudou a data, ajustar conforme o tipo de recorrência
      if (dto.startDate !== undefined) {
        // Parse date string as local date, not UTC
        const [year, month, day] = dto.startDate.split('-').map(Number);
        const newStartDate = new Date(year, month - 1, day);
        newStartDate.setHours(0, 0, 0, 0);

        console.log('[UpdateEventUseCase] Date parsing details', {
          inputString: dto.startDate,
          parsed: { year, month, day },
          dateObject: newStartDate,
          dateISOString: newStartDate.toISOString(),
          dateLocalString: newStartDate.toString(),
          dateGetDate: newStartDate.getDate(),
          dateGetMonth: newStartDate.getMonth(),
          dateGetFullYear: newStartDate.getFullYear(),
        });

        // Parse occurrence date as local date too
        let originalOccurrenceDate: Date;
        if (dto.occurrenceDate) {
          const [y, m, d] = dto.occurrenceDate.split('-').map(Number);
          originalOccurrenceDate = new Date(y, m - 1, d);
        } else {
          originalOccurrenceDate = new Date(existingEvent.startDate);
        }
        originalOccurrenceDate.setHours(0, 0, 0, 0);

        console.log('[UpdateEventUseCase] Processing startDate change', {
          dtoStartDate: dto.startDate,
          newStartDate,
          originalOccurrenceDate,
        });

        // Parse RRULE atual
        const currentRule = RRuleHelper.parse(existingEvent.recurrenceRule);

        console.log('[UpdateEventUseCase] Parsed RRULE', {
          currentRule,
          freq: currentRule?.freq,
        });

        if (currentRule) {
          if (currentRule.freq === 'DAILY') {
            // Para eventos DAILY, atualizar o startDate (isso move toda a série)
            console.log('[UpdateEventUseCase] DAILY event detected - updating startDate', {
              newStartDate,
            });
            updateData.startDate = newStartDate;
          } else if (currentRule.freq === 'WEEKLY') {
            // Para eventos WEEKLY, NÃO atualizar startDate (para preservar ocorrências anteriores)
            // Apenas atualizar o dia da semana no RRULE
            const newDayOfWeek = newStartDate.getDay();
            const originalDayOfWeek = originalOccurrenceDate.getDay();

            if (newDayOfWeek !== originalDayOfWeek && currentRule.byweekday) {
              // Substituir todos os dias antigos pelo novo dia
              const newRule = {
                ...currentRule,
                byweekday: currentRule.byweekday
                  .map(day => day === originalDayOfWeek ? newDayOfWeek : day)
                  .filter((day, index, arr) => arr.indexOf(day) === index) // Remove duplicatas
                  .sort(),
              };
              updateData.recurrenceRule = RRuleHelper.toString(newRule);
            }
          } else {
            // Para MONTHLY e YEARLY, atualizar startDate também
            updateData.startDate = newStartDate;
          }
        }
      }

      if (dto.endDate !== undefined) {
        if (dto.endDate) {
          const [y, m, d] = dto.endDate.split('-').map(Number);
          updateData.endDate = new Date(y, m - 1, d);
        } else {
          updateData.endDate = null;
        }
      }

      // Atualizar RRULE se passou novos parâmetros
      if (dto.recurrenceFrequency !== undefined) {
        const freq = dto.recurrenceFrequency.toUpperCase() as 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY';
        updateData.recurrenceRule = RRuleHelper.toString({
          freq,
          interval: dto.recurrenceInterval || 1,
          byweekday: dto.recurrenceDaysOfWeek,
          bymonthday: dto.recurrenceDayOfMonth,
          until: dto.recurrenceEndDate ? new Date(dto.recurrenceEndDate) : undefined,
        });
      }

      console.log('[UpdateEventUseCase] Final updateData before repository.update', {
        updateData,
        eventId: id,
      });

      const updatedEvent = await this.eventRepository.update(id, updateData);

      console.log('[UpdateEventUseCase] Event updated successfully', {
        updatedEventId: updatedEvent.id,
        updatedEventStartDate: updatedEvent.startDate,
        updatedEventRecurrenceRule: updatedEvent.recurrenceRule,
      });

      return updatedEvent;
    }

    throw new Error(`Invalid recurringEditScope: ${scope}`);
  }
}

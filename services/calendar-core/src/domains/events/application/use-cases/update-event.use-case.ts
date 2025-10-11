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
    if (dto.title !== undefined) updateData.title = dto.title;
    if (dto.description !== undefined) updateData.description = dto.description;
    if (dto.startTime !== undefined) updateData.startTime = dto.startTime;
    if (dto.endTime !== undefined) updateData.endTime = dto.endTime;
    if (dto.startDate !== undefined) updateData.startDate = new Date(dto.startDate);
    if (dto.endDate !== undefined) updateData.endDate = dto.endDate ? new Date(dto.endDate) : null;

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

    if (scope === 'this') {
      // Google Calendar Pattern: "Apenas este evento"
      // 1. Criar exceção na data original
      // 2. Criar novo evento one-time com as modificações

      const originalOccurrenceDate = dto.occurrenceDate
        ? new Date(dto.occurrenceDate)
        : new Date(existingEvent.startDate);

      // Criar exceção
      await this.prisma.recurrenceException.create({
        data: {
          eventId: id,
          exceptionDate: originalOccurrenceDate,
        },
      });

      // Criar novo evento one-time
      const newEvent = await this.prisma.event.create({
        data: {
          calendarId: dto.calendarId ?? existingEvent.calendarId,
          categoryId: dto.categoryId ?? existingEvent.categoryId,
          title: dto.title ?? existingEvent.title,
          description: dto.description ?? existingEvent.description,
          startTime: dto.startTime ?? existingEvent.startTime,
          endTime: dto.endTime ?? existingEvent.endTime,
          startDate: dto.startDate ? new Date(dto.startDate) : existingEvent.startDate,
          endDate: dto.endDate ? new Date(dto.endDate) : existingEvent.endDate,
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

      const originalOccurrenceDate = dto.occurrenceDate
        ? new Date(dto.occurrenceDate)
        : new Date(existingEvent.startDate);

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
      const newStartDate = dto.startDate ? new Date(dto.startDate) : originalOccurrenceDate;

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

      const newEvent = await this.prisma.event.create({
        data: {
          calendarId: dto.calendarId ?? existingEvent.calendarId,
          categoryId: dto.categoryId ?? existingEvent.categoryId,
          title: dto.title ?? existingEvent.title,
          description: dto.description ?? existingEvent.description,
          startTime: dto.startTime ?? existingEvent.startTime,
          endTime: dto.endTime ?? existingEvent.endTime,
          startDate: newStartDate,
          endDate: dto.endDate ? new Date(dto.endDate) : existingEvent.endDate,
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
      if (dto.title !== undefined) updateData.title = dto.title;
      if (dto.description !== undefined) updateData.description = dto.description;
      if (dto.startTime !== undefined) updateData.startTime = dto.startTime;
      if (dto.endTime !== undefined) updateData.endTime = dto.endTime;

      // Se mudou a data, ajustar RRULE MAS NÃO o startDate do master
      // O startDate do master deve ser preservado para manter ocorrências anteriores
      if (dto.startDate !== undefined) {
        const newStartDate = new Date(dto.startDate);
        const originalOccurrenceDate = dto.occurrenceDate
          ? new Date(dto.occurrenceDate)
          : new Date(existingEvent.startDate);

        // NÃO atualizar startDate do master - isso faria ocorrências anteriores desaparecerem
        // updateData.startDate = newStartDate;

        // Parse RRULE atual
        const currentRule = RRuleHelper.parse(existingEvent.recurrenceRule);
        if (currentRule && currentRule.freq === 'WEEKLY') {
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
        }
      }

      if (dto.endDate !== undefined) updateData.endDate = dto.endDate ? new Date(dto.endDate) : null;

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

      const updatedEvent = await this.eventRepository.update(id, updateData);
      return updatedEvent;
    }

    throw new Error(`Invalid recurringEditScope: ${scope}`);
  }
}

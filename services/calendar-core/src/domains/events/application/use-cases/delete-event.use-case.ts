import { Injectable, NotFoundException } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { PrismaClient } from '@prisma/client';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

@Injectable()
export class DeleteEventUseCase {
  private prisma: PrismaClient;

  constructor(private readonly eventRepository: EventRepository) {
    this.prisma = new PrismaClient();
  }

  async execute(id: string): Promise<void> {
    const event = await this.eventRepository.findById(id);

    if (!event) {
      throw new NotFoundException(`Event with ID ${id} not found`);
    }

    await this.eventRepository.delete(id);
  }

  async executeRecurring(
    id: string,
    scope: 'this' | 'future' | 'all',
    occurrenceDate: string,
  ): Promise<void> {
    const event = await this.eventRepository.findById(id);

    if (!event) {
      throw new NotFoundException(`Event with ID ${id} not found`);
    }

    if (!event.recurrenceRule) {
      // Se não é recorrente, deleta normalmente
      await this.eventRepository.delete(id);
      return;
    }

    const occurrenceDateObj = new Date(occurrenceDate);

    if (scope === 'this') {
      // "Apenas este evento"
      // Criar exceção na data específica
      await this.prisma.recurrenceException.create({
        data: {
          eventId: id,
          exceptionDate: occurrenceDateObj,
        },
      });
    } else if (scope === 'future') {
      // "Este e os próximos eventos"
      // Adicionar UNTIL no dia anterior à ocorrência
      const dayBefore = new Date(occurrenceDateObj);
      dayBefore.setDate(dayBefore.getDate() - 1);

      const currentRule = RRuleHelper.parse(event.recurrenceRule);
      if (!currentRule) {
        throw new Error('Invalid recurrence rule');
      }

      const updatedRule = RRuleHelper.toString({
        ...currentRule,
        until: dayBefore,
      });

      await this.eventRepository.update(id, {
        recurrenceRule: updatedRule,
      });
    } else if (scope === 'all') {
      // "Todos os eventos"
      // Deletar o evento master completamente
      await this.eventRepository.delete(id);
    }
  }
}

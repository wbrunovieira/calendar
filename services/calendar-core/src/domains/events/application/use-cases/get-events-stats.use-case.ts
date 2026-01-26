import { Injectable } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

export interface StatsData {
  // Identificador do grupo (data, categoria, tipo, etc)
  key: string;
  label: string;

  // Métricas básicas
  total: number;
  completed: number;
  pending: number;
  percentage: number;

  // Métricas adicionais opcionais
  averageCompletionTime?: number; // duração média dos eventos em minutos
  streak?: number; // dias consecutivos com 100% (apenas para groupBy=day)

  // Breakdown por subcategoria (quando agrupado por período)
  breakdown?: {
    [key: string]: {
      total: number;
      completed: number;
    };
  };
}

export interface GetEventsStatsFilters {
  // Filtros de período (obrigatórios)
  startDate: string; // YYYY-MM-DD
  endDate: string; // YYYY-MM-DD

  // Filtros de contexto (opcionais)
  calendarId?: string;
  categoryId?: string;
  categoryTypeId?: string;

  // Nível de agregação
  groupBy?: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';

  // Incluir breakdown (subcategorias dentro de cada grupo)
  includeBreakdown?: boolean;
}

@Injectable()
export class GetEventsStatsUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(filters: GetEventsStatsFilters): Promise<StatsData[]> {
    const groupBy = filters.groupBy || 'day';

    // Buscar eventos com filtros
    const events = await this.eventRepository.findAll({
      calendarId: filters.calendarId,
      categoryId: filters.categoryId,
    });

    // Garantir que datas sejam interpretadas no timezone local, não UTC
    const rangeStart = new Date(filters.startDate);
    rangeStart.setHours(0, 0, 0, 0);
    const rangeEnd = new Date(filters.endDate);
    rangeEnd.setHours(23, 59, 59, 999);

    // Coletar todas as ocorrências de eventos no período
    const occurrences = this.collectOccurrences(
      events,
      rangeStart,
      rangeEnd,
      filters.categoryTypeId,
    );

    // Agrupar conforme o groupBy solicitado
    switch (groupBy) {
      case 'day':
        return this.groupByDay(
          occurrences,
          rangeStart,
          rangeEnd,
          filters.includeBreakdown,
        );
      case 'week':
        return this.groupByWeek(occurrences, rangeStart, rangeEnd);
      case 'month':
        return this.groupByMonth(occurrences, rangeStart, rangeEnd);
      case 'category':
        return this.groupByCategory(occurrences);
      case 'categoryType':
        return this.groupByCategoryType(occurrences);
      case 'total':
        return this.groupByTotal(occurrences);
      default:
        return this.groupByDay(
          occurrences,
          rangeStart,
          rangeEnd,
          filters.includeBreakdown,
        );
    }
  }

  private collectOccurrences(
    events: any[],
    rangeStart: Date,
    rangeEnd: Date,
    categoryTypeIdFilter?: string,
  ) {
    const occurrences: Array<{
      date: string;
      eventId: string;
      categoryId: string;
      categoryName: string;
      categoryTypeId: string;
      categoryTypeName: string;
      completed: boolean;
      duration: number; // em minutos
    }> = [];

    for (const event of events) {
      // Filtrar por categoryTypeId se fornecido
      if (
        categoryTypeIdFilter &&
        event.categoryTypeId !== categoryTypeIdFilter
      ) {
        continue;
      }

      const duration = this.calculateDuration(event.startTime, event.endTime);

      // Evento one-time
      if (!event.recurrenceRule) {
        const eventDate = new Date(event.startDate);
        eventDate.setHours(0, 0, 0, 0); // Force local timezone
        const dateStr = eventDate.toISOString().split('T')[0];

        if (eventDate >= rangeStart && eventDate <= rangeEnd) {
          const execution = event.executions?.find(
            (exec) => exec.executionDate?.split('T')[0] === dateStr,
          );

          occurrences.push({
            date: dateStr,
            eventId: event.id,
            categoryId: event.categoryId || '',
            categoryName: event.category?.name || 'Sem categoria',
            categoryTypeId: event.categoryTypeId || '',
            categoryTypeName: event.categoryType?.name || 'Sem tipo',
            completed: execution?.completed || false,
            duration,
          });
        }
        continue;
      }

      // Evento recorrente: expandir usando RRULE
      const dates = RRuleHelper.generateOccurrences(
        event.recurrenceRule,
        new Date(event.startDate),
        rangeStart,
        rangeEnd,
      );

      const exceptions = new Set<string>();
      if (event.exceptions) {
        for (const exception of event.exceptions) {
          const exceptionDate = new Date(exception.exceptionDate);
          exceptions.add(exceptionDate.toISOString().split('T')[0]);
        }
      }

      for (const date of dates) {
        const dateStr = date.toISOString().split('T')[0];

        if (exceptions.has(dateStr)) {
          continue;
        }

        const completion = event.completions?.find(
          (c) =>
            new Date(c.occurrenceDate).toISOString().split('T')[0] === dateStr,
        );

        occurrences.push({
          date: dateStr,
          eventId: event.id,
          categoryId: event.categoryId || '',
          categoryName: event.category?.name || 'Sem categoria',
          categoryTypeId: event.categoryTypeId || '',
          categoryTypeName: event.categoryType?.name || 'Sem tipo',
          completed: completion?.completed || false,
          duration,
        });
      }
    }

    return occurrences;
  }

  private calculateDuration(startTime: string, endTime?: string): number {
    if (!endTime) return 60; // default 1 hora

    const [startHour, startMin] = startTime.split(':').map(Number);
    const [endHour, endMin] = endTime.split(':').map(Number);

    const startMinutes = startHour * 60 + startMin;
    const endMinutes = endHour * 60 + endMin;

    return endMinutes - startMinutes;
  }

  private groupByDay(
    occurrences: any[],
    rangeStart: Date,
    rangeEnd: Date,
    includeBreakdown?: boolean,
  ): StatsData[] {
    const statsMap = new Map<string, any>();

    // Inicializar todos os dias no range
    const currentDate = new Date(rangeStart);
    while (currentDate <= rangeEnd) {
      const dateStr = currentDate.toISOString().split('T')[0];
      statsMap.set(dateStr, {
        total: 0,
        completed: 0,
        durations: [],
        breakdown: includeBreakdown ? {} : undefined,
      });
      currentDate.setDate(currentDate.getDate() + 1);
    }

    // Processar ocorrências
    for (const occ of occurrences) {
      const stats = statsMap.get(occ.date);
      if (stats) {
        stats.total++;
        if (occ.completed) {
          stats.completed++;
          stats.durations.push(occ.duration);
        }

        if (includeBreakdown) {
          const key = `${occ.categoryName}`;
          if (!stats.breakdown[key]) {
            stats.breakdown[key] = { total: 0, completed: 0 };
          }
          stats.breakdown[key].total++;
          if (occ.completed) {
            stats.breakdown[key].completed++;
          }
        }
      }
    }

    // Converter para StatsData
    const result: StatsData[] = [];
    const sortedDates = Array.from(statsMap.keys()).sort();

    let currentStreak = 0;
    for (const date of sortedDates) {
      const stats = statsMap.get(date);
      const percentage =
        stats.total > 0 ? Math.round((stats.completed / stats.total) * 100) : 0;

      // Calcular streak
      if (percentage === 100 && stats.total > 0) {
        currentStreak++;
      } else {
        currentStreak = 0;
      }

      const avgDuration =
        stats.durations.length > 0
          ? Math.round(
              stats.durations.reduce((a, b) => a + b, 0) /
                stats.durations.length,
            )
          : undefined;

      result.push({
        key: date,
        label: date,
        total: stats.total,
        completed: stats.completed,
        pending: stats.total - stats.completed,
        percentage,
        averageCompletionTime: avgDuration,
        streak: currentStreak,
        breakdown: stats.breakdown,
      });
    }

    return result;
  }

  private groupByWeek(
    occurrences: any[],
    rangeStart: Date,
    rangeEnd: Date,
  ): StatsData[] {
    const weekMap = new Map<
      string,
      { total: number; completed: number; durations: number[] }
    >();

    for (const occ of occurrences) {
      const date = new Date(occ.date);
      const weekKey = this.getWeekKey(date);

      if (!weekMap.has(weekKey)) {
        weekMap.set(weekKey, { total: 0, completed: 0, durations: [] });
      }

      const stats = weekMap.get(weekKey)!;
      stats.total++;
      if (occ.completed) {
        stats.completed++;
        stats.durations.push(occ.duration);
      }
    }

    const result: StatsData[] = [];
    for (const [weekKey, stats] of weekMap.entries()) {
      const percentage =
        stats.total > 0 ? Math.round((stats.completed / stats.total) * 100) : 0;
      const avgDuration =
        stats.durations.length > 0
          ? Math.round(
              stats.durations.reduce((a, b) => a + b, 0) /
                stats.durations.length,
            )
          : undefined;

      result.push({
        key: weekKey,
        label: weekKey,
        total: stats.total,
        completed: stats.completed,
        pending: stats.total - stats.completed,
        percentage,
        averageCompletionTime: avgDuration,
      });
    }

    result.sort((a, b) => a.key.localeCompare(b.key));
    return result;
  }

  private groupByMonth(
    occurrences: any[],
    rangeStart: Date,
    rangeEnd: Date,
  ): StatsData[] {
    const monthMap = new Map<
      string,
      { total: number; completed: number; durations: number[] }
    >();

    for (const occ of occurrences) {
      const monthKey = occ.date.substring(0, 7); // YYYY-MM

      if (!monthMap.has(monthKey)) {
        monthMap.set(monthKey, { total: 0, completed: 0, durations: [] });
      }

      const stats = monthMap.get(monthKey)!;
      stats.total++;
      if (occ.completed) {
        stats.completed++;
        stats.durations.push(occ.duration);
      }
    }

    const result: StatsData[] = [];
    for (const [monthKey, stats] of monthMap.entries()) {
      const percentage =
        stats.total > 0 ? Math.round((stats.completed / stats.total) * 100) : 0;
      const avgDuration =
        stats.durations.length > 0
          ? Math.round(
              stats.durations.reduce((a, b) => a + b, 0) /
                stats.durations.length,
            )
          : undefined;

      result.push({
        key: monthKey,
        label: monthKey,
        total: stats.total,
        completed: stats.completed,
        pending: stats.total - stats.completed,
        percentage,
        averageCompletionTime: avgDuration,
      });
    }

    result.sort((a, b) => a.key.localeCompare(b.key));
    return result;
  }

  private groupByCategory(occurrences: any[]): StatsData[] {
    const categoryMap = new Map<
      string,
      { name: string; total: number; completed: number; durations: number[] }
    >();

    for (const occ of occurrences) {
      if (!categoryMap.has(occ.categoryId)) {
        categoryMap.set(occ.categoryId, {
          name: occ.categoryName,
          total: 0,
          completed: 0,
          durations: [],
        });
      }

      const stats = categoryMap.get(occ.categoryId)!;
      stats.total++;
      if (occ.completed) {
        stats.completed++;
        stats.durations.push(occ.duration);
      }
    }

    const result: StatsData[] = [];
    for (const [categoryId, stats] of categoryMap.entries()) {
      const percentage =
        stats.total > 0 ? Math.round((stats.completed / stats.total) * 100) : 0;
      const avgDuration =
        stats.durations.length > 0
          ? Math.round(
              stats.durations.reduce((a, b) => a + b, 0) /
                stats.durations.length,
            )
          : undefined;

      result.push({
        key: categoryId,
        label: stats.name,
        total: stats.total,
        completed: stats.completed,
        pending: stats.total - stats.completed,
        percentage,
        averageCompletionTime: avgDuration,
      });
    }

    result.sort((a, b) => b.total - a.total);
    return result;
  }

  private groupByCategoryType(occurrences: any[]): StatsData[] {
    const typeMap = new Map<
      string,
      { name: string; total: number; completed: number; durations: number[] }
    >();

    for (const occ of occurrences) {
      if (!typeMap.has(occ.categoryTypeId)) {
        typeMap.set(occ.categoryTypeId, {
          name: occ.categoryTypeName,
          total: 0,
          completed: 0,
          durations: [],
        });
      }

      const stats = typeMap.get(occ.categoryTypeId)!;
      stats.total++;
      if (occ.completed) {
        stats.completed++;
        stats.durations.push(occ.duration);
      }
    }

    const result: StatsData[] = [];
    for (const [typeId, stats] of typeMap.entries()) {
      const percentage =
        stats.total > 0 ? Math.round((stats.completed / stats.total) * 100) : 0;
      const avgDuration =
        stats.durations.length > 0
          ? Math.round(
              stats.durations.reduce((a, b) => a + b, 0) /
                stats.durations.length,
            )
          : undefined;

      result.push({
        key: typeId,
        label: stats.name,
        total: stats.total,
        completed: stats.completed,
        pending: stats.total - stats.completed,
        percentage,
        averageCompletionTime: avgDuration,
      });
    }

    result.sort((a, b) => b.total - a.total);
    return result;
  }

  private groupByTotal(occurrences: any[]): StatsData[] {
    let total = 0;
    let completed = 0;
    const durations: number[] = [];

    for (const occ of occurrences) {
      total++;
      if (occ.completed) {
        completed++;
        durations.push(occ.duration);
      }
    }

    const percentage = total > 0 ? Math.round((completed / total) * 100) : 0;
    const avgDuration =
      durations.length > 0
        ? Math.round(durations.reduce((a, b) => a + b, 0) / durations.length)
        : undefined;

    return [
      {
        key: 'total',
        label: 'Total Geral',
        total,
        completed,
        pending: total - completed,
        percentage,
        averageCompletionTime: avgDuration,
      },
    ];
  }

  private getWeekKey(date: Date): string {
    const year = date.getFullYear();
    const firstDayOfYear = new Date(year, 0, 1);
    const pastDaysOfYear =
      (date.getTime() - firstDayOfYear.getTime()) / 86400000;
    const weekNumber = Math.ceil(
      (pastDaysOfYear + firstDayOfYear.getDay() + 1) / 7,
    );
    return `${year}-W${weekNumber.toString().padStart(2, '0')}`;
  }
}

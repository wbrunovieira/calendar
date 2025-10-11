/**
 * RRULE Helper - Simplificado para suportar os casos de uso do calendário
 * Baseado no formato RFC 5545 (iCalendar)
 */

export interface RRuleOptions {
  freq: 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY';
  interval?: number;
  byweekday?: number[]; // 0=SU, 1=MO, 2=TU, 3=WE, 4=TH, 5=FR, 6=SA
  bymonthday?: number;
  until?: Date;
}

export class RRuleHelper {
  /**
   * Converte opções para string RRULE
   * Exemplo: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR"
   */
  static toString(options: RRuleOptions): string {
    const parts: string[] = [`FREQ=${options.freq}`];

    if (options.interval && options.interval > 1) {
      parts.push(`INTERVAL=${options.interval}`);
    }

    if (options.byweekday && options.byweekday.length > 0) {
      const days = options.byweekday.map(d => this.weekdayToRRule(d)).join(',');
      parts.push(`BYDAY=${days}`);
    }

    if (options.bymonthday) {
      parts.push(`BYMONTHDAY=${options.bymonthday}`);
    }

    if (options.until) {
      const untilStr = this.dateToRRuleFormat(options.until);
      parts.push(`UNTIL=${untilStr}`);
    }

    return parts.join(';');
  }

  /**
   * Parse string RRULE para opções
   */
  static parse(rrule: string): RRuleOptions | null {
    if (!rrule) return null;

    const parts = rrule.split(';');
    const options: any = {};

    for (const part of parts) {
      const [key, value] = part.split('=');

      switch (key) {
        case 'FREQ':
          options.freq = value as RRuleOptions['freq'];
          break;
        case 'INTERVAL':
          options.interval = parseInt(value);
          break;
        case 'BYDAY':
          options.byweekday = value.split(',').map(d => this.rruleToWeekday(d));
          break;
        case 'BYMONTHDAY':
          options.bymonthday = parseInt(value);
          break;
        case 'UNTIL':
          options.until = this.rruleFormatToDate(value);
          break;
      }
    }

    return options;
  }

  /**
   * Gera ocorrências de um evento recorrente dentro de um período
   */
  static generateOccurrences(
    rrule: string,
    startDate: Date,
    rangeStart: Date,
    rangeEnd: Date,
  ): Date[] {
    const options = this.parse(rrule);
    if (!options) return [];

    const occurrences: Date[] = [];
    const seriesStart = new Date(startDate);

    // Determinar data de término da série
    let seriesEnd: Date;
    if (options.until) {
      seriesEnd = new Date(options.until);
    } else {
      seriesEnd = rangeEnd;
    }

    // Ajustar início efetivo
    const effectiveStart = seriesStart > rangeStart ? seriesStart : rangeStart;
    const effectiveEnd = seriesEnd < rangeEnd ? seriesEnd : rangeEnd;

    switch (options.freq) {
      case 'DAILY':
        this.generateDaily(effectiveStart, effectiveEnd, options.interval || 1, occurrences);
        break;
      case 'WEEKLY':
        this.generateWeekly(seriesStart, effectiveStart, effectiveEnd, options, occurrences);
        break;
      case 'MONTHLY':
        this.generateMonthly(seriesStart, effectiveStart, effectiveEnd, options, occurrences);
        break;
      case 'YEARLY':
        this.generateYearly(seriesStart, effectiveStart, effectiveEnd, options, occurrences);
        break;
    }

    return occurrences;
  }

  private static generateDaily(
    start: Date,
    end: Date,
    interval: number,
    occurrences: Date[],
  ): void {
    const current = new Date(start);
    while (current <= end) {
      occurrences.push(new Date(current));
      current.setDate(current.getDate() + interval);
    }
  }

  private static generateWeekly(
    seriesStart: Date,
    start: Date,
    end: Date,
    options: RRuleOptions,
    occurrences: Date[],
  ): void {
    const interval = options.interval || 1;
    const byweekday = options.byweekday || [seriesStart.getDay()];

    // Começar da data de início da série (seriesStart)
    // e encontrar o domingo da semana que contém seriesStart
    const current = new Date(seriesStart);
    current.setDate(current.getDate() - current.getDay());

    while (current <= end) {
      for (const dayOfWeek of byweekday) {
        const occurrenceDate = new Date(current);
        occurrenceDate.setDate(occurrenceDate.getDate() + dayOfWeek);

        // Garantir que está dentro do range E após ou igual à data de início da série
        if (
          occurrenceDate >= start &&
          occurrenceDate <= end &&
          occurrenceDate >= seriesStart
        ) {
          occurrences.push(new Date(occurrenceDate));
        }
      }

      current.setDate(current.getDate() + 7 * interval);
    }
  }

  private static generateMonthly(
    seriesStart: Date,
    start: Date,
    end: Date,
    options: RRuleOptions,
    occurrences: Date[],
  ): void {
    const interval = options.interval || 1;
    const dayOfMonth = options.bymonthday || seriesStart.getDate();

    const current = new Date(start);
    current.setDate(1);

    while (current <= end) {
      const occurrenceDate = new Date(current);
      occurrenceDate.setDate(dayOfMonth);

      // Handle months with fewer days
      if (occurrenceDate.getMonth() !== current.getMonth()) {
        occurrenceDate.setDate(0);
      }

      if (
        occurrenceDate >= start &&
        occurrenceDate <= end &&
        occurrenceDate >= seriesStart
      ) {
        occurrences.push(new Date(occurrenceDate));
      }

      current.setMonth(current.getMonth() + interval);
    }
  }

  private static generateYearly(
    seriesStart: Date,
    start: Date,
    end: Date,
    options: RRuleOptions,
    occurrences: Date[],
  ): void {
    const interval = options.interval || 1;
    const current = new Date(seriesStart);

    while (current < start) {
      current.setFullYear(current.getFullYear() + interval);
    }

    while (current <= end) {
      occurrences.push(new Date(current));
      current.setFullYear(current.getFullYear() + interval);
    }
  }

  /**
   * Converte número do dia da semana (0=DOM, 6=SAB) para formato RRULE
   */
  private static weekdayToRRule(day: number): string {
    const days = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA'];
    return days[day];
  }

  /**
   * Converte formato RRULE para número do dia da semana
   */
  private static rruleToWeekday(day: string): number {
    const days = { SU: 0, MO: 1, TU: 2, WE: 3, TH: 4, FR: 5, SA: 6 };
    return days[day] || 0;
  }

  /**
   * Converte Date para formato RRULE (YYYYMMDD)
   */
  private static dateToRRuleFormat(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}${month}${day}`;
  }

  /**
   * Converte formato RRULE (YYYYMMDD) para Date
   */
  private static rruleFormatToDate(dateStr: string): Date {
    const year = parseInt(dateStr.substring(0, 4));
    const month = parseInt(dateStr.substring(4, 6)) - 1;
    const day = parseInt(dateStr.substring(6, 8));
    return new Date(year, month, day);
  }
}

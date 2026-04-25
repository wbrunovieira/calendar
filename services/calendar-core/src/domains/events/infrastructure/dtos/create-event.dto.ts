import {
  IsString,
  IsOptional,
  IsBoolean,
  IsArray,
  IsNumber,
  IsDateString,
  Matches,
  IsIn,
  Min,
  Max,
  ValidateNested,
} from 'class-validator';
import { Type } from 'class-transformer';
import { EventReminderDto } from './event-reminder.dto';

export class CreateEventDto {
  @IsString()
  calendarId: string;

  @IsOptional()
  @IsString()
  categoryId?: string;

  @IsOptional()
  @IsString()
  categoryTypeId?: string;

  @IsOptional()
  @IsString()
  labelId?: string;

  @IsString()
  title: string;

  @IsOptional()
  @IsString()
  description?: string;

  @IsString()
  @Matches(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, {
    message: 'startTime must be in HH:mm format',
  })
  startTime: string;

  @IsOptional()
  @IsString()
  @Matches(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, {
    message: 'endTime must be in HH:mm format',
  })
  endTime?: string;

  @IsOptional()
  @IsDateString()
  startDate?: string;

  @IsOptional()
  @IsDateString()
  endDate?: string;

  @IsOptional()
  @IsBoolean()
  isRecurring?: boolean;

  // Direct RRULE string (preferred for new frontend)
  @IsOptional()
  @IsString()
  recurrenceRule?: string; // e.g., 'FREQ=DAILY', 'FREQ=WEEKLY;BYDAY=MO,WE,FR'

  @IsOptional()
  @IsString()
  recurrenceFrequency?: string; // 'daily', 'weekly', 'monthly', 'yearly' (legacy)

  @IsOptional()
  @IsNumber()
  recurrenceInterval?: number;

  @IsOptional()
  @IsArray()
  @IsNumber({}, { each: true })
  recurrenceDaysOfWeek?: number[];

  @IsOptional()
  @IsNumber()
  recurrenceDayOfMonth?: number;

  @IsOptional()
  @IsNumber()
  recurrenceWeekOfMonth?: number;

  @IsOptional()
  @IsDateString()
  recurrenceEndDate?: string;

  // Tipo de evento: EVENT (compromisso), HABIT (habito), TODO (tarefa), REMINDER (lembrete)
  @IsOptional()
  @IsString()
  @IsIn(['EVENT', 'HABIT', 'TODO', 'REMINDER'])
  eventType?: string;

  // Para TODOs: prioridade (1=alta, 2=media, 3=baixa)
  @IsOptional()
  @IsNumber()
  @Min(1)
  @Max(3)
  priority?: number;

  // Para TODOs: data de vencimento
  @IsOptional()
  @IsDateString()
  dueDate?: string;

  // Para REMINDERs: dias de antecedência para lembrar (ex: [7, 3, 1, 0] = 7 dias antes, 3 dias antes, etc)
  @IsOptional()
  @IsArray()
  @IsNumber({}, { each: true })
  reminderDaysBefore?: number[];

  // Hábitos semanais flexíveis
  @IsOptional()
  @IsString()
  @IsIn(['FIXED', 'FLEXIBLE'])
  recurrenceType?: string; // "FIXED" | "FLEXIBLE"

  @IsOptional()
  @IsNumber()
  @Min(1)
  @Max(7)
  weeklyTargetCount?: number; // Meta semanal (ex: 2 vezes por semana)

  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  weeklyPreferredDays?: string[]; // Dias preferidos ["MO", "TU", "WE", "TH", "FR", "SA", "SU"]

  // Alertas do evento (ex: [{ minutesBefore: 30, method: 'notification' }])
  @IsOptional()
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => EventReminderDto)
  reminders?: EventReminderDto[];

  // Internal: set to 'google' when event is created by inbound sync (prevents loop)
  @IsOptional()
  @IsString()
  syncSource?: 'google' | 'internal';
}

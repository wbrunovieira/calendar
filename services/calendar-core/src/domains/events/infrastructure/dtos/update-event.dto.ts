import { IsOptional, IsString, IsBoolean, IsNumber, IsArray } from 'class-validator';

export class UpdateEventDto {
  @IsOptional()
  @IsString()
  calendarId?: string;

  @IsOptional()
  @IsString()
  categoryId?: string;

  @IsOptional()
  @IsString()
  title?: string;

  @IsOptional()
  @IsString()
  description?: string;

  @IsOptional()
  @IsString()
  startTime?: string;

  @IsOptional()
  @IsString()
  endTime?: string;

  @IsOptional()
  @IsString()
  startDate?: string;

  @IsOptional()
  @IsString()
  endDate?: string;

  @IsOptional()
  @IsBoolean()
  isRecurring?: boolean;

  @IsOptional()
  @IsString()
  recurrenceFrequency?: string;

  @IsOptional()
  @IsNumber()
  recurrenceInterval?: number;

  @IsOptional()
  @IsArray()
  recurrenceDaysOfWeek?: number[];

  @IsOptional()
  @IsNumber()
  recurrenceDayOfMonth?: number;

  @IsOptional()
  @IsString()
  recurrenceEndDate?: string;

  @IsOptional()
  @IsString()
  recurringEditScope?: 'this' | 'all' | 'future';

  @IsOptional()
  @IsString()
  occurrenceDate?: string; // The specific date of the occurrence being edited
}

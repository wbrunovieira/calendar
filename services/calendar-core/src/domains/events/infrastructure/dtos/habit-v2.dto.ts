import {
  IsDateString,
  IsIn,
  IsNumber,
  IsOptional,
  IsString,
  IsUUID,
  Matches,
  MaxLength,
  Min,
} from 'class-validator';

export const SLIP_TRIGGERS = [
  'Social',
  'Stress',
  'Boredom',
  'Tired',
  'Celebration',
  'Craving',
];

export class LogHabitAmountDto {
  @IsUUID()
  eventId: string;

  @IsDateString()
  executionDate: string;

  @IsNumber()
  @Min(0)
  value: number;
}

export class LogHabitSlipDto {
  @Matches(/^\d{4}-\d{2}-\d{2}$/, { message: 'date must be YYYY-MM-DD' })
  date: string;

  @IsOptional()
  @IsIn(SLIP_TRIGGERS)
  trigger?: string;

  @IsOptional()
  @IsString()
  @MaxLength(500)
  note?: string;
}

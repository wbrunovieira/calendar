import {
  IsBoolean,
  IsDateString,
  IsOptional,
  IsString,
  IsUUID,
} from 'class-validator';

export class ToggleEventExecutionDto {
  @IsUUID()
  eventId: string;

  @IsDateString()
  executionDate: string;

  @IsBoolean()
  completed: boolean;

  @IsOptional()
  @IsString()
  notes?: string;
}

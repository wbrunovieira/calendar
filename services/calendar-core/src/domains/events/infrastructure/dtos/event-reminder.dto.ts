import { IsNumber, IsOptional, IsString, Min } from 'class-validator';

export class EventReminderDto {
  @IsNumber()
  @Min(0)
  minutesBefore: number;

  @IsOptional()
  @IsString()
  method?: string;
}

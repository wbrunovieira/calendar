import { IsString, IsEmail, IsOptional, IsBoolean, IsUUID, Matches } from 'class-validator';

export class CreateCalendarDto {
  @IsUUID()
  userId: string;

  @IsString()
  name: string;

  @IsEmail()
  @IsOptional()
  email?: string;

  @IsString()
  @Matches(/^#([A-Fa-f0-9]{6})$/, {
    message: 'Color must be a valid hex color (e.g., #350545)',
  })
  color: string;

  @IsString()
  type: string; // 'professional' | 'personal'

  @IsString()
  @IsOptional()
  googleCalendarId?: string;

  @IsUUID()
  @IsOptional()
  financeProfileId?: string;

  @IsBoolean()
  @IsOptional()
  isActive?: boolean;
}

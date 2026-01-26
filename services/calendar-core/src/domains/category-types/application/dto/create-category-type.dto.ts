import {
  IsString,
  IsNotEmpty,
  IsOptional,
  Matches,
  IsUUID,
} from 'class-validator';

export class CreateCategoryTypeDto {
  @IsUUID()
  @IsNotEmpty()
  calendarId: string;

  @IsString()
  @IsNotEmpty()
  name: string;

  @IsString()
  @IsNotEmpty()
  @Matches(/^[a-z0-9-]+$/, {
    message: 'value must contain only lowercase letters, numbers, and hyphens',
  })
  value: string;

  @IsString()
  @IsOptional()
  icon?: string;

  @IsString()
  @IsNotEmpty()
  @Matches(/^#[0-9A-Fa-f]{6}$/, {
    message: 'color must be a valid hex color code',
  })
  color: string;
}

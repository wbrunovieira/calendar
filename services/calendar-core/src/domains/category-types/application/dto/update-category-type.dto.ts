import { IsString, IsOptional, Matches } from 'class-validator';

export class UpdateCategoryTypeDto {
  @IsString()
  @IsOptional()
  name?: string;

  @IsString()
  @IsOptional()
  @Matches(/^[a-z0-9-]+$/, {
    message: 'value must contain only lowercase letters, numbers, and hyphens',
  })
  value?: string;

  @IsString()
  @IsOptional()
  icon?: string;

  @IsString()
  @IsOptional()
  @Matches(/^#[0-9A-Fa-f]{6}$/, {
    message: 'color must be a valid hex color code',
  })
  color?: string;
}

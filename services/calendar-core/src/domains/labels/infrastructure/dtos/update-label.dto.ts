import { IsString, IsOptional, Matches } from 'class-validator';

export class UpdateLabelDto {
  @IsString()
  @IsOptional()
  name?: string;

  @IsString()
  @IsOptional()
  @Matches(/^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/, {
    message: 'color must be a valid hex color (e.g., #FF0000 or #F00)',
  })
  color?: string;
}

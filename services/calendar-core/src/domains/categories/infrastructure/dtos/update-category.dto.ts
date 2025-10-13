export class UpdateCategoryDto {
  name?: string;
  icon?: string;
  color?: string;
  type?: string; // DEPRECATED - usar typeIds
  typeIds?: string[]; // Array de IDs de CategoryType
}

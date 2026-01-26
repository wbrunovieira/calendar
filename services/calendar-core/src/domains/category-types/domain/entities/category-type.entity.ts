export class CategoryType {
  id: string;
  calendarId: string;
  name: string;
  value: string; // slug/key: work, health, leisure, etc
  icon?: string | null;
  color: string;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;

  constructor(partial: Partial<CategoryType>) {
    Object.assign(this, partial);
  }

  static create(data: {
    calendarId: string;
    name: string;
    value: string;
    color: string;
    icon?: string;
  }): CategoryType {
    return new CategoryType({
      ...data,
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
    });
  }

  update(
    data: Partial<Pick<CategoryType, 'name' | 'value' | 'icon' | 'color'>>,
  ): void {
    Object.assign(this, data);
    this.updatedAt = new Date();
  }

  deactivate(): void {
    this.isActive = false;
    this.updatedAt = new Date();
  }

  activate(): void {
    this.isActive = true;
    this.updatedAt = new Date();
  }
}

export class Category {
  id: string;
  calendarId: string;
  name: string;
  icon?: string | null;
  color: string;
  type?: string | null;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;

  constructor(partial: Partial<Category>) {
    Object.assign(this, partial);
  }

  static create(data: {
    calendarId: string;
    name: string;
    color: string;
    icon?: string;
    type?: string;
  }): Category {
    return new Category({
      ...data,
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
    });
  }

  update(data: Partial<Pick<Category, 'name' | 'icon' | 'color' | 'type'>>): void {
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

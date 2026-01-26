export interface LabelProps {
  id: string;
  calendarId: string;
  name: string;
  color: string;
  isActive?: boolean;
  createdAt: Date;
  updatedAt: Date;
}

export class Label {
  private _id: string;
  private _calendarId: string;
  private _name: string;
  private _color: string;
  private _isActive: boolean;
  private _createdAt: Date;
  private _updatedAt: Date;

  constructor(props: LabelProps) {
    this.validateCalendarId(props.calendarId);
    this.validateName(props.name);
    this.validateColor(props.color);

    this._id = props.id;
    this._calendarId = props.calendarId;
    this._name = props.name;
    this._color = props.color;
    this._isActive = props.isActive ?? true;
    this._createdAt = props.createdAt;
    this._updatedAt = props.updatedAt;
  }

  get id(): string {
    return this._id;
  }

  get calendarId(): string {
    return this._calendarId;
  }

  get name(): string {
    return this._name;
  }

  get color(): string {
    return this._color;
  }

  get isActive(): boolean {
    return this._isActive;
  }

  get createdAt(): Date {
    return this._createdAt;
  }

  get updatedAt(): Date {
    return this._updatedAt;
  }

  private validateCalendarId(calendarId: string): void {
    if (!calendarId || calendarId.trim() === '') {
      throw new Error('Calendar ID is required');
    }
  }

  private validateName(name: string): void {
    if (!name || name.trim() === '') {
      throw new Error('Label name is required');
    }
  }

  private validateColor(color: string): void {
    if (!color || color.trim() === '') {
      throw new Error('Label color is required');
    }

    const hexColorRegex = /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/;
    if (!hexColorRegex.test(color)) {
      throw new Error('Label color must be a valid hex color');
    }
  }

  update(props: Partial<Pick<LabelProps, 'name' | 'color'>>): void {
    if (props.name !== undefined) {
      this.validateName(props.name);
      this._name = props.name;
    }

    if (props.color !== undefined) {
      this.validateColor(props.color);
      this._color = props.color;
    }

    this._updatedAt = new Date();
  }

  deactivate(): void {
    this._isActive = false;
    this._updatedAt = new Date();
  }

  activate(): void {
    this._isActive = true;
    this._updatedAt = new Date();
  }

  toJSON(): LabelProps {
    return {
      id: this._id,
      calendarId: this._calendarId,
      name: this._name,
      color: this._color,
      isActive: this._isActive,
      createdAt: this._createdAt,
      updatedAt: this._updatedAt,
    };
  }
}

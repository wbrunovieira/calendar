export class Calendar {
  id: string;
  userId: string;
  name: string;
  email: string | null;
  color: string;
  type: string;
  googleCalendarId: string | null;
  googleSyncToken: string | null;
  lastSyncAt: Date | null;
  financeProfileId: string | null;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;

  private constructor(props: Calendar) {
    Object.assign(this, props);
  }

  static create(data: {
    id?: string;
    userId: string;
    name: string;
    email?: string | null;
    color: string;
    type: string;
    googleCalendarId?: string | null;
    googleSyncToken?: string | null;
    lastSyncAt?: Date | null;
    financeProfileId?: string | null;
    isActive?: boolean;
    createdAt?: Date;
    updatedAt?: Date;
  }): Calendar {
    return new Calendar({
      id: data.id || crypto.randomUUID(),
      userId: data.userId,
      name: data.name,
      email: data.email || null,
      color: data.color,
      type: data.type,
      googleCalendarId: data.googleCalendarId || null,
      googleSyncToken: data.googleSyncToken || null,
      lastSyncAt: data.lastSyncAt || null,
      financeProfileId: data.financeProfileId || null,
      isActive: data.isActive ?? true,
      createdAt: data.createdAt || new Date(),
      updatedAt: data.updatedAt || new Date(),
    });
  }
}

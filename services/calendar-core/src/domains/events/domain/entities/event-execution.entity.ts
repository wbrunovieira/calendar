export class EventExecution {
  id: string;
  eventId: string;
  executionDate: Date;
  completed: boolean;
  completedAt?: Date | null;
  notes?: string | null;
  createdAt: Date;
  updatedAt: Date;

  constructor(props: any) {
    this.id = props.id;
    this.eventId = props.eventId;
    this.executionDate = props.executionDate;
    this.completed = props.completed;
    this.completedAt = props.completedAt || undefined;
    this.notes = props.notes || undefined;
    this.createdAt = props.createdAt;
    this.updatedAt = props.updatedAt;
  }

  markAsCompleted(notes?: string): void {
    this.completed = true;
    this.completedAt = new Date();
    if (notes) {
      this.notes = notes;
    }
  }

  markAsIncomplete(): void {
    this.completed = false;
    this.completedAt = undefined;
  }
}

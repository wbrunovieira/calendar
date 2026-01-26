import { Label } from '../../domain/entities/label.entity';

export interface LabelRepository {
  create(label: Label): Promise<Label>;
  findById(id: string): Promise<Label | null>;
  findByCalendarId(calendarId: string): Promise<Label[]>;
  update(label: Label): Promise<Label>;
  delete(id: string): Promise<void>;
}

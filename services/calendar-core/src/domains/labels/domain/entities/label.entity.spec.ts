import { describe, it, expect } from 'vitest';
import { Label } from './label.entity';

describe('Label Entity', () => {
  const validProps = {
    id: 'label-123',
    calendarId: 'calendar-456',
    name: 'Urgente',
    color: '#FF0000',
    isActive: true,
    createdAt: new Date('2024-01-15'),
    updatedAt: new Date('2024-01-15'),
  };

  describe('constructor', () => {
    it('should create a label with all properties', () => {
      const label = new Label(validProps);

      expect(label.id).toBe('label-123');
      expect(label.calendarId).toBe('calendar-456');
      expect(label.name).toBe('Urgente');
      expect(label.color).toBe('#FF0000');
      expect(label.isActive).toBe(true);
      expect(label.createdAt).toEqual(new Date('2024-01-15'));
      expect(label.updatedAt).toEqual(new Date('2024-01-15'));
    });

    it('should create a label with default isActive as true', () => {
      const props = { ...validProps };
      delete (props as Partial<typeof validProps>).isActive;

      const label = new Label(props as typeof validProps);

      expect(label.isActive).toBe(true);
    });
  });

  describe('validation', () => {
    it('should throw error when name is empty', () => {
      expect(() => new Label({ ...validProps, name: '' })).toThrow('Label name is required');
    });

    it('should throw error when name is only whitespace', () => {
      expect(() => new Label({ ...validProps, name: '   ' })).toThrow('Label name is required');
    });

    it('should throw error when color is empty', () => {
      expect(() => new Label({ ...validProps, color: '' })).toThrow('Label color is required');
    });

    it('should throw error when color is invalid hex format', () => {
      expect(() => new Label({ ...validProps, color: 'red' })).toThrow('Label color must be a valid hex color');
      expect(() => new Label({ ...validProps, color: '#GGG' })).toThrow('Label color must be a valid hex color');
      expect(() => new Label({ ...validProps, color: '123456' })).toThrow('Label color must be a valid hex color');
    });

    it('should accept valid 3-digit hex color', () => {
      const label = new Label({ ...validProps, color: '#F00' });
      expect(label.color).toBe('#F00');
    });

    it('should accept valid 6-digit hex color', () => {
      const label = new Label({ ...validProps, color: '#FF0000' });
      expect(label.color).toBe('#FF0000');
    });

    it('should accept lowercase hex color', () => {
      const label = new Label({ ...validProps, color: '#ff0000' });
      expect(label.color).toBe('#ff0000');
    });

    it('should throw error when calendarId is empty', () => {
      expect(() => new Label({ ...validProps, calendarId: '' })).toThrow('Calendar ID is required');
    });
  });

  describe('update', () => {
    it('should update name', () => {
      const label = new Label(validProps);
      label.update({ name: 'Importante' });

      expect(label.name).toBe('Importante');
    });

    it('should update color', () => {
      const label = new Label(validProps);
      label.update({ color: '#00FF00' });

      expect(label.color).toBe('#00FF00');
    });

    it('should update multiple properties', () => {
      const label = new Label(validProps);
      label.update({ name: 'Importante', color: '#00FF00' });

      expect(label.name).toBe('Importante');
      expect(label.color).toBe('#00FF00');
    });

    it('should throw error when updating with invalid name', () => {
      const label = new Label(validProps);
      expect(() => label.update({ name: '' })).toThrow('Label name is required');
    });

    it('should throw error when updating with invalid color', () => {
      const label = new Label(validProps);
      expect(() => label.update({ color: 'invalid' })).toThrow('Label color must be a valid hex color');
    });
  });

  describe('deactivate', () => {
    it('should deactivate the label', () => {
      const label = new Label(validProps);
      label.deactivate();

      expect(label.isActive).toBe(false);
    });
  });

  describe('activate', () => {
    it('should activate the label', () => {
      const label = new Label({ ...validProps, isActive: false });
      label.activate();

      expect(label.isActive).toBe(true);
    });
  });

  describe('toJSON', () => {
    it('should return plain object representation', () => {
      const label = new Label(validProps);
      const json = label.toJSON();

      expect(json).toEqual({
        id: 'label-123',
        calendarId: 'calendar-456',
        name: 'Urgente',
        color: '#FF0000',
        isActive: true,
        createdAt: new Date('2024-01-15'),
        updatedAt: new Date('2024-01-15'),
      });
    });
  });
});

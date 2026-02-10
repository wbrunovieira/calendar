import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import EventRemindersField from './EventRemindersField';
import type { ReminderInput } from '@/types/calendar';

describe('EventRemindersField', () => {
  const defaultProps = {
    reminders: [] as ReminderInput[],
    onChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render the label', () => {
      render(<EventRemindersField {...defaultProps} />);
      expect(screen.getByText('Alertas')).toBeInTheDocument();
    });

    it('should render the preset selector', () => {
      render(<EventRemindersField {...defaultProps} />);
      expect(screen.getByText('+ Adicionar alerta')).toBeInTheDocument();
    });

    it('should render existing reminders', () => {
      const reminders: ReminderInput[] = [
        { minutesBefore: 30, method: 'notification' },
        { minutesBefore: 60, method: 'notification' },
      ];
      render(<EventRemindersField {...defaultProps} reminders={reminders} />);
      // Text appears in both the reminder list and the preset dropdown, so use getAllByText
      expect(screen.getAllByText('30 minutos antes').length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText('1 hora antes').length).toBeGreaterThanOrEqual(1);
    });

    it('should render "Na hora do evento" for 0 minutes', () => {
      const reminders: ReminderInput[] = [{ minutesBefore: 0, method: 'notification' }];
      render(<EventRemindersField {...defaultProps} reminders={reminders} />);
      expect(screen.getAllByText('Na hora do evento').length).toBeGreaterThanOrEqual(1);
    });

    it('should render "1 dia antes" for 1440 minutes', () => {
      const reminders: ReminderInput[] = [{ minutesBefore: 1440, method: 'notification' }];
      render(<EventRemindersField {...defaultProps} reminders={reminders} />);
      expect(screen.getAllByText('1 dia antes').length).toBeGreaterThanOrEqual(1);
    });
  });

  describe('adding reminders', () => {
    it('should call onChange when a preset is selected', () => {
      const onChange = vi.fn();
      render(<EventRemindersField reminders={[]} onChange={onChange} />);

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: '15' } });

      expect(onChange).toHaveBeenCalledWith([{ minutesBefore: 15, method: 'notification' }]);
    });

    it('should not add duplicate reminder', () => {
      const onChange = vi.fn();
      const reminders: ReminderInput[] = [{ minutesBefore: 30, method: 'notification' }];
      render(<EventRemindersField reminders={reminders} onChange={onChange} />);

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: '30' } });

      expect(onChange).not.toHaveBeenCalled();
    });

    it('should show custom input when "Personalizado..." is selected', () => {
      render(<EventRemindersField {...defaultProps} />);

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: 'custom' } });

      expect(screen.getByText('antes')).toBeInTheDocument();
      expect(screen.getByText('OK')).toBeInTheDocument();
      expect(screen.getByText('Cancelar')).toBeInTheDocument();
    });
  });

  describe('removing reminders', () => {
    it('should call onChange without the removed reminder', () => {
      const onChange = vi.fn();
      const reminders: ReminderInput[] = [
        { minutesBefore: 5, method: 'notification' },
        { minutesBefore: 30, method: 'notification' },
      ];
      render(<EventRemindersField reminders={reminders} onChange={onChange} />);

      // Click the first remove button (✕)
      const removeButtons = screen.getAllByRole('button');
      // Remove buttons have ✕ text
      const xButtons = removeButtons.filter(b => b.textContent === '✕');
      fireEvent.click(xButtons[0]);

      expect(onChange).toHaveBeenCalledWith([{ minutesBefore: 30, method: 'notification' }]);
    });
  });
});

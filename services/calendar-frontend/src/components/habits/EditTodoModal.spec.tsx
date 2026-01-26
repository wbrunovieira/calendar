import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import EditTodoModal from './EditTodoModal';
import type { Calendar, Event } from '@/types/calendar';

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    events: {
      update: vi.fn(),
    },
    categories: {
      list: vi.fn(),
    },
    labels: {
      list: vi.fn(),
      create: vi.fn(),
    },
  },
}));

import { api } from '@/lib/api';

describe('EditTodoModal', () => {
  const mockCalendars: Calendar[] = [
    {
      id: 'cal-1',
      name: 'Personal',
      email: 'personal@test.com',
      color: '#FF5733',
      type: 'personal',
      isActive: true,
    },
    {
      id: 'cal-2',
      name: 'Work',
      email: 'work@test.com',
      color: '#0077B5',
      type: 'professional',
      isActive: true,
    },
  ];

  const mockTodo: Event = {
    id: 'todo-1',
    calendarId: 'cal-1',
    title: 'Original Todo',
    description: 'Original description',
    startTime: '00:00',
    startDate: '2024-01-15',
    eventType: 'TODO',
    priority: 2,
    dueDate: '2024-01-20',
    isActive: true,
    isRecurring: false,
  };

  const mockOnClose = vi.fn();
  const mockOnUpdated = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.categories.list).mockResolvedValue([]);
    vi.mocked(api.labels.list).mockResolvedValue([]);
    vi.mocked(api.events.update).mockResolvedValue(mockTodo);
  });

  describe('rendering', () => {
    it('should render modal when isOpen is true and todo is provided', async () => {
      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      expect(screen.getByText('Editar Tarefa')).toBeInTheDocument();
    });

    it('should not render modal when isOpen is false', () => {
      render(
        <EditTodoModal
          isOpen={false}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      expect(screen.queryByText('Editar Tarefa')).not.toBeInTheDocument();
    });

    it('should not render modal when todo is null', () => {
      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={null}
          calendars={mockCalendars}
        />
      );

      expect(screen.queryByText('Editar Tarefa')).not.toBeInTheDocument();
    });

    it('should populate form with todo data', async () => {
      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Titulo')).toHaveValue('Original Todo');
        expect(screen.getByLabelText('Descricao (opcional)')).toHaveValue('Original description');
        expect(screen.getByLabelText('Prioridade')).toHaveValue('2');
      });
    });
  });

  describe('form submission', () => {
    it('should update todo with new values', async () => {
      const user = userEvent.setup();

      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Titulo')).toHaveValue('Original Todo');
      });

      const titleInput = screen.getByLabelText('Titulo');
      await user.clear(titleInput);
      await user.type(titleInput, 'Updated Todo');

      await user.click(screen.getByRole('button', { name: /salvar/i }));

      await waitFor(() => {
        expect(api.events.update).toHaveBeenCalledWith(
          'todo-1',
          expect.objectContaining({
            title: 'Updated Todo',
            calendarId: 'cal-1',
          })
        );
      });
    });

    it('should call onUpdated and onClose after successful update', async () => {
      const user = userEvent.setup();

      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Titulo')).toHaveValue('Original Todo');
      });

      await user.click(screen.getByRole('button', { name: /salvar/i }));

      await waitFor(() => {
        expect(mockOnUpdated).toHaveBeenCalled();
        expect(mockOnClose).toHaveBeenCalled();
      });
    });
  });

  describe('interactions', () => {
    it('should call onClose when cancel button is clicked', async () => {
      const user = userEvent.setup();

      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await user.click(screen.getByRole('button', { name: /cancelar/i }));

      expect(mockOnClose).toHaveBeenCalled();
    });

    it('should disable save button when title is empty', async () => {
      const user = userEvent.setup();

      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Titulo')).toHaveValue('Original Todo');
      });

      const titleInput = screen.getByLabelText('Titulo');
      await user.clear(titleInput);

      const saveButton = screen.getByRole('button', { name: /salvar/i });
      expect(saveButton).toBeDisabled();
    });

    it('should allow changing priority', async () => {
      const user = userEvent.setup();

      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Prioridade')).toHaveValue('2');
      });

      const prioritySelect = screen.getByLabelText('Prioridade');
      await user.selectOptions(prioritySelect, '1');

      expect(prioritySelect).toHaveValue('1');
    });

    it('should allow changing calendar', async () => {
      const user = userEvent.setup();

      render(
        <EditTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onUpdated={mockOnUpdated}
          todo={mockTodo}
          calendars={mockCalendars}
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Perfil')).toHaveValue('cal-1');
      });

      const calendarSelect = screen.getByLabelText('Perfil');
      await user.selectOptions(calendarSelect, 'cal-2');

      expect(calendarSelect).toHaveValue('cal-2');
    });
  });
});

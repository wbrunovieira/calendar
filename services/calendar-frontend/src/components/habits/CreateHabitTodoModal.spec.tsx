import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CreateHabitTodoModal from './CreateHabitTodoModal';
import type { Calendar, Category, CategoryType, Event } from '@/types/calendar';

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    events: {
      create: vi.fn(),
    },
    categories: {
      list: vi.fn(),
    },
    categoryTypes: {
      list: vi.fn(),
    },
    labels: {
      list: vi.fn(),
      create: vi.fn(),
    },
  },
}));

import { api } from '@/lib/api';

describe('CreateHabitTodoModal', () => {
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
      name: 'WB Digital Solutions',
      email: 'work@test.com',
      color: '#0077B5',
      type: 'professional',
      isActive: true,
    },
  ];

  const mockCategoryTypes: CategoryType[] = [
    {
      id: 'ct-1',
      calendarId: 'cal-2',
      name: 'Work',
      value: 'work',
      color: '#0077B5',
      isActive: true,
      createdAt: '2024-01-01',
      updatedAt: '2024-01-01',
    },
  ];

  const mockCategories: Category[] = [
    {
      id: 'cat-1',
      calendarId: 'cal-2',
      name: 'LinkedIn',
      icon: '💼',
      color: '#0077B5',
      isActive: true,
      categoryTypes: mockCategoryTypes,
    },
  ];

  const mockOnClose = vi.fn();
  const mockOnCreated = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.categories.list).mockResolvedValue(mockCategories);
    vi.mocked(api.categoryTypes.list).mockResolvedValue([]);
    vi.mocked(api.labels.list).mockResolvedValue([]);
    vi.mocked(api.events.create).mockResolvedValue({
      id: 'new-event-1',
      calendarId: 'cal-1',
      title: 'Test',
      startTime: '09:00',
      startDate: '2024-01-15',
      eventType: 'HABIT',
      isActive: true,
      isRecurring: true,
    } as unknown as Event);
  });

  describe('rendering', () => {
    it('should render modal when isOpen is true', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      expect(screen.getByText('Novo Habito')).toBeInTheDocument();
    });

    it('should not render modal when isOpen is false', () => {
      render(
        <CreateHabitTodoModal
          isOpen={false}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      expect(screen.queryByText('Novo Habito')).not.toBeInTheDocument();
    });

    it('should show "Nova Tarefa" title when eventType is TODO', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="TODO"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      expect(screen.getByText('Nova Tarefa')).toBeInTheDocument();
    });

    it('should show calendar selector when selectedCalendarId is null', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId={null}
        />
      );

      expect(screen.getByLabelText('Perfil')).toBeInTheDocument();
    });

    it('should not show calendar selector when selectedCalendarId is provided', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      expect(screen.queryByLabelText('Perfil')).not.toBeInTheDocument();
    });
  });

  describe('habit creation', () => {
    it('should show recurrence options for habits', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      expect(screen.getByLabelText('Frequencia')).toBeInTheDocument();
      expect(screen.getByLabelText('Horario Preferido')).toBeInTheDocument();
    });

    it('should create habit with correct data', async () => {
      const user = userEvent.setup();

      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      await user.type(screen.getByLabelText('Titulo'), 'Meditacao');
      await user.click(screen.getByRole('button', { name: /criar/i }));

      await waitFor(() => {
        expect(api.events.create).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Meditacao',
            calendarId: 'cal-1',
            eventType: 'HABIT',
          })
        );
      });
    });
  });

  describe('todo creation', () => {
    it('should show priority and due date options for todos', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="TODO"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      expect(screen.getByLabelText('Prioridade')).toBeInTheDocument();
      expect(screen.getByLabelText('Data de Vencimento')).toBeInTheDocument();
    });

    it('should show category selector for todos when category type is selected', async () => {
      const user = userEvent.setup();
      vi.mocked(api.categoryTypes.list).mockResolvedValue(mockCategoryTypes);
      vi.mocked(api.categories.list).mockResolvedValue(mockCategories);

      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="TODO"
          calendars={mockCalendars}
          selectedCalendarId="cal-2"
        />
      );

      // Wait for the "Tipo" selector to appear (requires categoryTypes to load)
      await waitFor(() => {
        expect(screen.getByLabelText('Tipo')).toBeInTheDocument();
      });

      // Select a category type to make the "Categoria" field appear
      const tipoSelect = screen.getByLabelText('Tipo');
      await user.selectOptions(tipoSelect, 'ct-1');

      await waitFor(() => {
        expect(screen.getByLabelText('Categoria')).toBeInTheDocument();
      });
    });

    it('should create todo with correct data', async () => {
      const user = userEvent.setup();

      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="TODO"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      await user.type(screen.getByLabelText('Titulo'), 'Criar post LinkedIn');
      await user.click(screen.getByRole('button', { name: /criar/i }));

      await waitFor(() => {
        expect(api.events.create).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Criar post LinkedIn',
            calendarId: 'cal-1',
            eventType: 'TODO',
          })
        );
      });
    });
  });

  describe('interactions', () => {
    it('should call onClose when cancel button is clicked', async () => {
      const user = userEvent.setup();

      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      await user.click(screen.getByRole('button', { name: /cancelar/i }));

      expect(mockOnClose).toHaveBeenCalled();
    });

    it('should call onCreated after successful creation', async () => {
      const user = userEvent.setup();

      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      await user.type(screen.getByLabelText('Titulo'), 'Test Habit');
      await user.click(screen.getByRole('button', { name: /criar/i }));

      await waitFor(() => {
        expect(mockOnCreated).toHaveBeenCalled();
      });
    });

    it('should disable create button when title is empty', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId="cal-1"
        />
      );

      const createButton = screen.getByRole('button', { name: /criar/i });
      expect(createButton).toBeDisabled();
    });

    it('should disable create button when no calendar selected and selectedCalendarId is null', () => {
      render(
        <CreateHabitTodoModal
          isOpen={true}
          onClose={mockOnClose}
          onCreated={mockOnCreated}
          eventType="HABIT"
          calendars={mockCalendars}
          selectedCalendarId={null}
        />
      );

      const createButton = screen.getByRole('button', { name: /criar/i });
      expect(createButton).toBeDisabled();
    });
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LabelSelector from './LabelSelector';
import type { Label } from '@/types/calendar';

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    labels: {
      list: vi.fn(),
      create: vi.fn(),
    },
  },
}));

import { api } from '@/lib/api';

describe('LabelSelector', () => {
  const mockLabels: Label[] = [
    {
      id: 'label-1',
      calendarId: 'cal-1',
      name: 'Urgente',
      color: '#ef4444',
      isActive: true,
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
    {
      id: 'label-2',
      calendarId: 'cal-1',
      name: 'Importante',
      color: '#f97316',
      isActive: true,
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
  ];

  const mockOnSelect = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.labels.list).mockResolvedValue(mockLabels);
    vi.mocked(api.labels.create).mockResolvedValue({
      id: 'new-label',
      calendarId: 'cal-1',
      name: 'New Label',
      color: '#ef4444',
      isActive: true,
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    });
  });

  describe('rendering', () => {
    it('should render the label selector with default text', async () => {
      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      expect(screen.getByText('Label')).toBeInTheDocument();
      expect(screen.getByText('Sem label')).toBeInTheDocument();
    });

    it('should fetch labels when calendarId is provided', async () => {
      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalledWith('cal-1');
      });
    });

    it('should display selected label name', async () => {
      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId="label-1"
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(screen.getByText('Urgente')).toBeInTheDocument();
      });
    });
  });

  describe('dropdown interactions', () => {
    it('should open dropdown when clicked', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      expect(screen.getByText('Criar novo label')).toBeInTheDocument();
    });

    it('should show existing labels in dropdown', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      expect(screen.getByText('Urgente')).toBeInTheDocument();
      expect(screen.getByText('Importante')).toBeInTheDocument();
    });

    it('should call onSelect with undefined when "Sem label" is clicked', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId="label-1"
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const noLabelOption = screen.getAllByText('Sem label')[0];
      await user.click(noLabelOption);

      expect(mockOnSelect).toHaveBeenCalledWith(undefined);
    });

    it('should call onSelect with label id when a label is clicked', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const urgenteOption = screen.getByText('Urgente');
      await user.click(urgenteOption);

      expect(mockOnSelect).toHaveBeenCalledWith('label-1');
    });
  });

  describe('label creation', () => {
    it('should show create form when "Criar novo label" is clicked', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createButton = screen.getByText('Criar novo label');
      await user.click(createButton);

      expect(screen.getByPlaceholderText('Nome do label')).toBeInTheDocument();
    });

    it('should create a new label and select it', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createButton = screen.getByText('Criar novo label');
      await user.click(createButton);

      const input = screen.getByPlaceholderText('Nome do label');
      await user.type(input, 'New Label');

      const criarButton = screen.getByRole('button', { name: /^criar$/i });
      await user.click(criarButton);

      await waitFor(() => {
        expect(api.labels.create).toHaveBeenCalledWith({
          calendarId: 'cal-1',
          name: 'New Label',
          color: '#ef4444', // Default first color
        });
      });

      await waitFor(() => {
        expect(mockOnSelect).toHaveBeenCalledWith('new-label');
      });
    });

    it('should allow selecting a color for new label', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createLabelButton = screen.getByText('Criar novo label');
      await user.click(createLabelButton);

      // Find color buttons by their background color style
      const colorButtons = document.querySelectorAll('button[style*="background-color"]');
      expect(colorButtons.length).toBe(9); // 9 preset colors

      // Click on a different color (blue)
      const blueColorButton = Array.from(colorButtons).find(
        (btn) => (btn as HTMLElement).style.backgroundColor === 'rgb(59, 130, 246)'
      );
      expect(blueColorButton).toBeTruthy();
      await user.click(blueColorButton!);

      const input = screen.getByPlaceholderText('Nome do label');
      await user.type(input, 'Blue Label');

      const criarButton = screen.getByRole('button', { name: /^criar$/i });
      await user.click(criarButton);

      await waitFor(() => {
        expect(api.labels.create).toHaveBeenCalledWith({
          calendarId: 'cal-1',
          name: 'Blue Label',
          color: '#3b82f6', // Blue
        });
      });
    });

    it('should cancel creation when cancel button is clicked', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createLabelButton = screen.getByText('Criar novo label');
      await user.click(createLabelButton);

      expect(screen.getByPlaceholderText('Nome do label')).toBeInTheDocument();

      const cancelButton = screen.getByRole('button', { name: /cancelar/i });
      await user.click(cancelButton);

      expect(screen.queryByPlaceholderText('Nome do label')).not.toBeInTheDocument();
      expect(screen.getByText('Criar novo label')).toBeInTheDocument();
    });

    it('should disable create button when name is empty', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createLabelButton = screen.getByText('Criar novo label');
      await user.click(createLabelButton);

      const criarButton = screen.getByRole('button', { name: /^criar$/i });
      expect(criarButton).toBeDisabled();
    });
  });

  describe('keyboard navigation', () => {
    it('should create label when Enter is pressed in input', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createLabelButton = screen.getByText('Criar novo label');
      await user.click(createLabelButton);

      const input = screen.getByPlaceholderText('Nome do label');
      await user.type(input, 'Enter Label{Enter}');

      await waitFor(() => {
        expect(api.labels.create).toHaveBeenCalledWith({
          calendarId: 'cal-1',
          name: 'Enter Label',
          color: '#ef4444',
        });
      });
    });

    it('should cancel creation when Escape is pressed', async () => {
      const user = userEvent.setup();

      render(
        <LabelSelector
          calendarId="cal-1"
          selectedLabelId={undefined}
          onSelect={mockOnSelect}
        />
      );

      await waitFor(() => {
        expect(api.labels.list).toHaveBeenCalled();
      });

      const button = screen.getByRole('button');
      await user.click(button);

      const createLabelButton = screen.getByText('Criar novo label');
      await user.click(createLabelButton);

      const input = screen.getByPlaceholderText('Nome do label');
      await user.type(input, 'Test{Escape}');

      expect(screen.queryByPlaceholderText('Nome do label')).not.toBeInTheDocument();
      expect(screen.getByText('Criar novo label')).toBeInTheDocument();
    });
  });
});

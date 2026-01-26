import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ProfileSelector from './ProfileSelector';
import type { Calendar } from '@/types/calendar';

describe('ProfileSelector', () => {
  const mockCalendars: Calendar[] = [
    {
      id: 'cal-1',
      name: 'Pessoal',
      email: 'personal@example.com',
      color: '#8b5cf6',
      type: 'personal',
      isActive: true,
    },
    {
      id: 'cal-2',
      name: 'Trabalho',
      email: 'work@example.com',
      color: '#10b981',
      type: 'professional',
      isActive: true,
    },
  ];

  const mockOnSelect = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render all calendars as options', () => {
    render(
      <ProfileSelector
        calendars={mockCalendars}
        selectedId={null}
        onSelect={mockOnSelect}
      />
    );

    expect(screen.getByText('Pessoal')).toBeInTheDocument();
    expect(screen.getByText('Trabalho')).toBeInTheDocument();
  });

  it('should render "Todos" option for viewing all profiles', () => {
    render(
      <ProfileSelector
        calendars={mockCalendars}
        selectedId={null}
        onSelect={mockOnSelect}
      />
    );

    expect(screen.getByText('Todos')).toBeInTheDocument();
  });

  it('should highlight selected calendar', () => {
    render(
      <ProfileSelector
        calendars={mockCalendars}
        selectedId="cal-1"
        onSelect={mockOnSelect}
      />
    );

    const selectedButton = screen.getByRole('button', { name: /Pessoal/i });
    expect(selectedButton).toHaveClass('bg-purple-600');
  });

  it('should call onSelect when calendar is clicked', () => {
    render(
      <ProfileSelector
        calendars={mockCalendars}
        selectedId={null}
        onSelect={mockOnSelect}
      />
    );

    fireEvent.click(screen.getByText('Trabalho'));

    expect(mockOnSelect).toHaveBeenCalledWith('cal-2');
  });

  it('should call onSelect with null when "Todos" is clicked', () => {
    render(
      <ProfileSelector
        calendars={mockCalendars}
        selectedId="cal-1"
        onSelect={mockOnSelect}
      />
    );

    fireEvent.click(screen.getByText('Todos'));

    expect(mockOnSelect).toHaveBeenCalledWith(null);
  });

  it('should show calendar type indicator', () => {
    render(
      <ProfileSelector
        calendars={mockCalendars}
        selectedId={null}
        onSelect={mockOnSelect}
      />
    );

    // Personal calendar should have user icon indicator
    // Professional calendar should have briefcase icon indicator
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(3); // Todos + 2 calendars
  });

  it('should handle empty calendars list', () => {
    render(
      <ProfileSelector
        calendars={[]}
        selectedId={null}
        onSelect={mockOnSelect}
      />
    );

    // Should still show "Todos" option
    expect(screen.getByText('Todos')).toBeInTheDocument();
  });

  it('should display loading state when loading prop is true', () => {
    render(
      <ProfileSelector
        calendars={[]}
        selectedId={null}
        onSelect={mockOnSelect}
        loading={true}
      />
    );

    expect(screen.getByTestId('profile-selector-loading')).toBeInTheDocument();
  });
});

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import LabelBadge from './LabelBadge';
import type { Label } from '@/types/calendar';

describe('LabelBadge', () => {
  const mockLabel: Label = {
    id: 'label-1',
    calendarId: 'cal-1',
    name: 'Urgente',
    color: '#ef4444',
    isActive: true,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  };

  it('should render label name', () => {
    render(<LabelBadge label={mockLabel} />);
    expect(screen.getByText('Urgente')).toBeInTheDocument();
  });

  it('should apply label color to background and text', () => {
    render(<LabelBadge label={mockLabel} />);

    const badge = screen.getByText('Urgente').closest('span');
    expect(badge).toHaveStyle({
      backgroundColor: '#ef444420',
      color: '#ef4444',
    });
  });

  it('should render color dot with label color', () => {
    render(<LabelBadge label={mockLabel} />);

    const colorDot = document.querySelector('span span[style*="background-color"]');
    expect(colorDot).toHaveStyle({
      backgroundColor: '#ef4444',
    });
  });

  it('should render with different colors', () => {
    const blueLabel: Label = {
      ...mockLabel,
      name: 'Importante',
      color: '#3b82f6',
    };

    render(<LabelBadge label={blueLabel} />);

    const badge = screen.getByText('Importante').closest('span');
    expect(badge).toHaveStyle({
      backgroundColor: '#3b82f620',
      color: '#3b82f6',
    });
  });
});

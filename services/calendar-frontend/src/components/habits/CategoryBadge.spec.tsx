import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import CategoryBadge from './CategoryBadge';
import type { Category } from '@/types/calendar';

describe('CategoryBadge', () => {
  const mockCategory: Category = {
    id: 'cat-1',
    calendarId: 'cal-1',
    name: 'LinkedIn',
    icon: '💼',
    color: '#0077B5',
    isActive: true,
  };

  it('should render category name', () => {
    render(<CategoryBadge category={mockCategory} />);
    expect(screen.getByText('LinkedIn')).toBeInTheDocument();
  });

  it('should render category icon when provided', () => {
    render(<CategoryBadge category={mockCategory} />);
    expect(screen.getByText('💼')).toBeInTheDocument();
  });

  it('should apply category color as background', () => {
    render(<CategoryBadge category={mockCategory} />);
    const badge = screen.getByTestId('category-badge');
    expect(badge).toHaveStyle({ backgroundColor: '#0077B5' });
  });

  it('should render without icon when icon is empty', () => {
    const categoryWithoutIcon: Category = {
      ...mockCategory,
      icon: '',
    };
    render(<CategoryBadge category={categoryWithoutIcon} />);
    expect(screen.getByText('LinkedIn')).toBeInTheDocument();
    expect(screen.queryByTestId('category-icon')).not.toBeInTheDocument();
  });

  it('should return null when category is undefined', () => {
    const { container } = render(<CategoryBadge category={undefined} />);
    expect(container.firstChild).toBeNull();
  });

  it('should have correct styling classes', () => {
    render(<CategoryBadge category={mockCategory} />);
    const badge = screen.getByTestId('category-badge');
    expect(badge).toHaveClass('rounded');
    expect(badge).toHaveClass('text-xs');
  });

  it('should handle category with special characters in name', () => {
    const categoryWithSpecialName: Category = {
      ...mockCategory,
      name: 'WB Digital & Solutions',
    };
    render(<CategoryBadge category={categoryWithSpecialName} />);
    expect(screen.getByText('WB Digital & Solutions')).toBeInTheDocument();
  });

  it('should be compact by default', () => {
    render(<CategoryBadge category={mockCategory} />);
    const badge = screen.getByTestId('category-badge');
    expect(badge).toHaveClass('px-2');
    expect(badge).toHaveClass('py-0.5');
  });
});

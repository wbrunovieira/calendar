'use client';

import type { Category } from '@/types/calendar';

interface CategoryBadgeProps {
  category?: Category;
}

export default function CategoryBadge({ category }: CategoryBadgeProps) {
  if (!category) {
    return null;
  }

  return (
    <span
      data-testid="category-badge"
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs text-white"
      style={{ backgroundColor: category.color }}
    >
      {category.icon && (
        <span data-testid="category-icon">{category.icon}</span>
      )}
      <span>{category.name}</span>
    </span>
  );
}

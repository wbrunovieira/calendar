'use client';

import type { CategoryType } from '@/types/calendar';

interface CategoryTypeBadgeProps {
  categoryType?: CategoryType;
}

export default function CategoryTypeBadge({ categoryType }: CategoryTypeBadgeProps) {
  if (!categoryType) {
    return null;
  }

  return (
    <span
      data-testid="category-type-badge"
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs text-white"
      style={{ backgroundColor: categoryType.color }}
    >
      {categoryType.icon && (
        <span data-testid="category-type-icon">{categoryType.icon}</span>
      )}
      <span>{categoryType.name}</span>
    </span>
  );
}

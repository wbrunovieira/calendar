'use client';

import type { Label } from '@/types/calendar';

interface LabelBadgeProps {
  label: Label;
}

export default function LabelBadge({ label }: LabelBadgeProps) {
  return (
    <span
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
      style={{
        backgroundColor: `${label.color}20`,
        color: label.color,
        borderColor: `${label.color}40`,
        borderWidth: '1px',
      }}
    >
      <span
        className="w-2 h-2 rounded-full"
        style={{ backgroundColor: label.color }}
      />
      {label.name}
    </span>
  );
}

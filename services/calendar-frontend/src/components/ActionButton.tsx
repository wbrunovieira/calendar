/**
 * Action Button Component
 * Reusable button for recurring event actions
 * Supports different visual variants (default, delete)
 */

import { ReactNode } from 'react';

interface ActionButtonProps {
  icon: ReactNode;
  iconColor: string;
  title: string;
  description: string;
  onClick: () => void;
  variant?: 'default' | 'delete';
}

const VARIANT_STYLES = {
  default: 'hover:bg-white/20 hover:border-white/30',
  delete: 'hover:bg-red-500/20 hover:border-red-500/50',
};

export default function ActionButton({
  icon,
  iconColor,
  title,
  description,
  onClick,
  variant = 'default',
}: ActionButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`w-full p-4 bg-white/10 border border-white/20 rounded-lg text-left transition-colors group ${VARIANT_STYLES[variant]}`}
    >
      <div className="flex items-start gap-3">
        <div className={`mt-1 flex-shrink-0 ${iconColor}`}>{icon}</div>
        <div className="flex-1">
          <div className="text-white font-semibold mb-1">{title}</div>
          <div className="text-white/70 text-sm">{description}</div>
        </div>
      </div>
    </button>
  );
}

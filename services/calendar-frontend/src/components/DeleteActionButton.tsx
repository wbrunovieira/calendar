/**
 * Delete Action Button Component
 * Reusable button for delete recurring event actions
 */

import { ReactNode } from 'react';

interface DeleteActionButtonProps {
  icon: ReactNode;
  iconColor: string;
  title: string;
  description: string;
  onClick: () => void;
}

export default function DeleteActionButton({
  icon,
  iconColor,
  title,
  description,
  onClick,
}: DeleteActionButtonProps) {
  return (
    <button
      onClick={onClick}
      className="w-full p-4 bg-white/10 hover:bg-red-500/20 border border-white/20 hover:border-red-500/50 rounded-lg text-left transition-colors group"
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

/**
 * Navigation Button Component
 * Reusable navigation button for calendar header
 */

interface NavigationButtonProps {
  direction: 'previous' | 'next';
  onClick: () => void;
  label: string;
}

export default function NavigationButton({ direction, onClick, label }: NavigationButtonProps) {
  const isPrevious = direction === 'previous';

  return (
    <button
      onClick={onClick}
      className="p-1 hover:bg-white/20 rounded transition-all duration-200"
      aria-label={label}
    >
      <svg className="w-4 h-4 md:w-5 md:h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d={isPrevious ? 'M15 19l-7-7 7-7' : 'M9 5l7 7-7 7'}
        />
      </svg>
    </button>
  );
}

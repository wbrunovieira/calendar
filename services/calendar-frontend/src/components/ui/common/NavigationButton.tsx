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
      className="p-2 hover:bg-white/20 rounded-lg transition-all duration-300 hover:scale-110 hover:shadow-md bg-white/10 border border-white/5 backdrop-blur-sm"
      aria-label={label}
    >
      <svg className="w-5 h-5 md:w-6 md:h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2.5}>
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d={isPrevious ? 'M15 19l-7-7 7-7' : 'M9 5l7 7-7 7'}
        />
      </svg>
    </button>
  );
}

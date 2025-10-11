/**
 * Floating Add Button Component
 * Fixed position button to create new events
 */

interface FloatingAddButtonProps {
  onClick: () => void;
}

export default function FloatingAddButton({ onClick }: FloatingAddButtonProps) {
  return (
    <button
      onClick={onClick}
      className="fixed top-6 right-6 w-14 h-14 bg-gradient-to-br from-[#792990] to-[#350545] hover:from-[#8b2fa0] hover:to-[#461556] text-white rounded-full shadow-2xl hover:shadow-[#792990]/50 transition-all duration-300 flex items-center justify-center z-50 hover:scale-110"
      title="Criar novo evento"
      aria-label="Criar novo evento"
    >
      <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
      </svg>
    </button>
  );
}

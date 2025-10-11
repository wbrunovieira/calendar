/**
 * Modal Header Component
 * Reusable header for modal dialogs
 */

interface ModalHeaderProps {
  title: string;
  onClose: () => void;
}

export default function ModalHeader({ title, onClose }: ModalHeaderProps) {
  return (
    <div className="sticky top-0 bg-gradient-to-r from-[#350545] to-[#792990] text-white px-6 py-4 flex items-center justify-between border-b border-white/10">
      <h2 className="text-2xl font-bold">{title}</h2>
      <button
        onClick={onClose}
        className="text-white hover:bg-white/20 rounded-full p-2 transition-colors"
        aria-label="Fechar"
      >
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  );
}

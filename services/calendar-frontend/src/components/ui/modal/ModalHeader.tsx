/**
 * Modal Header Component
 * Reusable header for modal dialogs
 */

interface ModalHeaderProps {
  title: string;
  onClose: () => void;
  variant?: 'default' | 'danger';
  size?: 'default' | 'small';
}

const VARIANT_STYLES = {
  default: 'bg-gradient-to-r from-[#350545] to-[#792990]',
  danger: 'bg-gradient-to-r from-red-900 to-red-700',
};

const SIZE_STYLES = {
  default: 'text-2xl',
  small: 'text-xl',
};

export default function ModalHeader({
  title,
  onClose,
  variant = 'default',
  size = 'default',
}: ModalHeaderProps) {
  return (
    <div
      className={`sticky top-0 ${VARIANT_STYLES[variant]} text-white px-6 py-4 flex items-center justify-between border-b border-white/10 ${variant === 'danger' ? 'rounded-t-2xl' : ''}`}
    >
      <h2 className={`${SIZE_STYLES[size]} font-bold`}>{title}</h2>
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

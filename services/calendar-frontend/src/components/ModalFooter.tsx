/**
 * Modal Footer Component
 * Reusable footer with action buttons for modal dialogs
 */

interface ModalFooterProps {
  onCancel: () => void;
  onSubmit?: () => void;
  submitText?: string;
  cancelText?: string;
  loading?: boolean;
  loadingText?: string;
  submitType?: 'button' | 'submit';
}

export default function ModalFooter({
  onCancel,
  onSubmit,
  submitText = 'Confirmar',
  cancelText = 'Cancelar',
  loading = false,
  loadingText = 'Carregando...',
  submitType = 'button',
}: ModalFooterProps) {
  return (
    <div className="flex gap-3 pt-2">
      <button
        type="button"
        onClick={onCancel}
        className="flex-1 px-6 py-3 bg-white/10 text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/20"
      >
        {cancelText}
      </button>
      <button
        type={submitType}
        onClick={submitType === 'button' ? onSubmit : undefined}
        disabled={loading}
        className="flex-1 px-6 py-3 bg-gradient-to-r from-[#792990] to-[#350545] text-white rounded-lg font-semibold hover:opacity-90 transition-opacity disabled:opacity-50"
      >
        {loading ? loadingText : submitText}
      </button>
    </div>
  );
}

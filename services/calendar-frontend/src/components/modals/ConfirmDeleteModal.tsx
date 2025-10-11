'use client';

import { Event } from '@/types/calendar';

interface ConfirmDeleteModalProps {
  isOpen: boolean;
  event: Event | null;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDeleteModal({
  isOpen,
  event,
  onConfirm,
  onCancel,
}: ConfirmDeleteModalProps) {
  if (!isOpen || !event) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-[#350545] rounded-2xl shadow-2xl max-w-md w-full">
        <div className="bg-gradient-to-r from-[#350545] to-[#792990] text-white px-6 py-4 border-b border-white/10">
          <h2 className="text-xl font-bold">Confirmar Exclusão</h2>
        </div>

        <div className="p-6">
          <p className="text-white mb-4">
            Tem certeza que deseja deletar o evento:
          </p>
          <div className="bg-white/10 rounded-lg p-4 mb-6">
            <p className="text-white font-semibold text-lg">{event.title}</p>
            {event.description && (
              <p className="text-white/70 text-sm mt-1">{event.description}</p>
            )}
            <p className="text-white/50 text-xs mt-2">
              {event.startDate.split('T')[0]} às {event.startTime}
            </p>
          </div>

          <div className="flex gap-3">
            <button
              onClick={onCancel}
              className="flex-1 px-6 py-3 bg-white/10 text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/20"
            >
              Cancelar
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 px-6 py-3 bg-red-600 text-white rounded-lg font-semibold hover:bg-red-700 transition-colors"
            >
              Deletar
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

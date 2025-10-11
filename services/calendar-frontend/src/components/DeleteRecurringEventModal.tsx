'use client';

import ModalContainer from './ModalContainer';
import ModalHeader from './ModalHeader';
import ActionButton from './ActionButton';

export type DeleteRecurringEventAction = 'this' | 'all' | 'future';

interface DeleteRecurringEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (action: DeleteRecurringEventAction) => void;
  eventTitle: string;
}

export default function DeleteRecurringEventModal({
  isOpen,
  onClose,
  onSelect,
  eventTitle,
}: DeleteRecurringEventModalProps) {
  return (
    <ModalContainer isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Deletar Evento Recorrente" onClose={onClose} variant="danger" size="small" />

      <div className="p-6 space-y-4">
        <div className="text-white mb-4">
          <p className="text-sm opacity-80">Você está deletando um evento recorrente:</p>
          <p className="font-semibold mt-1 text-lg">{eventTitle}</p>
        </div>

        <div className="space-y-3">
          <ActionButton
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>
            }
            iconColor="text-blue-400"
            title="Apenas este evento"
            description="Deleta somente esta ocorrência específica"
            onClick={() => onSelect('this')}
            variant="delete"
          />

          <ActionButton
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
              </svg>
            }
            iconColor="text-yellow-400"
            title="Este e os próximos eventos"
            description="Deleta esta ocorrência e todas as futuras (não deleta ocorrências anteriores)"
            onClick={() => onSelect('future')}
            variant="delete"
          />

          <ActionButton
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            }
            iconColor="text-red-400"
            title="Todos os eventos"
            description="Deleta todas as ocorrências (passadas, presente e futuras)"
            onClick={() => onSelect('all')}
            variant="delete"
          />
        </div>

        <div className="pt-4 border-t border-white/10">
          <button
            onClick={onClose}
            className="w-full px-6 py-3 bg-white/10 text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/20"
          >
            Cancelar
          </button>
        </div>
      </div>
    </ModalContainer>
  );
}

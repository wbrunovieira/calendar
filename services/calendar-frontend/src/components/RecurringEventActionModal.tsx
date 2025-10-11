'use client';

import ModalContainer from './ModalContainer';
import ModalHeader from './ModalHeader';
import ActionButton from './ActionButton';

export type RecurringEventAction = 'this' | 'all' | 'future';

interface RecurringEventActionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (action: RecurringEventAction) => void;
  eventTitle: string;
}

export default function RecurringEventActionModal({
  isOpen,
  onClose,
  onSelect,
  eventTitle,
}: RecurringEventActionModalProps) {
  return (
    <ModalContainer isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Editar Evento Recorrente" onClose={onClose} size="small" />

      <div className="p-6 space-y-4">
        <div className="text-white mb-4">
          <p className="text-sm opacity-80">Você está editando um evento recorrente:</p>
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
            description="Modifica somente esta ocorrência específica"
            onClick={() => onSelect('this')}
          />

          <ActionButton
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
              </svg>
            }
            iconColor="text-yellow-400"
            title="Este e os próximos eventos"
            description="Modifica esta ocorrência e todas as futuras"
            onClick={() => onSelect('future')}
          />

          <ActionButton
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                />
              </svg>
            }
            iconColor="text-green-400"
            title="Todos os eventos"
            description="Modifica todas as ocorrências (passadas e futuras)"
            onClick={() => onSelect('all')}
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

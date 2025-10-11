'use client';

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
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
    >
      <div className="bg-[#350545] rounded-2xl shadow-2xl max-w-md w-full">
        <div className="bg-gradient-to-r from-[#350545] to-[#792990] text-white px-6 py-4 flex items-center justify-between border-b border-white/10 rounded-t-2xl">
          <h2 className="text-xl font-bold">Editar Evento Recorrente</h2>
          <button
            onClick={onClose}
            className="text-white hover:bg-white/20 rounded-full p-2 transition-colors"
            aria-label="Fechar"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div className="text-white mb-4">
            <p className="text-sm opacity-80">Você está editando um evento recorrente:</p>
            <p className="font-semibold mt-1 text-lg">{eventTitle}</p>
          </div>

          <div className="space-y-3">
            <button
              onClick={() => onSelect('this')}
              className="w-full p-4 bg-white/10 hover:bg-white/20 border border-white/20 rounded-lg text-left transition-colors group"
            >
              <div className="flex items-start gap-3">
                <div className="mt-1 flex-shrink-0">
                  <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                </div>
                <div className="flex-1">
                  <div className="text-white font-semibold mb-1">Apenas este evento</div>
                  <div className="text-white/70 text-sm">
                    Modifica somente esta ocorrência específica
                  </div>
                </div>
              </div>
            </button>

            <button
              onClick={() => onSelect('future')}
              className="w-full p-4 bg-white/10 hover:bg-white/20 border border-white/20 rounded-lg text-left transition-colors group"
            >
              <div className="flex items-start gap-3">
                <div className="mt-1 flex-shrink-0">
                  <svg className="w-5 h-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                  </svg>
                </div>
                <div className="flex-1">
                  <div className="text-white font-semibold mb-1">Este e os próximos eventos</div>
                  <div className="text-white/70 text-sm">
                    Modifica esta ocorrência e todas as futuras
                  </div>
                </div>
              </div>
            </button>

            <button
              onClick={() => onSelect('all')}
              className="w-full p-4 bg-white/10 hover:bg-white/20 border border-white/20 rounded-lg text-left transition-colors group"
            >
              <div className="flex items-start gap-3">
                <div className="mt-1 flex-shrink-0">
                  <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </div>
                <div className="flex-1">
                  <div className="text-white font-semibold mb-1">Todos os eventos</div>
                  <div className="text-white/70 text-sm">
                    Modifica todas as ocorrências (passadas e futuras)
                  </div>
                </div>
              </div>
            </button>
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
      </div>
    </div>
  );
}

'use client';

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useRef,
  useEffect,
  type ReactNode,
} from 'react';

/* ─── Types ─────────────────────────────────────────────── */

type ToastType = 'success' | 'error' | 'info' | 'warning';

interface ToastItem {
  id: number;
  message: string;
  type: ToastType;
  leaving: boolean;
}

interface ConfirmState {
  message: string;
  description?: string;
  resolve: (value: boolean) => void;
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType) => void;
  confirm: (message: string, description?: string) => Promise<boolean>;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}

/* ─── Icons ─────────────────────────────────────────────── */

const icons: Record<ToastType, ReactNode> = {
  success: (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 6L9 17l-5-5" />
    </svg>
  ),
  error: (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <path d="M15 9l-6 6M9 9l6 6" />
    </svg>
  ),
  warning: (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0zM12 9v4M12 17h.01" />
    </svg>
  ),
  info: (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <path d="M12 16v-4M12 8h.01" />
    </svg>
  ),
};

const typeStyles: Record<ToastType, { bg: string; border: string; icon: string; bar: string }> = {
  success: {
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    icon: 'text-emerald-400',
    bar: 'bg-emerald-400',
  },
  error: {
    bg: 'bg-rose-500/10',
    border: 'border-rose-500/30',
    icon: 'text-rose-400',
    bar: 'bg-rose-400',
  },
  warning: {
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/30',
    icon: 'text-amber-400',
    bar: 'bg-amber-400',
  },
  info: {
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/30',
    icon: 'text-blue-400',
    bar: 'bg-blue-400',
  },
};

/* ─── Toast Notification ────────────────────────────────── */

function ToastNotification({
  item,
  onDismiss,
}: {
  item: ToastItem;
  onDismiss: (id: number) => void;
}) {
  const s = typeStyles[item.type];
  const progressRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (progressRef.current) {
      progressRef.current.style.transition = 'none';
      progressRef.current.style.width = '100%';
      requestAnimationFrame(() => {
        if (progressRef.current) {
          progressRef.current.style.transition = 'width 3500ms linear';
          progressRef.current.style.width = '0%';
        }
      });
    }
  }, []);

  return (
    <div
      className={`
        relative overflow-hidden
        flex items-center gap-3 px-4 py-3 rounded-xl
        border backdrop-blur-xl shadow-2xl shadow-black/40
        ${s.bg} ${s.border}
        ${item.leaving
          ? 'animate-[toastOut_300ms_ease-in_forwards]'
          : 'animate-[toastIn_400ms_cubic-bezier(0.16,1,0.3,1)_forwards]'
        }
      `}
    >
      <span className={s.icon}>{icons[item.type]}</span>
      <span className="text-white/90 text-sm font-medium flex-1">{item.message}</span>
      <button
        onClick={() => onDismiss(item.id)}
        className="text-white/30 hover:text-white/70 transition-colors ml-2"
      >
        <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
      <div className="absolute bottom-0 left-0 right-0 h-[2px] bg-white/5">
        <div ref={progressRef} className={`h-full ${s.bar} opacity-60`} />
      </div>
    </div>
  );
}

/* ─── Confirm Dialog ────────────────────────────────────── */

function ConfirmDialog({
  state,
  onClose,
}: {
  state: ConfirmState;
  onClose: (result: boolean) => void;
}) {
  const [leaving, setLeaving] = useState(false);

  const handleClose = useCallback(
    (result: boolean) => {
      setLeaving(true);
      setTimeout(() => onClose(result), 200);
    },
    [onClose],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') handleClose(false);
      if (e.key === 'Enter') handleClose(true);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [handleClose]);

  return (
    <div
      className={`fixed inset-0 z-[9999] flex items-center justify-center p-4
        ${leaving ? 'animate-[fadeOut_200ms_ease-in_forwards]' : 'animate-[fadeIn_200ms_ease-out_forwards]'}
      `}
    >
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => handleClose(false)} />
      <div
        className={`
          relative w-full max-w-sm rounded-2xl border border-white/10
          bg-[#141414] shadow-2xl shadow-black/60
          ${leaving ? 'animate-[dialogOut_200ms_ease-in_forwards]' : 'animate-[dialogIn_300ms_cubic-bezier(0.16,1,0.3,1)_forwards]'}
        `}
      >
        <div className="p-6">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-full bg-rose-500/15 flex items-center justify-center">
              <svg className="w-5 h-5 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                <path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </div>
            <h3 className="text-white font-semibold text-lg">Confirmar exclusao</h3>
          </div>
          <p className="text-white/80 text-sm mb-1">{state.message}</p>
          {state.description && (
            <p className="text-white/40 text-xs">{state.description}</p>
          )}
        </div>
        <div className="flex gap-3 px-6 pb-6">
          <button
            onClick={() => handleClose(false)}
            className="flex-1 px-4 py-2.5 rounded-xl text-sm font-medium
              bg-white/5 border border-white/10 text-white/70
              hover:bg-white/10 hover:text-white transition-all"
          >
            Cancelar
          </button>
          <button
            onClick={() => handleClose(true)}
            className="flex-1 px-4 py-2.5 rounded-xl text-sm font-medium
              bg-rose-500/20 border border-rose-500/30 text-rose-300
              hover:bg-rose-500/30 hover:text-rose-200 transition-all"
          >
            Excluir
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Provider ──────────────────────────────────────────── */

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [confirmState, setConfirmState] = useState<ConfirmState | null>(null);
  const nextId = useRef(0);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.map((t) => (t.id === id ? { ...t, leaving: true } : t)));
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 300);
  }, []);

  const toast = useCallback(
    (message: string, type: ToastType = 'success') => {
      const id = nextId.current++;
      setToasts((prev) => [...prev, { id, message, type, leaving: false }]);
      setTimeout(() => dismiss(id), 4000);
    },
    [dismiss],
  );

  const confirm = useCallback(
    (message: string, description?: string): Promise<boolean> =>
      new Promise((resolve) => {
        setConfirmState({ message, description, resolve });
      }),
    [],
  );

  const handleConfirmClose = useCallback(
    (result: boolean) => {
      confirmState?.resolve(result);
      setConfirmState(null);
    },
    [confirmState],
  );

  return (
    <ToastContext.Provider value={{ toast, confirm }}>
      {children}

      {/* Toast stack */}
      <div className="fixed bottom-6 right-6 z-[9998] flex flex-col-reverse gap-2 max-w-sm w-full pointer-events-none">
        {toasts.map((item) => (
          <div key={item.id} className="pointer-events-auto">
            <ToastNotification item={item} onDismiss={dismiss} />
          </div>
        ))}
      </div>

      {/* Confirm dialog */}
      {confirmState && (
        <ConfirmDialog state={confirmState} onClose={handleConfirmClose} />
      )}

      {/* Keyframe animations */}
      <style jsx global>{`
        @keyframes toastIn {
          from { opacity: 0; transform: translateY(16px) scale(0.95); }
          to { opacity: 1; transform: translateY(0) scale(1); }
        }
        @keyframes toastOut {
          from { opacity: 1; transform: translateX(0) scale(1); }
          to { opacity: 0; transform: translateX(100px) scale(0.9); }
        }
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @keyframes fadeOut {
          from { opacity: 1; }
          to { opacity: 0; }
        }
        @keyframes dialogIn {
          from { opacity: 0; transform: scale(0.9) translateY(10px); }
          to { opacity: 1; transform: scale(1) translateY(0); }
        }
        @keyframes dialogOut {
          from { opacity: 1; transform: scale(1); }
          to { opacity: 0; transform: scale(0.9) translateY(10px); }
        }
      `}</style>
    </ToastContext.Provider>
  );
}

'use client';

/**
 * Shown in place of page content when the gateway wants a login.
 *
 * It replaces the content rather than sitting above it because the data behind
 * it is not merely stale — it was never received. Rendering an empty dashboard
 * with a warning on top is how a logged-out session came to read as an empty
 * account.
 */
export default function SessionExpired({ loginUrl }: { loginUrl: string }) {
  return (
    <div className="flex justify-center py-16">
      <div className="w-full max-w-lg bg-white/10 border border-amber-400/30 rounded-2xl p-8 text-center">
        <div className="text-4xl mb-4" aria-hidden="true">
          🔒
        </div>
        <h2 className="text-xl font-bold text-white mb-2">Sessão expirada</h2>
        <p className="text-white/70 text-sm leading-relaxed mb-6">
          Seus dados estão intactos — o acesso é que precisa ser renovado. A sessão dura
          12&nbsp;horas e a sua venceu, então o servidor parou de responder às consultas desta
          tela.
        </p>
        <a
          href={loginUrl}
          className="inline-flex items-center gap-2 px-5 py-2.5 bg-emerald-500 hover:bg-emerald-400 text-emerald-950 font-semibold rounded-lg transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h5a3 3 0 013 3v1"
            />
          </svg>
          Entrar novamente
        </a>
        <p className="text-white/40 text-xs mt-5">
          Depois de entrar você volta direto para esta página. Se a tela continuar vazia,
          recarregue com Cmd/Ctrl&nbsp;+&nbsp;Shift&nbsp;+&nbsp;R para descartar o cache.
        </p>
      </div>
    </div>
  );
}

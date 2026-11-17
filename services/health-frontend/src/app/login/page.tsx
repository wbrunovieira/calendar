'use client';

import { Suspense, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { safeNextPath } from '@/lib/auth/authorize';

function LoginForm() {
  const params = useSearchParams();
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password, code }),
      });
      if (res.ok) {
        window.location.assign(safeNextPath(params.get('next')));
        return;
      }
      const body = await res.json().catch(() => null);
      setError(body?.message ?? 'Não foi possível entrar');
      setCode('');
    } catch {
      setError('Não foi possível entrar');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="w-full max-w-sm space-y-4 rounded-2xl border border-white/10 bg-white/5 p-8 shadow-xl backdrop-blur"
    >
      <h1 className="text-xl font-semibold text-white">Entrar</h1>
      <label className="block space-y-1">
        <span className="text-sm text-white/70">Senha</span>
        <input
          type="password"
          autoComplete="current-password"
          required
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full rounded-lg border border-white/15 bg-black/20 px-3 py-2 text-white outline-none focus:border-purple-400"
        />
      </label>
      <label className="block space-y-1">
        <span className="text-sm text-white/70">Código do autenticador</span>
        <input
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="\d{6}"
          maxLength={6}
          required
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
          className="w-full rounded-lg border border-white/15 bg-black/20 px-3 py-2 tracking-[0.4em] text-white outline-none focus:border-purple-400"
        />
      </label>
      {error && (
        <p role="alert" className="text-sm text-red-300">
          {error}
        </p>
      )}
      <button
        type="submit"
        disabled={submitting}
        className="w-full rounded-lg bg-purple-600 py-2 font-medium text-white transition hover:bg-purple-500 disabled:opacity-50"
      >
        {submitting ? 'Entrando…' : 'Entrar'}
      </button>
    </form>
  );
}

export default function LoginPage() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gradient-to-br from-[#1a0b2e] to-[#2d1450] p-4">
      <Suspense>
        <LoginForm />
      </Suspense>
    </main>
  );
}

'use client';

import { ReactNode } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

interface AppLayoutProps {
  children: ReactNode;
}

const navItems = [
  { href: '/', label: 'Dashboard', icon: '📊' },
  { href: '/wim-hof', label: 'Wim Hof', icon: '🧊' },
  { href: '/alongamento', label: 'Alongamento', icon: '🤸' },
  { href: '/workouts', label: 'Treinos', icon: '🏋️' },
  { href: '/activities', label: 'Atividades', icon: '🧘' },
  { href: '/exercises', label: 'Exercicios', icon: '💪' },
  { href: '/records', label: 'Records', icon: '🏆' },
  { href: '/configuracoes', label: 'Config', icon: '⚙️' },
];

export default function AppLayout({ children }: AppLayoutProps) {
  const pathname = usePathname();

  return (
    <div className="min-h-screen bg-gradient-to-br from-violet-900 via-purple-900 to-indigo-900">
      {/* Header */}
      <header className="bg-black/20 backdrop-blur-sm border-b border-white/10">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="text-3xl">💪</div>
              <div>
                <h1 className="text-2xl font-bold text-white">Health</h1>
                <p className="text-white/60 text-sm">Treinos, atividades e performance</p>
              </div>
            </div>
            <a
              href="http://localhost:3002"
              className="flex items-center gap-2 px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg transition-all border border-white/20"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
              </svg>
              <span className="text-sm font-semibold">Voltar ao Calendar</span>
            </a>
          </div>
          <nav className="mt-4 flex flex-wrap gap-2">
            {navItems.map((item) => {
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border transition-colors ${
                    isActive
                      ? 'bg-violet-500/30 border-violet-400/50 text-violet-200 font-medium'
                      : 'border-white/15 text-white/80 hover:bg-white/10'
                  }`}
                >
                  <span>{item.icon}</span>
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-6 py-6">
        {children}
      </main>
    </div>
  );
}

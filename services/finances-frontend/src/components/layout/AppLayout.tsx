'use client';

import { ReactNode } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

export type ProfileTheme = 'personal' | 'business';

interface AppLayoutProps {
  children: ReactNode;
  profileTheme?: ProfileTheme;
  profileName?: string;
}

const navItems = [
  { href: '/', label: 'Lancamentos' },
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/categories', label: 'Categorias' },
  { href: '/recurring', label: 'Fixas' },
  { href: '/budgets', label: 'Orcamentos' },
  { href: '/plan', label: 'Planejamento' },
  { href: '/visao', label: 'Visao Mensal' },
  { href: '/configuracoes', label: 'Configuracoes' },
];

// Theme configuration per profile type
const themeConfig = {
  personal: {
    gradient: 'from-emerald-900 via-teal-900 to-cyan-900',
    border: 'border-l-emerald-500',
    accent: 'emerald',
    accentBg: 'bg-emerald-500/30',
    accentBorder: 'border-emerald-400/50',
    accentText: 'text-emerald-200',
    badgeBg: 'bg-emerald-500/20',
    badgeText: 'text-emerald-300',
    badgeBorder: 'border-emerald-400/30',
  },
  business: {
    gradient: 'from-indigo-900 via-blue-900 to-slate-900',
    border: 'border-l-amber-500',
    accent: 'amber',
    accentBg: 'bg-amber-500/30',
    accentBorder: 'border-amber-400/50',
    accentText: 'text-amber-200',
    badgeBg: 'bg-amber-500/20',
    badgeText: 'text-amber-300',
    badgeBorder: 'border-amber-400/30',
  },
};

export default function AppLayout({ children, profileTheme = 'personal', profileName }: AppLayoutProps) {
  const pathname = usePathname();
  const theme = themeConfig[profileTheme];

  return (
    <div className={`min-h-screen bg-gradient-to-br ${theme.gradient} border-l-4 ${theme.border} transition-all duration-300`}>
      {/* Header */}
      <header className="bg-black/20 backdrop-blur-sm border-b border-white/10">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="text-3xl">💸</div>
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-2xl font-bold text-white">Financas</h1>
                  {/* Profile Badge */}
                  {profileName && (
                    <span className={`px-2.5 py-1 text-xs font-medium rounded-full border flex items-center gap-1.5 ${theme.badgeBg} ${theme.badgeText} ${theme.badgeBorder}`}>
                      {profileTheme === 'personal' ? (
                        <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                        </svg>
                      ) : (
                        <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M4 4a2 2 0 012-2h8a2 2 0 012 2v12a1 1 0 110 2h-3a1 1 0 01-1-1v-2a1 1 0 00-1-1H9a1 1 0 00-1 1v2a1 1 0 01-1 1H4a1 1 0 110-2V4zm3 1h2v2H7V5zm2 4H7v2h2V9zm2-4h2v2h-2V5zm2 4h-2v2h2V9z" clipRule="evenodd" />
                        </svg>
                      )}
                      {profileName}
                    </span>
                  )}
                </div>
                <p className="text-white/60 text-sm">Custos diarios, semanais e mensais</p>
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
                  className={`px-3 py-1.5 text-sm rounded-lg border transition-colors ${
                    isActive
                      ? `${theme.accentBg} ${theme.accentBorder} ${theme.accentText} font-medium`
                      : 'border-white/15 text-white/80 hover:bg-white/10'
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-6">
        {children}
      </main>
    </div>
  );
}

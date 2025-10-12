'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import CalendarIcon from '../ui/common/CalendarIcon';

interface SidebarProps {
  className?: string;
  onToggle?: (collapsed: boolean) => void;
}

export default function Sidebar({ className = '', onToggle }: SidebarProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const pathname = usePathname();

  const toggleSidebar = () => {
    const newState = !isCollapsed;
    setIsCollapsed(newState);
    if (onToggle) {
      onToggle(newState);
    }
  };

  return (
    <aside
      className={`fixed left-0 top-0 h-screen bg-gradient-to-b from-[#350545] via-[#4a0860] to-[#792990] shadow-2xl shadow-purple-900/50 transition-all duration-300 ease-in-out z-50 border-r border-purple-600/20 ${
        isCollapsed ? 'w-20' : 'w-64'
      } ${className}`}
    >
      {/* Header with Logo/Title and Toggle Button */}
      <div className="flex items-center justify-between px-4 py-6 border-b border-white/10">
        {!isCollapsed && (
          <div className="flex items-center gap-3 overflow-hidden">
            <CalendarIcon size="md" />
            <div className="overflow-hidden">
              <h1 className="text-white font-bold text-lg whitespace-nowrap">Calendar</h1>
              <p className="text-white/60 text-xs whitespace-nowrap">Organize sua vida</p>
            </div>
          </div>
        )}

        {isCollapsed && (
          <div className="mx-auto">
            <CalendarIcon size="md" />
          </div>
        )}
      </div>

      {/* Toggle Button */}
      <button
        onClick={toggleSidebar}
        className="absolute -right-3 top-8 w-6 h-6 bg-white/20 hover:bg-white/30 rounded-full flex items-center justify-center backdrop-blur-sm border border-white/10 shadow-md transition-all duration-300 hover:scale-110"
        aria-label={isCollapsed ? 'Expandir sidebar' : 'Recolher sidebar'}
      >
        <svg
          className={`w-3 h-3 text-white transition-transform duration-300 ${
            isCollapsed ? 'rotate-180' : ''
          }`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2.5}
            d="M15 19l-7-7 7-7"
          />
        </svg>
      </button>

      {/* Navigation Items */}
      <nav className="mt-6 px-3">
        <ul className="space-y-2">
          {/* Home Button */}
          <li>
            <Link
              href="/"
              className={`flex items-center gap-3 px-3 py-3 rounded-lg text-white hover:bg-white/30 transition-all duration-300 shadow-md hover:shadow-lg border group relative ${
                pathname === '/'
                  ? 'bg-white/30 border-white/30 shadow-lg'
                  : 'bg-white/10 border-white/10'
              }`}
            >
              {pathname === '/' && (
                <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-white rounded-r-full"></div>
              )}
              <div className="w-6 h-6 flex items-center justify-center flex-shrink-0">
                <svg
                  className="w-6 h-6 transition-transform duration-300 group-hover:scale-110"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
                  />
                </svg>
              </div>
              {!isCollapsed && (
                <span className="font-semibold text-sm whitespace-nowrap overflow-hidden">
                  Home
                </span>
              )}
            </Link>
          </li>

          {/* Finances Button - External App */}
          <li>
            <a
              href="http://localhost:3003"
              target="_self"
              className="flex items-center gap-3 px-3 py-3 rounded-lg text-white hover:bg-white/30 transition-all duration-300 shadow-md hover:shadow-lg border bg-white/10 border-white/10 group relative"
            >
              <div className="w-6 h-6 flex items-center justify-center flex-shrink-0">
                <svg
                  className="w-6 h-6 transition-transform duration-300 group-hover:scale-110"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
              {!isCollapsed && (
                <span className="font-semibold text-sm whitespace-nowrap overflow-hidden">
                  Finanças
                </span>
              )}
              {!isCollapsed && (
                <svg className="w-3 h-3 ml-auto opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              )}
            </a>
          </li>

          {/* Settings Button */}
          <li>
            <Link
              href="/settings"
              className={`flex items-center gap-3 px-3 py-3 rounded-lg text-white hover:bg-white/30 transition-all duration-300 shadow-md hover:shadow-lg border group relative ${
                pathname === '/settings'
                  ? 'bg-white/30 border-white/30 shadow-lg'
                  : 'bg-white/10 border-white/10'
              }`}
            >
              {pathname === '/settings' && (
                <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-white rounded-r-full"></div>
              )}
              <div className="w-6 h-6 flex items-center justify-center flex-shrink-0">
                <svg
                  className="w-6 h-6 transition-transform duration-300 group-hover:scale-110"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                </svg>
              </div>
              {!isCollapsed && (
                <span className="font-semibold text-sm whitespace-nowrap overflow-hidden">
                  Configurações
                </span>
              )}
            </Link>
          </li>
        </ul>
      </nav>

      {/* Footer with collapsed indicator */}
      {!isCollapsed && (
        <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-white/10">
          <div className="text-white/50 text-xs text-center">
            v1.0.0
          </div>
        </div>
      )}
    </aside>
  );
}

'use client';

import { useState } from 'react';
import Link from 'next/link';
import CalendarIcon from '../ui/common/CalendarIcon';

interface SidebarProps {
  className?: string;
  onToggle?: (collapsed: boolean) => void;
}

export default function Sidebar({ className = '', onToggle }: SidebarProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

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
              className="flex items-center gap-3 px-3 py-3 rounded-lg bg-white/20 text-white hover:bg-white/30 transition-all duration-300 shadow-md hover:shadow-lg border border-white/10 group"
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

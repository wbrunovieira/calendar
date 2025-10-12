'use client';

import { useState, ReactNode } from 'react';
import Sidebar from './Sidebar';

interface AppLayoutProps {
  children: ReactNode;
}

export default function AppLayout({ children }: AppLayoutProps) {
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  const handleSidebarToggle = (collapsed: boolean) => {
    setIsSidebarCollapsed(collapsed);
  };

  return (
    <div className="min-h-screen flex relative overflow-hidden">
      {/* Sidebar */}
      <Sidebar onToggle={handleSidebarToggle} />

      {/* Main Content */}
      <div
        className={`flex-1 flex flex-col py-8 px-4 relative overflow-hidden transition-all duration-300 ${
          isSidebarCollapsed ? 'ml-20' : 'ml-64'
        }`}
        style={{
          background: 'linear-gradient(135deg, #350545 0%, #792990 50%, #350545 100%)',
        }}
      >
        {/* Animated gradient overlay */}
        <div
          className="absolute inset-0 opacity-30"
          style={{
            background: 'radial-gradient(circle at 20% 50%, #792990 0%, transparent 50%), radial-gradient(circle at 80% 80%, #350545 0%, transparent 50%)',
          }}
        />

        <div className="max-w-7xl mx-auto w-full flex flex-col flex-1 relative z-10">
          {children}
        </div>
      </div>
    </div>
  );
}

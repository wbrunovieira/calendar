'use client';

import { ReactNode } from 'react';
import { ProfileProvider } from '@/contexts/ProfileContext';
import { ToastProvider } from '@/components/ui/Toast';

interface ProvidersProps {
  children: ReactNode;
}

export default function Providers({ children }: ProvidersProps) {
  return (
    <ProfileProvider>
      <ToastProvider>
        {children}
      </ToastProvider>
    </ProfileProvider>
  );
}

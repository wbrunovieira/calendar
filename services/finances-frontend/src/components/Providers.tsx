'use client';

import { ReactNode } from 'react';
import { ProfileProvider } from '@/contexts/ProfileContext';

interface ProvidersProps {
  children: ReactNode;
}

export default function Providers({ children }: ProvidersProps) {
  return (
    <ProfileProvider>
      {children}
    </ProfileProvider>
  );
}

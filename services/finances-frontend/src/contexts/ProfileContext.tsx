'use client';

import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import { api } from '@/lib/api';
import { sessionStateFromError, ACTIVE_SESSION, type SessionState } from '@/lib/session';
import type { Profile } from '@/types/finances';

interface ProfileContextType {
  profiles: Profile[];
  selectedProfileId: string | null;
  selectedProfile: Profile | null;
  setSelectedProfileId: (id: string) => void;
  isLoading: boolean;
  /**
   * True when the load failed because the gateway wants a login. Without it a
   * screen cannot tell "this account has no profiles" from "you are logged out"
   * and reports the wrong one — which is what every screen did until 07/11/2026.
   */
  authRequired: boolean;
  loginUrl: string;
}

const ProfileContext = createContext<ProfileContextType | undefined>(undefined);

const STORAGE_KEY = 'finances_selected_profile_id';

export function ProfileProvider({ children }: { children: ReactNode }) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileIdState] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [session, setSession] = useState<SessionState>(ACTIVE_SESSION);

  // Load profiles on mount
  useEffect(() => {
    const loadProfiles = async () => {
      try {
        const data = await api.get<{ data: Profile[] }>('/profiles');
        const profileList: Profile[] = data.data || [];
        setProfiles(profileList);

        // Try to restore from localStorage first
        const savedProfileId = localStorage.getItem(STORAGE_KEY);
        if (savedProfileId && profileList.some((p) => p.id === savedProfileId)) {
          setSelectedProfileIdState(savedProfileId);
        } else if (profileList.length > 0) {
          // Default to "Bruno Pessoal" or first profile
          const defaultProfile = profileList.find((p) => p.name === 'Bruno Pessoal') || profileList[0];
          setSelectedProfileIdState(defaultProfile.id);
        }
      } catch (error) {
        console.error('Erro ao carregar perfis:', error);
        setSession(sessionStateFromError(error));
      } finally {
        setIsLoading(false);
      }
    };

    loadProfiles();
  }, []);

  const setSelectedProfileId = useCallback((id: string) => {
    setSelectedProfileIdState(id);
    localStorage.setItem(STORAGE_KEY, id);
  }, []);

  const selectedProfile = profiles.find((p) => p.id === selectedProfileId) || null;

  return (
    <ProfileContext.Provider
      value={{
        profiles,
        selectedProfileId,
        selectedProfile,
        setSelectedProfileId,
        isLoading,
        authRequired: session.authRequired,
        loginUrl: session.loginUrl,
      }}
    >
      {children}
    </ProfileContext.Provider>
  );
}

export function useProfile() {
  const context = useContext(ProfileContext);
  if (context === undefined) {
    throw new Error('useProfile must be used within a ProfileProvider');
  }
  return context;
}

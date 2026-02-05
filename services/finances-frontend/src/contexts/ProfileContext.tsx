'use client';

import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import { API_BASE } from '@/lib/api';

export interface Profile {
  id: string;
  name: string;
  type: 'PERSONAL' | 'BUSINESS';
}

interface ProfileContextType {
  profiles: Profile[];
  selectedProfileId: string | null;
  selectedProfile: Profile | null;
  setSelectedProfileId: (id: string) => void;
  isLoading: boolean;
}

const ProfileContext = createContext<ProfileContextType | undefined>(undefined);

const STORAGE_KEY = 'finances_selected_profile_id';

export function ProfileProvider({ children }: { children: ReactNode }) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileIdState] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Load profiles on mount
  useEffect(() => {
    const loadProfiles = async () => {
      try {
        const response = await fetch(`${API_BASE}/profiles`);
        const data = await response.json();
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

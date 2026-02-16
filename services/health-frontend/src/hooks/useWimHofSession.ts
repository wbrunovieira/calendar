'use client';

import { useState, useRef, useCallback, useEffect } from 'react';
import type { ActivityFormData } from '@/types/health';

export type Phase = 'SETUP' | 'BREATHING' | 'HOLD' | 'RECOVERY' | 'PUSHUPS' | 'SUMMARY';

export interface WimHofSessionState {
  phase: Phase;
  currentRound: number;
  totalRounds: number;
  breathsPerRound: number;
  currentBreath: number;
  holdTimeSeconds: number;
  recoveryTimeSeconds: number;
  roundBreaths: number[];
  retentionTimes: number[];
  pushUpCount: number | null;
  sessionStartTime: Date | null;
  sessionEndTime: Date | null;
}

const RECOVERY_DURATION = 15;

const initialState: WimHofSessionState = {
  phase: 'SETUP',
  currentRound: 1,
  totalRounds: 4,
  breathsPerRound: 30,
  currentBreath: 0,
  holdTimeSeconds: 0,
  recoveryTimeSeconds: RECOVERY_DURATION,
  roundBreaths: [],
  retentionTimes: [],
  pushUpCount: null,
  sessionStartTime: null,
  sessionEndTime: null,
};

export function useWimHofSession() {
  const [state, setState] = useState<WimHofSessionState>(initialState);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const timerStartRef = useRef<number>(0);

  const clearTimer = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => clearTimer();
  }, [clearTimer]);

  const setTotalRounds = useCallback((n: number) => {
    setState(prev => ({ ...prev, totalRounds: Math.max(1, Math.min(10, n)) }));
  }, []);

  const setBreathsPerRound = useCallback((n: number) => {
    setState(prev => ({ ...prev, breathsPerRound: Math.max(10, Math.min(60, n)) }));
  }, []);

  const startRecoveryTimer = useCallback(() => {
    clearTimer();
    setState(prev => ({ ...prev, phase: 'RECOVERY', recoveryTimeSeconds: RECOVERY_DURATION }));
    timerStartRef.current = Date.now();

    intervalRef.current = setInterval(() => {
      const elapsed = (Date.now() - timerStartRef.current) / 1000;
      const remaining = Math.max(0, RECOVERY_DURATION - elapsed);

      if (remaining <= 0) {
        clearTimer();
        setState(prev => {
          const isLastRound = prev.currentRound >= prev.totalRounds;
          if (isLastRound) {
            return { ...prev, phase: 'PUSHUPS', recoveryTimeSeconds: 0 };
          }
          return {
            ...prev,
            phase: 'BREATHING',
            currentRound: prev.currentRound + 1,
            currentBreath: 0,
            recoveryTimeSeconds: 0,
          };
        });
      } else {
        setState(prev => ({ ...prev, recoveryTimeSeconds: Math.ceil(remaining) }));
      }
    }, 100);
  }, [clearTimer]);

  const startSession = useCallback(() => {
    clearTimer();
    setState(prev => ({
      ...prev,
      phase: 'BREATHING',
      currentRound: 1,
      currentBreath: 0,
      holdTimeSeconds: 0,
      recoveryTimeSeconds: RECOVERY_DURATION,
      roundBreaths: [],
      retentionTimes: [],
      pushUpCount: null,
      sessionStartTime: new Date(),
      sessionEndTime: null,
    }));
  }, [clearTimer]);

  const nextBreath = useCallback(() => {
    setState(prev => {
      if (prev.phase !== 'BREATHING') return prev;
      const next = prev.currentBreath + 1;
      if (next >= prev.breathsPerRound) {
        // Last breath — transition to HOLD
        clearTimer();
        timerStartRef.current = Date.now();
        intervalRef.current = setInterval(() => {
          const elapsed = (Date.now() - timerStartRef.current) / 1000;
          setState(s => ({ ...s, holdTimeSeconds: elapsed }));
        }, 100);

        return {
          ...prev,
          phase: 'HOLD',
          currentBreath: next,
          holdTimeSeconds: 0,
          roundBreaths: [...prev.roundBreaths, next],
        };
      }
      return { ...prev, currentBreath: next };
    });
  }, [clearTimer]);

  const endHold = useCallback(() => {
    clearTimer();
    setState(prev => {
      if (prev.phase !== 'HOLD') return prev;
      const retentionSeconds = Math.round(prev.holdTimeSeconds);
      return {
        ...prev,
        retentionTimes: [...prev.retentionTimes, retentionSeconds],
        holdTimeSeconds: retentionSeconds,
      };
    });
    // Start recovery after state update
    startRecoveryTimer();
  }, [clearTimer, startRecoveryTimer]);

  const setPushUpCount = useCallback((n: number) => {
    setState(prev => ({ ...prev, pushUpCount: Math.max(0, n) }));
  }, []);

  const confirmPushUps = useCallback(() => {
    setState(prev => ({
      ...prev,
      phase: 'SUMMARY',
      sessionEndTime: new Date(),
    }));
  }, []);

  const skipPushUps = useCallback(() => {
    setState(prev => ({
      ...prev,
      pushUpCount: null,
      phase: 'SUMMARY',
      sessionEndTime: new Date(),
    }));
  }, []);

  const getSessionData = useCallback((profileId: string, rating?: number, notes?: string): ActivityFormData => {
    const duration = state.sessionStartTime && state.sessionEndTime
      ? Math.round((state.sessionEndTime.getTime() - state.sessionStartTime.getTime()) / 60000)
      : 0;

    const startTime = state.sessionStartTime
      ? state.sessionStartTime.toTimeString().substring(0, 5)
      : undefined;

    return {
      profileId,
      name: `Wim Hof Method - ${state.totalRounds} rounds`,
      activityType: 'WIM_HOF',
      activityDate: new Date().toISOString().split('T')[0],
      startTime,
      durationMinutes: duration || undefined,
      rounds: state.totalRounds,
      rating: rating || undefined,
      notes: notes || undefined,
      metrics: {
        breathingRounds: state.totalRounds,
        breathsPerRound: state.roundBreaths,
        retentionTimes: state.retentionTimes,
        pushUps: state.pushUpCount ?? undefined,
      },
    };
  }, [state]);

  const reset = useCallback(() => {
    clearTimer();
    setState(prev => ({
      ...initialState,
      totalRounds: prev.totalRounds,
      breathsPerRound: prev.breathsPerRound,
    }));
  }, [clearTimer]);

  return {
    state,
    setTotalRounds,
    setBreathsPerRound,
    startSession,
    nextBreath,
    endHold,
    setPushUpCount,
    confirmPushUps,
    skipPushUps,
    getSessionData,
    reset,
  };
}

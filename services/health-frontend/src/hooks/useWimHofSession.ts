'use client';

import { useState, useRef, useCallback, useEffect } from 'react';
import type { ActivityFormData } from '@/types/health';

export type Phase = 'SETUP' | 'BREATHING' | 'HOLD' | 'PUSHUPS' | 'SUMMARY';

export interface WimHofSessionState {
  phase: Phase;
  currentRound: number;
  totalRounds: number;
  breathsPerRound: number;
  currentBreath: number;
  holdTimeSeconds: number;
  roundBreaths: number[];
  retentionTimes: number[];
  pushUpCount: number | null;
  sessionStartTime: Date | null;
  sessionEndTime: Date | null;
}

const BREATH_INTERVAL_MS = 2000;

const initialState: WimHofSessionState = {
  phase: 'SETUP',
  currentRound: 1,
  totalRounds: 4,
  breathsPerRound: 30,
  currentBreath: 0,
  holdTimeSeconds: 0,
  roundBreaths: [],
  retentionTimes: [],
  pushUpCount: null,
  sessionStartTime: null,
  sessionEndTime: null,
};

export function useWimHofSession() {
  const [state, setState] = useState<WimHofSessionState>(initialState);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const holdStartRef = useRef<number>(0);

  const clearTimer = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  // Start timers based on phase changes
  useEffect(() => {
    clearTimer();

    if (state.phase === 'BREATHING' && state.sessionStartTime) {
      // Auto-count breaths every 2s
      setState(prev => ({ ...prev, currentBreath: 1 }));
      intervalRef.current = setInterval(() => {
        setState(prev => {
          if (prev.phase !== 'BREATHING') return prev;
          const next = prev.currentBreath + 1;
          if (next > prev.breathsPerRound) {
            // Reached target — transition to HOLD
            return {
              ...prev,
              phase: 'HOLD' as Phase,
              holdTimeSeconds: 0,
              roundBreaths: [...prev.roundBreaths, prev.currentBreath],
            };
          }
          return { ...prev, currentBreath: next };
        });
      }, BREATH_INTERVAL_MS);
    }

    if (state.phase === 'HOLD') {
      // Stopwatch counting up
      holdStartRef.current = Date.now();
      intervalRef.current = setInterval(() => {
        const elapsed = (Date.now() - holdStartRef.current) / 1000;
        setState(prev => {
          if (prev.phase !== 'HOLD') return prev;
          return { ...prev, holdTimeSeconds: elapsed };
        });
      }, 100);
    }

    return () => clearTimer();
  }, [state.phase, state.currentRound]); // eslint-disable-line react-hooks/exhaustive-deps

  const setTotalRounds = useCallback((n: number) => {
    setState(prev => ({ ...prev, totalRounds: Math.max(1, Math.min(10, n)) }));
  }, []);

  const setBreathsPerRound = useCallback((n: number) => {
    setState(prev => ({ ...prev, breathsPerRound: Math.max(10, Math.min(60, n)) }));
  }, []);

  const startSession = useCallback(() => {
    setState(prev => ({
      ...prev,
      phase: 'BREATHING',
      currentRound: 1,
      currentBreath: 0,
      holdTimeSeconds: 0,
      roundBreaths: [],
      retentionTimes: [],
      pushUpCount: null,
      sessionStartTime: new Date(),
      sessionEndTime: null,
    }));
  }, []);

  const skipToHold = useCallback(() => {
    setState(prev => {
      if (prev.phase !== 'BREATHING' || prev.currentBreath === 0) return prev;
      return {
        ...prev,
        phase: 'HOLD' as Phase,
        holdTimeSeconds: 0,
        roundBreaths: [...prev.roundBreaths, prev.currentBreath],
      };
    });
  }, []);

  const endHold = useCallback(() => {
    setState(prev => {
      if (prev.phase !== 'HOLD') return prev;
      const retentionSeconds = Math.round(prev.holdTimeSeconds);
      const isLastRound = prev.currentRound >= prev.totalRounds;
      if (isLastRound) {
        return {
          ...prev,
          phase: 'PUSHUPS' as Phase,
          retentionTimes: [...prev.retentionTimes, retentionSeconds],
          holdTimeSeconds: retentionSeconds,
        };
      }
      return {
        ...prev,
        phase: 'BREATHING' as Phase,
        currentRound: prev.currentRound + 1,
        currentBreath: 0,
        retentionTimes: [...prev.retentionTimes, retentionSeconds],
        holdTimeSeconds: retentionSeconds,
      };
    });
  }, []);

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
    skipToHold,
    endHold,
    setPushUpCount,
    confirmPushUps,
    skipPushUps,
    getSessionData,
    reset,
  };
}

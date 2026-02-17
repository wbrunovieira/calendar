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
const BREATH_INTERVAL_MS = 2000;

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

  useEffect(() => {
    return () => clearTimer();
  }, [clearTimer]);

  const setTotalRounds = useCallback((n: number) => {
    setState(prev => ({ ...prev, totalRounds: Math.max(1, Math.min(10, n)) }));
  }, []);

  const setBreathsPerRound = useCallback((n: number) => {
    setState(prev => ({ ...prev, breathsPerRound: Math.max(10, Math.min(60, n)) }));
  }, []);

  // Use refs to break circular dependency between timer functions
  const startBreathingRef = useRef<() => void>(() => {});
  const startHoldRef = useRef<() => void>(() => {});
  const startRecoveryRef = useRef<() => void>(() => {});

  startHoldRef.current = () => {
    clearTimer();
    timerStartRef.current = Date.now();
    intervalRef.current = setInterval(() => {
      const elapsed = (Date.now() - timerStartRef.current) / 1000;
      setState(s => ({ ...s, holdTimeSeconds: elapsed }));
    }, 100);
  };

  startBreathingRef.current = () => {
    clearTimer();
    timerStartRef.current = Date.now();
    setState(prev => ({ ...prev, currentBreath: 1 }));

    intervalRef.current = setInterval(() => {
      let transitionToHold = false;
      setState(prev => {
        if (prev.phase !== 'BREATHING') return prev;
        const next = prev.currentBreath + 1;
        if (next > prev.breathsPerRound) {
          transitionToHold = true;
          return {
            ...prev,
            phase: 'HOLD' as Phase,
            holdTimeSeconds: 0,
            roundBreaths: [...prev.roundBreaths, prev.currentBreath],
          };
        }
        return { ...prev, currentBreath: next };
      });
      // Side effects outside setState — clear breathing interval, start hold timer
      if (transitionToHold) {
        clearTimer();
        startHoldRef.current();
      }
    }, BREATH_INTERVAL_MS);
  };

  startRecoveryRef.current = () => {
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
        setTimeout(() => {
          setState(prev => {
            if (prev.phase === 'BREATHING') {
              startBreathingRef.current();
            }
            return prev;
          });
        }, 0);
      } else {
        setState(prev => ({ ...prev, recoveryTimeSeconds: Math.ceil(remaining) }));
      }
    }, 100);
  };

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
    setTimeout(() => startBreathingRef.current(), 0);
  }, [clearTimer]);

  const skipToHold = useCallback(() => {
    clearTimer();
    setState(prev => {
      if (prev.phase !== 'BREATHING' || prev.currentBreath === 0) return prev;
      return {
        ...prev,
        phase: 'HOLD' as Phase,
        holdTimeSeconds: 0,
        roundBreaths: [...prev.roundBreaths, prev.currentBreath],
      };
    });
    startHoldRef.current();
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
    startRecoveryRef.current();
  }, [clearTimer]);

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

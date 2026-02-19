'use client';

import { useState, useRef, useCallback, useEffect } from 'react';
import type { StretchSequence } from '@/types/health';

export type StretchPhase = 'IDLE' | 'ACTIVE' | 'REST' | 'COMPLETE';

export interface MovementResult {
  movementId: string;
  movementName: string;
  plannedDuration: number;
  actualDuration: number;
  reps: number;
  skipped: boolean;
  orderIndex: number;
}

export interface StretchSessionState {
  phase: StretchPhase;
  sequence: StretchSequence | null;
  currentMovementIndex: number;
  elapsedSeconds: number;
  restCountdown: number;
  movementResults: MovementResult[];
  sessionStartTime: number | null;
  totalElapsedSeconds: number;
}

const REST_DURATION = 5;

const initialState: StretchSessionState = {
  phase: 'IDLE',
  sequence: null,
  currentMovementIndex: 0,
  elapsedSeconds: 0,
  restCountdown: REST_DURATION,
  movementResults: [],
  sessionStartTime: null,
  totalElapsedSeconds: 0,
};

export function useStretchSession() {
  const [state, setState] = useState<StretchSessionState>(initialState);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const startTimeRef = useRef<number>(0);

  const clearTimer = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  // Timer effect for ACTIVE and REST phases
  useEffect(() => {
    clearTimer();

    if (state.phase === 'ACTIVE') {
      startTimeRef.current = Date.now();
      timerRef.current = setInterval(() => {
        const elapsed = Math.floor((Date.now() - startTimeRef.current) / 1000);
        setState(prev => ({ ...prev, elapsedSeconds: elapsed }));
      }, 100);
    } else if (state.phase === 'REST') {
      startTimeRef.current = Date.now();
      timerRef.current = setInterval(() => {
        const elapsed = Math.floor((Date.now() - startTimeRef.current) / 1000);
        const remaining = REST_DURATION - elapsed;
        if (remaining <= 0) {
          // Auto-advance to next movement
          setState(prev => {
            const movements = prev.sequence?.movements || [];
            const nextIndex = prev.currentMovementIndex + 1;
            if (nextIndex >= movements.length) {
              return {
                ...prev,
                phase: 'COMPLETE',
                restCountdown: 0,
                totalElapsedSeconds: prev.sessionStartTime
                  ? Math.floor((Date.now() - prev.sessionStartTime) / 1000)
                  : 0,
              };
            }
            return {
              ...prev,
              phase: 'ACTIVE',
              currentMovementIndex: nextIndex,
              elapsedSeconds: 0,
              restCountdown: REST_DURATION,
            };
          });
        } else {
          setState(prev => ({ ...prev, restCountdown: remaining }));
        }
      }, 100);
    }

    return clearTimer;
  }, [state.phase, state.currentMovementIndex, clearTimer]);

  const startSession = useCallback((sequence: StretchSequence) => {
    if (!sequence.movements || sequence.movements.length === 0) return;
    setState({
      ...initialState,
      phase: 'ACTIVE',
      sequence,
      sessionStartTime: Date.now(),
    });
  }, []);

  const completeMovement = useCallback(() => {
    setState(prev => {
      if (prev.phase !== 'ACTIVE' || !prev.sequence?.movements) return prev;
      const movement = prev.sequence.movements[prev.currentMovementIndex];
      const result: MovementResult = {
        movementId: movement.id,
        movementName: movement.name,
        plannedDuration: movement.durationSeconds,
        actualDuration: prev.elapsedSeconds,
        reps: 1,
        skipped: false,
        orderIndex: prev.currentMovementIndex,
      };
      const results = [...prev.movementResults, result];
      const isLast = prev.currentMovementIndex >= prev.sequence.movements.length - 1;

      if (isLast) {
        return {
          ...prev,
          phase: 'COMPLETE',
          movementResults: results,
          totalElapsedSeconds: prev.sessionStartTime
            ? Math.floor((Date.now() - prev.sessionStartTime) / 1000)
            : 0,
        };
      }

      return {
        ...prev,
        phase: 'REST',
        movementResults: results,
        elapsedSeconds: 0,
        restCountdown: REST_DURATION,
      };
    });
  }, []);

  const skipMovement = useCallback(() => {
    setState(prev => {
      if (prev.phase !== 'ACTIVE' || !prev.sequence?.movements) return prev;
      const movement = prev.sequence.movements[prev.currentMovementIndex];
      const result: MovementResult = {
        movementId: movement.id,
        movementName: movement.name,
        plannedDuration: movement.durationSeconds,
        actualDuration: 0,
        reps: 0,
        skipped: true,
        orderIndex: prev.currentMovementIndex,
      };
      const results = [...prev.movementResults, result];
      const isLast = prev.currentMovementIndex >= prev.sequence.movements.length - 1;

      if (isLast) {
        return {
          ...prev,
          phase: 'COMPLETE',
          movementResults: results,
          totalElapsedSeconds: prev.sessionStartTime
            ? Math.floor((Date.now() - prev.sessionStartTime) / 1000)
            : 0,
        };
      }

      return {
        ...prev,
        phase: 'REST',
        movementResults: results,
        elapsedSeconds: 0,
        restCountdown: REST_DURATION,
      };
    });
  }, []);

  const addRep = useCallback(() => {
    // Reset timer for another rep of the same movement
    startTimeRef.current = Date.now();
    setState(prev => ({
      ...prev,
      elapsedSeconds: 0,
    }));
  }, []);

  const getSessionData = useCallback(() => {
    if (!state.sequence) return null;
    return {
      profileId: state.sequence.profileId,
      sequenceId: state.sequence.id,
      sessionDate: new Date().toISOString().split('T')[0],
      durationSeconds: state.totalElapsedSeconds,
      movements: state.movementResults.map(r => ({
        movementId: r.movementId,
        movementName: r.movementName,
        plannedDurationSeconds: r.plannedDuration,
        actualDurationSeconds: r.actualDuration,
        reps: r.reps,
        skipped: r.skipped,
        orderIndex: r.orderIndex,
      })),
    };
  }, [state.sequence, state.totalElapsedSeconds, state.movementResults]);

  const reset = useCallback(() => {
    clearTimer();
    setState(initialState);
  }, [clearTimer]);

  const currentMovement = state.sequence?.movements?.[state.currentMovementIndex] || null;
  const totalMovements = state.sequence?.movements?.length || 0;
  const completedCount = state.movementResults.length;
  const skippedCount = state.movementResults.filter(r => r.skipped).length;

  return {
    state,
    currentMovement,
    totalMovements,
    completedCount,
    skippedCount,
    startSession,
    completeMovement,
    skipMovement,
    addRep,
    getSessionData,
    reset,
  };
}

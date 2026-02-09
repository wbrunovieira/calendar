'use client';

import { useEffect, useRef, useCallback } from 'react';
import { api } from '@/lib/api';

const POLL_INTERVAL = 30_000; // 30 seconds
const WINDOW_MINUTES = 2;

interface UpcomingReminder {
  eventId: string;
  eventTitle: string;
  startTime: string;
  startDate: string;
  minutesBefore: number;
  triggerAt: string;
}

function formatReminderBody(reminder: UpcomingReminder): string {
  if (reminder.minutesBefore === 0) {
    return `Agora: ${reminder.startTime}`;
  }
  if (reminder.minutesBefore < 60) {
    return `Em ${reminder.minutesBefore} minutos (${reminder.startTime})`;
  }
  if (reminder.minutesBefore < 1440) {
    const hours = Math.floor(reminder.minutesBefore / 60);
    return `Em ${hours} hora${hours > 1 ? 's' : ''} (${reminder.startTime})`;
  }
  const days = Math.floor(reminder.minutesBefore / 1440);
  return `Em ${days} dia${days > 1 ? 's' : ''} (${reminder.startDate} ${reminder.startTime})`;
}

export function useEventAlerts() {
  const firedRef = useRef<Set<string>>(new Set());
  const audioRef = useRef<HTMLAudioElement | null>(null);

  // Request notification permission on mount
  useEffect(() => {
    if (typeof window !== 'undefined' && 'Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }
  }, []);

  // Initialize audio element
  useEffect(() => {
    if (typeof window !== 'undefined') {
      audioRef.current = new Audio('/sounds/alert.mp3');
      audioRef.current.volume = 0.5;
    }
  }, []);

  const fireAlert = useCallback((reminder: UpcomingReminder) => {
    const key = `${reminder.eventId}-${reminder.minutesBefore}`;
    if (firedRef.current.has(key)) return;
    firedRef.current.add(key);

    // Play sound
    if (audioRef.current) {
      audioRef.current.currentTime = 0;
      audioRef.current.play().catch(() => {
        // Audio play may fail if user hasn't interacted with page yet
      });
    }

    // Show notification
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification(reminder.eventTitle, {
        body: formatReminderBody(reminder),
        icon: '/icon.svg',
        tag: key,
      });
    }
  }, []);

  // Polling
  useEffect(() => {
    const checkReminders = async () => {
      try {
        const reminders = await api.events.upcomingReminders(WINDOW_MINUTES);
        for (const reminder of reminders) {
          fireAlert(reminder);
        }
      } catch {
        // Silently fail — don't break the app if the endpoint is unavailable
      }
    };

    // Initial check
    checkReminders();

    // Poll every 30s
    const intervalId = setInterval(checkReminders, POLL_INTERVAL);

    return () => clearInterval(intervalId);
  }, [fireAlert]);

  // Clean up old fired alerts every 10 minutes to prevent memory leak
  useEffect(() => {
    const cleanupId = setInterval(() => {
      firedRef.current.clear();
    }, 10 * 60 * 1000);

    return () => clearInterval(cleanupId);
  }, []);
}

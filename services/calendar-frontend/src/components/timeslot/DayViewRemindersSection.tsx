'use client';

/**
 * Day View Reminders Section
 * Shows reminders for the current day (including those with reminderDaysBefore)
 * Collapsible with smooth animation
 * Groups reminders by profile/calendar
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import { api } from '@/lib/api';
import { formatDateToString } from '@/utils/calendar/dateHelpers';
import { calendars } from '@/data/calendars';
import type { Event } from '@/types/calendar';
import CreateReminderModal from '@/components/reminders/CreateReminderModal';

interface DayViewRemindersSectionProps {
  date: Date;
  onReminderToggled?: () => void;
}

const getPriorityInfo = (priority?: number) => {
  switch (priority) {
    case 1:
      return { label: 'Urgente', color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' };
    case 2:
      return { label: 'Importante', color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' };
    case 3:
      return { label: 'Normal', color: 'text-blue-400', bg: 'bg-blue-500/20', border: 'border-blue-500/30' };
    default:
      return null;
  }
};

const getDaysUntil = (targetDate: string, todayString: string): number => {
  const target = new Date(targetDate);
  const today = new Date(todayString);
  const diffTime = target.getTime() - today.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  return diffDays;
};

const getReminderStatus = (startDate: string, todayString: string, _reminderDaysBefore?: number[]) => {
  const cleanDate = startDate.includes('T') ? startDate.split('T')[0] : startDate;
  const daysUntil = getDaysUntil(cleanDate, todayString);

  if (daysUntil < 0) {
    return { label: 'Passou', color: 'text-white/40', icon: 'clock' };
  }
  if (daysUntil === 0) {
    return { label: 'Hoje!', color: 'text-amber-400', icon: 'alert' };
  }
  if (daysUntil === 1) {
    return { label: 'Amanha', color: 'text-orange-400', icon: 'soon' };
  }
  return { label: `${daysUntil} dias`, color: 'text-blue-400', icon: 'calendar' };
};

const getNextReminderDay = (startDate: string, todayString: string, reminderDaysBefore?: number[]): string | null => {
  if (!reminderDaysBefore || reminderDaysBefore.length === 0) return null;

  const cleanDate = startDate.includes('T') ? startDate.split('T')[0] : startDate;
  const daysUntil = getDaysUntil(cleanDate, todayString);

  // Find the next reminder day after today
  const sortedDays = [...reminderDaysBefore].sort((a, b) => b - a); // descending
  const nextDay = sortedDays.find(d => d < daysUntil && d > 0);

  if (nextDay === 1) return 'amanhã';
  if (nextDay) return `em ${nextDay} dias`;
  if (daysUntil > 0) return 'no dia do evento';
  return null;
};

export default function DayViewRemindersSection({ date, onReminderToggled }: DayViewRemindersSectionProps) {
  const [reminders, setReminders] = useState<Event[]>([]);
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [isExpanded, setIsExpanded] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const dateString = formatDateToString(date);

  const fetchReminders = useCallback(async () => {
    try {
      setLoading(true);
      // Fetch reminders in a wide range to catch those with reminderDaysBefore
      const startRange = new Date(date);
      startRange.setDate(startRange.getDate() - 7);
      const endRange = new Date(date);
      endRange.setDate(endRange.getDate() + 30);

      const data = await api.events.listReminders({
        startDate: formatDateToString(startRange),
        endDate: formatDateToString(endRange),
      });

      // Filter reminders that should appear today:
      // 1. Reminders whose startDate is today
      // 2. Reminders whose startDate is in the future AND today is in their reminderDaysBefore
      const relevantReminders = data.filter(r => {
        const reminderDate = r.startDate ? r.startDate.split('T')[0] : null;
        if (!reminderDate) return false;

        // Show if reminder is for today
        if (reminderDate === dateString) return true;

        // Show if today is a reminder day (X days before the event)
        if (r.reminderDaysBefore && r.reminderDaysBefore.length > 0) {
          const daysUntil = getDaysUntil(reminderDate, dateString);
          if (daysUntil > 0 && r.reminderDaysBefore.includes(daysUntil)) {
            return true;
          }
        }

        return false;
      });

      setReminders(relevantReminders);

      // Check which reminders are completed for today
      const completed = new Set<string>();
      for (const reminder of relevantReminders) {
        const isCompleted = reminder.executions?.some(e => {
          const execDate = e.executionDate?.split('T')[0];
          return execDate === dateString && e.completed;
        });
        if (isCompleted) {
          completed.add(reminder.id);
        }
      }
      setCompletedIds(completed);
    } catch (error) {
      console.error('Failed to fetch reminders:', error);
    } finally {
      setLoading(false);
    }
  }, [date, dateString]);

  useEffect(() => {
    fetchReminders();
  }, [fetchReminders]);

  // Group reminders by calendar/profile
  const remindersByCalendar = useMemo(() => {
    const grouped = new Map<string, Event[]>();

    for (const reminder of reminders) {
      const calendarId = reminder.calendarId;
      if (!grouped.has(calendarId)) {
        grouped.set(calendarId, []);
      }
      grouped.get(calendarId)!.push(reminder);
    }

    return grouped;
  }, [reminders]);

  const hasMultipleProfiles = remindersByCalendar.size > 1;

  const handleToggle = async (reminder: Event) => {
    const isCompleted = completedIds.has(reminder.id);

    setToggling(reminder.id);
    try {
      await api.events.toggleExecution(reminder.id, dateString, !isCompleted);

      setCompletedIds(prev => {
        const next = new Set(prev);
        if (isCompleted) {
          next.delete(reminder.id);
        } else {
          next.add(reminder.id);
        }
        return next;
      });

      if (onReminderToggled) {
        onReminderToggled();
      }
    } catch (error) {
      console.error('Failed to toggle reminder:', error);
    } finally {
      setToggling(null);
    }
  };

  const pendingCount = reminders.filter(r => !completedIds.has(r.id)).length;
  const totalCount = reminders.length;

  const renderReminderItem = (reminder: Event) => {
    const isCompleted = completedIds.has(reminder.id);
    const isToggling = toggling === reminder.id;
    const priority = getPriorityInfo(reminder.priority);
    const status = getReminderStatus(reminder.startDate, dateString, reminder.reminderDaysBefore);
    const reminderDate = reminder.startDate?.split('T')[0];
    const isForToday = reminderDate === dateString;
    const nextReminder = getNextReminderDay(reminder.startDate, dateString, reminder.reminderDaysBefore);

    return (
      <li key={reminder.id}>
        <div
          className={`w-full flex items-start gap-3 px-3 py-3 rounded-lg transition-all duration-200 ${
            isCompleted
              ? 'bg-green-500/20 border border-green-500/30'
              : isForToday
                ? 'bg-amber-500/20 border border-amber-500/30'
                : 'bg-white/5 border border-white/10 hover:border-white/20'
          }`}
        >
          {/* Bell icon */}
          <span className="text-lg flex-shrink-0 mt-0.5">
            {isCompleted ? '✅' : isForToday ? '🔔' : '⏰'}
          </span>

          {/* Title and info */}
          <div className="flex-1 min-w-0">
            <span
              className={`block text-sm font-medium ${
                isCompleted ? 'text-white/60 line-through' : 'text-white'
              }`}
            >
              {reminder.title}
            </span>
            {reminder.description && (
              <span className="text-white/50 text-xs block mt-0.5">
                {reminder.description}
              </span>
            )}
            <div className="flex items-center gap-2 mt-1.5 flex-wrap">
              {priority && (
                <span className={`text-xs px-1.5 py-0.5 rounded ${priority.bg} ${priority.color} ${priority.border} border`}>
                  ⚡ {priority.label}
                </span>
              )}
              {!isCompleted && (
                <span className={`text-xs font-medium ${status.color}`}>
                  📅 {status.label}
                </span>
              )}
            </div>
            {/* Show next reminder info when marked as seen */}
            {isCompleted && nextReminder && (
              <span className="text-white/40 text-xs block mt-1">
                Lembrarei novamente {nextReminder}
              </span>
            )}
          </div>

          {/* Action button */}
          <div className="flex-shrink-0">
            {isToggling ? (
              <div className="w-16 h-8 flex items-center justify-center">
                <div className="w-4 h-4 border-2 border-amber-400/30 border-t-amber-400 rounded-full animate-spin" />
              </div>
            ) : isCompleted ? (
              <button
                onClick={() => handleToggle(reminder)}
                className="px-3 py-1.5 text-xs font-medium text-white/60 bg-white/10 hover:bg-white/20 rounded-lg transition-colors flex items-center gap-1"
                title="Marcar como não visto"
              >
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                </svg>
                Desfazer
              </button>
            ) : (
              <button
                onClick={() => handleToggle(reminder)}
                className="px-3 py-1.5 text-xs font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg transition-colors flex items-center gap-1.5 shadow-sm"
                title="Marcar como visto - o lembrete aparecerá novamente nos próximos dias configurados"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                Já vi
              </button>
            )}
          </div>
        </div>
      </li>
    );
  };

  return (
    <>
    <div className="mt-4 bg-gradient-to-br from-amber-900/20 to-orange-900/20 rounded-xl border border-amber-500/20 overflow-hidden shadow-lg">
      {/* Header */}
      <div className="px-4 py-3 border-b border-amber-500/20 flex items-center justify-between bg-amber-500/10">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-2 hover:opacity-80 transition-opacity"
        >
          <span className="text-lg">🔔</span>
          <span className="text-white font-medium">Lembretes</span>
        </button>
        <div className="flex items-center gap-3">
          {!loading && totalCount > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-white/90 font-semibold text-lg">
                {pendingCount > 0 ? `${pendingCount} pendente${pendingCount > 1 ? 's' : ''}` : 'Tudo visto!'}
              </span>
              {pendingCount === 0 && totalCount > 0 && (
                <span className="text-green-400">✓</span>
              )}
            </div>
          )}
          {/* Add button */}
          <button
            onClick={() => setShowCreateModal(true)}
            className="w-7 h-7 rounded-lg bg-amber-600 hover:bg-amber-700 text-white flex items-center justify-center transition-colors"
            title="Criar lembrete"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          </button>
          {/* Chevron icon with rotation animation */}
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="p-1 hover:bg-white/10 rounded transition-colors"
          >
            <svg
              className={`w-5 h-5 text-white/60 transition-transform duration-300 ease-in-out ${
                isExpanded ? 'rotate-180' : 'rotate-0'
              }`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>
      </div>

      {/* Collapsible Content with smooth animation */}
      <div
        className={`transition-all duration-300 ease-in-out overflow-hidden ${
          isExpanded ? 'max-h-[500px] opacity-100' : 'max-h-0 opacity-0'
        }`}
      >
        {/* Content */}
        <div className="p-3 max-h-64 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-6">
              <div className="w-6 h-6 border-2 border-amber-400/30 border-t-amber-400 rounded-full animate-spin" />
            </div>
          ) : totalCount === 0 ? (
            <p className="text-white/40 text-sm text-center py-4">
              Nenhum lembrete para hoje
            </p>
          ) : hasMultipleProfiles ? (
            // Grouped by profile
            <div className="space-y-4">
              {Array.from(remindersByCalendar.entries()).map(([calendarId, calendarReminders]) => {
                const calendar = calendars.find(c => c.id === calendarId);
                const calendarPendingCount = calendarReminders.filter(r => !completedIds.has(r.id)).length;

                return (
                  <div key={calendarId}>
                    {/* Profile header */}
                    <div className="flex items-center gap-2 mb-2 px-1">
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: calendar?.color || '#666' }}
                      />
                      <span className="text-white/70 text-xs font-medium">
                        {calendar?.name || 'Desconhecido'}
                      </span>
                      <span className="text-white/40 text-xs">
                        ({calendarPendingCount} pendente{calendarPendingCount !== 1 ? 's' : ''})
                      </span>
                    </div>
                    {/* Reminders list */}
                    <ul className="space-y-2">
                      {calendarReminders.map(reminder => renderReminderItem(reminder))}
                    </ul>
                  </div>
                );
              })}
            </div>
          ) : (
            // Single profile - no grouping needed
            <ul className="space-y-2">
              {reminders.map(reminder => renderReminderItem(reminder))}
            </ul>
          )}
        </div>
      </div>
    </div>

    {/* Create Reminder Modal */}
    <CreateReminderModal
      isOpen={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      onCreated={() => {
        fetchReminders();
        if (onReminderToggled) onReminderToggled();
      }}
      calendars={calendars}
    />
    </>
  );
}

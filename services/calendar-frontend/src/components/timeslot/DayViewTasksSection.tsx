'use client';

/**
 * Day View Tasks Section
 * Shows today's tasks with completion toggle below the habits section
 * Collapsible with smooth animation
 * Groups tasks by profile/calendar
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import { api } from '@/lib/api';
import { formatDateToString } from '@/utils/calendar/dateHelpers';
import { calendars } from '@/data/calendars';
import type { Event } from '@/types/calendar';

interface DayViewTasksSectionProps {
  date: Date;
  onTaskToggled?: () => void;
  onEditTask?: (task: Event) => void;
}

const getPriorityInfo = (priority?: number) => {
  switch (priority) {
    case 1:
      return { label: 'Alta', color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' };
    case 2:
      return { label: 'Media', color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' };
    case 3:
      return { label: 'Baixa', color: 'text-blue-400', bg: 'bg-blue-500/20', border: 'border-blue-500/30' };
    default:
      return null;
  }
};

const getDueDateStatus = (dueDate?: string, todayString?: string) => {
  if (!dueDate || !todayString) return null;
  const cleanDate = dueDate.includes('T') ? dueDate.split('T')[0] : dueDate;

  if (cleanDate < todayString) return { label: 'Atrasado', color: 'text-red-400' };
  if (cleanDate === todayString) return { label: 'Hoje', color: 'text-amber-400' };
  return null;
};

export default function DayViewTasksSection({ date, onTaskToggled, onEditTask }: DayViewTasksSectionProps) {
  const [tasks, setTasks] = useState<Event[]>([]);
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [isExpanded, setIsExpanded] = useState(true);
  const [draggedId, setDraggedId] = useState<string | null>(null);

  const dateString = formatDateToString(date);

  const fetchTasks = useCallback(async () => {
    try {
      setLoading(true);
      // Fetch all tasks (they don't have recurrence like habits)
      const startRange = new Date(date);
      startRange.setDate(startRange.getDate() - 30);
      const endRange = new Date(date);
      endRange.setDate(endRange.getDate() + 30);

      const data = await api.events.listTodos({
        startDate: formatDateToString(startRange),
        endDate: formatDateToString(endRange),
      });

      // Filter tasks: show tasks that are due today or tasks without due date created today
      // Also show overdue tasks
      const relevantTasks = data.filter(t => {
        const dueDate = t.dueDate ? t.dueDate.split('T')[0] : null;
        const startDate = t.startDate ? t.startDate.split('T')[0] : null;

        // Show if due today
        if (dueDate === dateString) return true;
        // Show if overdue (and not completed)
        if (dueDate && dueDate < dateString) {
          const isCompleted = t.executions?.some(e => e.completed);
          if (!isCompleted) return true;
        }
        // Show if created today and no due date
        if (!dueDate && startDate === dateString) return true;

        return false;
      });

      // Sort by displayOrder (for drag and drop)
      const sortedTasks = [...relevantTasks].sort((a, b) =>
        (a.displayOrder || 0) - (b.displayOrder || 0)
      );

      setTasks(sortedTasks);

      // Check which tasks are completed
      const completed = new Set<string>();
      for (const task of sortedTasks) {
        const isCompleted = task.executions?.some(e => e.completed);
        if (isCompleted) {
          completed.add(task.id);
        }
      }
      setCompletedIds(completed);
    } catch (error) {
      console.error('Failed to fetch tasks:', error);
    } finally {
      setLoading(false);
    }
  }, [date, dateString]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  // Group tasks by calendar/profile
  const tasksByCalendar = useMemo(() => {
    const grouped = new Map<string, Event[]>();

    for (const task of tasks) {
      const calendarId = task.calendarId;
      if (!grouped.has(calendarId)) {
        grouped.set(calendarId, []);
      }
      grouped.get(calendarId)!.push(task);
    }

    return grouped;
  }, [tasks]);

  // Check if we have multiple profiles
  const hasMultipleProfiles = tasksByCalendar.size > 1;

  // Drag and drop handlers
  const handleDragStart = (e: React.DragEvent, id: string) => {
    setDraggedId(id);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = async (e: React.DragEvent, targetId: string, list: Event[]) => {
    e.preventDefault();
    if (!draggedId || draggedId === targetId) {
      setDraggedId(null);
      return;
    }

    const draggedIndex = list.findIndex(item => item.id === draggedId);
    const targetIndex = list.findIndex(item => item.id === targetId);

    if (draggedIndex === -1 || targetIndex === -1) {
      setDraggedId(null);
      return;
    }

    // Reorder the list
    const newList = [...list];
    const [draggedItem] = newList.splice(draggedIndex, 1);
    newList.splice(targetIndex, 0, draggedItem);

    // Get the IDs in the new order
    const orderedIds = newList.map(item => item.id);

    try {
      await api.events.reorder(orderedIds);
      await fetchTasks();
    } catch (error) {
      console.error('Error reordering:', error);
    }

    setDraggedId(null);
  };

  const handleDragEnd = () => {
    setDraggedId(null);
  };

  const handleToggle = async (task: Event) => {
    const isCompleted = completedIds.has(task.id);

    // Optimistic update — update UI immediately
    setCompletedIds(prev => {
      const next = new Set(prev);
      if (isCompleted) {
        next.delete(task.id);
      } else {
        next.add(task.id);
      }
      return next;
    });

    try {
      await api.events.toggleExecution(task.id, dateString, !isCompleted);
    } catch (error) {
      console.error('Failed to toggle task:', error);
      // Revert on error
      setCompletedIds(prev => {
        const next = new Set(prev);
        if (isCompleted) {
          next.add(task.id);
        } else {
          next.delete(task.id);
        }
        return next;
      });
    }
  };

  const pendingTasks = tasks.filter(t => !completedIds.has(t.id));
  const completedTasks = tasks.filter(t => completedIds.has(t.id));
  const pendingCount = pendingTasks.length;
  const totalCount = tasks.length;

  // Render a single task item
  const renderTaskItem = (task: Event, list: Event[]) => {
    const isCompleted = completedIds.has(task.id);
    const priority = getPriorityInfo(task.priority);
    const dueStatus = getDueDateStatus(task.dueDate, dateString);
    const isDragged = draggedId === task.id;

    return (
      <li key={task.id}>
        <div
          draggable
          onDragStart={(e) => handleDragStart(e, task.id)}
          onDragOver={handleDragOver}
          onDrop={(e) => handleDrop(e, task.id, list)}
          onDragEnd={handleDragEnd}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200 cursor-grab active:cursor-grabbing ${
            isCompleted
              ? 'bg-green-500/20 border border-green-500/30'
              : 'bg-white/5 border border-white/10 hover:border-white/20'
          } ${isDragged ? 'opacity-50' : ''}`}
        >
          {/* Drag handle */}
          <svg className="w-4 h-4 text-white/30 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
          </svg>

          {/* Checkbox */}
          <button
            onClick={() => handleToggle(task)}
            className={`w-5 h-5 rounded-md border-2 flex-shrink-0 flex items-center justify-center transition-all duration-200 ${
              isCompleted
                ? 'bg-green-500 border-green-500'
                : 'border-white/40 hover:border-white/60'
            }`}
          >
            {isCompleted ? (
              <svg className="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
              </svg>
            ) : null}
          </button>

          {/* Title and info */}
          <div className="flex-1 min-w-0">
            <span
              className={`block text-sm font-medium truncate ${
                isCompleted ? 'text-white/60 line-through' : 'text-white'
              }`}
            >
              {task.title}
            </span>
            <div className="flex items-center gap-2 mt-0.5">
              {priority && (
                <span className={`text-xs ${priority.color}`}>
                  {priority.label}
                </span>
              )}
              {dueStatus && !isCompleted && (
                <span className={`text-xs ${dueStatus.color}`}>
                  {dueStatus.label}
                </span>
              )}
              {task.label && (
                <span
                  className="text-xs px-1.5 py-0.5 rounded-full"
                  style={{
                    backgroundColor: `${task.label.color}20`,
                    color: task.label.color,
                  }}
                >
                  {task.label.name}
                </span>
              )}
            </div>
          </div>

          {/* Edit button */}
          {onEditTask && (
            <button
              onClick={() => onEditTask(task)}
              className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
              title="Editar tarefa"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
          )}

          {/* Completed indicator */}
          {isCompleted && (
            <span className="text-green-400 text-xs font-medium">Concluido</span>
          )}
        </div>
      </li>
    );
  };

  return (
    <div className="mt-4 bg-gradient-to-br from-white/10 to-white/5 rounded-xl border border-white/20 overflow-hidden shadow-lg">
      {/* Header - Clickable to expand/collapse */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 py-3 border-b border-white/10 flex items-center justify-between bg-white/5 hover:bg-white/10 transition-colors duration-200"
      >
        <div className="flex items-center gap-2">
          <span className="text-lg">📋</span>
          <span className="text-white font-medium">Tarefas do Dia</span>
        </div>
        <div className="flex items-center gap-3">
          {!loading && totalCount > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-white/90 font-semibold text-lg">
                {pendingCount > 0 ? `${pendingCount} pendente${pendingCount > 1 ? 's' : ''}` : 'Tudo feito!'}
              </span>
              {pendingCount === 0 && totalCount > 0 && (
                <span className="text-green-400">✓</span>
              )}
            </div>
          )}
          {/* Chevron icon with rotation animation */}
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
        </div>
      </button>

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
              <div className="w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            </div>
          ) : tasks.length === 0 ? (
            <p className="text-white/40 text-sm text-center py-4">
              Nenhuma tarefa para hoje
            </p>
          ) : hasMultipleProfiles ? (
            // Grouped by profile
            <div className="space-y-4">
              {Array.from(tasksByCalendar.entries()).map(([calendarId, calendarTasks]) => {
                const calendar = calendars.find(c => c.id === calendarId);
                const calendarPendingCount = calendarTasks.filter(t => !completedIds.has(t.id)).length;

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
                    {/* Tasks list */}
                    <ul className="space-y-2">
                      {calendarTasks.map(task => renderTaskItem(task, calendarTasks))}
                    </ul>
                  </div>
                );
              })}
            </div>
          ) : (
            // Single profile - no grouping needed
            <ul className="space-y-2">
              {/* Pending tasks first */}
              {pendingTasks.map(task => renderTaskItem(task, pendingTasks))}
              {/* Completed tasks */}
              {completedTasks.length > 0 && pendingTasks.length > 0 && (
                <li className="border-t border-white/10 pt-2 mt-2">
                  <span className="text-white/40 text-xs px-1">Concluidas ({completedTasks.length})</span>
                </li>
              )}
              {completedTasks.map(task => renderTaskItem(task, completedTasks))}
            </ul>
          )}
        </div>

        {/* Progress bar */}
        {!loading && totalCount > 0 && (
          <div className="px-4 pb-3">
            <div className="h-2 bg-white/10 rounded-full overflow-hidden">
              <div
                className={`h-full transition-all duration-500 ${
                  pendingCount === 0
                    ? 'bg-gradient-to-r from-green-500 to-emerald-400'
                    : 'bg-gradient-to-r from-orange-500 to-amber-400'
                }`}
                style={{ width: `${totalCount > 0 ? ((totalCount - pendingCount) / totalCount) * 100 : 0}%` }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

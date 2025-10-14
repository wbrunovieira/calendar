/**
 * Time Slot Event Card Component
 * Displays an event card in the TimeSlotView with drag, resize, and action buttons
 */

import React from 'react';
import { Event, Category, CategoryType } from '@/types/calendar';
import { PIXELS_PER_HOUR } from '@/constants/timeSlotView';
import { timeToPixels, calculateEventHeight } from '@/utils/timeCalculations';
import { formatDateToString } from '@/utils/calendar/dateHelpers';

interface Calendar {
  id: string;
  name: string;
  color: string;
  type: string;
}

interface TimeSlotEventCardProps {
  event: Event;
  date: Date;
  category?: Category;
  categoryType?: CategoryType;
  calendar?: Calendar;
  daysCount: number;
  positionInfo: { column: number; totalColumns: number };
  resizingEventId?: string;
  onDragStart: (e: React.DragEvent) => void;
  onDragEnd: (e: React.DragEvent) => void;
  onResizeStart: (edge: 'top' | 'bottom', e: React.MouseEvent, height: number, top: number) => void;
  onToggleExecution: (e: React.MouseEvent) => void;
  onEditClick?: (e: React.MouseEvent) => void;
  onDeleteClick: (e: React.MouseEvent) => void;
}

export default function TimeSlotEventCard({
  event,
  date,
  category,
  categoryType,
  calendar,
  daysCount,
  positionInfo,
  resizingEventId,
  onDragStart,
  onDragEnd,
  onResizeStart,
  onToggleExecution,
  onEditClick,
  onDeleteClick,
}: TimeSlotEventCardProps) {
  // Check execution status from event executions
  const executionDate = formatDateToString(date);
  const execution = event.executions?.find(exec =>
    exec.executionDate?.split('T')[0] === executionDate
  );
  const isCompleted = execution?.completed || false;

  // Calculate position and height based on time
  let topPosition = 0;
  let eventHeight = PIXELS_PER_HOUR; // Default 1 hour

  try {
    topPosition = timeToPixels(event.startTime);

    if (event.endTime) {
      eventHeight = calculateEventHeight(event.startTime, event.endTime);
    }
  } catch {
    // Error parsing time for event
  }

  const padding = daysCount === 7 ? 'px-2 py-2' : daysCount <= 3 ? 'px-3 py-2' : 'px-4 py-3';

  const widthPercentage = 100 / positionInfo.totalColumns;
  const leftPercentage = positionInfo.column * widthPercentage;

  // Add gap between overlapping events
  const gap = positionInfo.totalColumns >= 2 ? 4 : 0; // 4px gap
  const widthWithGap = positionInfo.totalColumns >= 2
    ? `calc(${widthPercentage}% - ${gap}px)`
    : `${widthPercentage}%`;

  // Debug positioning for overlapping events
  if (positionInfo.totalColumns >= 2 && daysCount === 1) {
    console.log(`Event "${event.title}": column=${positionInfo.column}, totalColumns=${positionInfo.totalColumns}, width=${widthPercentage}%, left=${leftPercentage}%`);
  }

  const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

  // Convert hex color to rgba for better browser compatibility
  const hexToRgba = (hex: string | undefined, opacity: number): string => {
    if (!hex) return `rgba(128, 128, 128, ${opacity})`;

    // Remove # if present
    const cleanHex = hex.replace('#', '');

    // Parse hex values
    const r = parseInt(cleanHex.substring(0, 2), 16);
    const g = parseInt(cleanHex.substring(2, 4), 16);
    const b = parseInt(cleanHex.substring(4, 6), 16);

    return `rgba(${r}, ${g}, ${b}, ${opacity})`;
  };

  const bgColor = isCompleted
    ? hexToRgba(category?.color, 0.38)  // 60 in hex = 0.38 opacity
    : hexToRgba(category?.color, 0.56); // 90 in hex = 0.56 opacity

  // Define threshold for "very short" events (< 30 minutes)
  const VERY_SHORT_EVENT_THRESHOLD = PIXELS_PER_HOUR / 2; // 48px = 30 minutes
  const FIFTEEN_MINUTE_THRESHOLD = PIXELS_PER_HOUR / 4; // 24px = 15 minutes

  // Calculate minimum height based on actual event duration
  const getMinHeight = () => {
    // No minHeight for any event to prevent visual overlap
    return undefined;
  };

  return (
    <div
      key={event.id}
      data-event-id={event.id}
      draggable
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      className={`absolute rounded-lg text-white flex flex-col group ${
        isCompleted
          ? 'ring-2 ring-green-500 shadow-lg shadow-green-500/50'
          : ''
      } transition-all duration-300`}
      style={{
        backgroundColor: bgColor,
        top: `${topPosition}px`,
        height: `${eventHeight}px`,
        minHeight: getMinHeight(),
        left: `${leftPercentage}%`,
        width: widthWithGap,
        zIndex: 10 + positionInfo.column,
        cursor: resizingEventId === event.id ? 'ns-resize' : 'move',
        overflow: 'visible',
      }}
    >
      {/* Calendar icon badge - positioned at top left, size based on event duration */}
      <div
        className="absolute -top-2 -left-2 rounded-full flex items-center justify-center shadow-lg z-30"
        style={{
          backgroundColor: calendar?.color,
          width: eventHeight < VERY_SHORT_EVENT_THRESHOLD ? '20px' : eventHeight < PIXELS_PER_HOUR ? '24px' : daysCount === 7 ? '24px' : daysCount <= 3 ? '28px' : '32px',
          height: eventHeight < VERY_SHORT_EVENT_THRESHOLD ? '20px' : eventHeight < PIXELS_PER_HOUR ? '24px' : daysCount === 7 ? '24px' : daysCount <= 3 ? '28px' : '32px',
          fontSize: eventHeight < VERY_SHORT_EVENT_THRESHOLD ? '10px' : eventHeight < PIXELS_PER_HOUR ? '12px' : daysCount === 7 ? '12px' : daysCount <= 3 ? '14px' : '16px'
        }}
      >
        {calendarIcon}
      </div>

      {/* Top resize handle - only for events >= 30 minutes */}
      {eventHeight >= VERY_SHORT_EVENT_THRESHOLD && (
        <div
          className="absolute top-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
          onMouseDown={(e) => onResizeStart('top', e, eventHeight, topPosition)}
          title="Arrastar para ajustar início"
        >
          <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-t-lg" />
        </div>
      )}

      {/* Event content */}
      <div className="flex flex-1 cursor-move relative bg-transparent" style={{ overflow: eventHeight < VERY_SHORT_EVENT_THRESHOLD ? 'hidden' : 'visible', alignItems: eventHeight < PIXELS_PER_HOUR ? 'center' : 'flex-start' }}>
        {eventHeight < PIXELS_PER_HOUR && positionInfo.totalColumns >= 2 && daysCount === 1 ? (
          /* Layout ultra-compacto para eventos curtos sobrepostos (15m, 20m, 30m) - Duas colunas */
          <>
            {/* Coluna Esquerda - Ícone, Título e Horário */}
            <div className={`flex-1 px-1.5 py-1 flex flex-col justify-center gap-0.5 ${isCompleted ? 'relative' : ''}`}>
              <div className="flex items-center gap-1">
                <span className="text-lg flex-shrink-0 leading-none">
                  {category?.icon || '📅'}
                </span>
                <div className={`text-xs font-bold truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                  {event.title}
                </div>
              </div>
              <div className={`text-xs opacity-60 ml-5 ${isCompleted ? 'line-through opacity-40' : ''}`}>
                ⏰ {event.startTime}{event.endTime && `-${event.endTime}`}
              </div>

              {isCompleted && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <div className="bg-green-500 rounded-full p-1 shadow-lg animate-pulse">
                    <svg className="w-4 h-4" fill="white" viewBox="0 0 24 24">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                    </svg>
                  </div>
                </div>
              )}
            </div>

            {/* Coluna Direita - Categoria e Tipo */}
            {category?.name && (
              <div className="flex flex-col gap-0.5 px-1.5 py-1 justify-center border-l border-white/20 min-w-[80px]">
                <div className="flex items-center gap-1">
                  <div
                    className="w-2 h-2 rounded-sm flex-shrink-0"
                    style={{ backgroundColor: category.color }}
                  />
                  <span className={`text-xs font-semibold truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {category.name}
                  </span>
                </div>
                {categoryType && (
                  <div className="flex items-center gap-1">
                    <span className="text-xs opacity-70 capitalize truncate">
                      {categoryType.icon && categoryType.icon.trim()} {categoryType.name}
                    </span>
                  </div>
                )}
              </div>
            )}
          </>
        ) : eventHeight < VERY_SHORT_EVENT_THRESHOLD && eventHeight > FIFTEEN_MINUTE_THRESHOLD && daysCount !== 3 && daysCount !== 7 ? (
          /* Layout compacto para eventos curtos (16-29 minutos) - Tudo em linha (exceto 3 e 7 dias) */
          <div className="flex-1 px-1.5 flex items-center gap-1.5">
            <span className="text-base flex-shrink-0 leading-none">
              {category?.icon || '📅'}
            </span>
            <div className={`text-xs font-bold truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
            {category?.name && (
              <>
                <div className="w-px h-3 bg-white/20 mx-2" />
                <div
                  className="w-2 h-2 rounded-sm flex-shrink-0"
                  style={{ backgroundColor: category.color }}
                />
                <span className={`text-xs opacity-70 truncate ${isCompleted ? 'line-through opacity-50' : ''}`}>
                  {category.name}
                </span>
              </>
            )}
            {categoryType && (
              <span className="text-xs leading-none opacity-70 flex items-center gap-1">
                {categoryType.icon && categoryType.icon.trim()}
                {categoryType.name && <span className="capitalize">{categoryType.name}</span>}
              </span>
            )}
            <div className={`text-xs opacity-60 whitespace-nowrap flex-shrink-0 ml-auto ${isCompleted ? 'line-through opacity-40' : ''}`}>
              {event.startTime}{event.endTime && ` - ${event.endTime}`}
            </div>
            {isCompleted && (
              <div className="flex-shrink-0">
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
              </div>
            )}
          </div>
        ) : eventHeight < PIXELS_PER_HOUR && daysCount === 1 ? (
          /* Layout para eventos de 30-60 minutos - Tudo em linha */
          <div className="flex-1 px-2 flex items-center gap-2">
            <span className="text-lg flex-shrink-0 leading-none">
              {category?.icon || '📅'}
            </span>
            <div className={`text-sm font-bold truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
            {category?.name && (
              <>
                <div className="w-px h-3 bg-white/20 mx-2" />
                <div
                  className="w-2.5 h-2.5 rounded-sm flex-shrink-0"
                  style={{ backgroundColor: category.color }}
                />
                <span className={`text-sm opacity-70 truncate ${isCompleted ? 'line-through opacity-50' : ''}`}>
                  {category.name}
                </span>
              </>
            )}
            {categoryType && (
              <span className="text-sm leading-none opacity-70 flex items-center gap-1">
                {categoryType.icon && categoryType.icon.trim()}
                {categoryType.name && <span className="capitalize">{categoryType.name}</span>}
              </span>
            )}
            <div className={`text-sm opacity-60 whitespace-nowrap flex-shrink-0 ml-auto ${isCompleted ? 'line-through opacity-40' : ''}`}>
              {event.startTime}{event.endTime && ` - ${event.endTime}`}
            </div>
            {isCompleted && (
              <div className="flex-shrink-0">
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
              </div>
            )}
          </div>
        ) : eventHeight >= PIXELS_PER_HOUR && positionInfo.totalColumns >= 2 && daysCount === 1 ? (
          /* Layout para eventos de 1 hora+ sobrepostos (2 ou mais colunas) - Layout compacto */
          <>
            {/* Coluna única compacta */}
            <div className={`flex-1 px-2 py-2 flex flex-col justify-center gap-1 ${isCompleted ? 'relative' : ''}`}>
              {/* Linha 1: Ícone + Título */}
              <div className="flex items-center gap-2">
                <span className="text-2xl flex-shrink-0">
                  {category?.icon || '📅'}
                </span>
                <div className="flex-1 min-w-0">
                  <div className={`font-bold text-sm leading-tight truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {event.title}
                  </div>
                  <div className={`text-xs mt-0.5 opacity-90 ${isCompleted ? 'line-through opacity-60' : ''}`}>
                    ⏰ {event.startTime}{event.endTime && ` - ${event.endTime}`}
                  </div>
                </div>
              </div>

              {/* Linha 2: Categoria + Tipo */}
              {category?.name && (
                <div className="flex items-center gap-1.5 ml-8">
                  <div
                    className="w-3 h-3 rounded-md flex-shrink-0"
                    style={{ backgroundColor: category.color }}
                  />
                  <span className={`text-xs font-semibold truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {category.name}
                  </span>
                  {categoryType && (
                    <>
                      <span className="text-xs opacity-50">•</span>
                      <span className="text-xs opacity-70 capitalize truncate">
                        {categoryType.icon && categoryType.icon.trim()} {categoryType.name}
                      </span>
                    </>
                  )}
                </div>
              )}

              {isCompleted && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <div className="bg-green-500 rounded-full p-2 shadow-lg animate-pulse">
                    <svg className="w-8 h-8" fill="white" viewBox="0 0 24 24">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                    </svg>
                  </div>
                </div>
              )}
            </div>
          </>
        ) : daysCount === 1 ? (
          /* Layout para visualização de 1 dia - Duas colunas */
          <>
            {/* Coluna Esquerda - Informações Principais */}
            <div className={`flex-1 ${padding} flex flex-col justify-center ${isCompleted ? 'relative' : ''}`}>
              {/* Category Icon - Grande e proeminente */}
              <div className="flex items-center gap-3 pb-3">
                <span className="text-5xl">
                  {category?.icon || '📅'}
                </span>
                <div className="flex-1">
                  {/* Event Title - Bold */}
                  <div className={`font-bold text-xl leading-tight ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {event.title}
                  </div>
                  {/* Time */}
                  <div className={`text-base mt-1 opacity-90 ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    ⏰ {event.startTime}{event.endTime && ` - ${event.endTime}`}
                  </div>
                </div>
              </div>

              {isCompleted && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <div className="bg-green-500 rounded-full p-3 shadow-lg animate-pulse">
                    <svg className="w-12 h-12" fill="white" viewBox="0 0 24 24">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                    </svg>
                  </div>
                </div>
              )}
            </div>

            {/* Coluna Direita - Hierarquia Visual */}
            <div className="flex flex-col gap-2 px-4 py-4 justify-center border-l border-white/20 min-w-[200px]">
              {/* Category (Nível Superior) - Maior, mais proeminente */}
              {category?.name && (
                <div className="flex items-center gap-2.5">
                  <div
                    className="w-5 h-5 rounded-lg flex-shrink-0 shadow-md border-2 border-white/30"
                    style={{ backgroundColor: category.color }}
                  />
                  <div className={`text-base font-bold leading-tight ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {category.name}
                  </div>
                </div>
              )}

              {/* Type (Sub-nível) - Menor, indentado com conector */}
              {(category?.type || categoryType) && (
                <div className="flex items-start gap-2 ml-1">
                  {/* Conector visual em L */}
                  <div className="flex flex-col items-center pt-0.5">
                    <div className="w-0.5 h-2 bg-white/20" />
                    <div className="w-3 h-0.5 bg-white/20" />
                  </div>
                  {/* Ícone do CategoryType se disponível, senão badge colorido */}
                  {categoryType?.icon ? (
                    <span className="text-lg leading-none pt-0.5">{categoryType.icon.trim()}</span>
                  ) : (
                    <div
                      className="w-3 h-3 rounded flex-shrink-0 mt-0.5 shadow-sm border border-white/20"
                      style={{ backgroundColor: categoryType?.color || category?.color, opacity: 0.8 }}
                    />
                  )}
                  <div className={`text-sm capitalize opacity-80 leading-tight pt-0.5 ${isCompleted ? 'line-through opacity-50' : ''}`}>
                    {categoryType?.name || category?.type}
                  </div>
                </div>
              )}
            </div>
          </>
        ) : eventHeight <= FIFTEEN_MINUTE_THRESHOLD && daysCount === 3 ? (
          /* Layout ultra-compacto para eventos de 15 minutos na visualização de 3 dias - Tudo em linha */
          <div className="flex-1 px-1 flex items-center gap-1">
            <span className="text-sm flex-shrink-0 leading-none">
              {categoryType?.icon ? categoryType.icon.trim() : (category?.icon || '📅')}
            </span>
            <div className={`text-xs font-bold truncate flex-1 min-w-0 ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
            <div className={`text-xs opacity-60 flex-shrink-0 whitespace-nowrap ${isCompleted ? 'line-through opacity-40' : ''}`}>
              {event.startTime}{event.endTime && `-${event.endTime}`}
            </div>
          </div>
        ) : eventHeight < PIXELS_PER_HOUR && daysCount === 3 ? (
          /* Layout compacto para eventos de 16-60 minutos na visualização de 3 dias */
          <div className="flex-1 px-1.5 py-1 flex items-center gap-1.5">
            <span className="text-xl flex-shrink-0 leading-none">
              {category?.icon || '📅'}
            </span>
            <div className="flex-1 min-w-0">
              <div className={`text-xs font-bold truncate leading-tight ${isCompleted ? 'line-through opacity-70' : ''}`}>
                {event.title}
              </div>
              <div className={`text-xs opacity-60 leading-tight ${isCompleted ? 'line-through opacity-40' : ''}`}>
                {event.startTime}{event.endTime && `-${event.endTime}`}
              </div>
            </div>
            {categoryType?.icon && (
              <span className="text-sm opacity-70 flex-shrink-0 leading-none">
                {categoryType.icon.trim()}
              </span>
            )}
          </div>
        ) : eventHeight >= PIXELS_PER_HOUR && positionInfo.totalColumns >= 2 && daysCount === 3 ? (
          /* Layout ultra-compacto para eventos de 1 hora+ sobrepostos na visualização de 3 dias */
          <div className={`flex-1 px-1.5 py-1.5 flex flex-col justify-center gap-0.5 ${isCompleted ? 'relative' : ''}`}>
            {/* Linha 1: Ícone + Título */}
            <div className="flex items-center gap-1.5">
              <span className="text-xl flex-shrink-0 leading-none">
                {category?.icon || '📅'}
              </span>
              <div className={`text-xs font-bold truncate flex-1 min-w-0 ${isCompleted ? 'line-through opacity-70' : ''}`}>
                {event.title}
              </div>
            </div>

            {/* Linha 2: Horário */}
            <div className={`text-xs opacity-80 ml-6 ${isCompleted ? 'line-through opacity-60' : ''}`}>
              ⏰ {event.startTime}{event.endTime && ` - ${event.endTime}`}
            </div>

            {/* Linha 3: Apenas ícone do tipo (sem nome da categoria) */}
            {categoryType?.icon && (
              <div className="flex items-center gap-1 ml-6">
                <span className="text-sm opacity-70 leading-none">
                  {categoryType.icon.trim()}
                </span>
              </div>
            )}

            {isCompleted && (
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div className="bg-green-500 rounded-full p-1.5 shadow-lg animate-pulse">
                  <svg className="w-5 h-5" fill="white" viewBox="0 0 24 24">
                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                  </svg>
                </div>
              </div>
            )}
          </div>
        ) : eventHeight >= PIXELS_PER_HOUR && daysCount === 3 ? (
          /* Layout compacto para eventos de 1 hora+ na visualização de 3 dias */
          <div className={`flex-1 px-2 py-2 flex flex-col justify-center gap-1 ${isCompleted ? 'relative' : ''}`}>
            {/* Linha 1: Ícone + Título */}
            <div className="flex items-center gap-2">
              <span className="text-2xl flex-shrink-0 leading-none">
                {category?.icon || '📅'}
              </span>
              <div className={`text-sm font-bold truncate flex-1 min-w-0 ${isCompleted ? 'line-through opacity-70' : ''}`}>
                {event.title}
              </div>
            </div>

            {/* Linha 2: Horário */}
            <div className={`text-xs opacity-80 ml-8 ${isCompleted ? 'line-through opacity-60' : ''}`}>
              ⏰ {event.startTime}{event.endTime && ` - ${event.endTime}`}
            </div>

            {/* Linha 3: Categoria + Tipo */}
            {category?.name && (
              <div className="flex items-center gap-1.5 ml-8">
                <div
                  className="w-2.5 h-2.5 rounded-sm flex-shrink-0"
                  style={{ backgroundColor: category.color }}
                />
                <span className={`text-xs font-semibold truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                  {category.name}
                </span>
                {categoryType?.icon && (
                  <>
                    <span className="text-xs opacity-50">•</span>
                    <span className="text-xs opacity-70">
                      {categoryType.icon.trim()}
                    </span>
                  </>
                )}
              </div>
            )}

            {isCompleted && (
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div className="bg-green-500 rounded-full p-2 shadow-lg animate-pulse">
                  <svg className="w-6 h-6" fill="white" viewBox="0 0 24 24">
                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                  </svg>
                </div>
              </div>
            )}
          </div>
        ) : daysCount === 3 ? (
          /* Layout para visualização de 3 dias - Híbrido com hierarquia visual */
          <>
            <div className={`flex-1 ${padding} flex flex-col justify-center ${isCompleted ? 'relative' : ''}`}>
              {/* Icon e Title lado a lado */}
              <div className="flex items-center gap-2 pb-2">
                <span className="text-3xl flex-shrink-0">
                  {category?.icon || '📅'}
                </span>
                <div className="flex-1 min-w-0">
                  <div className={`font-bold text-sm leading-tight truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {event.title}
                  </div>
                  <div className={`text-xs mt-0.5 opacity-90 ${isCompleted ? 'line-through opacity-60' : ''}`}>
                    ⏰ {event.startTime}{event.endTime && `-${event.endTime}`}
                  </div>
                </div>
              </div>

              {isCompleted && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <div className="bg-green-500 rounded-full p-2.5 shadow-lg animate-pulse">
                    <svg className="w-8 h-8" fill="white" viewBox="0 0 24 24">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                    </svg>
                  </div>
                </div>
              )}
            </div>

            {/* Coluna direita compacta - Hierarquia visual */}
            <div className="flex flex-col gap-1.5 px-2 py-3 justify-center border-l border-white/20 min-w-[140px]">
              {/* Category */}
              {category?.name && (
                <div className="flex items-center gap-1.5">
                  <div
                    className="w-3.5 h-3.5 rounded-md flex-shrink-0 shadow-sm border border-white/30"
                    style={{ backgroundColor: category.color }}
                  />
                  <div className={`text-xs font-bold leading-tight truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>
                    {category.name}
                  </div>
                </div>
              )}

              {/* Type com conector */}
              {(category?.type || categoryType) && (
                <div className="flex items-start gap-1 ml-0.5">
                  <div className="flex flex-col items-center pt-0.5">
                    <div className="w-0.5 h-1.5 bg-white/20" />
                    <div className="w-2 h-0.5 bg-white/20" />
                  </div>
                  {/* Ícone do CategoryType se disponível, senão badge colorido */}
                  {categoryType?.icon ? (
                    <span className="text-sm leading-none pt-0.5">{categoryType.icon.trim()}</span>
                  ) : (
                    <div
                      className="w-2 h-2 rounded-sm flex-shrink-0 mt-0.5 shadow-sm border border-white/15"
                      style={{ backgroundColor: categoryType?.color || category?.color, opacity: 0.75 }}
                    />
                  )}
                  <div className={`text-xs capitalize opacity-75 leading-tight pt-0.5 truncate ${isCompleted ? 'line-through opacity-50' : ''}`}>
                    {categoryType?.name || category?.type}
                  </div>
                </div>
              )}
            </div>
          </>
        ) : eventHeight <= FIFTEEN_MINUTE_THRESHOLD && positionInfo.totalColumns >= 2 && daysCount === 7 ? (
          /* Layout ultra-compacto para eventos de 15 minutos SOBREPOSTOS na visualização de 7 dias - Apenas horário */
          <div className="flex-1 px-0 flex items-center justify-center">
            <div className={`opacity-80 leading-none ${isCompleted ? 'line-through opacity-50' : ''}`} style={{ fontSize: '8px' }}>
              {event.startTime}
            </div>
          </div>
        ) : eventHeight < VERY_SHORT_EVENT_THRESHOLD && positionInfo.totalColumns >= 2 && daysCount === 7 ? (
          /* Layout ultra-compacto para eventos de 16-29 minutos SOBREPOSTOS na visualização de 7 dias - Apenas horário */
          <div className="flex-1 px-0 flex items-center justify-center">
            <div className={`opacity-80 leading-none ${isCompleted ? 'line-through opacity-50' : ''}`} style={{ fontSize: '8px' }}>
              {event.startTime}
            </div>
          </div>
        ) : eventHeight < PIXELS_PER_HOUR && positionInfo.totalColumns >= 2 && daysCount === 7 ? (
          /* Layout compacto para eventos de 30-60 minutos SOBREPOSTOS na visualização de 7 dias - Horário e título em duas linhas muito compactas */
          <div className="flex-1 px-0 flex flex-col items-center justify-center gap-0.5">
            <div className={`opacity-70 leading-none ${isCompleted ? 'line-through opacity-50' : ''}`} style={{ fontSize: '8px' }}>
              {event.startTime}
            </div>
            <div className={`font-semibold leading-none truncate w-full text-center px-0.5 ${isCompleted ? 'line-through opacity-70' : ''}`} style={{ fontSize: '9px' }}>
              {event.title}
            </div>
          </div>
        ) : positionInfo.totalColumns >= 2 && daysCount === 7 ? (
          /* Layout compacto para eventos de 1h+ SOBREPOSTOS na visualização de 7 dias - Apenas horário e título vertical */
          <div className="flex-1 px-0 py-1 flex flex-col items-center justify-center gap-0.5">
            <div className={`opacity-80 leading-none ${isCompleted ? 'line-through opacity-60' : ''}`} style={{ fontSize: '9px' }}>
              {event.startTime}
            </div>
            <div className={`font-semibold leading-tight truncate w-full text-center px-0.5 ${isCompleted ? 'line-through opacity-70' : ''}`} style={{ fontSize: '10px' }}>
              {event.title}
            </div>
          </div>
        ) : eventHeight <= FIFTEEN_MINUTE_THRESHOLD && daysCount === 7 ? (
          /* Layout ultra-compacto para eventos de 15 minutos na visualização de 7 dias - Horário e título em linha */
          <div className="flex-1 px-0.5 flex items-center justify-center gap-1">
            <div className={`opacity-70 leading-none flex-shrink-0 ${isCompleted ? 'line-through opacity-50' : ''}`} style={{ fontSize: '10px' }}>
              {event.startTime}{event.endTime && `-${event.endTime}`}
            </div>
            <div className={`text-xs font-semibold leading-none truncate flex-1 min-w-0 ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
          </div>
        ) : eventHeight < VERY_SHORT_EVENT_THRESHOLD && daysCount === 7 ? (
          /* Layout compacto para eventos de 16-29 minutos na visualização de 7 dias - Horário e título em linha */
          <div className="flex-1 px-0.5 flex items-center justify-center gap-1">
            <div className={`opacity-70 leading-none flex-shrink-0 ${isCompleted ? 'line-through opacity-50' : ''}`} style={{ fontSize: '10px' }}>
              {event.startTime}{event.endTime && `-${event.endTime}`}
            </div>
            <div className={`text-xs font-semibold leading-none truncate flex-1 min-w-0 ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
          </div>
        ) : eventHeight < PIXELS_PER_HOUR && daysCount === 7 ? (
          /* Layout compacto para eventos de 30-60 minutos na visualização de 7 dias - Horário e título em duas linhas */
          <div className="flex-1 px-0.5 flex flex-col items-center justify-center gap-1.5">
            <div className={`opacity-70 leading-none ${isCompleted ? 'line-through opacity-50' : ''}`} style={{ fontSize: '10px' }}>
              {event.startTime}{event.endTime && `-${event.endTime}`}
            </div>
            <div className={`text-xs font-semibold leading-none truncate w-full text-center ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
          </div>
        ) : daysCount === 7 ? (
          /* Layout para eventos de 1 hora+ na visualização de 7 dias */
          <div className={`flex-1 ${padding} flex flex-col ${isCompleted ? 'relative' : ''}`}>
            {/* Category Icon - centered, prominent */}
            <div className="text-center pb-1">
              <span className="text-2xl">
                {category?.icon || '📅'}
              </span>
            </div>

            {/* Time - single line, bold */}
            <div className={`font-bold text-xs text-center leading-none whitespace-nowrap pb-2 ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.startTime}{event.endTime && ` - ${event.endTime}`}
            </div>

            {/* Subtle divider */}
            <div className="w-full h-px bg-white/20 mb-2"></div>

            {/* Event Title - more space */}
            <div className={`text-sm leading-relaxed ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>

            {isCompleted && (
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div className="bg-green-500 rounded-full p-2 shadow-lg animate-pulse">
                  <svg className="w-6 h-6" fill="white" viewBox="0 0 24 24">
                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                  </svg>
                </div>
              </div>
            )}
          </div>
        ) : (
          /* Fallback layout */
          <div className={`flex-1 ${padding} flex flex-col ${isCompleted ? 'relative' : ''}`}>
            <div className="text-center pb-1">
              <span className="text-2xl">
                {category?.icon || '📅'}
              </span>
            </div>
            <div className={`font-bold text-xs text-center leading-none whitespace-nowrap pb-2 ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.startTime}{event.endTime && ` - ${event.endTime}`}
            </div>
            <div className="w-full h-px bg-white/20 mb-2"></div>
            <div className={`text-sm leading-relaxed ${isCompleted ? 'line-through opacity-70' : ''}`}>
              {event.title}
            </div>
          </div>
        )}

        {/* Action buttons - horizontal for events < 1 hour, vertical for 1 hour+ */}
        <div className={`${eventHeight < PIXELS_PER_HOUR ? 'flex-row gap-0.5 pr-0.5 pt-0.5' : 'flex-col gap-1 pr-1 pt-1'} flex`}>
          <button
            onClick={onToggleExecution}
            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-green-600 rounded"
            title={isCompleted ? "Marcar como não realizado" : "Marcar como realizado"}
          >
            {isCompleted ? (
              <svg className={eventHeight < PIXELS_PER_HOUR ? "w-3 h-3" : daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="currentColor" viewBox="0 0 24 24">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
              </svg>
            ) : (
              <svg className={eventHeight < PIXELS_PER_HOUR ? "w-3 h-3" : daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <rect x="3" y="3" width="18" height="18" rx="2" strokeWidth={2} />
              </svg>
            )}
          </button>
          {onEditClick && (
            <button
              onClick={onEditClick}
              className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-blue-600 rounded"
              title="Editar"
            >
              <svg className={eventHeight < PIXELS_PER_HOUR ? "w-3 h-3" : daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
          )}
          <button
            onClick={onDeleteClick}
            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-600 rounded"
            title="Deletar"
          >
            <svg className={eventHeight < PIXELS_PER_HOUR ? "w-3 h-3" : daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      {/* Bottom resize handle - only for events >= 30 minutes */}
      {eventHeight >= VERY_SHORT_EVENT_THRESHOLD && (
        <div
          className="absolute bottom-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
          onMouseDown={(e) => onResizeStart('bottom', e, eventHeight, topPosition)}
          title="Arrastar para ajustar fim"
        >
          <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-b-lg mt-2" />
        </div>
      )}
    </div>
  );
}
